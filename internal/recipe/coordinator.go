package recipe

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wyw14/cry-127/internal/model"
)

type EvidenceKey struct {
	OperationID      string
	RecipeRevision   string
	ProbeID          string
	CalibrationEpoch string
}

type EvidenceBook struct {
	mu      sync.RWMutex
	samples map[EvidenceKey][]model.TemperatureSample
}

func NewEvidenceBook() *EvidenceBook {
	return &EvidenceBook{samples: map[EvidenceKey][]model.TemperatureSample{}}
}

func (b *EvidenceBook) Record(revision Revision, sample model.TemperatureSample) error {
	if sample.OperationID == "" || sample.ProbeID == "" || sample.CalibrationEpoch == "" {
		return errors.New("temperature evidence identity is incomplete")
	}
	if !revision.RequiredProbes[sample.ProbeID] && sample.Required {
		return fmt.Errorf("probe %s is not part of revision %s", sample.ProbeID, revision.ID)
	}
	key := EvidenceKey{
		OperationID:      sample.OperationID,
		RecipeRevision:   revision.ID,
		ProbeID:          sample.ProbeID,
		CalibrationEpoch: sample.CalibrationEpoch,
	}
	b.mu.Lock()
	b.samples[key] = append(b.samples[key], sample)
	b.mu.Unlock()
	return nil
}

func (b *EvidenceBook) Samples(key EvidenceKey) []model.TemperatureSample {
	b.mu.RLock()
	result := append([]model.TemperatureSample(nil), b.samples[key]...)
	b.mu.RUnlock()
	return result
}

func (b *EvidenceBook) ResetOperation(operationID string) int {
	b.mu.Lock()
	removed := 0
	for key := range b.samples {
		if key.OperationID == operationID {
			removed += len(b.samples[key])
			delete(b.samples, key)
		}
	}
	b.mu.Unlock()
	return removed
}
