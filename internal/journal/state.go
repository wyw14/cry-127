package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type SnapshotStore struct {
	mu  sync.Mutex
	dir string
}

func NewSnapshotStore(dir string) (*SnapshotStore, error) {
	if dir == "" {
		return nil, errors.New("snapshot directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create snapshot directory: %w", err)
	}
	return &SnapshotStore{dir: dir}, nil
}

func (s *SnapshotStore) Save(name string, value any) error {
	if name == "" {
		return errors.New("snapshot name is required")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	target := filepath.Join(s.dir, name+".json")
	temporary := target + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open snapshot temporary file: %w", err)
	}
	if _, err = file.Write(encoded); err != nil {
		file.Close()
		return fmt.Errorf("write snapshot: %w", err)
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync snapshot: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close snapshot: %w", err)
	}
	if err = os.Rename(temporary, target); err != nil {
		return fmt.Errorf("rename snapshot: %w", err)
	}
	return nil
}

func (s *SnapshotStore) Load(name string, destination any) (bool, error) {
	if name == "" {
		return false, errors.New("snapshot name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := os.ReadFile(filepath.Join(s.dir, name+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read snapshot: %w", err)
	}
	if err := json.Unmarshal(encoded, destination); err != nil {
		return false, fmt.Errorf("decode snapshot: %w", err)
	}
	return true, nil
}
