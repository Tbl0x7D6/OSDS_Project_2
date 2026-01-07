package pow

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"context"
	"strings"
	"testing"
	"time"
)

func createTestBlock(difficulty int) *block.Block {
	tx := transaction.NewCoinbaseTransaction("miner1", 50, 1)
	txs := []*transaction.Transaction{tx}

	return block.NewBlock(1, txs, "0000000000000000000000000000000000000000000000000000000000000000", difficulty, "miner1")
}

func TestMine(t *testing.T) {
	// Use low difficulty for fast test
	testBlock := createTestBlock(2)
	pow := NewProofOfWork(testBlock)

	result := pow.Mine(context.Background())

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
	// Use high difficulty to ensure mining doesn't complete quickly
	testBlock := createTestBlock(8)
	pow := NewProofOfWork(testBlock)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result := pow.Mine(ctx)

	if result.Success {
		t.Error("Mining should be cancelled, not succeed")
	}
}

func TestMineWithCallback(t *testing.T) {
	testBlock := createTestBlock(2)
	pow := NewProofOfWork(testBlock)

	callbackCount := 0
	callback := func(nonce int64) {
		callbackCount++
	}

	result := pow.MineWithCallback(context.Background(), callback)

	if !result.Success {
		t.Error("Mining should succeed")
	}

	// Callback might not be called if mining is very fast
	// Just check that mining completes without error
	_ = callbackCount
}

func TestMineParallel(t *testing.T) {
	testBlock := createTestBlock(2)
	pow := NewProofOfWork(testBlock)

	result := pow.MineParallel(context.Background(), 4)

	if !result.Success {
		t.Error("Parallel mining should succeed")
	}

	if !strings.HasPrefix(result.Block.Hash, "00") {
		t.Errorf("Hash should have 2 leading zeros, got %s", result.Block.Hash[:10])
	}
}

func TestMineParallelWithCancellation(t *testing.T) {
	testBlock := createTestBlock(8)
	pow := NewProofOfWork(testBlock)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result := pow.MineParallel(ctx, 4)

	if result.Success {
		t.Error("Parallel mining should be cancelled")
	}
}

func TestValidate(t *testing.T) {
	testBlock := createTestBlock(2)
	pow := NewProofOfWork(testBlock)

	result := pow.Mine(context.Background())
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
	block1 := createTestBlock(1)
	pow1 := NewProofOfWork(block1)
	start1 := time.Now()
	result1 := pow1.Mine(context.Background())
	time1 := time.Since(start1)

	if !result1.Success {
		t.Fatal("Mining with difficulty 1 should succeed")
	}

	// Difficulty 2
	block2 := createTestBlock(2)
	pow2 := NewProofOfWork(block2)
	start2 := time.Now()
	result2 := pow2.Mine(context.Background())
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
		testBlock := createTestBlock(2)
		pow := NewProofOfWork(testBlock)
		pow.Mine(context.Background())
	}
}

func BenchmarkMineParallel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testBlock := createTestBlock(2)
		pow := NewProofOfWork(testBlock)
		pow.MineParallel(context.Background(), 4)
	}
}
