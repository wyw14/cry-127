package recipe

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Revision struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	TargetC        float64            `json:"target_c"`
	ToleranceC     float64            `json:"tolerance_c"`
	SoakDuration   time.Duration      `json:"soak_duration"`
	RequiredProbes map[string]bool    `json:"required_probes"`
	CreatedAt      time.Time          `json:"created_at"`
	ZoneTargets    map[string]float64 `json:"zone_targets"`
}

type Registry struct {
	mu        sync.RWMutex
	revisions map[string]Revision
}

func NewRegistry() *Registry {
	return &Registry{revisions: map[string]Revision{}}
}

func (r *Registry) Create(name string, target, tolerance float64, duration time.Duration, probes []string, now time.Time) (Revision, error) {
	if name == "" || duration <= 0 || len(probes) == 0 {
		return Revision{}, errors.New("recipe definition is incomplete")
	}
	required := make(map[string]bool, len(probes))
	for _, probe := range probes {
		if probe == "" {
			return Revision{}, errors.New("probe identity is required")
		}
		required[probe] = true
	}
	revision := Revision{
		ID:             uuid.NewString(),
		Name:           name,
		TargetC:        target,
		ToleranceC:     tolerance,
		SoakDuration:   duration,
		RequiredProbes: required,
		CreatedAt:      now.UTC(),
		ZoneTargets:    map[string]float64{"zone-a": target, "zone-b": target, "zone-c": target},
	}
	r.mu.Lock()
	r.revisions[revision.ID] = revision
	r.mu.Unlock()
	return cloneRevision(revision), nil
}

func (r *Registry) Get(id string) (Revision, bool) {
	r.mu.RLock()
	revision, ok := r.revisions[id]
	r.mu.RUnlock()
	return cloneRevision(revision), ok
}

func (r *Registry) All() []Revision {
	r.mu.RLock()
	result := make([]Revision, 0, len(r.revisions))
	for _, revision := range r.revisions {
		result = append(result, cloneRevision(revision))
	}
	r.mu.RUnlock()
	return result
}

func cloneRevision(revision Revision) Revision {
	revision.RequiredProbes = cloneBools(revision.RequiredProbes)
	revision.ZoneTargets = cloneFloats(revision.ZoneTargets)
	return revision
}

func cloneBools(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneFloats(source map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
