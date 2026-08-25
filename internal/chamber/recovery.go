package chamber

import (
	"errors"

	"github.com/wyw14/cry-127/internal/journal"
)

type Recovery struct {
	States []State `json:"states"`
}

func Save(snapshotStore *journal.SnapshotStore, registry *Registry) error {
	if snapshotStore == nil || registry == nil {
		return errors.New("chamber recovery dependencies are required")
	}
	return snapshotStore.Save("chambers", Recovery{States: registry.All()})
}

func Restore(snapshotStore *journal.SnapshotStore, registry *Registry) (bool, error) {
	if snapshotStore == nil || registry == nil {
		return false, errors.New("chamber recovery dependencies are required")
	}
	var recovery Recovery
	found, err := snapshotStore.Load("chambers", &recovery)
	if err != nil || !found {
		return found, err
	}
	for _, state := range recovery.States {
		if err := registry.Put(state); err != nil {
			return false, err
		}
	}
	return true, nil
}
