package batch

import (
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/recipe"
)

type Coordinator struct {
	store   *Store
	recipes *recipe.Registry
	journal *journal.Store
	clock   func() time.Time
}

func NewCoordinator(store *Store, recipes *recipe.Registry, eventStore *journal.Store, clock func() time.Time) (*Coordinator, error) {
	if store == nil || recipes == nil || eventStore == nil || clock == nil {
		return nil, errors.New("batch coordinator dependencies are required")
	}
	return &Coordinator{store: store, recipes: recipes, journal: eventStore, clock: clock}, nil
}

func (c *Coordinator) Start(furnaceID, recipeRevision string) (model.Operation, error) {
	if _, ok := c.recipes.Get(recipeRevision); !ok {
		return model.Operation{}, fmt.Errorf("recipe revision %s does not exist", recipeRevision)
	}
	operation, err := model.NewOperation(furnaceID, recipeRevision, c.clock())
	if err != nil {
		return model.Operation{}, err
	}
	if err := c.store.Put(operation); err != nil {
		return model.Operation{}, err
	}
	if _, err := c.journal.Append(operation.ID, "operation.created", operation, c.clock()); err != nil {
		return model.Operation{}, err
	}
	return operation, nil
}

func (c *Coordinator) Advance(id string, next model.BatchStage) (model.Operation, error) {
	updated, err := c.store.Update(id, func(operation model.Operation) (model.Operation, error) {
		return model.Advance(operation, next, c.clock())
	})
	if err != nil {
		return model.Operation{}, err
	}
	if _, err := c.journal.Append(id, "operation.updated", updated, c.clock()); err != nil {
		return model.Operation{}, err
	}
	return updated, nil
}

func (c *Coordinator) Abort(id, cause string) (model.Operation, error) {
	updated, err := c.store.Update(id, func(operation model.Operation) (model.Operation, error) {
		return model.Abort(operation, cause, c.clock())
	})
	if err != nil {
		return model.Operation{}, err
	}
	if _, err := c.journal.Append(id, "operation.updated", updated, c.clock()); err != nil {
		return model.Operation{}, err
	}
	return updated, nil
}

func (c *Coordinator) ReplaceSeal(id, sealRevision string) (model.Operation, error) {
	updated, err := c.store.Update(id, func(operation model.Operation) (model.Operation, error) {
		return model.ReplaceSeal(operation, sealRevision, c.clock())
	})
	if err != nil {
		return model.Operation{}, err
	}
	if _, err := c.journal.Append(id, "operation.updated", updated, c.clock()); err != nil {
		return model.Operation{}, err
	}
	return updated, nil
}

func (c *Coordinator) Get(id string) (model.Operation, bool) {
	return c.store.Get(id)
}

func (c *Coordinator) List() []model.Operation {
	return c.store.List()
}
