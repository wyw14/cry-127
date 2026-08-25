package vacuum

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type RecoveryState struct {
	Topologies []Topology          `json:"topologies"`
	Proofs     []model.VacuumProof `json:"proofs"`
}

func SaveRecovery(snapshotStore *journal.SnapshotStore, controller *Controller, proofs *ProofBook, operationIDs []string) error {
	if snapshotStore == nil || controller == nil || proofs == nil {
		return errors.New("vacuum recovery dependencies are required")
	}
	state := RecoveryState{Topologies: []Topology{}, Proofs: proofs.All()}
	for _, operationID := range operationIDs {
		if topology, ok := controller.Current(operationID); ok {
			state.Topologies = append(state.Topologies, topology)
		}
	}
	return snapshotStore.Save("vacuum", state)
}

func RestoreRecovery(snapshotStore *journal.SnapshotStore, controller *Controller, proofs *ProofBook) (bool, error) {
	if snapshotStore == nil || controller == nil || proofs == nil {
		return false, errors.New("vacuum recovery dependencies are required")
	}
	var state RecoveryState
	found, err := snapshotStore.Load("vacuum", &state)
	if err != nil || !found {
		return found, err
	}
	for _, topology := range state.Topologies {
		if err := controller.Configure(topology); err != nil {
			return false, err
		}
	}
	for _, proof := range state.Proofs {
		if err := proofs.Record(proof); err != nil {
			return false, err
		}
	}
	return true, nil
}

func HeatingEligibility(controller *Controller, operation model.Operation, restoredAt time.Time) error {
	if restoredAt.Before(operation.UpdatedAt) {
		return errors.New("vacuum recovery is older than operation state")
	}
	if !controller.SafeForHeating(operation.ID, operation.SealRevision) {
		return errors.New("vacuum topology must load before heating")
	}
	return nil
}
