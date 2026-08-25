package recipe

import (
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry-127/internal/model"
)

type Readiness struct {
	OperationID    string
	RevisionID     string
	ReadyZones     map[string]time.Time
	RequiredZones  map[string]float64
	LastEvaluation time.Time
}

func NewReadiness(operationID string, revision Revision) (Readiness, error) {
	if operationID == "" || revision.ID == "" {
		return Readiness{}, errors.New("readiness identity is incomplete")
	}
	return Readiness{
		OperationID:   operationID,
		RevisionID:    revision.ID,
		ReadyZones:    map[string]time.Time{},
		RequiredZones: cloneFloats(revision.ZoneTargets),
	}, nil
}

func (r Readiness) Accept(revision Revision, zone string, sample model.TemperatureSample) (Readiness, error) {
	if r.RevisionID != revision.ID {
		return Readiness{}, fmt.Errorf("readiness belongs to recipe %s", r.RevisionID)
	}
	target, exists := r.RequiredZones[zone]
	if !exists {
		return Readiness{}, fmt.Errorf("zone %s is not required", zone)
	}
	r.ReadyZones = cloneTimes(r.ReadyZones)
	if sample.Online && sample.TemperatureC >= target-revision.ToleranceC {
		r.ReadyZones[zone] = sample.RecordedAt.UTC()
	} else {
		delete(r.ReadyZones, zone)
	}
	r.LastEvaluation = sample.RecordedAt.UTC()
	return r, nil
}

func (r Readiness) Complete() bool {
	if len(r.RequiredZones) == 0 || len(r.ReadyZones) != len(r.RequiredZones) {
		return false
	}
	for zone := range r.RequiredZones {
		if r.ReadyZones[zone].IsZero() {
			return false
		}
	}
	return true
}

func cloneTimes(source map[string]time.Time) map[string]time.Time {
	result := make(map[string]time.Time, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
