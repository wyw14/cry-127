package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/wyw14/cry-127/internal/model"
)

type Replay struct {
	Operations map[string]model.Operation
	Incidents  []model.Incident
	LastEvent  string
}

func Rebuild(events []Event) (Replay, error) {
	replay := Replay{Operations: map[string]model.Operation{}, Incidents: []model.Incident{}}
	ordered := append([]Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].RecordedAt.Equal(ordered[j].RecordedAt) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].RecordedAt.Before(ordered[j].RecordedAt)
	})
	for _, event := range ordered {
		switch event.Kind {
		case "operation.created", "operation.updated":
			var operation model.Operation
			if err := json.Unmarshal(event.Payload, &operation); err != nil {
				return Replay{}, fmt.Errorf("decode operation event %s: %w", event.ID, err)
			}
			if err := operation.Validate(); err != nil {
				return Replay{}, fmt.Errorf("invalid operation event %s: %w", event.ID, err)
			}
			replay.Operations[operation.ID] = operation
		case "incident.recorded":
			var incident model.Incident
			if err := json.Unmarshal(event.Payload, &incident); err != nil {
				return Replay{}, fmt.Errorf("decode incident event %s: %w", event.ID, err)
			}
			replay.Incidents = append(replay.Incidents, incident)
		}
		replay.LastEvent = event.ID
	}
	return replay, nil
}

func MergeErrors(results ...error) error {
	joined := make([]error, 0, len(results))
	for _, result := range results {
		if result != nil {
			joined = append(joined, result)
		}
	}
	return errors.Join(joined...)
}

func OperationList(replay Replay) []model.Operation {
	result := make([]model.Operation, 0, len(replay.Operations))
	for _, operation := range replay.Operations {
		result = append(result, operation)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}
