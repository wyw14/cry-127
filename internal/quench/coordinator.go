package quench

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Coordinator struct {
	leases     *LeaseManager
	workers    *WorkerBook
	atmosphere *atmosphere.StateStore
	chambers   *chamber.Controller
	interlocks *interlock.Integration
	journal    *journal.Store
	now        func() time.Time
}

func NewCoordinator(leases *LeaseManager, workers *WorkerBook, gases *atmosphere.StateStore, chambers *chamber.Controller, interlocks *interlock.Integration, eventStore *journal.Store, now func() time.Time) (*Coordinator, error) {
	if leases == nil || workers == nil || gases == nil || chambers == nil || interlocks == nil || eventStore == nil || now == nil {
		return nil, errors.New("quench coordinator dependencies are required")
	}
	return &Coordinator{leases: leases, workers: workers, atmosphere: gases, chambers: chambers, interlocks: interlocks, journal: eventStore, now: now}, nil
}

func (c *Coordinator) Start(ctx context.Context, resourceID, operationID string) (model.ResourceLease, interlock.Decision, error) {
	lease, err := c.leases.Acquire(resourceID, operationID, c.now())
	if err != nil {
		return model.ResourceLease{}, interlock.Decision{}, err
	}
	gasProof, ok := c.atmosphere.Proof(operationID)
	if !ok {
		c.leases.Release(lease, c.now())
		return model.ResourceLease{}, interlock.Decision{}, errors.New("gas proof not found")
	}
	chamberState, ok := c.chambers.Current(operationID)
	if !ok {
		c.leases.Release(lease, c.now())
		return model.ResourceLease{}, interlock.Decision{}, errors.New("chamber state not found")
	}
	decision, err := c.interlocks.QuenchStart(operationID, lease, gasProof, chamberState, c.now())
	if err != nil || !decision.Allowed {
		c.leases.Release(lease, c.now())
		if err == nil {
			err = errors.New("quench start denied")
		}
		return model.ResourceLease{}, decision, err
	}
	if err := c.workers.Start(ctx, operationID, c.now); err != nil {
		c.leases.Release(lease, c.now())
		return model.ResourceLease{}, decision, err
	}
	if _, err := c.journal.Append(operationID, "quench.started", lease, c.now()); err != nil {
		c.workers.Stop(operationID)
		c.leases.Release(lease, c.now())
		return model.ResourceLease{}, decision, err
	}
	return lease, decision, nil
}

func (c *Coordinator) Abort(operationID string) bool {
	return c.workers.Stop(operationID)
}

func (c *Coordinator) Release(lease model.ResourceLease) error {
	c.workers.Stop(lease.OperationID)
	if err := c.leases.Release(lease, c.now()); err != nil {
		return err
	}
	_, err := c.journal.Append(lease.OperationID, "quench.released", lease, c.now())
	return err
}

func (c *Coordinator) Current(resourceID string) (model.ResourceLease, bool) {
	return c.leases.Current(resourceID)
}
