package model

import (
	"errors"
	"fmt"
	"time"
)

var stageEdges = map[BatchStage]map[BatchStage]bool{
	StageLoaded:     {StageEvacuating: true, StageAborted: true},
	StageEvacuating: {StageHeating: true, StageAborted: true},
	StageHeating:    {StageSoaking: true, StageAborted: true},
	StageSoaking:    {StageQuenching: true, StageAborted: true},
	StageQuenching:  {StageCooling: true, StageAborted: true},
	StageCooling:    {StageComplete: true, StageAborted: true},
	StageComplete:   {},
	StageAborted:    {},
}

func Advance(operation Operation, next BatchStage, now time.Time) (Operation, error) {
	if err := operation.Validate(); err != nil {
		return Operation{}, err
	}
	allowed, exists := stageEdges[operation.Stage]
	if !exists || !allowed[next] {
		return Operation{}, fmt.Errorf("transition %s to %s is not allowed", operation.Stage, next)
	}
	operation.Stage = next
	operation.UpdatedAt = now.UTC()
	if next != StageAborted {
		operation.Failure = ""
	}
	return operation, nil
}

func Abort(operation Operation, cause string, now time.Time) (Operation, error) {
	if cause == "" {
		return Operation{}, errors.New("abort cause is required")
	}
	if operation.Stage == StageComplete || operation.Stage == StageAborted {
		return Operation{}, fmt.Errorf("operation in %s cannot be aborted", operation.Stage)
	}
	operation.Stage = StageAborted
	operation.Failure = cause
	operation.UpdatedAt = now.UTC()
	return operation, nil
}

func ReplaceSeal(operation Operation, sealRevision string, now time.Time) (Operation, error) {
	if sealRevision == "" {
		return Operation{}, errors.New("replacement seal revision is required")
	}
	operation.SealRevision = sealRevision
	operation.UpdatedAt = now.UTC()
	return operation, nil
}

func Terminal(stage BatchStage) bool {
	return stage == StageComplete || stage == StageAborted
}
