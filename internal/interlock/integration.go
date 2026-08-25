package interlock

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/model"
)

type Integration struct {
	arbiter *Arbiter
}

func NewIntegration(arbiter *Arbiter) (*Integration, error) {
	if arbiter == nil {
		return nil, errors.New("interlock arbiter is required")
	}
	return &Integration{arbiter: arbiter}, nil
}

func (i *Integration) DoorRelease(operationID string, chamberState chamber.State, gasProof model.GasProof, now time.Time) (Decision, error) {
	checks := map[string]bool{
		"chamber pressure is not near atmosphere": chamberState.PressurePa >= 95000 && chamberState.PressurePa <= 105000,
		"backfill gas path is not isolated":       gasProof.ActiveLine == "" && gasProof.PriorIsolated && !gasProof.BackfillClosing,
		"gas proof belongs to another operation":  gasProof.OperationID == operationID,
	}
	return i.arbiter.Decide(operationID, "door.release", checks, now)
}

func (i *Integration) QuenchStart(operationID string, lease model.ResourceLease, gasProof model.GasProof, chamberState chamber.State, now time.Time) (Decision, error) {
	checks := map[string]bool{
		"quench lease belongs to another operation":  lease.OperationID == operationID && lease.ReleasedAt.IsZero(),
		"gas path is not isolated":                   gasProof.PriorIsolated && gasProof.PurgeComplete,
		"chamber state belongs to another operation": chamberState.OperationID == operationID,
	}
	return i.arbiter.Decide(operationID, "quench.start", checks, now)
}

func (i *Integration) AtmospherePermit(operation model.Operation, gasProof model.GasProof, vacuumProof model.VacuumProof, now time.Time) (Decision, error) {
	checks := map[string]bool{
		"vacuum proof is not current": vacuumProof.OperationID == operation.ID && vacuumProof.Durable,
		"gas path is not ready":       atmosphere.Permit(gasProof, vacuumProof, vacuumProof.SealRevision) == nil,
	}
	return i.arbiter.Decide(operation.ID, "atmosphere.permit", checks, now)
}
