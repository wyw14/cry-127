package service

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/wyw14/cry-127/internal/atmosphere"
	"github.com/wyw14/cry-127/internal/batch"
	"github.com/wyw14/cry-127/internal/chamber"
	"github.com/wyw14/cry-127/internal/door"
	"github.com/wyw14/cry-127/internal/heater"
	"github.com/wyw14/cry-127/internal/interlock"
	"github.com/wyw14/cry-127/internal/journal"
	"github.com/wyw14/cry-127/internal/model"
	"github.com/wyw14/cry-127/internal/quench"
	"github.com/wyw14/cry-127/internal/recipe"
	"github.com/wyw14/cry-127/internal/sensor"
	"github.com/wyw14/cry-127/internal/vacuum"
)

type Runtime struct {
	mu                sync.RWMutex
	dataDir           string
	now               func() time.Time
	events            *journal.Store
	snapshots         *journal.SnapshotStore
	recipes           *recipe.Registry
	evidence          *recipe.EvidenceBook
	calibrations      *sensor.CalibrationBook
	receiver          *sensor.Receiver
	windows           *heater.WindowBook
	heater            *heater.Coordinator
	heaterIntegration *heater.Integrator
	batches           *batch.Coordinator
	batchStore        *batch.Store
	progress          *batch.Progressor
	vacuumController  *vacuum.Controller
	vacuumProofs      *vacuum.ProofBook
	vacuum            *vacuum.Coordinator
	chamberRegistry   *chamber.Registry
	chambers          *chamber.Controller
	atmosphereStates  *atmosphere.StateStore
	valves            *atmosphere.ValveBank
	atmosphere        *atmosphere.Coordinator
	interlockArbiter  *interlock.Arbiter
	interlocks        *interlock.Integration
	interlockRecovery *interlock.Recovery
	doorRegistry      *door.Registry
	doors             *door.Coordinator
	leases            *quench.LeaseManager
	workers           *quench.WorkerBook
	quench            *quench.Coordinator
	incidents         []model.Incident
	defaultRecipe     string
}

func NewRuntime(dataDir string, now func() time.Time) (*Runtime, error) {
	if dataDir == "" || now == nil {
		return nil, errors.New("runtime data directory and clock are required")
	}
	events, err := journal.Open(filepath.Join(dataDir, "journal"))
	if err != nil {
		return nil, err
	}
	snapshots, err := journal.NewSnapshotStore(filepath.Join(dataDir, "snapshots"))
	if err != nil {
		return nil, err
	}
	recipes, err := recipe.Restore(snapshots)
	if err != nil {
		return nil, err
	}
	if len(recipes.All()) == 0 {
		if _, err := recipes.Create("standard-carburize", 900, 8, 50*time.Millisecond, []string{"load-a", "load-b", "load-c"}, now()); err != nil {
			return nil, err
		}
	}
	batchStore, err := batch.Reconcile(events, snapshots)
	if err != nil {
		return nil, err
	}
	batches, err := batch.NewCoordinator(batchStore, recipes, events, now)
	if err != nil {
		return nil, err
	}
	progress, err := batch.NewProgressor(batches, events)
	if err != nil {
		return nil, err
	}
	calibrations := sensor.NewCalibrationBook()
	for _, probeID := range []string{"load-a", "load-b", "load-c"} {
		if _, err := calibrations.Calibrate(probeID, 0, 1, now()); err != nil {
			return nil, err
		}
	}
	receiver, err := sensor.NewReceiver(calibrations, events)
	if err != nil {
		return nil, err
	}
	windows := heater.NewWindowBook()
	heaterCoordinator, err := heater.NewCoordinator(windows)
	if err != nil {
		return nil, err
	}
	heaterIntegration, err := heater.NewIntegrator(receiver, windows)
	if err != nil {
		return nil, err
	}
	vacuumController := vacuum.NewController()
	vacuumProofs := vacuum.NewProofBook()
	traceWriter, err := vacuum.NewFileTraceWriter(filepath.Join(dataDir, "leak-traces"))
	if err != nil {
		return nil, err
	}
	vacuumCoordinator, err := vacuum.NewCoordinator(vacuumController, vacuumProofs, traceWriter, events)
	if err != nil {
		return nil, err
	}
	chamberRegistry := chamber.NewRegistry()
	chambers, err := chamber.NewController(chamberRegistry, events)
	if err != nil {
		return nil, err
	}
	atmosphereStates := atmosphere.NewStateStore()
	valves, err := atmosphere.NewValveBank([]string{"nitrogen", "methane", "argon"})
	if err != nil {
		return nil, err
	}
	atmosphereCoordinator, err := atmosphere.NewCoordinator(atmosphereStates, valves, chambers, events)
	if err != nil {
		return nil, err
	}
	interlockArbiter, err := interlock.NewArbiter(events)
	if err != nil {
		return nil, err
	}
	interlocks, err := interlock.NewIntegration(interlockArbiter)
	if err != nil {
		return nil, err
	}
	doorRegistry := door.NewRegistry()
	doors, err := door.NewCoordinator(doorRegistry, atmosphereStates, chambers, interlocks, events)
	if err != nil {
		return nil, err
	}
	leases, err := quench.NewLeaseManager([]string{"quench-vessel-1", "high-pressure-header"})
	if err != nil {
		return nil, err
	}
	workers := quench.NewWorkerBook()
	quenchCoordinator, err := quench.NewCoordinator(leases, workers, atmosphereStates, chambers, interlocks, events, now)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		dataDir: dataDir, now: now, events: events, snapshots: snapshots,
		recipes: recipes, evidence: recipe.NewEvidenceBook(), calibrations: calibrations,
		receiver: receiver, windows: windows, heater: heaterCoordinator, heaterIntegration: heaterIntegration,
		batches: batches, batchStore: batchStore, progress: progress,
		vacuumController: vacuumController, vacuumProofs: vacuumProofs, vacuum: vacuumCoordinator,
		chamberRegistry: chamberRegistry, chambers: chambers,
		atmosphereStates: atmosphereStates, valves: valves, atmosphere: atmosphereCoordinator,
		interlockArbiter: interlockArbiter, interlocks: interlocks, interlockRecovery: interlock.NewRecovery(),
		doorRegistry: doorRegistry, doors: doors, leases: leases, workers: workers, quench: quenchCoordinator,
	}
	runtime.defaultRecipe = recipes.All()[0].ID
	if err := runtime.Recover(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) DataDirectory() string {
	return r.dataDir
}
