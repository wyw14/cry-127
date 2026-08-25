package chamber

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Controller struct {
	registry *Registry
	journal  *journal.Store
}

func NewController(registry *Registry, eventStore *journal.Store) (*Controller, error) {
	if registry == nil || eventStore == nil {
		return nil, errors.New("chamber controller dependencies are required")
	}
	return &Controller{registry: registry, journal: eventStore}, nil
}

func (c *Controller) Register(operation model.Operation, now time.Time) (State, error) {
	state := State{OperationID: operation.ID, SealRevision: operation.SealRevision, PressurePa: 101325, UpdatedAt: now.UTC()}
	if err := c.registry.Put(state); err != nil {
		return State{}, err
	}
	if _, err := c.journal.Append(operation.ID, "chamber.registered", state, now); err != nil {
		return State{}, err
	}
	return state, nil
}

func (c *Controller) SetPressure(operationID string, pressure float64, now time.Time) (State, error) {
	if pressure < 0 {
		return State{}, errors.New("chamber pressure cannot be negative")
	}
	state, err := c.registry.Update(operationID, func(state State) (State, error) {
		state.PressurePa = pressure
		state.UpdatedAt = now.UTC()
		return state, nil
	})
	if err != nil {
		return State{}, err
	}
	if _, err := c.journal.Append(operationID, "chamber.pressure", state, now); err != nil {
		return State{}, err
	}
	return state, nil
}

func (c *Controller) SetGas(operationID, line string, now time.Time) (State, error) {
	state, err := c.registry.Update(operationID, func(state State) (State, error) {
		state.ActiveGasLine = line
		state.UpdatedAt = now.UTC()
		return state, nil
	})
	if err != nil {
		return State{}, err
	}
	if _, err := c.journal.Append(operationID, "chamber.gas", state, now); err != nil {
		return State{}, err
	}
	return state, nil
}

func (c *Controller) Current(operationID string) (State, bool) {
	return c.registry.Get(operationID)
}
