package heater

import (
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry-127/internal/recipe"
	"github.com/wyw14/cry-127/internal/sensor"
)

type Evaluation struct {
	Window       Window            `json:"window"`
	Assessment   sensor.Assessment `json:"assessment"`
	TargetC      float64           `json:"target_c"`
	ToleranceC   float64           `json:"tolerance_c"`
	SoakComplete bool              `json:"soak_complete"`
}

func Evaluate(book *WindowBook, revision recipe.Revision, assessment sensor.Assessment, now time.Time) (Evaluation, error) {
	if book == nil {
		return Evaluation{}, errors.New("window book is required")
	}
	if revision.ID == "" || assessment.OperationID == "" {
		return Evaluation{}, errors.New("evaluation identity is incomplete")
	}
	epoch, err := sensor.RequireEpoch(assessment)
	if err != nil {
		book.Invalidate(assessment.OperationID, now)
		return Evaluation{Assessment: assessment, TargetC: revision.TargetC, ToleranceC: revision.ToleranceC}, fmt.Errorf("probe evidence is incomplete: %w", err)
	}
	withinTarget := assessment.AverageC >= revision.TargetC-revision.ToleranceC
	window, err := book.Record(WindowKey{
		OperationID:      assessment.OperationID,
		RecipeRevision:   revision.ID,
		CalibrationEpoch: epoch,
	}, now, withinTarget)
	if err != nil {
		return Evaluation{}, err
	}
	return Evaluation{
		Window:       window,
		Assessment:   assessment,
		TargetC:      revision.TargetC,
		ToleranceC:   revision.ToleranceC,
		SoakComplete: window.Valid && window.Duration() >= revision.SoakDuration,
	}, nil
}

func Eligible(evaluation Evaluation) bool {
	return evaluation.Assessment.Complete && evaluation.Window.Valid && evaluation.SoakComplete
}
