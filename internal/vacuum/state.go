package vacuum

import (
	"errors"
	"sync"

	"github.com/wyw14/cry-127/internal/model"
)

type ProofBook struct {
	mu     sync.RWMutex
	proofs map[string]model.VacuumProof
}

func NewProofBook() *ProofBook {
	return &ProofBook{proofs: map[string]model.VacuumProof{}}
}

func (b *ProofBook) Record(proof model.VacuumProof) error {
	if proof.OperationID == "" || proof.SealRevision == "" || proof.ProofID == "" {
		return errors.New("vacuum proof identity is incomplete")
	}
	if !proof.Durable {
		return errors.New("vacuum proof must be durable")
	}
	b.mu.Lock()
	b.proofs[proof.OperationID] = proof
	b.mu.Unlock()
	return nil
}

func (b *ProofBook) Current(operationID, sealRevision string) (model.VacuumProof, bool) {
	b.mu.RLock()
	proof, ok := b.proofs[operationID]
	b.mu.RUnlock()
	if !ok || proof.SealRevision != sealRevision || !proof.Durable {
		return model.VacuumProof{}, false
	}
	return proof, true
}

func (b *ProofBook) Invalidate(operationID string) bool {
	b.mu.Lock()
	_, ok := b.proofs[operationID]
	delete(b.proofs, operationID)
	b.mu.Unlock()
	return ok
}

func (b *ProofBook) All() []model.VacuumProof {
	b.mu.RLock()
	result := make([]model.VacuumProof, 0, len(b.proofs))
	for _, proof := range b.proofs {
		result = append(result, proof)
	}
	b.mu.RUnlock()
	return result
}
