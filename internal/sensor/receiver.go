package sensor

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

type Receiver struct {
	mu          sync.RWMutex
	calibration *CalibrationBook
	journal     *journal.Store
	latest      map[string]model.TemperatureSample
}

func NewReceiver(calibrations *CalibrationBook, eventStore *journal.Store) (*Receiver, error) {
	if calibrations == nil || eventStore == nil {
		return nil, errors.New("sensor receiver dependencies are required")
	}
	return &Receiver{calibration: calibrations, journal: eventStore, latest: map[string]model.TemperatureSample{}}, nil
}

func (r *Receiver) Receive(operationID, probeID string, raw float64, required, online bool, now time.Time) (model.TemperatureSample, error) {
	if operationID == "" || probeID == "" {
		return model.TemperatureSample{}, errors.New("sample identity is incomplete")
	}
	temperature, calibration, err := r.calibration.Apply(probeID, raw)
	if err != nil {
		return model.TemperatureSample{}, fmt.Errorf("apply calibration: %w", err)
	}
	sample := model.TemperatureSample{
		OperationID:      operationID,
		ProbeID:          probeID,
		CalibrationEpoch: calibration.Epoch,
		TemperatureC:     temperature,
		Required:         required,
		Online:           online,
		RecordedAt:       now.UTC(),
	}
	if _, err := r.journal.Append(operationID, "sensor.sample", sample, now); err != nil {
		return model.TemperatureSample{}, err
	}
	r.mu.Lock()
	r.latest[operationID+"/"+probeID] = sample
	r.mu.Unlock()
	return sample, nil
}

func (r *Receiver) Latest(operationID, probeID string) (model.TemperatureSample, bool) {
	r.mu.RLock()
	sample, ok := r.latest[operationID+"/"+probeID]
	r.mu.RUnlock()
	return sample, ok
}

func (r *Receiver) OperationSamples(operationID string) []model.TemperatureSample {
	r.mu.RLock()
	result := []model.TemperatureSample{}
	for _, sample := range r.latest {
		if sample.OperationID == operationID {
			result = append(result, sample)
		}
	}
	r.mu.RUnlock()
	return result
}
