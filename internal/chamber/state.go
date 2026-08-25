package chamber

import (
	"errors"
	"sync"
	"time"
)

type State struct {
	OperationID   string    `json:"operation_id"`
	SealRevision  string    `json:"seal_revision"`
	PressurePa    float64   `json:"pressure_pa"`
	ActiveGasLine string    `json:"active_gas_line"`
	Quenching     bool      `json:"quenching"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Registry struct {
	mu     sync.RWMutex
	states map[string]State
}

func NewRegistry() *Registry {
	return &Registry{states: map[string]State{}}
}

func (r *Registry) Put(state State) error {
	if state.OperationID == "" || state.SealRevision == "" {
		return errors.New("chamber state identity is incomplete")
	}
	r.mu.Lock()
	r.states[state.OperationID] = state
	r.mu.Unlock()
	return nil
}

func (r *Registry) Get(operationID string) (State, bool) {
	r.mu.RLock()
	state, ok := r.states[operationID]
	r.mu.RUnlock()
	return state, ok
}

func (r *Registry) Update(operationID string, apply func(State) (State, error)) (State, error) {
	if apply == nil {
		return State{}, errors.New("chamber update function is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.states[operationID]
	if !ok {
		return State{}, errors.New("chamber state not found")
	}
	updated, err := apply(state)
	if err != nil {
		return State{}, err
	}
	r.states[operationID] = updated
	return updated, nil
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
