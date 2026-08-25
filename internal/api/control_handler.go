package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-127/internal/service"
)

type coolingRequest struct {
	DiffusionError string `json:"diffusion_error"`
	RoughingError  string `json:"roughing_error"`
}

type topologyRequest struct {
	VentOpen bool `json:"vent_open"`
}

func (s *Server) isolateAtmosphere(writer http.ResponseWriter, request *http.Request) {
	state, err := s.runtime.IsolateAtmosphere(chi.URLParam(request, "operationID"))
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, state)
}

func (s *Server) retryRecipe(writer http.ResponseWriter, request *http.Request) {
	result, err := s.runtime.RetryRecipe(chi.URLParam(request, "operationID"))
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) completeCooling(writer http.ResponseWriter, request *http.Request) {
	var input coolingRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	operation, err := s.runtime.CompleteCooling(
		chi.URLParam(request, "operationID"),
		service.RuntimeError(input.DiffusionError),
		service.RuntimeError(input.RoughingError),
	)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (s *Server) setVacuumTopology(writer http.ResponseWriter, request *http.Request) {
	var input topologyRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if err := s.runtime.ConfigureVacuumVent(chi.URLParam(request, "operationID"), input.VentOpen); err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"vent_open": input.VentOpen})
}

func (s *Server) reloadRecovery(writer http.ResponseWriter, request *http.Request) {
	if err := s.runtime.ReloadPersistentState(); err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"recovery_ready": s.runtime.RecoveryReady()})
}
