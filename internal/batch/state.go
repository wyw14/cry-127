package batch

import (
	"errors"
	"sort"
	"sync"

	"github.com/wyw14/cry-127/internal/model"
)

type Store struct {
	mu         sync.RWMutex
	operations map[string]model.Operation
}

func NewStore() *Store {
	return &Store{operations: map[string]model.Operation{}}
}

func (s *Store) Put(operation model.Operation) error {
	if err := operation.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	s.operations[operation.ID] = operation
	s.mu.Unlock()
	return nil
}

func (s *Store) Get(id string) (model.Operation, bool) {
	s.mu.RLock()
	operation, ok := s.operations[id]
	s.mu.RUnlock()
	return operation, ok
}

func (s *Store) Update(id string, apply func(model.Operation) (model.Operation, error)) (model.Operation, error) {
	if apply == nil {
		return model.Operation{}, errors.New("update function is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.operations[id]
	if !ok {
		return model.Operation{}, errors.New("operation not found")
	}
	updated, err := apply(operation)
	if err != nil {
		return model.Operation{}, err
	}
	if err := updated.Validate(); err != nil {
		return model.Operation{}, err
	}
	s.operations[id] = updated
	return updated, nil
}

func (s *Store) List() []model.Operation {
	s.mu.RLock()
	result := make([]model.Operation, 0, len(s.operations))
	for _, operation := range s.operations {
		result = append(result, operation)
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (s *Store) Active() []model.Operation {
	all := s.List()
	result := all[:0]
	for _, operation := range all {
		if !model.Terminal(operation.Stage) {
			result = append(result, operation)
		}
	}
	return result
}
