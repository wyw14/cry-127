package quench

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry-127/internal/model"
)

type LeaseManager struct {
	mu      sync.Mutex
	current map[string]model.ResourceLease
	fences  map[string]uint64
}

func NewLeaseManager(resources []string) (*LeaseManager, error) {
	if len(resources) == 0 {
		return nil, errors.New("quench resources are required")
	}
	manager := &LeaseManager{current: map[string]model.ResourceLease{}, fences: map[string]uint64{}}
	for _, resource := range resources {
		if resource == "" {
			return nil, errors.New("quench resource identity is required")
		}
		manager.fences[resource] = 0
	}
	return manager, nil
}

func (m *LeaseManager) Acquire(resourceID, operationID string, now time.Time) (model.ResourceLease, error) {
	if resourceID == "" || operationID == "" {
		return model.ResourceLease{}, errors.New("lease identity is incomplete")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.fences[resourceID]; !exists {
		return model.ResourceLease{}, errors.New("quench resource does not exist")
	}
	if current, busy := m.current[resourceID]; busy && current.ReleasedAt.IsZero() {
		return model.ResourceLease{}, errors.New("quench resource is busy")
	}
	m.fences[resourceID]++
	lease := model.ResourceLease{ID: uuid.NewString(), ResourceID: resourceID, OperationID: operationID, Fence: m.fences[resourceID], AcquiredAt: now.UTC()}
	m.current[resourceID] = lease
	return lease, nil
}

func (m *LeaseManager) Release(lease model.ResourceLease, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.current[lease.ResourceID]
	if !ok {
		return errors.New("lease is not the current fenced owner")
	}
	current.ReleasedAt = now.UTC()
	m.current[lease.ResourceID] = current
	return nil
}

func (m *LeaseManager) Current(resourceID string) (model.ResourceLease, bool) {
	m.mu.Lock()
	lease, ok := m.current[resourceID]
	m.mu.Unlock()
	return lease, ok && lease.ReleasedAt.IsZero()
}

func (m *LeaseManager) All() []model.ResourceLease {
	m.mu.Lock()
	result := make([]model.ResourceLease, 0, len(m.current))
	for _, lease := range m.current {
		result = append(result, lease)
	}
	m.mu.Unlock()
	return result
}
