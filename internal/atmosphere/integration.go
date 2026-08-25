package atmosphere

import (
	"errors"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Recovery struct {
	States []LineState `json:"states"`
}

func Save(snapshotStore *journal.SnapshotStore, states *StateStore) error {
	if snapshotStore == nil || states == nil {
		return errors.New("atmosphere recovery dependencies are required")
	}
	return snapshotStore.Save("atmosphere", Recovery{States: states.All()})
}

func Restore(snapshotStore *journal.SnapshotStore, states *StateStore) (bool, error) {
	if snapshotStore == nil || states == nil {
		return false, errors.New("atmosphere recovery dependencies are required")
	}
	var recovery Recovery
	found, err := snapshotStore.Load("atmosphere", &recovery)
	if err != nil || !found {
		return found, err
	}
	for _, state := range recovery.States {
		if err := states.Put(state); err != nil {
			return false, err
		}
	}
	return true, nil
}

func Permit(proof model.GasProof, vacuumProof model.VacuumProof, sealRevision string) error {
	if proof.OperationID == "" || proof.OperationID != vacuumProof.OperationID {
		return errors.New("gas and vacuum proof operation mismatch")
	}
	if vacuumProof.SealRevision != sealRevision || !vacuumProof.Durable {
		return errors.New("current seal vacuum proof is required")
	}
	if !proof.PriorIsolated || !proof.PurgeComplete || proof.ActiveLine == "" {
		return errors.New("gas path is not ready")
	}
	return nil
}
