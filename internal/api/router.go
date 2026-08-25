package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/cry-127/internal/service"
)

type Server struct {
	runtime *service.Runtime
	router  chi.Router
}

func NewServer(runtime *service.Runtime) (*Server, error) {
	server := &Server{runtime: runtime, router: chi.NewRouter()}
	server.routes()
	return server, nil
}

func (s *Server) routes() {
	s.router.Get("/healthz", s.health)
	s.router.Route("/api", func(router chi.Router) {
		router.Get("/operations", s.listOperations)
		router.Post("/operations", s.startOperation)
		router.Get("/operations/{operationID}", s.getOperation)
		router.Post("/operations/{operationID}/vacuum", s.qualifyVacuum)
		router.Post("/operations/{operationID}/temperatures", s.recordTemperatures)
		router.Post("/operations/{operationID}/atmosphere", s.changeAtmosphere)
		router.Post("/operations/{operationID}/atmosphere/isolate", s.isolateAtmosphere)
		router.Post("/operations/{operationID}/recipe/retry", s.retryRecipe)
		router.Post("/operations/{operationID}/quench", s.startQuench)
		router.Post("/operations/{operationID}/quench/finish", s.finishQuench)
		router.Post("/operations/{operationID}/cooling/complete", s.completeCooling)
		router.Post("/operations/{operationID}/calibration", s.recalibrate)
		router.Post("/operations/{operationID}/pressure", s.setPressure)
		router.Post("/operations/{operationID}/vacuum/topology", s.setVacuumTopology)
		router.Post("/operations/{operationID}/door/release", s.releaseDoor)
		router.Post("/operations/{operationID}/door/reseal", s.resealDoor)
		router.Post("/operations/{operationID}/abort", s.abortOperation)
		router.Post("/recovery/reload", s.reloadRecovery)
		router.Get("/equipment", s.equipment)
		router.Get("/interlocks", s.interlocks)
		router.Get("/incidents", s.incidents)
	})
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) health(writer http.ResponseWriter, request *http.Request) {
	if err := s.runtime.ValidatePersistentState(); err != nil {
		writeError(writer, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"status":         "ok",
		"recovery_ready": s.runtime.RecoveryReady(),
		"data_dir":       s.runtime.DataDirectory(),
		"default_recipe": s.runtime.DefaultRecipe(),
	})
}

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}
