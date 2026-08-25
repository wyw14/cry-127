package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-127/internal/model"
)

type startRequest struct {
	FurnaceID      string `json:"furnace_id"`
	RecipeRevision string `json:"recipe_revision"`
}

type vacuumRequest struct {
	PressurePa float64 `json:"pressure_pa"`
	LeakRate   float64 `json:"leak_rate"`
}

type temperatureRequest struct {
	Values map[string]float64 `json:"values"`
	At     time.Time          `json:"at"`
}

type atmosphereRequest struct {
	Line string `json:"line"`
}

type abortRequest struct {
	Cause string `json:"cause"`
}

type calibrationRequest struct {
	ProbeID string  `json:"probe_id"`
	OffsetC float64 `json:"offset_c"`
	Scale   float64 `json:"scale"`
}

type pressureRequest struct {
	PressurePa float64 `json:"pressure_pa"`
}

type finishQuenchRequest struct {
	Lease model.ResourceLease `json:"lease"`
}

func (s *Server) listOperations(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"operations": s.runtime.Operations()})
}

func (s *Server) startOperation(writer http.ResponseWriter, request *http.Request) {
	var input startRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	operation, err := s.runtime.StartOperation(input.FurnaceID, input.RecipeRevision)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusCreated, operation)
}

func (s *Server) getOperation(writer http.ResponseWriter, request *http.Request) {
	operation, ok := s.runtime.Operation(chi.URLParam(request, "operationID"))
	if !ok {
		writeError(writer, http.StatusNotFound, errors.New("operation not found"))
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (s *Server) qualifyVacuum(writer http.ResponseWriter, request *http.Request) {
	var input vacuumRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	operation, err := s.runtime.QualifyVacuum(chi.URLParam(request, "operationID"), input.PressurePa, input.LeakRate)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (s *Server) recordTemperatures(writer http.ResponseWriter, request *http.Request) {
	var input temperatureRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if input.At.IsZero() {
		input.At = time.Now().UTC()
	}
	evaluation, operation, err := s.runtime.RecordTemperatures(chi.URLParam(request, "operationID"), input.Values, input.At)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"evaluation": evaluation, "operation": operation})
}

func (s *Server) changeAtmosphere(writer http.ResponseWriter, request *http.Request) {
	var input atmosphereRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := s.runtime.ChangeAtmosphereChecked(chi.URLParam(request, "operationID"), input.Line)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) startQuench(writer http.ResponseWriter, request *http.Request) {
	lease, decision, err := s.runtime.StartQuench(request.Context(), chi.URLParam(request, "operationID"))
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"lease": lease, "decision": decision})
}

func (s *Server) finishQuench(writer http.ResponseWriter, request *http.Request) {
	var input finishQuenchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if input.Lease.OperationID != chi.URLParam(request, "operationID") {
		writeError(writer, http.StatusBadRequest, errors.New("lease operation mismatch"))
		return
	}
	operation, err := s.runtime.FinishQuench(input.Lease)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (s *Server) recalibrate(writer http.ResponseWriter, request *http.Request) {
	var input calibrationRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if err := s.runtime.Recalibrate(chi.URLParam(request, "operationID"), input.ProbeID, input.OffsetC, input.Scale); err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "calibrated"})
}

func (s *Server) setPressure(writer http.ResponseWriter, request *http.Request) {
	var input pressureRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if err := s.runtime.SetChamberPressure(chi.URLParam(request, "operationID"), input.PressurePa); err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) releaseDoor(writer http.ResponseWriter, request *http.Request) {
	decision, err := s.runtime.ReleaseDoor(chi.URLParam(request, "operationID"))
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, decision)
}

func (s *Server) resealDoor(writer http.ResponseWriter, request *http.Request) {
	operation, err := s.runtime.Reseal(chi.URLParam(request, "operationID"))
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (s *Server) abortOperation(writer http.ResponseWriter, request *http.Request) {
	var input abortRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	operation, err := s.runtime.Abort(chi.URLParam(request, "operationID"), input.Cause)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}
