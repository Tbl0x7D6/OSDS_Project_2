// Package pow implements the Proof of Work consensus algorithm
package pow

import (
	"blockchain/pkg/block"
	"context"
	"strings"
	"sync/atomic"
)

// ProofOfWork represents the PoW mining algorithm
type ProofOfWork struct {
	Block      *block.Block
	Difficulty int
}

// MiningResult represents the result of a mining operation
type MiningResult struct {
	Block   *block.Block
	Success bool
	Nonce   int64
}

// NewProofOfWork creates a new PoW instance for a block
func NewProofOfWork(b *block.Block) *ProofOfWork {
	return &ProofOfWork{
		Block:      b,
		Difficulty: b.Difficulty,
	}
}

// GetTarget returns the target prefix string for the given difficulty
func GetTarget(difficulty int) string {
	return strings.Repeat("0", difficulty)
}

// Mine performs the mining operation to find a valid nonce
// It returns when a valid nonce is found or the context is cancelled
func (pow *ProofOfWork) Mine(ctx context.Context) *MiningResult {
	var nonce int64 = 0
	target := GetTarget(pow.Difficulty)

	for {
		select {
		case <-ctx.Done():
			// Mining was cancelled
			return &MiningResult{
				Block:   pow.Block,
				Success: false,
				Nonce:   nonce,
			}
		default:
			pow.Block.Nonce = nonce
			hash := pow.Block.CalculateHash()

			if strings.HasPrefix(hash, target) {
				pow.Block.Hash = hash
				return &MiningResult{
					Block:   pow.Block,
					Success: true,
					Nonce:   nonce,
				}
			}
			nonce++
		}
	}
}

// MineWithCallback performs mining with periodic callback for progress reporting
func (pow *ProofOfWork) MineWithCallback(ctx context.Context, callback func(nonce int64)) *MiningResult {
	var nonce int64 = 0
	target := GetTarget(pow.Difficulty)
	reportInterval := int64(100000) // Report every 100k attempts

	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return &MiningResult{
					Block:   pow.Block,
					Success: false,
					Nonce:   nonce,
				}
			default:
			}
		}

		pow.Block.Nonce = nonce
		hash := pow.Block.CalculateHash()

		if strings.HasPrefix(hash, target) {
			pow.Block.Hash = hash
			return &MiningResult{
				Block:   pow.Block,
				Success: true,
				Nonce:   nonce,
			}
		}

		if callback != nil && nonce%reportInterval == 0 {
			callback(nonce)
		}
		nonce++
	}
}

// MineParallel performs parallel mining using multiple goroutines
func (pow *ProofOfWork) MineParallel(ctx context.Context, workers int) *MiningResult {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultChan := make(chan *MiningResult, workers)
	var found int32 = 0

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			// Each worker starts from different nonce to avoid duplication
			var nonce int64 = int64(workerID)
			target := GetTarget(pow.Difficulty)

			// Create a copy of the block for this worker
			workerBlock := pow.Block.Clone()

			for {
				// Check if someone else found the solution
				if atomic.LoadInt32(&found) == 1 {
					return
				}

				select {
				case <-ctx.Done():
					return
				default:
					workerBlock.Nonce = nonce
					hash := workerBlock.CalculateHash()

					if strings.HasPrefix(hash, target) {
						// Found a valid solution
						if atomic.CompareAndSwapInt32(&found, 0, 1) {
							workerBlock.Hash = hash
							resultChan <- &MiningResult{
								Block:   workerBlock,
								Success: true,
								Nonce:   nonce,
							}
							cancel()
						}
						return
					}
					nonce += int64(workers) // Skip nonces handled by other workers
				}
			}
		}(i)
	}

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return &MiningResult{
			Block:   pow.Block,
			Success: false,
			Nonce:   0,
		}
	}
}

// Validate checks if a block has a valid proof of work
func Validate(b *block.Block) bool {
	if b.Hash != b.CalculateHash() {
		return false
	}
	target := GetTarget(b.Difficulty)
	return strings.HasPrefix(b.Hash, target)
}

// ValidateHash checks if a hash meets the difficulty requirement
func ValidateHash(hash string, difficulty int) bool {
	target := GetTarget(difficulty)
	return strings.HasPrefix(hash, target)
}
