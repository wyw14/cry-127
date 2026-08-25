package atmosphere

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/journal"
)

type Coordinator struct {
	states   *StateStore
	valves   *ValveBank
	chambers *chamber.Controller
	journal  *journal.Store
}

func NewCoordinator(states *StateStore, valves *ValveBank, chambers *chamber.Controller, eventStore *journal.Store) (*Coordinator, error) {
	if states == nil || valves == nil || chambers == nil || eventStore == nil {
		return nil, errors.New("atmosphere coordinator dependencies are required")
	}
	return &Coordinator{states: states, valves: valves, chambers: chambers, journal: eventStore}, nil
}

func (c *Coordinator) Change(operationID, nextLine string, purge func() error, now time.Time) (LineState, error) {
	state, ok := c.states.Current(operationID)
	if !ok {
		var err error
		state, err = c.states.Begin(operationID, now)
		if err != nil {
			return LineState{}, err
		}
	}
	prior := state.ActiveLine
	if prior != "" {
		if err := c.valves.Close(prior); err != nil {
			return LineState{}, err
		}
		state.IsolatedLines[prior] = true
	}
	if purge == nil {
		return LineState{}, errors.New("purge step is required")
	}
	if err := purge(); err != nil {
		return LineState{}, err
	}
	state.PurgeComplete = true
	if err := c.valves.Open(nextLine); err != nil {
		return LineState{}, err
	}
	state.ActiveLine = nextLine
	state.Revision = uuid.NewString()
	state.UpdatedAt = now.UTC()
	if err := c.states.Put(state); err != nil {
		return LineState{}, err
	}
	if _, err := c.chambers.SetGas(operationID, nextLine, now); err != nil {
		return LineState{}, err
	}
	if _, err := c.journal.Append(operationID, "atmosphere.changed", state, now); err != nil {
		return LineState{}, err
	}
	return state, nil
}

func (c *Coordinator) IsolateBackfill(operationID string, now time.Time) (LineState, error) {
	state, ok := c.states.Current(operationID)
	if !ok {
		return LineState{}, errors.New("atmosphere state not found")
	}
	if state.ActiveLine != "" {
		if err := c.valves.Close(state.ActiveLine); err != nil {
			return LineState{}, err
		}
		state.IsolatedLines[state.ActiveLine] = true
	}
	state.ActiveLine = ""
	state.UpdatedAt = now.UTC()
	if err := c.states.Put(state); err != nil {
		return LineState{}, err
	}
	return state, nil
}
