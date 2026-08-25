package heater

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/recipe"
)

type ZoneAck struct {
	OperationID    string    `json:"operation_id"`
	RecipeRevision string    `json:"recipe_revision"`
	Zone           string    `json:"zone"`
	ReadyAt        time.Time `json:"ready_at"`
}

type Coordinator struct {
	mu    sync.RWMutex
	acks  map[string]ZoneAck
	book  *WindowBook
	power map[string]bool
}

func NewCoordinator(book *WindowBook) (*Coordinator, error) {
	if book == nil {
		return nil, errors.New("window book is required")
	}
	return &Coordinator{acks: map[string]ZoneAck{}, book: book, power: map[string]bool{}}, nil
}

func (c *Coordinator) Acknowledge(operation model.Operation, revision recipe.Revision, zone string, ready bool, now time.Time) (ZoneAck, error) {
	if operation.RecipeRevision != revision.ID {
		return ZoneAck{}, errors.New("operation recipe revision does not match acknowledgement")
	}
	if _, ok := revision.ZoneTargets[zone]; !ok {
		return ZoneAck{}, fmt.Errorf("zone %s is not in recipe", zone)
	}
	key := operation.ID + "/" + revision.ID + "/" + zone
	c.mu.Lock()
	defer c.mu.Unlock()
	if !ready {
		delete(c.acks, key)
		return ZoneAck{}, nil
	}
	ack := ZoneAck{OperationID: operation.ID, RecipeRevision: revision.ID, Zone: zone, ReadyAt: now.UTC()}
	c.acks[key] = ack
	return ack, nil
}

func (c *Coordinator) Ready(operationID, revisionID string, zones []string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, zone := range zones {
		if _, ok := c.acks[operationID+"/"+revisionID+"/"+zone]; !ok {
			return false
		}
	}
	return len(zones) > 0
}

func (c *Coordinator) SetPower(operationID string, enabled bool) {
	c.mu.Lock()
	c.power[operationID] = enabled
	c.mu.Unlock()
}

func (c *Coordinator) Powered(operationID string) bool {
	c.mu.RLock()
	enabled := c.power[operationID]
	c.mu.RUnlock()
	return enabled
}

func (c *Coordinator) Stop(operationID string, now time.Time) {
	c.SetPower(operationID, false)
	c.book.Invalidate(operationID, now)
}
