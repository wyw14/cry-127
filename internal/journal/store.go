package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          string          `json:"id"`
	OperationID string          `json:"operation_id"`
	Kind        string          `json:"kind"`
	Payload     json.RawMessage `json:"payload"`
	RecordedAt  time.Time       `json:"recorded_at"`
}

type Store struct {
	mu       sync.Mutex
	dir      string
	eventLog string
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("journal directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create journal directory: %w", err)
	}
	return &Store{dir: dir, eventLog: filepath.Join(dir, "events.jsonl")}, nil
}

func (s *Store) Append(operationID, kind string, value any, now time.Time) (Event, error) {
	if operationID == "" || kind == "" {
		return Event{}, errors.New("event identity is incomplete")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return Event{}, fmt.Errorf("encode event payload: %w", err)
	}
	event := Event{ID: uuid.NewString(), OperationID: operationID, Kind: kind, Payload: payload, RecordedAt: now.UTC()}
	encoded, err := json.Marshal(event)
	if err != nil {
		return Event{}, fmt.Errorf("encode event: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.OpenFile(s.eventLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Event{}, fmt.Errorf("open event log: %w", err)
	}
	if _, err = file.Write(append(encoded, '\n')); err != nil {
		file.Close()
		return Event{}, fmt.Errorf("append event: %w", err)
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return Event{}, fmt.Errorf("sync event: %w", err)
	}
	if err = file.Close(); err != nil {
		return Event{}, fmt.Errorf("close event log: %w", err)
	}
	return event, nil
}

func (s *Store) Events() ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.Open(s.eventLog)
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()
	result := []Event{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		result = append(result, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan event log: %w", err)
	}
	return result, nil
}
