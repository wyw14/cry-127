package door

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type State struct {
	OperationID  string    `json:"operation_id"`
	SealRevision string    `json:"seal_revision"`
	Locked       bool      `json:"locked"`
	Open         bool      `json:"open"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Registry struct {
	mu     sync.RWMutex
	states map[string]State
}

func NewRegistry() *Registry {
	return &Registry{states: map[string]State{}}
}

func (r *Registry) Seal(operationID string, now time.Time) (State, error) {
	if operationID == "" {
		return State{}, errors.New("operation identity is required")
	}
	state := State{OperationID: operationID, SealRevision: uuid.NewString(), Locked: true, Open: false, UpdatedAt: now.UTC()}
	r.mu.Lock()
	r.states[operationID] = state
	r.mu.Unlock()
	return state, nil
}

func (r *Registry) Open(operationID string, now time.Time) (State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.states[operationID]
	if !ok {
		return State{}, errors.New("door state not found")
	}
	if state.Locked {
		return State{}, errors.New("door is locked")
	}
	state.Open = true
	state.UpdatedAt = now.UTC()
	r.states[operationID] = state
	return state, nil
}

func (r *Registry) Put(state State) error {
	if state.OperationID == "" || state.SealRevision == "" {
		return errors.New("door state identity is incomplete")
	}
	r.mu.Lock()
	r.states[state.OperationID] = state
	r.mu.Unlock()
	return nil
}

func (r *Registry) Current(operationID string) (State, bool) {
	r.mu.RLock()
	state, ok := r.states[operationID]
	r.mu.RUnlock()
	return state, ok
}

func (r *Registry) All() []State {
	r.mu.RLock()
	result := make([]State, 0, len(r.states))
	for _, state := range r.states {
		result = append(result, state)
	}
	r.mu.RUnlock()
	return result
}
