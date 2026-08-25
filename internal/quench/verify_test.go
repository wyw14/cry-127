package quench_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/api"
	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/service"
)

func TestRetiredFurnaceCannotReleaseReassignedQuenchHeader(t *testing.T) {
	server := fencingHTTPServer(t)
	first := fencingReady(t, server, "furnace-1")
	firstStart := fencingRequest[struct {
		Lease model.ResourceLease `json:"lease"`
	}](t, server, http.MethodPost, "/api/operations/"+first.ID+"/quench", nil, http.StatusOK)
	fencingRequest[model.Operation](t, server, http.MethodPost, "/api/operations/"+first.ID+"/quench/finish", map[string]any{"lease": firstStart.Lease}, http.StatusOK)
	second := fencingReady(t, server, "furnace-2")
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+second.ID+"/quench", nil, http.StatusOK)
	third := fencingReady(t, server, "furnace-3")
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+first.ID+"/quench/finish", map[string]any{"lease": firstStart.Lease}, http.StatusConflict)
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+third.ID+"/quench", nil, http.StatusConflict)
}

func fencingReady(t *testing.T, server *httptest.Server, furnace string) model.Operation {
	operation := fencingRequest[model.Operation](t, server, http.MethodPost, "/api/operations", map[string]any{"furnace_id": furnace}, http.StatusCreated)
	fencingRequest[model.Operation](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/vacuum", map[string]any{"pressure_pa": 5, "leak_rate": 0.001}, http.StatusOK)
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/atmosphere", map[string]any{"line": "nitrogen"}, http.StatusOK)
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/atmosphere/isolate", nil, http.StatusOK)
	start := time.Now().UTC()
	values := map[string]float64{"load-a": 900, "load-b": 900, "load-c": 900}
	fencingRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/temperatures", map[string]any{"values": values, "at": start}, http.StatusOK)
	result := fencingRequest[struct {
		Operation model.Operation `json:"operation"`
	}](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/temperatures", map[string]any{"values": values, "at": start.Add(time.Second)}, http.StatusOK)
	return result.Operation
}

func fencingHTTPServer(t *testing.T) *httptest.Server {
	runtime, err := service.NewRuntime(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	server, err := api.NewServer(runtime)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return httpServer
}

func fencingRequest[T any](t *testing.T, server *httptest.Server, method, path string, input any, expected int) T {
	t.Helper()
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	encoded, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != expected {
		t.Fatalf("%s %s returned %d: %s", method, path, response.StatusCode, encoded)
	}
	var output T
	if len(encoded) != 0 {
		if err := json.Unmarshal(encoded, &output); err != nil {
			t.Fatal(err)
		}
	}
	return output
}
