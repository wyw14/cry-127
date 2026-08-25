package door

import (
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
)

// newTestCoordinator wires a door Coordinator against a shared journal so the
// chamber controller, interlock arbiter, and door all record into the same
// event log, mirroring the production runtime wiring.
func newTestCoordinator(t *testing.T, atmosphereStates *atmosphere.StateStore) (*Coordinator, *chamber.Controller, *interlock.Arbiter) {
	t.Helper()
	if atmosphereStates == nil {
		atmosphereStates = atmosphere.NewStateStore()
	}
	journalStore, err := journal.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	chambers, err := chamber.NewController(chamber.NewRegistry(), journalStore)
	if err != nil {
		t.Fatal(err)
	}
	arbiter, err := interlock.NewArbiter(journalStore)
	if err != nil {
		t.Fatal(err)
	}
	integration, err := interlock.NewIntegration(arbiter)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewCoordinator(NewRegistry(), atmosphereStates, chambers, integration, journalStore)
	if err != nil {
		t.Fatal(err)
	}
	return coordinator, chambers, arbiter
}

// TestReleaseDoorLockedDuringBackfillClosing reproduces the field report: at
// the end of cooling the chamber pressure has returned to atmosphere, yet the
// backfill nitrogen valve still reports closing. The door interlock must keep
// the door locked until this round of gas-path confirmation has actually
// finished isolating the backfill line.
func TestReleaseDoorLockedDuringBackfillClosing(t *testing.T) {
	now := time.Now()
	atmosphereStates := atmosphere.NewStateStore()
	coordinator, chambers, arbiter := newTestCoordinator(t, atmosphereStates)

	operation, err := model.NewOperation("furnace-a", "recipe-1", now)
	if err != nil {
		t.Fatal(err)
	}
	operationID := operation.ID
	if _, err := coordinator.Seal(operationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := chambers.Register(operation, now); err != nil {
		t.Fatal(err)
	}
	if _, err := atmosphereStates.Begin(operationID, now); err != nil {
		t.Fatal(err)
	}

	// Cooling finishes: chamber pressure is back to atmosphere.
	if _, err := chambers.SetPressure(operationID, 101325, now); err != nil {
		t.Fatal(err)
	}

	// The backfill nitrogen line is still active: the gas path has not been
	// isolated and the valve is still closing.
	if err := atmosphereStates.Put(atmosphere.LineState{
		OperationID:   operationID,
		Revision:      "rev-1",
		ActiveLine:    "nitrogen",
		IsolatedLines: map[string]bool{},
		PurgeComplete: true,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}

	state, decision, err := coordinator.Release(operationID, now)
	if err == nil {
		t.Fatal("door release must be denied while the backfill valve is still closing")
	}
	if decision.Allowed {
		t.Fatalf("interlock granted door release, want denied: %+v", decision)
	}
	// State must remain untouched when the decision denies release.
	current, ok := coordinator.registry.Current(operationID)
	if !ok || !current.Locked {
		t.Fatalf("door lock state must remain locked, got %+v", current)
	}
	if !state.Locked {
		t.Fatalf("returned state must remain locked, got %+v", state)
	}

	latest, ok := arbiter.Latest(operationID, "door.release")
	if !ok {
		t.Fatal("expected a recorded door.release decision")
	}
	foundReason := false
	for _, reason := range latest.Reasons {
		if reason == "backfill gas path is not isolated" {
			foundReason = true
		}
	}
	if !foundReason {
		t.Fatalf("decision reasons must cite the backfill gas path, got %v", latest.Reasons)
	}
}

// TestReleaseDoorAllowedAfterBackfillIsolated confirms that isolating the
// backfill line clears the lock and the door may then be released.
func TestReleaseDoorAllowedAfterBackfillIsolated(t *testing.T) {
	now := time.Now()
	atmosphereStates := atmosphere.NewStateStore()
	coordinator, chambers, _ := newTestCoordinator(t, atmosphereStates)

	operation, err := model.NewOperation("furnace-a", "recipe-1", now)
	if err != nil {
		t.Fatal(err)
	}
	operationID := operation.ID
	if _, err := coordinator.Seal(operationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := chambers.Register(operation, now); err != nil {
		t.Fatal(err)
	}
	if _, err := atmosphereStates.Begin(operationID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := chambers.SetPressure(operationID, 101325, now); err != nil {
		t.Fatal(err)
	}

	// Gas path confirmed complete: no active line and the prior nitrogen line
	// has been isolated.
	if err := atmosphereStates.Put(atmosphere.LineState{
		OperationID:   operationID,
		Revision:      "rev-2",
		ActiveLine:    "",
		IsolatedLines: map[string]bool{"nitrogen": true},
		PurgeComplete: true,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}

	state, decision, err := coordinator.Release(operationID, now)
	if err != nil {
		t.Fatalf("door release should succeed after isolation: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("interlock denied door release after isolation: %+v", decision)
	}
	if state.Locked {
		t.Fatalf("door must be unlocked after release, got %+v", state)
	}
}
