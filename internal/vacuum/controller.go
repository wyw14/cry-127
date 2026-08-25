package vacuum

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Topology struct {
	OperationID   string    `json:"operation_id"`
	SealRevision  string    `json:"seal_revision"`
	RoughingOpen  bool      `json:"roughing_open"`
	DiffusionOpen bool      `json:"diffusion_open"`
	VentOpen      bool      `json:"vent_open"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Controller struct {
	mu         sync.RWMutex
	topologies map[string]Topology
}

func NewController() *Controller {
	return &Controller{topologies: map[string]Topology{}}
}

func (c *Controller) Configure(topology Topology) error {
	if topology.OperationID == "" || topology.SealRevision == "" {
		return errors.New("vacuum topology identity is incomplete")
	}
	if topology.VentOpen && (topology.RoughingOpen || topology.DiffusionOpen) {
		return errors.New("vent cannot open while a pump path is connected")
	}
	c.mu.Lock()
	c.topologies[topology.OperationID] = topology
	c.mu.Unlock()
	return nil
}

func (c *Controller) Current(operationID string) (Topology, bool) {
	c.mu.RLock()
	topology, ok := c.topologies[operationID]
	c.mu.RUnlock()
	return topology, ok
}

func (c *Controller) SafeForHeating(operationID, sealRevision string) bool {
	topology, ok := c.Current(operationID)
	return ok && topology.SealRevision == sealRevision && !topology.VentOpen && (topology.RoughingOpen || topology.DiffusionOpen)
}

type TracePoint struct {
	At         time.Time `json:"at"`
	PressurePa float64   `json:"pressure_pa"`
	LeakRate   float64   `json:"leak_rate"`
}

type TraceWriter interface {
	Write(operationID string, points []TracePoint) (string, error)
}

type FileTraceWriter struct {
	dir string
}

func NewFileTraceWriter(dir string) (*FileTraceWriter, error) {
	if dir == "" {
		return nil, errors.New("trace directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create trace directory: %w", err)
	}
	return &FileTraceWriter{dir: dir}, nil
}

func (w *FileTraceWriter) Write(operationID string, points []TracePoint) (string, error) {
	if operationID == "" || len(points) == 0 {
		return "", errors.New("trace identity and points are required")
	}
	encoded, err := json.Marshal(points)
	if err != nil {
		return "", fmt.Errorf("encode leak trace: %w", err)
	}
	target := filepath.Join(w.dir, operationID+".json")
	temporary := target + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open leak trace: %w", err)
	}
	if _, err = file.Write(encoded); err != nil {
		file.Close()
		return "", fmt.Errorf("write leak trace: %w", err)
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return "", fmt.Errorf("sync leak trace: %w", err)
	}
	if err = file.Close(); err != nil {
		return "", fmt.Errorf("close leak trace: %w", err)
	}
	if err = os.Rename(temporary, target); err != nil {
		return "", fmt.Errorf("rename leak trace: %w", err)
	}
	return target, nil
}
