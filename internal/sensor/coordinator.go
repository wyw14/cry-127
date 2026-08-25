package sensor

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/wyw14/cry-127/internal/model"
)

type ProbeHealth struct {
	ProbeID   string  `json:"probe_id"`
	Required  bool    `json:"required"`
	Online    bool    `json:"online"`
	ValueC    float64 `json:"value_c"`
	Epoch     string  `json:"epoch"`
	Operation string  `json:"operation"`
}

type Assessment struct {
	OperationID string        `json:"operation_id"`
	Probes      []ProbeHealth `json:"probes"`
	Complete    bool          `json:"complete"`
	ColdestC    float64       `json:"coldest_c"`
	AverageC    float64       `json:"average_c"`
}

func Assess(operationID string, required map[string]bool, samples []model.TemperatureSample) (Assessment, error) {
	if operationID == "" || len(required) == 0 {
		return Assessment{}, errors.New("assessment identity and required probes are needed")
	}
	byProbe := map[string]model.TemperatureSample{}
	for _, sample := range samples {
		if sample.OperationID == operationID {
			byProbe[sample.ProbeID] = sample
		}
	}
	assessment := Assessment{OperationID: operationID, Complete: true}
	total := 0.0
	count := 0
	for probeID := range required {
		sample, ok := byProbe[probeID]
		health := ProbeHealth{ProbeID: probeID, Required: true, Operation: operationID}
		if ok {
			health.Online = sample.Online
			health.ValueC = sample.TemperatureC
			health.Epoch = sample.CalibrationEpoch
		}
		if !ok || !sample.Online {
			assessment.Complete = false
		} else {
			total += sample.TemperatureC
			count++
			if count == 1 || sample.TemperatureC < assessment.ColdestC {
				assessment.ColdestC = sample.TemperatureC
			}
		}
		assessment.Probes = append(assessment.Probes, health)
	}
	if count > 0 {
		assessment.AverageC = total / float64(count)
	}
	sort.Slice(assessment.Probes, func(i, j int) bool { return assessment.Probes[i].ProbeID < assessment.Probes[j].ProbeID })
	return assessment, nil
}

func RequireEpoch(assessment Assessment) (string, error) {
	if !assessment.Complete {
		return "", errors.New("required probe set is incomplete")
	}
	epochs := make([]string, 0, len(assessment.Probes))
	for _, probe := range assessment.Probes {
		if probe.Epoch == "" {
			return "", fmt.Errorf("probe %s has no calibration epoch", probe.ProbeID)
		}
		epochs = append(epochs, probe.ProbeID+"="+probe.Epoch)
	}
	sort.Strings(epochs)
	return strings.Join(epochs, "|"), nil
}
