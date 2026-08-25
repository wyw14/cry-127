package atmosphere

import (
	"errors"
	"fmt"
	"sync"
)

type ValveBank struct {
	mu    sync.RWMutex
	lines map[string]bool
}

func NewValveBank(lines []string) (*ValveBank, error) {
	if len(lines) == 0 {
		return nil, errors.New("at least one gas line is required")
	}
	bank := &ValveBank{lines: map[string]bool{}}
	for _, line := range lines {
		if line == "" {
			return nil, errors.New("gas line identity is required")
		}
		bank.lines[line] = false
	}
	return bank, nil
}

func (b *ValveBank) Open(line string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.lines[line]; !ok {
		return fmt.Errorf("gas line %s does not exist", line)
	}
	for other, opened := range b.lines {
		if other != line && opened {
			return fmt.Errorf("gas line %s is still open", other)
		}
	}
	b.lines[line] = true
	return nil
}

func (b *ValveBank) Close(line string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.lines[line]; !ok {
		return fmt.Errorf("gas line %s does not exist", line)
	}
	b.lines[line] = false
	return nil
}

func (b *ValveBank) OpenLines() []string {
	b.mu.RLock()
	result := []string{}
	for line, opened := range b.lines {
		if opened {
			result = append(result, line)
		}
	}
	b.mu.RUnlock()
	return result
}
