package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BatchStage string

const (
	StageLoaded     BatchStage = "loaded"
	StageEvacuating BatchStage = "evacuating"
	StageHeating    BatchStage = "heating"
	StageSoaking    BatchStage = "soaking"
	StageQuenching  BatchStage = "quenching"
	StageCooling    BatchStage = "cooling"
	StageComplete   BatchStage = "complete"
	StageAborted    BatchStage = "aborted"
)

type Operation struct {
	ID             string     `json:"id"`
	FurnaceID      string     `json:"furnace_id"`
	RecipeRevision string     `json:"recipe_revision"`
	Stage          BatchStage `json:"stage"`
	SealRevision   string     `json:"seal_revision"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Failure        string     `json:"failure,omitempty"`
}

func NewOperation(furnaceID, recipeRevision string, now time.Time) (Operation, error) {
	if furnaceID == "" {
		return Operation{}, errors.New("furnace id is required")
	}
	if recipeRevision == "" {
		return Operation{}, errors.New("recipe revision is required")
	}
	return Operation{
		ID:             uuid.NewString(),
		FurnaceID:      furnaceID,
		RecipeRevision: recipeRevision,
		Stage:          StageLoaded,
		SealRevision:   uuid.NewString(),
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}, nil
}

func (o Operation) Validate() error {
	if o.ID == "" || o.FurnaceID == "" || o.RecipeRevision == "" {
		return errors.New("operation identity is incomplete")
	}
	if !KnownStage(o.Stage) {
		return fmt.Errorf("unknown operation stage %q", o.Stage)
	}
	if o.SealRevision == "" {
		return errors.New("seal revision is required")
	}
	return nil
}

func KnownStage(stage BatchStage) bool {
	switch stage {
	case StageLoaded, StageEvacuating, StageHeating, StageSoaking, StageQuenching, StageCooling, StageComplete, StageAborted:
		return true
	default:
		return false
	}
}
