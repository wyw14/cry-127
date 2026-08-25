package heater

import (
	"errors"
	"sync"
	"time"
)

type WindowKey struct {
	OperationID      string `json:"operation_id"`
	RecipeRevision   string `json:"recipe_revision"`
	CalibrationEpoch string `json:"calibration_epoch"`
}

type Window struct {
	Key       WindowKey `json:"key"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Valid     bool      `json:"valid"`
	Samples   int       `json:"samples"`
}

type WindowBook struct {
	mu      sync.RWMutex
	windows map[string]Window
}

func NewWindowBook() *WindowBook {
	return &WindowBook{windows: map[string]Window{}}
}

func (b *WindowBook) Record(key WindowKey, at time.Time, withinTarget bool) (Window, error) {
	if key.OperationID == "" || key.RecipeRevision == "" || key.CalibrationEpoch == "" {
		return Window{}, errors.New("soak window identity is incomplete")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	current, ok := b.windows[key.OperationID]
	if !ok || current.Key != key || !withinTarget {
		current = Window{Key: key, StartedAt: at.UTC(), UpdatedAt: at.UTC(), Valid: withinTarget}
		if withinTarget {
			current.Samples = 1
		}
		b.windows[key.OperationID] = current
		return current, nil
	}
	current.UpdatedAt = at.UTC()
	current.Valid = true
	current.Samples++
	b.windows[key.OperationID] = current
	return current, nil
}

func (b *WindowBook) Invalidate(operationID string, at time.Time) bool {
	b.mu.Lock()
	window, ok := b.windows[operationID]
	if ok {
		window.Valid = false
		window.UpdatedAt = at.UTC()
		b.windows[operationID] = window
	}
	b.mu.Unlock()
	return ok
}

func (b *WindowBook) Current(operationID string) (Window, bool) {
	b.mu.RLock()
	window, ok := b.windows[operationID]
	b.mu.RUnlock()
	return window, ok
}

func (w Window) Duration() time.Duration {
	if !w.Valid || w.UpdatedAt.Before(w.StartedAt) {
		return 0
	}
	return w.UpdatedAt.Sub(w.StartedAt)
}
