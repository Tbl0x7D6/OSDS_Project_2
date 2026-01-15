package difficulty

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"testing"
	"time"
)

// Helper function to create test blocks with specific timestamps
func createTestBlock(index int64, timestamp int64, difficulty int) *block.Block {
	tx := transaction.NewCoinbaseTransaction("miner1", 50, index)
	txs := []*transaction.Transaction{tx}
	b := &block.Block{
		Index:        index,
		Timestamp:    timestamp,
		Transactions: txs,
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Hash:         "0000000000000000000000000000000000000000000000000000000000000001",
		Nonce:        0,
		Difficulty:   difficulty,
		MinerID:      "miner1",
	}
	return b
}

// Helper function to create a series of test blocks with given intervals
func createTestBlocks(count int, intervalSeconds int64, startDifficulty int) []*block.Block {
	blocks := make([]*block.Block, count)
	baseTime := time.Now().UnixNano()

	for i := 0; i < count; i++ {
		timestamp := baseTime + int64(i)*intervalSeconds*int64(time.Second)
		blocks[i] = createTestBlock(int64(i), timestamp, startDifficulty)
	}
	return blocks
}

func TestNewDifficultyAdjuster(t *testing.T) {
	da := NewDifficultyAdjuster(10, true)

	if da.GetDifficulty() != 10 {
		t.Errorf("Expected difficulty 10, got %d", da.GetDifficulty())
	}

	if !da.IsEnabled() {
		t.Error("Expected dynamic difficulty to be enabled")
	}
}

func TestDifficultyAdjusterEnableDisable(t *testing.T) {
	da := NewDifficultyAdjuster(10, false)

	if da.IsEnabled() {
		t.Error("Expected dynamic difficulty to be disabled initially")
	}

	da.SetEnabled(true)
	if !da.IsEnabled() {
		t.Error("Expected dynamic difficulty to be enabled after SetEnabled(true)")
	}

	da.SetEnabled(false)
	if da.IsEnabled() {
		t.Error("Expected dynamic difficulty to be disabled after SetEnabled(false)")
	}
}

func TestShouldAdjust(t *testing.T) {
	tests := []struct {
		blockIndex int64
		expected   bool
	}{
		{0, false},   // Genesis block, do not adjust
		{1, false},   // Too early
		{5, false},   // Not at interval
		{6, true},    // At interval (6 blocks)
		{12, true},   // At interval
		{18, true},   // At interval
		{7, false},   // Not at interval
		{100, false}, // Not at interval (100 % 6 = 4)
		{102, true},  // At interval (102 % 6 = 0)
	}

	for _, tt := range tests {
		result := ShouldAdjust(tt.blockIndex)
		if result != tt.expected {
			t.Errorf("ShouldAdjust(%d) = %v, expected %v", tt.blockIndex, result, tt.expected)
		}
	}
}

func TestCalculateNewDifficulty_BlocksTooFast(t *testing.T) {
	// Blocks mined every 2 seconds (target is 10 seconds)
	blocks := createTestBlocks(7, 2, 10) // 6 intervals of 2 seconds each = 12 seconds total

	newDifficulty := CalculateNewDifficulty(blocks, 10)

	// Blocks are too fast, difficulty should increase
	if newDifficulty <= 10 {
		t.Errorf("Difficulty should increase when blocks are too fast, got %d", newDifficulty)
	}
}

func TestCalculateNewDifficulty_BlocksTooSlow(t *testing.T) {
	// Blocks mined every 30 seconds (target is 10 seconds)
	blocks := createTestBlocks(7, 30, 10) // 6 intervals of 30 seconds each = 180 seconds total

	newDifficulty := CalculateNewDifficulty(blocks, 10)

	// Blocks are too slow, difficulty should decrease
	if newDifficulty >= 10 {
		t.Errorf("Difficulty should decrease when blocks are too slow, got %d", newDifficulty)
	}
}

func TestCalculateNewDifficulty_BlocksOnTarget(t *testing.T) {
	// Blocks mined every 10 seconds (exactly on target)
	blocks := createTestBlocks(7, 10, 10)

	newDifficulty := CalculateNewDifficulty(blocks, 10)

	// Blocks are on target, difficulty should stay the same
	if newDifficulty != 10 {
		t.Errorf("Difficulty should stay at 10 when blocks are on target, got %d", newDifficulty)
	}
}

func TestCalculateNewDifficulty_MinDifficulty(t *testing.T) {
	// Even very slow blocks should not go below MinDifficulty
	blocks := createTestBlocks(7, 1000, 1) // Very slow blocks

	newDifficulty := CalculateNewDifficulty(blocks, 1)

	if newDifficulty < MinDifficulty {
		t.Errorf("Difficulty should not go below %d, got %d", MinDifficulty, newDifficulty)
	}
}

func TestCalculateNewDifficulty_MaxDifficulty(t *testing.T) {
	// Even very fast blocks should not go above MaxDifficulty
	blocks := createTestBlocks(7, 1, MaxDifficulty) // Very fast blocks

	newDifficulty := CalculateNewDifficulty(blocks, MaxDifficulty)

	if newDifficulty > MaxDifficulty {
		t.Errorf("Difficulty should not go above %d, got %d", MaxDifficulty, newDifficulty)
	}
}

func TestCalculateNewDifficulty_InsufficientBlocks(t *testing.T) {
	// With only 1 block, difficulty should remain unchanged
	blocks := createTestBlocks(1, 10, 10)

	newDifficulty := CalculateNewDifficulty(blocks, 10)

	if newDifficulty != 10 {
		t.Errorf("Difficulty should remain at 10 with insufficient blocks, got %d", newDifficulty)
	}
}

func TestCalculateAverageBlockTime(t *testing.T) {
	// 7 blocks with 10-second intervals (6 intervals total)
	blocks := createTestBlocks(7, 10, 10)

	avgTime := CalculateAverageBlockTime(blocks)

	// Should be approximately 10 seconds
	expected := 10 * time.Second
	tolerance := 100 * time.Millisecond

	diff := avgTime - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("Average block time should be ~%v, got %v", expected, avgTime)
	}
}

func TestCalculateAverageBlockTime_InsufficientBlocks(t *testing.T) {
	blocks := createTestBlocks(1, 10, 10)

	avgTime := CalculateAverageBlockTime(blocks)

	if avgTime != 0 {
		t.Errorf("Average block time should be 0 with insufficient blocks, got %v", avgTime)
	}
}

func TestGetBlocksPerMinute(t *testing.T) {
	// 7 blocks with 10-second intervals = 60 seconds total time for 6 blocks
	// That is 6 blocks per minute
	blocks := createTestBlocks(7, 10, 10)

	bpm := GetBlocksPerMinute(blocks)

	// Should be approximately 6 blocks per minute
	expected := 6.0
	tolerance := 0.1

	diff := bpm - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("Blocks per minute should be ~%v, got %v", expected, bpm)
	}
}

func TestCalculateAdjustment(t *testing.T) {
	// Test with blocks mined too fast
	blocks := createTestBlocks(7, 2, 10)

	info := CalculateAdjustment(blocks, 10)

	if info.OldDifficulty != 10 {
		t.Errorf("OldDifficulty should be 10, got %d", info.OldDifficulty)
	}

	if info.BlocksAnalyzed != 7 {
		t.Errorf("BlocksAnalyzed should be 7, got %d", info.BlocksAnalyzed)
	}

	if info.TargetBlockTime != TargetBlockTime {
		t.Errorf("TargetBlockTime should be %v, got %v", TargetBlockTime, info.TargetBlockTime)
	}

	// Since blocks are too fast, new difficulty should be higher
	if info.NewDifficulty <= info.OldDifficulty {
		t.Errorf("NewDifficulty should be higher than OldDifficulty when blocks are too fast")
	}
}

func TestClampDifficulty(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-5, MinDifficulty},
		{0, MinDifficulty},
		{1, 1},
		{10, 10},
		{MaxDifficulty, MaxDifficulty},
		{MaxDifficulty + 10, MaxDifficulty},
	}

	for _, tt := range tests {
		result := clampDifficulty(tt.input)
		if result != tt.expected {
			t.Errorf("clampDifficulty(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestDifficultyAdjusterSetGetDifficulty(t *testing.T) {
	da := NewDifficultyAdjuster(10, true)

	da.SetDifficulty(15)
	if da.GetDifficulty() != 15 {
		t.Errorf("Expected difficulty 15, got %d", da.GetDifficulty())
	}

	// Test clamping
	da.SetDifficulty(-5)
	if da.GetDifficulty() != MinDifficulty {
		t.Errorf("Expected difficulty to be clamped to %d, got %d", MinDifficulty, da.GetDifficulty())
	}

	da.SetDifficulty(100)
	if da.GetDifficulty() != MaxDifficulty {
		t.Errorf("Expected difficulty to be clamped to %d, got %d", MaxDifficulty, da.GetDifficulty())
	}
}

// Benchmark test
func BenchmarkCalculateNewDifficulty(b *testing.B) {
	blocks := createTestBlocks(7, 10, 10)

	for i := 0; i < b.N; i++ {
		CalculateNewDifficulty(blocks, 10)
	}
}
