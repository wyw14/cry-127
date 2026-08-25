package api

import (
	"net/http"
	"strings"
)

func (s *Server) equipment(writer http.ResponseWriter, request *http.Request) {
	operationID := strings.TrimSpace(request.URL.Query().Get("operation_id"))
	if operationID != "" {
		view, err := s.runtime.Equipment(operationID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, view)
		return
	}
	views := []any{}
	for _, operation := range s.runtime.Operations() {
		view, err := s.runtime.Equipment(operation.ID)
		if err == nil {
			views = append(views, view)
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{"equipment": views})
}
