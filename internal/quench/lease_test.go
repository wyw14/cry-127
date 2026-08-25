package quench

import (
	"testing"
	"time"

	"github.com/wyw14/cry-127/internal/model"
)

// TestReleaseRejectsRetiredCycle reproduces the shared-header handoff bug:
// furnace-2 has taken over the header and is pressurizing, then a late
// completion message from furnace-1's retired cycle must not free the header
// that furnace-2 currently holds. A retired cycle may only release its own
// lease, never the lease the current cycle is using.
func TestReleaseRejectsRetiredCycle(t *testing.T) {
	manager, err := NewLeaseManager([]string{"quench-vessel-1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()

	// Furnace-1 acquires, then completes and releases correctly.
	furnaceOne, err := manager.Acquire("quench-vessel-1", "operation-furnace-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Release(furnaceOne, now); err != nil {
		t.Fatal(err)
	}

	// Furnace-2 takes over the header and begins pressurizing.
	furnaceTwo, err := manager.Acquire("quench-vessel-1", "operation-furnace-2", now)
	if err != nil {
		t.Fatal(err)
	}

	// The late completion message from furnace-1's retired cycle arrives.
	// It must not release the header furnace-2 currently holds.
	err = manager.Release(furnaceOne, now.Add(time.Second))
	if err == nil {
		t.Fatal("retired cycle released the current cycle's header")
	}

	// Furnace-2 must still be the active, pressurizing owner.
	current, busy := manager.Current("quench-vessel-1")
	if !busy {
		t.Fatal("header was freed by a retired cycle")
	}
	if current.ID != furnaceTwo.ID || current.Fence != furnaceTwo.Fence {
		t.Fatalf("header ownership changed: want furnace-2 lease %s fence %d, got %s fence %d",
			furnaceTwo.ID, furnaceTwo.Fence, current.ID, current.Fence)
	}
	if !current.ReleasedAt.IsZero() {
		t.Fatal("current cycle lease was marked released by a retired cycle")
	}

	// Furnace-3 must be denied access while furnace-2 still holds the header.
	if _, err := manager.Acquire("quench-vessel-1", "operation-furnace-3", now); err == nil {
		t.Fatal("furnace-3 was granted access to an occupied header")
	}

	// Furnace-2 releasing its own lease still works and frees the header.
	if err := manager.Release(furnaceTwo, now.Add(2*time.Second)); err != nil {
		t.Fatalf("current cycle could not release its own lease: %v", err)
	}
	if _, busy := manager.Current("quench-vessel-1"); busy {
		t.Fatal("header remained busy after the current cycle released")
	}
}

// TestReleaseRejectsDuplicateAndForeignLeases verifies the owner check on the
// remaining handoff failure modes: a duplicate release of an already-freed
// lease and a release for a resource the caller never held must both be
// rejected without disturbing a later cycle's ownership.
func TestReleaseRejectsDuplicateAndForeignLeases(t *testing.T) {
	manager, err := NewLeaseManager([]string{"quench-vessel-1", "high-pressure-header"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()

	// Furnace-1 acquires and completes its cycle.
	first, err := manager.Acquire("quench-vessel-1", "operation-first", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Release(first, now); err != nil {
		t.Fatal(err)
	}

	// A duplicate completion for the already-freed lease must be rejected.
	if err := manager.Release(first, now.Add(time.Second)); err == nil {
		t.Fatal("duplicate release of an already-freed lease succeeded")
	}

	// Furnace-2 takes over the header while furnace-1's duplicate is ignored.
	second, err := manager.Acquire("quench-vessel-1", "operation-second", now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	// A foreign lease (forged identity) for an unheld resource must be rejected.
	foreign := model.ResourceLease{ID: "foreign-lease", ResourceID: "high-pressure-header", OperationID: "operation-foreign", Fence: 99}
	if err := manager.Release(foreign, now); err == nil {
		t.Fatal("foreign lease released an unheld header")
	}

	// Furnace-2 must still be the active owner and can release its own lease.
	current, busy := manager.Current("quench-vessel-1")
	if !busy || current.ID != second.ID {
		t.Fatal("active lease was disturbed by rejected releases")
	}
	if err := manager.Release(second, now.Add(3*time.Second)); err != nil {
		t.Fatalf("active cycle could not release its own lease: %v", err)
	}
}

// TestAcquireRespectsActiveLease is a sanity check that acquiring an occupied
// header is rejected, which is the guard the retired-cycle fix relies on.
func TestAcquireRespectsActiveLease(t *testing.T) {
	manager, err := NewLeaseManager([]string{"quench-vessel-1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := manager.Acquire("quench-vessel-1", "operation-a", now); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Acquire("quench-vessel-1", "operation-b", now)
	if err == nil {
		t.Fatal("second acquire on occupied header succeeded")
	}
	if err.Error() != "quench resource is busy" {
		t.Fatalf("unexpected acquire error: %v", err)
	}
}
