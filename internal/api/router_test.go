package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/service"
)

func TestOperationHTTPFlow(t *testing.T) {
	runtime, err := service.NewRuntime(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(runtime)
	if err != nil {
		t.Fatal(err)
	}
	start := requestJSON(t, server.Handler(), http.MethodPost, "/api/operations", map[string]any{"furnace_id": "furnace-a"})
	if start.Code != http.StatusCreated {
		t.Fatalf("start status %d body %s", start.Code, start.Body.String())
	}
	var operation model.Operation
	if err := json.Unmarshal(start.Body.Bytes(), &operation); err != nil {
		t.Fatal(err)
	}
	vacuum := requestJSON(t, server.Handler(), http.MethodPost, "/api/operations/"+operation.ID+"/vacuum", map[string]any{"pressure_pa": 5.0, "leak_rate": 0.001})
	if vacuum.Code != http.StatusOK {
		t.Fatalf("vacuum status %d body %s", vacuum.Code, vacuum.Body.String())
	}
	equipment := requestJSON(t, server.Handler(), http.MethodGet, "/api/equipment?operation_id="+operation.ID, nil)
	if equipment.Code != http.StatusOK {
		t.Fatalf("equipment status %d body %s", equipment.Code, equipment.Body.String())
	}
}

func TestHealthAndCollections(t *testing.T) {
	runtime, err := service.NewRuntime(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(runtime)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/healthz", "/api/operations", "/api/equipment", "/api/interlocks", "/api/incidents"} {
		response := requestJSON(t, server.Handler(), http.MethodGet, path, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("path %s status %d body %s", path, response.Code, response.Body.String())
		}
	}
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded := []byte{}
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
