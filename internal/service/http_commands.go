package service

import (
	"errors"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/vacuum"
)

type RecipeRetry struct {
	Operation model.Operation `json:"operation"`
	ZoneReady bool            `json:"zone_ready"`
}

func (r *Runtime) IsolateAtmosphere(operationID string) (atmosphere.LineState, error) {
	return r.atmosphere.IsolateBackfill(operationID, r.now())
}

func (r *Runtime) ChangeAtmosphereChecked(operationID, line string) (atmosphere.LineState, error) {
	state, err := r.atmosphere.Change(operationID, line, func() error {
		if len(r.valves.OpenLines()) != 0 {
			return errors.New("gas path remained connected during purge")
		}
		return nil
	}, r.now())
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

func (r *Runtime) RetryRecipe(operationID string) (RecipeRetry, error) {
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return RecipeRetry{}, errors.New("operation not found")
	}
	current, ok := r.recipes.Get(operation.RecipeRevision)
	if !ok {
		return RecipeRetry{}, errors.New("recipe revision not found")
	}
	probes := make([]string, 0, len(current.RequiredProbes))
	for probeID := range current.RequiredProbes {
		probes = append(probes, probeID)
	}
	revision, err := r.recipes.Create(current.Name+"-retry", current.TargetC, current.ToleranceC, current.SoakDuration, probes, r.now())
	if err != nil {
		return RecipeRetry{}, err
	}
	updated, err := r.batchStore.Update(operationID, func(value model.Operation) (model.Operation, error) {
		value.RecipeRevision = revision.ID
		value.UpdatedAt = r.now().UTC()
		return value, nil
	})
	if err != nil {
		return RecipeRetry{}, err
	}
	if _, err := r.events.Append(operationID, "operation.updated", updated, r.now()); err != nil {
		return RecipeRetry{}, err
	}
	ready := r.heater.Ready(operationID, revision.ID, []string{"zone-a", "zone-b", "zone-c"})
	return RecipeRetry{Operation: updated, ZoneReady: ready}, nil
}

func (r *Runtime) ConfigureVacuumVent(operationID string, open bool) error {
	operation, ok := r.batches.Get(operationID)
	if !ok {
		return errors.New("operation not found")
	}
	return r.vacuumController.Configure(vacuum.Topology{
		OperationID:  operationID,
		SealRevision: operation.SealRevision,
		RoughingOpen: !open,
		VentOpen:     open,
		UpdatedAt:    r.now().UTC(),
	})
}

func (r *Runtime) ReloadPersistentState() error {
	if err := r.Snapshot(); err != nil {
		return err
	}
	return r.Recover()
}

func RuntimeError(message string) error {
	if message == "" {
		return nil
	}
	return errors.New(message)
}
