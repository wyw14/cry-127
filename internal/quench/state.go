package quench

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Circulation struct {
	OperationID string    `json:"operation_id"`
	Speed       int       `json:"speed"`
	Running     bool      `json:"running"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorkerBook struct {
	mu      sync.RWMutex
	states  map[string]Circulation
	cancels map[string]context.CancelFunc
}

func NewWorkerBook() *WorkerBook {
	return &WorkerBook{states: map[string]Circulation{}, cancels: map[string]context.CancelFunc{}}
}

func (b *WorkerBook) Start(parent context.Context, operationID string, now func() time.Time) error {
	if parent == nil || operationID == "" || now == nil {
		return errors.New("circulation worker input is incomplete")
	}
	ctx, cancel := context.WithCancel(parent)
	b.mu.Lock()
	if existing := b.cancels[operationID]; existing != nil {
		existing()
	}
	b.cancels[operationID] = cancel
	b.states[operationID] = Circulation{OperationID: operationID, Running: true, UpdatedAt: now().UTC()}
	b.mu.Unlock()
	go b.run(ctx, operationID, now)
	return nil
}

func (b *WorkerBook) run(ctx context.Context, operationID string, now func() time.Time) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			state := b.states[operationID]
			state.Running = false
			state.Speed = 0
			state.UpdatedAt = now().UTC()
			b.states[operationID] = state
			delete(b.cancels, operationID)
			b.mu.Unlock()
			return
		case <-ticker.C:
			b.mu.Lock()
			state := b.states[operationID]
			if state.Speed < 100 {
				state.Speed += 10
			}
			state.UpdatedAt = now().UTC()
			b.states[operationID] = state
			b.mu.Unlock()
		}
	}
}

func (b *WorkerBook) Stop(operationID string) bool {
	b.mu.RLock()
	cancel := b.cancels[operationID]
	b.mu.RUnlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (b *WorkerBook) State(operationID string) (Circulation, bool) {
	b.mu.RLock()
	state, ok := b.states[operationID]
	b.mu.RUnlock()
	return state, ok
}
