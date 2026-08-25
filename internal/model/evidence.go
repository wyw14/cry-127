package model

import "time"

type VacuumProof struct {
	OperationID  string    `json:"operation_id"`
	SealRevision string    `json:"seal_revision"`
	ProofID      string    `json:"proof_id"`
	PressurePa   float64   `json:"pressure_pa"`
	LeakRate     float64   `json:"leak_rate"`
	Durable      bool      `json:"durable"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type GasProof struct {
	OperationID     string    `json:"operation_id"`
	Revision        string    `json:"revision"`
	ActiveLine      string    `json:"active_line"`
	PriorLine       string    `json:"prior_line"`
	PriorIsolated   bool      `json:"prior_isolated"`
	PurgeComplete   bool      `json:"purge_complete"`
	BackfillClosing bool      `json:"backfill_closing"`
	RecordedAt      time.Time `json:"recorded_at"`
}

type DoorProof struct {
	OperationID  string    `json:"operation_id"`
	SealRevision string    `json:"seal_revision"`
	Locked       bool      `json:"locked"`
	PressurePa   float64   `json:"pressure_pa"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type TemperatureSample struct {
	OperationID      string    `json:"operation_id"`
	ProbeID          string    `json:"probe_id"`
	CalibrationEpoch string    `json:"calibration_epoch"`
	TemperatureC     float64   `json:"temperature_c"`
	Required         bool      `json:"required"`
	Online           bool      `json:"online"`
	RecordedAt       time.Time `json:"recorded_at"`
}

type ResourceLease struct {
	ID          string    `json:"id"`
	ResourceID  string    `json:"resource_id"`
	OperationID string    `json:"operation_id"`
	Fence       uint64    `json:"fence"`
	AcquiredAt  time.Time `json:"acquired_at"`
	ReleasedAt  time.Time `json:"released_at,omitempty"`
}

type Incident struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operation_id"`
	Component   string    `json:"component"`
	Message     string    `json:"message"`
	OccurredAt  time.Time `json:"occurred_at"`
}
