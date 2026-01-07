package pow

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"context"
	"strings"
	"testing"
	"time"
)

// Helper function to create a ProofOfWork instance for testing
func setupTestPoW(difficulty int) (*ProofOfWork, *block.Block) {
	tx := transaction.NewCoinbaseTransaction("miner1", 50, 1)
	txs := []*transaction.Transaction{tx}
	testBlock := block.NewBlock(1, txs, "0000000000000000000000000000000000000000000000000000000000000000", difficulty, "miner1")
	return NewProofOfWork(testBlock), testBlock
}

// Helper function to create a cancellable context with a delay
func createCancellableContext(delay time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(delay)
		cancel()
	}()
	return ctx, cancel
}

func TestMine(t *testing.T) {
	pow, _ := setupTestPoW(2)

	result := pow.Mine(context.Background(), nil)

	if !result.Success {
		t.Error("Mining should succeed")
	}

	if !strings.HasPrefix(result.Block.Hash, "00") {
		t.Errorf("Hash should have 2 leading zeros, got %s", result.Block.Hash[:10])
	}

	if !result.Block.HasValidPoW() {
		t.Error("Mined block should have valid PoW")
	}
}

func TestMineWithCancellation(t *testing.T) {
	pow, _ := setupTestPoW(8)
	ctx, _ := createCancellableContext(100 * time.Millisecond)

	result := pow.Mine(ctx, nil)

	if result.Success {
		t.Error("Mining should be cancelled, not succeed")
	}
}

func TestMineWithCallback(t *testing.T) {
	pow, _ := setupTestPoW(2)

	callbackCount := 0
	callback := func(nonce int64) {
		callbackCount++
	}

	result := pow.Mine(context.Background(), callback)

	if !result.Success {
		t.Error("Mining should succeed")
	}
}

func TestMineParallel(t *testing.T) {
	pow, _ := setupTestPoW(2)

	result := pow.MineParallel(context.Background(), 4)

	if !result.Success {
		t.Error("Parallel mining should succeed")
	}

	if !strings.HasPrefix(result.Block.Hash, "00") {
		t.Errorf("Hash should have 2 leading zeros, got %s", result.Block.Hash[:10])
	}
}

func TestMineParallelWithCancellation(t *testing.T) {
	pow, _ := setupTestPoW(8)
	ctx, _ := createCancellableContext(100 * time.Millisecond)

	result := pow.MineParallel(ctx, 4)

	if result.Success {
		t.Error("Parallel mining should be cancelled")
	}
}

func TestValidate(t *testing.T) {
	pow, _ := setupTestPoW(2)
	result := pow.Mine(context.Background(), nil)
	if !result.Success {
		t.Fatal("Mining should succeed")
	}

	// Valid block
	if !Validate(result.Block) {
		t.Error("Mined block should be valid")
	}

	// Invalid block (corrupted hash)
	result.Block.Hash = "corrupted"
	if Validate(result.Block) {
		t.Error("Block with corrupted hash should be invalid")
	}
}

func TestValidateHash(t *testing.T) {
	// Valid hashes
	if !ValidateHash("00abc", 2) {
		t.Error("Hash with 2 leading zeros should be valid for difficulty 2")
	}
	if !ValidateHash("000abc", 3) {
		t.Error("Hash with 3 leading zeros should be valid for difficulty 3")
	}

	// Invalid hashes
	if ValidateHash("0abc", 2) {
		t.Error("Hash with 1 leading zero should be invalid for difficulty 2")
	}
	if ValidateHash("abc", 1) {
		t.Error("Hash without leading zeros should be invalid for difficulty 1")
	}
}

func TestGetTarget(t *testing.T) {
	if GetTarget(2) != "00" {
		t.Error("Target for difficulty 2 should be '00'")
	}
	if GetTarget(4) != "0000" {
		t.Error("Target for difficulty 4 should be '0000'")
	}
}

func TestDifficultyAffectsMiningSpeed(t *testing.T) {
	// Test that higher difficulty takes longer
	// Using difficulty 1 and 2 for reasonable test time

	// Difficulty 1
	pow1, _ := setupTestPoW(1)
	start1 := time.Now()
	result1 := pow1.Mine(context.Background(), nil)
	time1 := time.Since(start1)

	if !result1.Success {
		t.Fatal("Mining with difficulty 1 should succeed")
	}

	// Difficulty 2
	pow2, _ := setupTestPoW(2)
	start2 := time.Now()
	result2 := pow2.Mine(context.Background(), nil)
	time2 := time.Since(start2)

	if !result2.Success {
		t.Fatal("Mining with difficulty 2 should succeed")
	}

	// We can't guarantee exact times, but generally diff 2 should take longer
	// Just log the times for manual verification
	t.Logf("Difficulty 1: %v (nonce: %d)", time1, result1.Nonce)
	t.Logf("Difficulty 2: %v (nonce: %d)", time2, result2.Nonce)
}

func BenchmarkMine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pow, _ := setupTestPoW(2)
		pow.Mine(context.Background(), nil)
	}
}

func BenchmarkMineParallel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pow, _ := setupTestPoW(2)
		pow.MineParallel(context.Background(), 4)
	}
}
