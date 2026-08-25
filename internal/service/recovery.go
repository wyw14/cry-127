package service

import (
	"errors"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/batch"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/door"
	"github.com/wyw14/cry-127/internal/heater"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/recipe"
	"github.com/wyw14/cry-127/internal/vacuum"
)

func (r *Runtime) Snapshot() error {
	operationIDs := make([]string, 0, len(r.batchStore.List()))
	for _, operation := range r.batchStore.List() {
		operationIDs = append(operationIDs, operation.ID)
	}
	return journal.MergeErrors(
		batch.Save(r.snapshots, r.batchStore),
		recipe.Save(r.snapshots, r.recipes),
		chamber.Save(r.snapshots, r.chamberRegistry),
		atmosphere.Save(r.snapshots, r.atmosphereStates),
		vacuum.SaveRecovery(r.snapshots, r.vacuumController, r.vacuumProofs, operationIDs),
		door.Save(r.snapshots, r.doorRegistry),
		heater.SaveRecovery(r.snapshots, r.windows, r.heater, operationIDs),
		interlock.SaveRecovery(r.snapshots, r.interlockRecovery),
	)
}

func (r *Runtime) Recover() error {
	generation := r.interlockRecovery.Begin()
	if _, err := vacuum.RestoreRecovery(r.snapshots, r.vacuumController, r.vacuumProofs); err != nil {
		return err
	}
	if _, err := chamber.Restore(r.snapshots, r.chamberRegistry); err != nil {
		return err
	}
	if _, err := atmosphere.Restore(r.snapshots, r.atmosphereStates); err != nil {
		return err
	}
	if _, err := door.Restore(r.snapshots, r.doorRegistry); err != nil {
		return err
	}
	for _, operation := range r.batchStore.Active() {
		if operation.Stage == "heating" || operation.Stage == "soaking" {
			if err := vacuum.HeatingEligibility(r.vacuumController, operation, time.Now().UTC()); err != nil {
				r.heater.Stop(operation.ID, r.now())
			}
		}
	}
	if _, err := heater.RestoreRecovery(r.snapshots, r.windows, r.heater); err != nil {
		return err
	}
	return r.interlockRecovery.Complete(generation)
}

func (r *Runtime) RecoveryReady() bool {
	generation := r.interlockRecovery.Begin()
	if err := r.interlockRecovery.Complete(generation); err != nil {
		return false
	}
	return r.interlockRecovery.Ready(generation)
}

func (r *Runtime) ValidatePersistentState() error {
	events, err := r.events.Events()
	if err != nil {
		return err
	}
	replay, err := journal.Rebuild(events)
	if err != nil {
		return err
	}
	if len(replay.Operations) < len(r.batchStore.List()) {
		return errors.New("journal contains fewer operations than runtime")
	}
	journal.OperationList(replay)
	if revision, err := r.Recipe(r.defaultRecipe); err == nil {
		if _, err := recipe.Encode(revision); err != nil {
			return err
		}
	}
	return nil
}
