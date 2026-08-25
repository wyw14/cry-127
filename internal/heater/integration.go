package heater

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/recipe"
	"github.com/wyw14/cry-127/internal/sensor"
)

type Integrator struct {
	receiver *sensor.Receiver
	windows  *WindowBook
}

func NewIntegrator(receiver *sensor.Receiver, windows *WindowBook) (*Integrator, error) {
	if receiver == nil || windows == nil {
		return nil, errors.New("heater integration dependencies are required")
	}
	return &Integrator{receiver: receiver, windows: windows}, nil
}

func (i *Integrator) EvaluateOperation(operation model.Operation, revision recipe.Revision, now time.Time) (Evaluation, error) {
	samples := i.receiver.OperationSamples(operation.ID)
	assessment, err := sensor.Assess(operation.ID, revision.RequiredProbes, samples)
	if err != nil {
		return Evaluation{}, err
	}
	return Evaluate(i.windows, revision, assessment, now)
}

func (i *Integrator) InvalidateForCalibration(operationID string, now time.Time) bool {
	return i.windows.Invalidate(operationID, now)
}

func (i *Integrator) Current(operationID string) (Window, bool) {
	return i.windows.Current(operationID)
}
