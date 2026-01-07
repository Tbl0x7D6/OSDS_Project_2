package block

import (
	"blockchain/pkg/transaction"
	"testing"
)

func TestNewBlock(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	if block.Index != 1 {
		t.Errorf("Expected index 1, got %d", block.Index)
	}
	if block.PrevHash != "prev_hash" {
		t.Errorf("Expected prev_hash 'prev_hash', got '%s'", block.PrevHash)
	}
	if block.Difficulty != 2 {
		t.Errorf("Expected difficulty 2, got %d", block.Difficulty)
	}
	if block.MinerID != "miner1" {
		t.Errorf("Expected miner 'miner1', got '%s'", block.MinerID)
	}
	if len(block.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(block.Transactions))
	}
}

func TestGenesisBlock(t *testing.T) {
	genesis := NewGenesisBlock(2)

	if genesis.Index != 0 {
		t.Errorf("Genesis block should have index 0, got %d", genesis.Index)
	}
	if genesis.PrevHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Error("Genesis block should have all-zero previous hash")
	}
	if genesis.MinerID != "genesis" {
		t.Errorf("Genesis block miner should be 'genesis', got '%s'", genesis.MinerID)
	}
}

func TestBlockHashConsistency(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.Nonce = 12345
	block.SetHash()

	hash1 := block.CalculateHash()
	hash2 := block.CalculateHash()

	if hash1 != hash2 {
		t.Error("Block hash should be deterministic")
	}
}

func TestHasValidHash(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.Nonce = 12345
	block.SetHash()

	if !block.HasValidHash() {
		t.Error("Block should have valid hash")
	}

	// Corrupt the hash
	block.Hash = "corrupted_hash"
	if block.HasValidHash() {
		t.Error("Block should have invalid hash after corruption")
	}
}

func TestHasValidPoW(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	// Find a valid nonce manually (for low difficulty)
	for nonce := int64(0); nonce < 1000000; nonce++ {
		block.Nonce = nonce
		block.SetHash()
		if block.HasValidPoW() {
			break
		}
	}

	if !block.HasValidPoW() {
		t.Error("Block should have valid PoW after mining")
	}

	// Test with invalid hash
	block.Hash = "ffff" + block.Hash[4:]
	if block.HasValidPoW() {
		t.Error("Block should have invalid PoW with non-zero prefix")
	}
}

func TestValidateTransactions(t *testing.T) {
	// Valid transactions
	tx1 := transaction.NewTransaction("alice", "bob", 10.0)
	tx1.Sign("key")
	txs := []*transaction.Transaction{tx1}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	if !block.ValidateTransactions() {
		t.Error("Block with valid transactions should pass validation")
	}

	// Invalid transaction (unsigned)
	tx2 := transaction.NewTransaction("alice", "bob", 10.0)
	block.Transactions = append(block.Transactions, tx2)
	if block.ValidateTransactions() {
		t.Error("Block with invalid transactions should fail validation")
	}
}

func TestBlockSerialization(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.Nonce = 12345
	block.SetHash()

	// Serialize
	data, err := block.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize block: %v", err)
	}

	// Deserialize
	block2, err := DeserializeBlock(data)
	if err != nil {
		t.Fatalf("Failed to deserialize block: %v", err)
	}

	// Compare
	if block.Index != block2.Index {
		t.Error("Index mismatch after deserialization")
	}
	if block.Hash != block2.Hash {
		t.Error("Hash mismatch after deserialization")
	}
	if block.PrevHash != block2.PrevHash {
		t.Error("PrevHash mismatch after deserialization")
	}
	if block.Nonce != block2.Nonce {
		t.Error("Nonce mismatch after deserialization")
	}
	if len(block.Transactions) != len(block2.Transactions) {
		t.Error("Transaction count mismatch after deserialization")
	}
}

func TestBlockClone(t *testing.T) {
	tx := transaction.NewTransaction("alice", "bob", 10.0)
	tx.Sign("key")
	txs := []*transaction.Transaction{tx}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.SetHash()

	clone := block.Clone()

	// Modify clone
	clone.Nonce = 99999
	clone.SetHash()

	// Original should be unchanged
	if block.Nonce == clone.Nonce {
		t.Error("Clone should be independent of original")
	}
}
