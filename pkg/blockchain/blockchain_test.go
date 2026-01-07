package blockchain

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"testing"
)

func createValidBlock(bc *Blockchain, minerID string) *block.Block {
	// Use coinbase transaction for mining reward
	coinbase := transaction.NewCoinbaseTransaction(minerID, 5000000000, bc.GetLatestBlock().Index+1)
	txs := []*transaction.Transaction{coinbase}

	newBlock := bc.CreateBlock(txs, minerID)

	// Mine the block (find valid PoW)
	target := ""
	for i := 0; i < bc.Difficulty; i++ {
		target += "0"
	}

	for nonce := int64(0); ; nonce++ {
		newBlock.Nonce = nonce
		hash := newBlock.CalculateHash()
		if len(hash) >= bc.Difficulty && hash[:bc.Difficulty] == target {
			newBlock.Hash = hash
			break
		}
	}

	return newBlock
}

func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain(2)

	if bc.GetLength() != 1 {
		t.Errorf("New blockchain should have 1 block (genesis), got %d", bc.GetLength())
	}

	genesis := bc.GetLatestBlock()
	if genesis.Index != 0 {
		t.Errorf("Genesis block should have index 0, got %d", genesis.Index)
	}
}

func TestAddValidBlock(t *testing.T) {
	bc := NewBlockchain(2)

	newBlock := createValidBlock(bc, "miner1")
	err := bc.AddBlock(newBlock)
	if err != nil {
		t.Fatalf("Failed to add valid block: %v", err)
	}

	if bc.GetLength() != 2 {
		t.Errorf("Chain should have 2 blocks, got %d", bc.GetLength())
	}
}

func TestAddBlockWithInvalidIndex(t *testing.T) {
	bc := NewBlockchain(2)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	// Create block with wrong index
	latestBlock := bc.GetLatestBlock()
	newBlock := block.NewBlock(
		5, // Wrong index
		txs,
		latestBlock.Hash,
		bc.Difficulty,
		"miner1",
	)

	// Mine the block
	for nonce := int64(0); ; nonce++ {
		newBlock.Nonce = nonce
		hash := newBlock.CalculateHash()
		if len(hash) >= bc.Difficulty && hash[:bc.Difficulty] == "00" {
			newBlock.Hash = hash
			break
		}
	}

	err := bc.AddBlock(newBlock)
	if err != ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex, got %v", err)
	}
}

func TestAddBlockWithInvalidPrevHash(t *testing.T) {
	bc := NewBlockchain(2)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	// Create block with wrong previous hash
	newBlock := block.NewBlock(
		1,
		txs,
		"wrong_prev_hash", // Wrong previous hash
		bc.Difficulty,
		"miner1",
	)

	// Mine the block
	for nonce := int64(0); ; nonce++ {
		newBlock.Nonce = nonce
		hash := newBlock.CalculateHash()
		if len(hash) >= bc.Difficulty && hash[:bc.Difficulty] == "00" {
			newBlock.Hash = hash
			break
		}
	}

	err := bc.AddBlock(newBlock)
	if err != ErrInvalidPrevHash {
		t.Errorf("Expected ErrInvalidPrevHash, got %v", err)
	}
}

func TestAddBlockWithInvalidPoW(t *testing.T) {
	bc := NewBlockchain(2)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	latestBlock := bc.GetLatestBlock()
	newBlock := block.NewBlock(
		1,
		txs,
		latestBlock.Hash,
		bc.Difficulty,
		"miner1",
	)

	// Set an invalid hash (doesn't meet PoW requirement)
	newBlock.Nonce = 1
	newBlock.Hash = newBlock.CalculateHash() // Will not have leading zeros

	err := bc.AddBlock(newBlock)
	if err != ErrInvalidPoW {
		t.Errorf("Expected ErrInvalidPoW, got %v", err)
	}
}

func TestValidateChain(t *testing.T) {
	bc := NewBlockchain(2)

	// Add a few valid blocks
	for i := 0; i < 3; i++ {
		newBlock := createValidBlock(bc, "miner1")
		err := bc.AddBlock(newBlock)
		if err != nil {
			t.Fatalf("Failed to add block %d: %v", i, err)
		}
	}

	err := bc.ValidateChain()
	if err != nil {
		t.Errorf("Valid chain should pass validation, got error: %v", err)
	}
}

func TestValidateCorruptedChain(t *testing.T) {
	bc := NewBlockchain(2)

	// Add a valid block
	newBlock := createValidBlock(bc, "miner1")
	err := bc.AddBlock(newBlock)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// Corrupt a block
	bc.Blocks[1].Hash = "corrupted_hash"

	err = bc.ValidateChain()
	if err == nil {
		t.Error("Corrupted chain should fail validation")
	}
}

func TestLongestChainRule(t *testing.T) {
	bc := NewBlockchain(2)

	// Add 2 blocks to original chain
	for i := 0; i < 2; i++ {
		newBlock := createValidBlock(bc, "miner1")
		bc.AddBlock(newBlock)
	}

	if bc.GetLength() != 3 {
		t.Fatalf("Expected chain length 3, got %d", bc.GetLength())
	}

	// Create a longer chain (4 blocks including genesis)
	longerChain := NewBlockchain(2)
	for i := 0; i < 3; i++ {
		newBlock := createValidBlock(longerChain, "miner2")
		longerChain.AddBlock(newBlock)
	}

	if longerChain.GetLength() != 4 {
		t.Fatalf("Expected longer chain length 4, got %d", longerChain.GetLength())
	}

	// Replace with longer chain
	err := bc.ReplaceChain(longerChain.GetBlocks())
	if err != nil {
		t.Fatalf("Failed to replace chain: %v", err)
	}

	if bc.GetLength() != 4 {
		t.Errorf("Chain should be replaced with longer chain, length should be 4, got %d", bc.GetLength())
	}
}

func TestRejectShorterChain(t *testing.T) {
	bc := NewBlockchain(2)

	// Add 3 blocks
	for i := 0; i < 3; i++ {
		newBlock := createValidBlock(bc, "miner1")
		bc.AddBlock(newBlock)
	}

	// Create a shorter chain
	shorterChain := NewBlockchain(2)
	for i := 0; i < 1; i++ {
		newBlock := createValidBlock(shorterChain, "miner2")
		shorterChain.AddBlock(newBlock)
	}

	// Try to replace with shorter chain
	err := bc.ReplaceChain(shorterChain.GetBlocks())
	if err != ErrChainTooShort {
		t.Errorf("Expected ErrChainTooShort, got %v", err)
	}
}

func TestGetBlockByIndex(t *testing.T) {
	bc := NewBlockchain(2)

	newBlock := createValidBlock(bc, "miner1")
	bc.AddBlock(newBlock)

	// Get existing block
	block := bc.GetBlockByIndex(1)
	if block == nil {
		t.Error("Expected to find block at index 1")
	}
	if block.Index != 1 {
		t.Errorf("Expected block index 1, got %d", block.Index)
	}

	// Get non-existing block
	block = bc.GetBlockByIndex(100)
	if block != nil {
		t.Error("Expected nil for non-existing block index")
	}
}

func TestGetBlockByHash(t *testing.T) {
	bc := NewBlockchain(2)

	newBlock := createValidBlock(bc, "miner1")
	bc.AddBlock(newBlock)

	// Get existing block
	block := bc.GetBlockByHash(newBlock.Hash)
	if block == nil {
		t.Error("Expected to find block by hash")
	}

	// Get non-existing block
	block = bc.GetBlockByHash("nonexistent_hash")
	if block != nil {
		t.Error("Expected nil for non-existing block hash")
	}
}

func TestSetDifficulty(t *testing.T) {
	bc := NewBlockchain(2)

	bc.SetDifficulty(4)
	if bc.GetDifficulty() != 4 {
		t.Errorf("Expected difficulty 4, got %d", bc.GetDifficulty())
	}
}
