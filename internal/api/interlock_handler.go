package api

import "net/http"

func (s *Server) interlocks(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"decisions": s.runtime.Decisions()})
}
