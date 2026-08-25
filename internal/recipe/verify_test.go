package recipe_test

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

func TestCalibrationChangeResetsSoakEvidenceWindow(t *testing.T) {
	server := calibrationHTTPServer(t)
	operation := calibrationRequest[model.Operation](t, server, http.MethodPost, "/api/operations", map[string]any{"furnace_id": "furnace-1"}, http.StatusCreated)
	calibrationRequest[model.Operation](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/vacuum", map[string]any{"pressure_pa": 5, "leak_rate": 0.001}, http.StatusOK)
	start := time.Now().UTC()
	values := map[string]float64{"load-a": 900, "load-b": 900, "load-c": 900}
	calibrationRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/temperatures", map[string]any{"values": values, "at": start}, http.StatusOK)
	calibrationRequest[map[string]any](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/calibration", map[string]any{"probe_id": "load-c", "offset_c": 0, "scale": 1}, http.StatusOK)
	result := calibrationRequest[struct {
		Operation model.Operation `json:"operation"`
	}](t, server, http.MethodPost, "/api/operations/"+operation.ID+"/temperatures", map[string]any{"values": values, "at": start.Add(time.Second)}, http.StatusOK)
	if result.Operation.Stage != model.StageSoaking {
		t.Fatalf("calibration change reused prior soak duration: %+v", result.Operation)
	}
}

func calibrationHTTPServer(t *testing.T) *httptest.Server {
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

func calibrationRequest[T any](t *testing.T, server *httptest.Server, method, path string, input any, expected int) T {
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
