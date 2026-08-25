package heater

import (
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/recipe"
)

// TestReadyRejectsAcknowledgementsFromPriorRecipeGeneration locks in the
// soak-confirmation ownership fix: a "ready" receipt that belongs to an old
// recipe generation must not satisfy a Ready() call made for a new revision.
//
// Field scenario: an operator field-revises the recipe and restarts the soak
// while zone-b has not yet reached the new target. The historical ready
// receipts (recorded against the previous revision) must not advance the new
// soak window.
func TestReadyRejectsAcknowledgementsFromPriorRecipeGeneration(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	book := NewWindowBook()
	coordinator, err := NewCoordinator(book)
	if err != nil {
		t.Fatalf("new coordinator: %v", err)
	}
	zones := []string{"zone-a", "zone-b", "zone-c"}

	// Original recipe generation rev-a; the operation is bound to it.
	revA := recipe.Revision{ID: "rev-a", ToleranceC: 8, ZoneTargets: map[string]float64{
		"zone-a": 900, "zone-b": 900, "zone-c": 900,
	}}
	operation := model.Operation{
		ID:             "op-1",
		FurnaceID:      "furnace-a",
		RecipeRevision: revA.ID,
		Stage:          model.StageHeating,
		SealRevision:   "seal-1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// All three zones reached the original target under rev-a.
	for _, zone := range zones {
		if _, err := coordinator.Acknowledge(operation, revA, zone, true, now); err != nil {
			t.Fatalf("acknowledge %s under rev-a: %v", zone, err)
		}
	}
	if !coordinator.Ready(operation.ID, revA.ID, zones) {
		t.Fatal("expected zones ready under the owning revision rev-a")
	}

	// Field revision: a new recipe generation is minted and bound to the same
	// operation, but the thermocouples have not been re-evaluated against the
	// new target. The stale rev-a receipts must not satisfy the new window.
	revB := recipe.Revision{ID: "rev-b", ToleranceC: 8, ZoneTargets: map[string]float64{
		"zone-a": 940, "zone-b": 940, "zone-c": 940,
	}}
	updated := operation
	updated.RecipeRevision = revB.ID
	updated.UpdatedAt = now

	if coordinator.Ready(updated.ID, revB.ID, zones) {
		t.Fatal("stale rev-a ready receipts must not satisfy the new rev-b soak window")
	}

	// Only once each zone is freshly acknowledged against the new revision may
	// the new window advance.
	for _, zone := range zones {
		if _, err := coordinator.Acknowledge(updated, revB, zone, true, now.Add(time.Second)); err != nil {
			t.Fatalf("acknowledge %s under rev-b: %v", zone, err)
		}
	}
	if !coordinator.Ready(updated.ID, revB.ID, zones) {
		t.Fatal("expected zones ready once re-acknowledged under rev-b")
	}

	// A mixed generation (one zone still on the old revision) is not ready.
	if _, err := coordinator.Acknowledge(updated, revB, "zone-a", true, now.Add(2*time.Second)); err != nil {
		t.Fatalf("acknowledge zone-a under rev-b: %v", err)
	}
	if _, err := coordinator.Acknowledge(updated, revB, "zone-b", true, now.Add(2*time.Second)); err != nil {
		t.Fatalf("acknowledge zone-b under rev-b: %v", err)
	}
	// zone-c left carrying the prior rev-b ack only is fine here, but dropping
	// it back to not-ready must invalidate readiness.
	if _, err := coordinator.Acknowledge(updated, revB, "zone-c", false, now.Add(2*time.Second)); err != nil {
		t.Fatalf("clear zone-c under rev-b: %v", err)
	}
	if coordinator.Ready(updated.ID, revB.ID, zones) {
		t.Fatal("expected readiness to drop when a zone clears under the new revision")
	}
}

// TestReadyRequiresRevisionAndZones guards the identity preconditions so the
// ownership check cannot be bypassed with an empty revision or zone set.
func TestReadyRequiresRevisionAndZones(t *testing.T) {
	book := NewWindowBook()
	coordinator, err := NewCoordinator(book)
	if err != nil {
		t.Fatalf("new coordinator: %v", err)
	}
	rev := recipe.Revision{ID: "rev-x", ZoneTargets: map[string]float64{"zone-a": 900}}
	operation := model.Operation{
		ID: "op-2", FurnaceID: "furnace-a", RecipeRevision: rev.ID,
		Stage: model.StageHeating, SealRevision: "seal-2",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if _, err := coordinator.Acknowledge(operation, rev, "zone-a", true, time.Now()); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if coordinator.Ready(operation.ID, "", []string{"zone-a"}) {
		t.Fatal("Ready must reject an empty revision id")
	}
	if coordinator.Ready(operation.ID, rev.ID, nil) {
		t.Fatal("Ready must reject an empty zone set")
	}
}
