package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/heater"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/quench"
	"github.com/wyw14/cry-127/internal/recipe"
	"github.com/wyw14/cry-127/internal/vacuum"
)

type EquipmentView struct {
	Operation   model.Operation    `json:"operation"`
	Chamber     any                `json:"chamber,omitempty"`
	Door        any                `json:"door,omitempty"`
	Vacuum      any                `json:"vacuum,omitempty"`
	Gas         any                `json:"gas,omitempty"`
	Heater      any                `json:"heater,omitempty"`
	HeaterPower bool               `json:"heater_power"`
	Quench      quench.Circulation `json:"quench,omitempty"`
}

func (r *Runtime) StartOperation(furnaceID, revisionID string) (model.Operation, error) {
	if revisionID == "" {
		revisionID = r.defaultRecipe
	}
	operation, err := r.batches.Start(furnaceID, revisionID)
	if err != nil {
		return model.Operation{}, err
	}
	doorState, err := r.doors.Seal(operation.ID, r.now())
	if err != nil {
		return model.Operation{}, err
	}
	operation, err = r.batches.ReplaceSeal(operation.ID, doorState.SealRevision)
	if err != nil {
		return model.Operation{}, err
	}
	if _, err := r.chambers.Register(operation, r.now()); err != nil {
		return model.Operation{}, err
	}
	if _, err := r.atmosphereStates.Begin(operation.ID, r.now()); err != nil {
		return model.Operation{}, err
	}
	if _, err := r.vacuum.Begin(operation, r.now()); err != nil {
		return model.Operation{}, err
	}
	return r.batches.Advance(operation.ID, model.StageEvacuating)
}

func (r *Runtime) QualifyVacuum(operationID string, pressure, leakRate float64) (model.Operation, error) {
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return model.Operation{}, errors.New("operation not found")
	}
	now := r.now()
	points := []vacuum.TracePoint{
		{At: now.Add(-time.Second), PressurePa: pressure * 1.05, LeakRate: leakRate * 1.1},
		{At: now, PressurePa: pressure, LeakRate: leakRate},
	}
	proof, err := r.vacuum.Qualify(operation, points, 10, 0.01, now)
	if err != nil {
		return model.Operation{}, err
	}
	updated, err := r.progress.ApplyVacuum(operationID, proof)
	if err == nil {
		r.heater.SetPower(operationID, true)
	}
	return updated, err
}

func (r *Runtime) RecordTemperatures(operationID string, values map[string]float64, at time.Time) (heater.Evaluation, model.Operation, error) {
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return heater.Evaluation{}, model.Operation{}, errors.New("operation not found")
	}
	revision, ok := r.recipes.Get(operation.RecipeRevision)
	if !ok {
		return heater.Evaluation{}, model.Operation{}, errors.New("recipe revision not found")
	}
	readiness, err := recipe.NewReadiness(operationID, revision)
	if err != nil {
		return heater.Evaluation{}, model.Operation{}, err
	}
	zones := []string{"zone-a", "zone-b", "zone-c"}
	probes := []string{"load-a", "load-b", "load-c"}
	for index, probeID := range probes {
		value, exists := values[probeID]
		if !exists {
			value = 0
		}
		sample, receiveErr := r.receiver.Receive(operationID, probeID, value, true, exists, at)
		if receiveErr != nil {
			return heater.Evaluation{}, model.Operation{}, receiveErr
		}
		if err := r.evidence.Record(revision, sample); err != nil {
			return heater.Evaluation{}, model.Operation{}, err
		}
		readiness, err = readiness.Accept(revision, zones[index], sample)
		if err != nil {
			return heater.Evaluation{}, model.Operation{}, err
		}
		if _, err := r.heater.Acknowledge(operation, revision, zones[index], sample.Online && sample.TemperatureC >= revision.ZoneTargets[zones[index]]-revision.ToleranceC, at); err != nil {
			return heater.Evaluation{}, model.Operation{}, err
		}
	}
	if operation.Stage == model.StageHeating && readiness.Complete() && r.heater.Ready(operationID, revision.ID, zones) {
		operation, err = r.batches.Advance(operationID, model.StageSoaking)
		if err != nil {
			return heater.Evaluation{}, model.Operation{}, err
		}
	}
	evaluation, err := r.heaterIntegration.EvaluateOperation(operation, revision, at)
	if err != nil {
		return evaluation, operation, err
	}
	if operation.Stage == model.StageSoaking && heater.Eligible(evaluation) {
		operation, err = r.progress.ApplySoak(operationID, evaluation, at)
	}
	return evaluation, operation, err
}

func (r *Runtime) Recalibrate(operationID, probeID string, offset, scale float64) error {
	if _, err := r.calibrations.Calibrate(probeID, offset, scale, r.now()); err != nil {
		return err
	}
	r.heaterIntegration.InvalidateForCalibration(operationID, r.now())
	r.evidence.ResetOperation(operationID)
	return nil
}

func (r *Runtime) ChangeAtmosphere(operationID, line string) (atmosphere.LineState, error) {
	state, err := r.atmosphere.Change(operationID, line, func() error { return nil }, r.now())
	if err != nil {
		return atmosphere.LineState{}, err
	}
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return atmosphere.LineState{}, errors.New("operation not found")
	}
	gasProof, ok := r.atmosphereStates.Proof(operationID)
	if !ok {
		return atmosphere.LineState{}, errors.New("gas proof not found")
	}
	vacuumProof, ok := r.vacuum.Proof(operationID, operation.SealRevision)
	if !ok {
		return atmosphere.LineState{}, errors.New("vacuum proof not found")
	}
	decision, err := r.interlocks.AtmospherePermit(operation, gasProof, vacuumProof, r.now())
	if err != nil {
		return atmosphere.LineState{}, err
	}
	if !decision.Allowed {
		return atmosphere.LineState{}, errors.New("atmosphere permit denied")
	}
	return state, nil
}

func (r *Runtime) SetChamberPressure(operationID string, pressure float64) error {
	_, err := r.chambers.SetPressure(operationID, pressure, r.now())
	return err
}

func (r *Runtime) StartQuench(ctx context.Context, operationID string) (model.ResourceLease, interlock.Decision, error) {
	operation, ok := r.batches.Get(operationID)
	if !ok || operation.Stage != model.StageQuenching {
		return model.ResourceLease{}, interlock.Decision{}, errors.New("operation is not ready for quench")
	}
	return r.quench.Start(ctx, "quench-vessel-1", operationID)
}

func (r *Runtime) FinishQuench(lease model.ResourceLease) (model.Operation, error) {
	if err := r.quench.Release(lease); err != nil {
		return model.Operation{}, err
	}
	return r.batches.Advance(lease.OperationID, model.StageCooling)
}

func (r *Runtime) Abort(operationID, cause string) (model.Operation, error) {
	operation, err := r.batches.Abort(operationID, cause)
	if err != nil {
		return model.Operation{}, err
	}
	r.quench.Abort(operationID)
	r.heater.Stop(operationID, r.now())
	r.RecordIncident(operationID, "service", cause)
	return operation, nil
}

func (r *Runtime) ReleaseDoor(operationID string) (interlock.Decision, error) {
	_, decision, err := r.doors.Release(operationID, r.now())
	return decision, err
}

func (r *Runtime) Reseal(operationID string) (model.Operation, error) {
	if _, err := r.doors.Open(operationID, r.now()); err != nil {
		return model.Operation{}, err
	}
	state, err := r.doors.Seal(operationID, r.now())
	if err != nil {
		return model.Operation{}, err
	}
	r.vacuumProofs.Invalidate(operationID)
	return r.batches.ReplaceSeal(operationID, state.SealRevision)
}

func (r *Runtime) CompleteCooling(operationID string, diffusionErr, roughingErr error) (model.Operation, error) {
	err := r.vacuum.Shutdown(operationID, func() error { return diffusionErr }, func() error { return roughingErr }, func() error { return nil })
	if err != nil {
		r.RecordIncident(operationID, "vacuum", err.Error())
		return model.Operation{}, err
	}
	return r.progress.CompleteCooling(operationID)
}

func (r *Runtime) RecordIncident(operationID, component, message string) model.Incident {
	incident := model.Incident{ID: uuid.NewString(), OperationID: operationID, Component: component, Message: message, OccurredAt: r.now().UTC()}
	r.mu.Lock()
	r.incidents = append(r.incidents, incident)
	r.mu.Unlock()
	r.events.Append(operationID, "incident.recorded", incident, r.now())
	return incident
}

func (r *Runtime) Operation(id string) (model.Operation, bool) {
	return r.batches.Get(id)
}

func (r *Runtime) Operations() []model.Operation {
	return r.batches.List()
}

func (r *Runtime) Decisions() []interlock.Decision {
	return r.interlockArbiter.All()
}

func (r *Runtime) Incidents() []model.Incident {
	r.mu.RLock()
	result := append([]model.Incident(nil), r.incidents...)
	r.mu.RUnlock()
	return result
}

func (r *Runtime) Equipment(operationID string) (EquipmentView, error) {
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return EquipmentView{}, errors.New("operation not found")
	}
	view := EquipmentView{Operation: operation}
	view.Chamber, _ = r.chambers.Current(operationID)
	view.Door, _ = r.doorRegistry.Current(operationID)
	view.Vacuum, _ = r.vacuumController.Current(operationID)
	view.Gas, _ = r.atmosphereStates.Current(operationID)
	if gas, ok := view.Gas.(atmosphere.LineState); ok {
		gas.IsolatedLines["open-lines"] = len(r.valves.OpenLines()) == 0
		view.Gas = gas
	}
	view.Heater, _ = r.windows.Current(operationID)
	view.HeaterPower = r.heater.Powered(operationID)
	view.Quench, _ = r.workers.State(operationID)
	return view, nil
}

func (r *Runtime) DefaultRecipe() string {
	return r.defaultRecipe
}

func (r *Runtime) Recipe(id string) (recipe.Revision, error) {
	revision, ok := r.recipes.Get(id)
	if !ok {
		return recipe.Revision{}, fmt.Errorf("recipe %s not found", id)
	}
	return revision, nil
}
