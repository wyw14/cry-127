package vacuum

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Coordinator struct {
	controller *Controller
	proofs     *ProofBook
	traces     TraceWriter
	journal    *journal.Store
}

func NewCoordinator(controller *Controller, proofs *ProofBook, traces TraceWriter, eventStore *journal.Store) (*Coordinator, error) {
	if controller == nil || proofs == nil || traces == nil || eventStore == nil {
		return nil, errors.New("vacuum coordinator dependencies are required")
	}
	return &Coordinator{controller: controller, proofs: proofs, traces: traces, journal: eventStore}, nil
}

func (c *Coordinator) Begin(operation model.Operation, now time.Time) (Topology, error) {
	topology := Topology{
		OperationID:  operation.ID,
		SealRevision: operation.SealRevision,
		RoughingOpen: true,
		UpdatedAt:    now.UTC(),
	}
	if err := c.controller.Configure(topology); err != nil {
		return Topology{}, err
	}
	if _, err := c.journal.Append(operation.ID, "vacuum.topology", topology, now); err != nil {
		return Topology{}, err
	}
	return topology, nil
}

func (c *Coordinator) Qualify(operation model.Operation, points []TracePoint, maxPressure, maxLeak float64, now time.Time) (model.VacuumProof, error) {
	if len(points) == 0 {
		return model.VacuumProof{}, errors.New("leak trace is required")
	}
	last := points[len(points)-1]
	if last.PressurePa > maxPressure || last.LeakRate > maxLeak {
		return model.VacuumProof{}, fmt.Errorf("vacuum not qualified: pressure %.3f leak %.6f", last.PressurePa, last.LeakRate)
	}
	if !c.controller.SafeForHeating(operation.ID, operation.SealRevision) {
		return model.VacuumProof{}, errors.New("vacuum topology is not safe for heating")
	}
	if _, err := c.traces.Write(operation.ID, points); err != nil {
		return model.VacuumProof{}, err
	}
	proof := model.VacuumProof{
		OperationID:  operation.ID,
		SealRevision: operation.SealRevision,
		ProofID:      uuid.NewString(),
		PressurePa:   last.PressurePa,
		LeakRate:     last.LeakRate,
		Durable:      true,
		RecordedAt:   now.UTC(),
	}
	if err := c.proofs.Record(proof); err != nil {
		return model.VacuumProof{}, err
	}
	if _, err := c.journal.Append(operation.ID, "vacuum.qualified", proof, now); err != nil {
		c.proofs.Invalidate(operation.ID)
		return model.VacuumProof{}, err
	}
	return proof, nil
}

func (c *Coordinator) Shutdown(operationID string, stopDiffusion, stopRoughing, finishCooling func() error) error {
	if stopDiffusion == nil || stopRoughing == nil || finishCooling == nil {
		return errors.New("shutdown steps are required")
	}
	diffusionErr := stopDiffusion()
	roughingErr := stopRoughing()
	coolingErr := finishCooling()
	return journal.MergeErrors(diffusionErr, roughingErr, coolingErr)
}

func (c *Coordinator) Proof(operationID, sealRevision string) (model.VacuumProof, bool) {
	return c.proofs.Current(operationID, sealRevision)
}
