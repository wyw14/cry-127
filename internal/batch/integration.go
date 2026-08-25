package batch

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/heater"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Progressor struct {
	batches *Coordinator
	journal *journal.Store
}

func NewProgressor(coordinator *Coordinator, eventStore *journal.Store) (*Progressor, error) {
	if coordinator == nil || eventStore == nil {
		return nil, errors.New("batch progress dependencies are required")
	}
	return &Progressor{batches: coordinator, journal: eventStore}, nil
}

func (p *Progressor) ApplySoak(operationID string, evaluation heater.Evaluation, now time.Time) (model.Operation, error) {
	if !heater.Eligible(evaluation) {
		return model.Operation{}, errors.New("soak evidence is not complete")
	}
	operation, err := p.batches.Advance(operationID, model.StageQuenching)
	if err != nil {
		return model.Operation{}, err
	}
	if _, err := p.journal.Append(operationID, "soak.completed", evaluation, now); err != nil {
		return model.Operation{}, err
	}
	return operation, nil
}

func (p *Progressor) ApplyVacuum(operationID string, proof model.VacuumProof) (model.Operation, error) {
	if !proof.Durable || proof.OperationID != operationID {
		return model.Operation{}, errors.New("durable vacuum proof is required")
	}
	return p.batches.Advance(operationID, model.StageHeating)
}

func (p *Progressor) CompleteCooling(operationID string) (model.Operation, error) {
	return p.batches.Advance(operationID, model.StageComplete)
}
