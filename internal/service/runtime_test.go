package service

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/model"
)

func TestRuntimePersistsAndCoordinatesCycle(t *testing.T) {
	clock := time.Now
	runtime, err := NewRuntime(t.TempDir(), clock)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := runtime.StartOperation("furnace-a", "")
	if err != nil {
		t.Fatal(err)
	}
	operation, err = runtime.QualifyVacuum(operation.ID, 5, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Stage != model.StageHeating {
		t.Fatalf("stage %s", operation.Stage)
	}
	values := map[string]float64{"load-a": 905, "load-b": 906, "load-c": 907}
	first := time.Now()
	if _, _, err := runtime.RecordTemperatures(operation.ID, values, first); err != nil {
		t.Fatal(err)
	}
	if _, operation, err = runtime.RecordTemperatures(operation.ID, values, first.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if operation.Stage != model.StageQuenching {
		t.Fatalf("stage %s", operation.Stage)
	}
	if _, err := runtime.ChangeAtmosphere(operation.ID, "nitrogen"); err != nil {
		t.Fatal(err)
	}
	lease, decision, err := runtime.StartQuench(context.Background(), operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed {
		t.Fatal("quench decision denied")
	}
	if _, err := runtime.FinishQuench(lease); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.ValidatePersistentState(); err != nil {
		t.Fatal(err)
	}
}

func TestAbortStopsRuntimeWork(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := runtime.StartOperation("furnace-b", "")
	if err != nil {
		t.Fatal(err)
	}
	operation, err = runtime.Abort(operation.ID, "pressure alarm")
	if err != nil {
		t.Fatal(err)
	}
	if operation.Stage != model.StageAborted {
		t.Fatalf("stage %s", operation.Stage)
	}
	if len(runtime.Incidents()) != 1 {
		t.Fatalf("incident count %d", len(runtime.Incidents()))
	}
}
