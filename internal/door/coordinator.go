package door

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Coordinator struct {
	registry   *Registry
	atmosphere *atmosphere.StateStore
	chambers   *chamber.Controller
	interlocks *interlock.Integration
	journal    *journal.Store
}

func NewCoordinator(registry *Registry, gases *atmosphere.StateStore, chambers *chamber.Controller, interlocks *interlock.Integration, eventStore *journal.Store) (*Coordinator, error) {
	if registry == nil || gases == nil || chambers == nil || interlocks == nil || eventStore == nil {
		return nil, errors.New("door coordinator dependencies are required")
	}
	return &Coordinator{registry: registry, atmosphere: gases, chambers: chambers, interlocks: interlocks, journal: eventStore}, nil
}

func (c *Coordinator) Seal(operationID string, now time.Time) (State, error) {
	state, err := c.registry.Seal(operationID, now)
	if err != nil {
		return State{}, err
	}
	if _, err := c.journal.Append(operationID, "door.sealed", state, now); err != nil {
		return State{}, err
	}
	return state, nil
}

func (c *Coordinator) Release(operationID string, now time.Time) (State, interlock.Decision, error) {
	state, ok := c.registry.Current(operationID)
	if !ok {
		return State{}, interlock.Decision{}, errors.New("door state not found")
	}
	chamberState, ok := c.chambers.Current(operationID)
	if !ok {
		return State{}, interlock.Decision{}, errors.New("chamber state not found")
	}
	gasProof, ok := c.atmosphere.Proof(operationID)
	if !ok {
		return State{}, interlock.Decision{}, errors.New("gas isolation proof not found")
	}
	gasProof.BackfillClosing = false
	decision, err := c.interlocks.DoorRelease(operationID, chamberState, gasProof, now)
	if err != nil {
		return State{}, interlock.Decision{}, err
	}
	if !decision.Allowed {
		return state, decision, errors.New("door release denied")
	}
	state.Locked = false
	state.UpdatedAt = now.UTC()
	if err := c.registry.Put(state); err != nil {
		return State{}, interlock.Decision{}, err
	}
	proof := model.DoorProof{OperationID: operationID, SealRevision: state.SealRevision, Locked: state.Locked, PressurePa: chamberState.PressurePa, RecordedAt: now.UTC()}
	if _, err := c.journal.Append(operationID, "door.released", proof, now); err != nil {
		return State{}, interlock.Decision{}, err
	}
	return state, decision, nil
}

func (c *Coordinator) Open(operationID string, now time.Time) (State, error) {
	return c.registry.Open(operationID, now)
}
