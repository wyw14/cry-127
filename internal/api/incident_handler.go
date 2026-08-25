package api

import "net/http"

func (s *Server) incidents(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"incidents": s.runtime.Incidents()})
}
