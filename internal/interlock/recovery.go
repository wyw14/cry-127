package interlock

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-127/internal/journal"
)

type Recovery struct {
	mu         sync.RWMutex
	generation uint64
	restored   bool
}

func NewRecovery() *Recovery {
	return &Recovery{}
}

func (r *Recovery) Begin() uint64 {
	r.mu.Lock()
	r.generation++
	r.restored = false
	generation := r.generation
	r.mu.Unlock()
	return generation
}

func (r *Recovery) Complete(generation uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if generation != r.generation {
		return errors.New("interlock recovery generation is stale")
	}
	r.restored = true
	return nil
}

func (r *Recovery) Ready(generation uint64) bool {
	r.mu.RLock()
	ready := r.restored && r.generation == generation
	r.mu.RUnlock()
	return ready
}

func SaveRecovery(snapshotStore *journal.SnapshotStore, recovery *Recovery) error {
	if snapshotStore == nil || recovery == nil {
		return errors.New("interlock recovery dependencies are required")
	}
	recovery.mu.RLock()
	state := struct {
		Generation uint64 `json:"generation"`
		Restored   bool   `json:"restored"`
	}{Generation: recovery.generation, Restored: recovery.restored}
	recovery.mu.RUnlock()
	return snapshotStore.Save("interlock", state)
}
