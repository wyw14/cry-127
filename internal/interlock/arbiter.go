package interlock

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/journal"
)

type Decision struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operation_id"`
	Action      string    `json:"action"`
	Allowed     bool      `json:"allowed"`
	Reasons     []string  `json:"reasons"`
	DecidedAt   time.Time `json:"decided_at"`
}

type Arbiter struct {
	mu        sync.RWMutex
	journal   *journal.Store
	decisions map[string]Decision
}

func NewArbiter(eventStore *journal.Store) (*Arbiter, error) {
	if eventStore == nil {
		return nil, errors.New("interlock journal is required")
	}
	return &Arbiter{journal: eventStore, decisions: map[string]Decision{}}, nil
}

func (a *Arbiter) Decide(operationID, action string, checks map[string]bool, now time.Time) (Decision, error) {
	if operationID == "" || action == "" || len(checks) == 0 {
		return Decision{}, errors.New("interlock decision input is incomplete")
	}
	reasons := []string{}
	for reason, passed := range checks {
		if !passed {
			reasons = append(reasons, reason)
		}
	}
	sort.Strings(reasons)
	decision := Decision{ID: uuid.NewString(), OperationID: operationID, Action: action, Allowed: len(reasons) == 0, Reasons: reasons, DecidedAt: now.UTC()}
	if _, err := a.journal.Append(operationID, "interlock.decision", decision, now); err != nil {
		return Decision{}, err
	}
	a.mu.Lock()
	a.decisions[operationID+"/"+action] = decision
	a.mu.Unlock()
	return decision, nil
}

func (a *Arbiter) Latest(operationID, action string) (Decision, bool) {
	a.mu.RLock()
	decision, ok := a.decisions[operationID+"/"+action]
	a.mu.RUnlock()
	return decision, ok
}

func (a *Arbiter) All() []Decision {
	a.mu.RLock()
	result := make([]Decision, 0, len(a.decisions))
	for _, decision := range a.decisions {
		result = append(result, decision)
	}
	a.mu.RUnlock()
	return result
}
