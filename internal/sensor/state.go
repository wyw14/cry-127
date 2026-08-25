package sensor

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Calibration struct {
	ProbeID    string    `json:"probe_id"`
	Epoch      string    `json:"epoch"`
	OffsetC    float64   `json:"offset_c"`
	Scale      float64   `json:"scale"`
	RecordedAt time.Time `json:"recorded_at"`
}

type CalibrationBook struct {
	mu      sync.RWMutex
	current map[string]Calibration
}

func NewCalibrationBook() *CalibrationBook {
	return &CalibrationBook{current: map[string]Calibration{}}
}

func (b *CalibrationBook) Calibrate(probeID string, offset, scale float64, now time.Time) (Calibration, error) {
	if probeID == "" {
		return Calibration{}, errors.New("probe identity is required")
	}
	if scale == 0 {
		return Calibration{}, errors.New("calibration scale cannot be zero")
	}
	calibration := Calibration{ProbeID: probeID, Epoch: uuid.NewString(), OffsetC: offset, Scale: scale, RecordedAt: now.UTC()}
	b.mu.Lock()
	b.current[probeID] = calibration
	b.mu.Unlock()
	return calibration, nil
}

func (b *CalibrationBook) Current(probeID string) (Calibration, bool) {
	b.mu.RLock()
	calibration, ok := b.current[probeID]
	b.mu.RUnlock()
	return calibration, ok
}

func (b *CalibrationBook) Apply(probeID string, raw float64) (float64, Calibration, error) {
	calibration, ok := b.Current(probeID)
	if !ok {
		return 0, Calibration{}, errors.New("probe has no calibration")
	}
	return raw*calibration.Scale + calibration.OffsetC, calibration, nil
}

func (b *CalibrationBook) All() []Calibration {
	b.mu.RLock()
	result := make([]Calibration, 0, len(b.current))
	for _, calibration := range b.current {
		result = append(result, calibration)
	}
	b.mu.RUnlock()
	return result
}
