package atmosphere

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/model"
)

type LineState struct {
	OperationID   string          `json:"operation_id"`
	Revision      string          `json:"revision"`
	ActiveLine    string          `json:"active_line"`
	IsolatedLines map[string]bool `json:"isolated_lines"`
	PurgeComplete bool            `json:"purge_complete"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type StateStore struct {
	mu     sync.RWMutex
	states map[string]LineState
}

func NewStateStore() *StateStore {
	return &StateStore{states: map[string]LineState{}}
}

func (s *StateStore) Begin(operationID string, now time.Time) (LineState, error) {
	if operationID == "" {
		return LineState{}, errors.New("operation identity is required")
	}
	state := LineState{OperationID: operationID, Revision: uuid.NewString(), IsolatedLines: map[string]bool{}, UpdatedAt: now.UTC()}
	s.mu.Lock()
	s.states[operationID] = state
	s.mu.Unlock()
	return cloneState(state), nil
}

func (s *StateStore) Current(operationID string) (LineState, bool) {
	s.mu.RLock()
	state, ok := s.states[operationID]
	s.mu.RUnlock()
	return cloneState(state), ok
}

func (s *StateStore) Put(state LineState) error {
	if state.OperationID == "" || state.Revision == "" {
		return errors.New("atmosphere state identity is incomplete")
	}
	s.mu.Lock()
	s.states[state.OperationID] = cloneState(state)
	s.mu.Unlock()
	return nil
}

func (s *StateStore) Proof(operationID string) (model.GasProof, bool) {
	state, ok := s.Current(operationID)
	if !ok {
		return model.GasProof{}, false
	}
	priorIsolated := true
	for _, isolated := range state.IsolatedLines {
		priorIsolated = priorIsolated && isolated
	}
	return model.GasProof{OperationID: operationID, Revision: state.Revision, ActiveLine: state.ActiveLine, PriorIsolated: priorIsolated, PurgeComplete: state.PurgeComplete, RecordedAt: state.UpdatedAt}, true
}

func (s *StateStore) All() []LineState {
	s.mu.RLock()
	result := make([]LineState, 0, len(s.states))
	for _, state := range s.states {
		result = append(result, cloneState(state))
	}
	s.mu.RUnlock()
	return result
}

func cloneState(state LineState) LineState {
	isolated := make(map[string]bool, len(state.IsolatedLines))
	for line, value := range state.IsolatedLines {
		isolated[line] = value
	}
	state.IsolatedLines = isolated
	return state
}
