package service_test

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

func TestRecoveryLoadsVacuumTopologyBeforeHeating(t *testing.T) {
	server := recoveryHTTPServer(t)
	operation := recoveryRequest[model.Operation](t, server, http.MethodPost, "/api/operations", map[string]any{"furnace_id": "furnace-1"}, http.StatusCreated)
	recoveryRequest[model.Operation](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/vacuum", map[string]any{"pressure_pa": 5, "leak_rate": 0.001}, http.StatusOK)
	recoveryRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/vacuum/topology", map[string]any{"vent_open": true}, http.StatusOK)
	recoveryRequest[map[string]any](t, server, http.MethodPost, "/api/recovery/reload", nil, http.StatusOK)
	view := recoveryRequest[service.EquipmentView](t, server, http.MethodGet, "/api/equipment?operation_id="+operation.ID, nil, http.StatusOK)
	if view.HeaterPower {
		t.Fatal("heater restarted before restored vacuum topology became safe")
	}
}

func recoveryHTTPServer(t *testing.T) *httptest.Server {
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

func recoveryRequest[T any](t *testing.T, server *httptest.Server, method, path string, input any, expected int) T {
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
