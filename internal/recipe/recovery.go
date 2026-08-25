package recipe

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wyw14/cry-127/internal/journal"
)

type Snapshot struct {
	Revisions []Revision `json:"revisions"`
}

func Save(snapshotStore *journal.SnapshotStore, registry *Registry) error {
	if snapshotStore == nil || registry == nil {
		return errors.New("recipe recovery dependencies are required")
	}
	return snapshotStore.Save("recipes", Snapshot{Revisions: registry.All()})
}

func Restore(snapshotStore *journal.SnapshotStore) (*Registry, error) {
	if snapshotStore == nil {
		return nil, errors.New("snapshot store is required")
	}
	var snapshot Snapshot
	found, err := snapshotStore.Load("recipes", &snapshot)
	if err != nil {
		return nil, err
	}
	registry := NewRegistry()
	if !found {
		return registry, nil
	}
	for _, revision := range snapshot.Revisions {
		if revision.ID == "" || revision.Name == "" {
			return nil, errors.New("restored recipe is incomplete")
		}
		registry.revisions[revision.ID] = cloneRevision(revision)
	}
	return registry, nil
}

func Encode(revision Revision) ([]byte, error) {
	encoded, err := json.Marshal(revision)
	if err != nil {
		return nil, fmt.Errorf("encode recipe revision: %w", err)
	}
	return encoded, nil
}
