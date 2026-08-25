package heater

import (
	"errors"

	"github.com/wyw14/cry-127/internal/journal"
)

type RecoveryState struct {
	Windows []Window        `json:"windows"`
	Power   map[string]bool `json:"power"`
}

func SaveRecovery(snapshotStore *journal.SnapshotStore, book *WindowBook, coordinator *Coordinator, operationIDs []string) error {
	if snapshotStore == nil || book == nil || coordinator == nil {
		return errors.New("heater recovery dependencies are required")
	}
	state := RecoveryState{Windows: []Window{}, Power: map[string]bool{}}
	for _, operationID := range operationIDs {
		if window, ok := book.Current(operationID); ok {
			state.Windows = append(state.Windows, window)
		}
		state.Power[operationID] = coordinator.Powered(operationID)
	}
	return snapshotStore.Save("heater", state)
}

func RestoreRecovery(snapshotStore *journal.SnapshotStore, book *WindowBook, coordinator *Coordinator) (bool, error) {
	if snapshotStore == nil || book == nil || coordinator == nil {
		return false, errors.New("heater recovery dependencies are required")
	}
	var state RecoveryState
	found, err := snapshotStore.Load("heater", &state)
	if err != nil || !found {
		return found, err
	}
	book.mu.Lock()
	for _, window := range state.Windows {
		book.windows[window.Key.OperationID] = window
	}
	book.mu.Unlock()
	for operationID, enabled := range state.Power {
		coordinator.SetPower(operationID, enabled)
	}
	return true, nil
}
