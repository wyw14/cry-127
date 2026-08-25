package batch

import (
	"errors"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type RecoverySnapshot struct {
	Operations []model.Operation `json:"operations"`
}

func Save(snapshotStore *journal.SnapshotStore, store *Store) error {
	if snapshotStore == nil || store == nil {
		return errors.New("batch recovery dependencies are required")
	}
	return snapshotStore.Save("operations", RecoverySnapshot{Operations: store.List()})
}

func Restore(snapshotStore *journal.SnapshotStore) (*Store, bool, error) {
	if snapshotStore == nil {
		return nil, false, errors.New("snapshot store is required")
	}
	var snapshot RecoverySnapshot
	found, err := snapshotStore.Load("operations", &snapshot)
	if err != nil {
		return nil, false, err
	}
	store := NewStore()
	if !found {
		return store, false, nil
	}
	for _, operation := range snapshot.Operations {
		if err := store.Put(operation); err != nil {
			return nil, false, err
		}
	}
	return store, true, nil
}

func Reconcile(eventStore *journal.Store, snapshotStore *journal.SnapshotStore) (*Store, error) {
	store, _, err := Restore(snapshotStore)
	if err != nil {
		return nil, err
	}
	events, err := eventStore.Events()
	if err != nil {
		return nil, err
	}
	replay, err := journal.Rebuild(events)
	if err != nil {
		return nil, err
	}
	for _, operation := range replay.Operations {
		if err := store.Put(operation); err != nil {
			return nil, err
		}
	}
	return store, nil
}
