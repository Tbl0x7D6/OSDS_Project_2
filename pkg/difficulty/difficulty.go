// Package difficulty implements dynamic difficulty adjustment for the blockchain
package difficulty

import (
	"blockchain/pkg/block"
	"sync"
	"time"
)

const (
	// TargetBlockTime is the target time between blocks (10 seconds)
	TargetBlockTime = 10 * time.Second

	// AdjustmentInterval is the number of blocks between difficulty adjustments
	// Check every 6 blocks (approximately 1 minute at target rate)
	AdjustmentInterval = 6

	// MinDifficulty is the minimum allowed difficulty
	MinDifficulty = 1

	// MaxDifficulty is the maximum allowed difficulty
	MaxDifficulty = 32

	// MaxAdjustmentFactor limits how much difficulty can change at once
	// This prevents drastic changes
	MaxAdjustmentFactor = 2.0
)

// DifficultyAdjuster handles dynamic difficulty adjustment
type DifficultyAdjuster struct {
	enabled           bool
	currentDifficulty int
	mu                sync.RWMutex
}

// NewDifficultyAdjuster creates a new difficulty adjuster
func NewDifficultyAdjuster(initialDifficulty int, enabled bool) *DifficultyAdjuster {
	return &DifficultyAdjuster{
		enabled:           enabled,
		currentDifficulty: initialDifficulty,
	}
}

// IsEnabled returns whether dynamic difficulty is enabled
func (da *DifficultyAdjuster) IsEnabled() bool {
	da.mu.RLock()
	defer da.mu.RUnlock()
	return da.enabled
}

// SetEnabled enables or disables dynamic difficulty adjustment
func (da *DifficultyAdjuster) SetEnabled(enabled bool) {
	da.mu.Lock()
	defer da.mu.Unlock()
	da.enabled = enabled
}

// GetDifficulty returns the current difficulty
func (da *DifficultyAdjuster) GetDifficulty() int {
	da.mu.RLock()
	defer da.mu.RUnlock()
	return da.currentDifficulty
}

// SetDifficulty sets the current difficulty
func (da *DifficultyAdjuster) SetDifficulty(difficulty int) {
	da.mu.Lock()
	defer da.mu.Unlock()
	da.currentDifficulty = clampDifficulty(difficulty)
}

// ShouldAdjust returns true if it is time to adjust difficulty based on block index
func ShouldAdjust(blockIndex int64) bool {
	return blockIndex > 0 && blockIndex%AdjustmentInterval == 0
}

// CalculateNewDifficulty calculates the new difficulty based on recent block times
// blocks should be the last AdjustmentInterval blocks
func CalculateNewDifficulty(blocks []*block.Block, currentDifficulty int) int {
	if len(blocks) < 2 {
		return currentDifficulty
	}

	// Calculate actual time taken for these blocks
	firstBlock := blocks[0]
	lastBlock := blocks[len(blocks)-1]

	// Block timestamps are in nanoseconds
	actualTime := time.Duration(lastBlock.Timestamp - firstBlock.Timestamp)

	// Expected time for these blocks
	expectedTime := time.Duration(len(blocks)-1) * TargetBlockTime

	if actualTime <= 0 {
		// If blocks were mined too fast (same timestamp), increase difficulty
		return clampDifficulty(currentDifficulty + 1)
	}

	// Calculate adjustment ratio
	// If blocks are mined too fast (actualTime < expectedTime), increase difficulty
	// If blocks are mined too slow (actualTime > expectedTime), decrease difficulty
	ratio := float64(expectedTime) / float64(actualTime)

	// Clamp the ratio to prevent drastic changes
	if ratio > MaxAdjustmentFactor {
		ratio = MaxAdjustmentFactor
	}
	if ratio < 1.0/MaxAdjustmentFactor {
		ratio = 1.0 / MaxAdjustmentFactor
	}

	// Calculate new difficulty
	// For bit-based difficulty, we adjust by 1 bit at a time
	var newDifficulty int
	if ratio > 1.2 {
		// Blocks are being mined too fast, increase difficulty
		newDifficulty = currentDifficulty + 1
	} else if ratio < 0.8 {
		// Blocks are being mined too slow, decrease difficulty
		newDifficulty = currentDifficulty - 1
	} else {
		// Within acceptable range, no change
		newDifficulty = currentDifficulty
	}

	return clampDifficulty(newDifficulty)
}

// CalculateAverageBlockTime calculates the average time between blocks
func CalculateAverageBlockTime(blocks []*block.Block) time.Duration {
	if len(blocks) < 2 {
		return 0
	}

	firstBlock := blocks[0]
	lastBlock := blocks[len(blocks)-1]

	totalTime := time.Duration(lastBlock.Timestamp - firstBlock.Timestamp)
	blockCount := len(blocks) - 1

	if blockCount <= 0 {
		return 0
	}

	return totalTime / time.Duration(blockCount)
}

// clampDifficulty ensures difficulty stays within allowed bounds
func clampDifficulty(difficulty int) int {
	if difficulty < MinDifficulty {
		return MinDifficulty
	}
	if difficulty > MaxDifficulty {
		return MaxDifficulty
	}
	return difficulty
}

// GetBlocksPerMinute calculates the blocks per minute rate based on recent blocks
func GetBlocksPerMinute(blocks []*block.Block) float64 {
	if len(blocks) < 2 {
		return 0
	}

	firstBlock := blocks[0]
	lastBlock := blocks[len(blocks)-1]

	totalTime := time.Duration(lastBlock.Timestamp - firstBlock.Timestamp)
	if totalTime <= 0 {
		return 0
	}

	blockCount := len(blocks) - 1
	minutes := totalTime.Minutes()
	if minutes <= 0 {
		return 0
	}

	return float64(blockCount) / minutes
}

// AdjustmentInfo contains information about a difficulty adjustment
type AdjustmentInfo struct {
	OldDifficulty   int
	NewDifficulty   int
	ActualBlockTime time.Duration
	TargetBlockTime time.Duration
	BlocksAnalyzed  int
}

// CalculateAdjustment calculates the difficulty adjustment and returns detailed info
func CalculateAdjustment(blocks []*block.Block, currentDifficulty int) *AdjustmentInfo {
	info := &AdjustmentInfo{
		OldDifficulty:   currentDifficulty,
		TargetBlockTime: TargetBlockTime,
		BlocksAnalyzed:  len(blocks),
	}

	if len(blocks) < 2 {
		info.NewDifficulty = currentDifficulty
		return info
	}

	info.ActualBlockTime = CalculateAverageBlockTime(blocks)
	info.NewDifficulty = CalculateNewDifficulty(blocks, currentDifficulty)

	return info
}
