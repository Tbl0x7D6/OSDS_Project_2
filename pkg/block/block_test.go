package block

import (
	"blockchain/pkg/config"
	"blockchain/pkg/transaction"
	"testing"
)

func TestNewBlock(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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
	// Valid coinbase transaction
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	if !block.ValidateTransactions() {
		t.Error("Block with valid coinbase should pass validation")
	}

	// Invalid transaction (unsigned regular tx)
	badTx := &transaction.Transaction{
		ID:      "bad",
		Inputs:  []transaction.TxInput{{TxID: "abc", OutIndex: 0, ScriptSig: ""}},
		Outputs: []transaction.TxOutput{{Value: 1000, ScriptPubKey: "bob"}},
	}
	block.Transactions = append(block.Transactions, badTx)
	if block.ValidateTransactions() {
		t.Error("Block with invalid transactions should fail validation")
	}
}

func TestBlockSerialization(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

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

	// MerkleRoot should be preserved
	if clone.MerkleRoot != block.MerkleRoot {
		t.Error("Clone should preserve MerkleRoot")
	}
}

func TestMerkleRootCalculation(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	// MerkleRoot should be set
	if block.MerkleRoot == "" {
		t.Error("MerkleRoot should be calculated on block creation")
	}

	// MerkleRoot should be consistent
	root1 := block.CalculateMerkleRoot()
	root2 := block.CalculateMerkleRoot()
	if root1 != root2 {
		t.Error("MerkleRoot calculation should be deterministic")
	}
}

func TestHasValidMerkleRoot(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	if !block.HasValidMerkleRoot() {
		t.Error("Block should have valid merkle root")
	}

	// Tamper with merkle root
	block.MerkleRoot = "tampered_root"
	if block.HasValidMerkleRoot() {
		t.Error("Tampered merkle root should be invalid")
	}
}

func TestMerkleRootMultipleTransactions(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	tx1 := transaction.NewCoinbaseTransaction("addr1", 1000, 1)
	tx2 := transaction.NewCoinbaseTransaction("addr2", 2000, 1)

	txs := []*transaction.Transaction{coinbase, tx1, tx2}
	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	if block.MerkleRoot == "" {
		t.Error("MerkleRoot should be calculated for multiple transactions")
	}

	if !block.HasValidMerkleRoot() {
		t.Error("Block should have valid merkle root with multiple transactions")
	}
}

func TestGetMerkleTree(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	tx1 := transaction.NewCoinbaseTransaction("addr1", 1000, 1)

	txs := []*transaction.Transaction{coinbase, tx1}
	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	tree, err := block.GetMerkleTree()
	if err != nil {
		t.Fatalf("Failed to get merkle tree: %v", err)
	}

	if tree.GetRootHash() != block.MerkleRoot {
		t.Error("MerkleTree root should match block's MerkleRoot")
	}
}

func TestGenerateSPVProof(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	tx1 := transaction.NewCoinbaseTransaction("addr1", 1000, 1)
	tx2 := transaction.NewCoinbaseTransaction("addr2", 2000, 1)

	txs := []*transaction.Transaction{coinbase, tx1, tx2}
	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	// Generate proof for tx1
	proof, err := block.GenerateSPVProof(tx1.ID)
	if err != nil {
		t.Fatalf("Failed to generate SPV proof: %v", err)
	}

	if proof.TxHash != tx1.ID {
		t.Error("Proof should be for the correct transaction")
	}

	if proof.MerkleRoot != block.MerkleRoot {
		t.Error("Proof merkle root should match block's merkle root")
	}
}

func TestVerifyTransactionInBlock(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	tx1 := transaction.NewCoinbaseTransaction("addr1", 1000, 1)
	tx2 := transaction.NewCoinbaseTransaction("addr2", 2000, 1)
	tx3 := transaction.NewCoinbaseTransaction("addr3", 3000, 1)

	txs := []*transaction.Transaction{coinbase, tx1, tx2, tx3}
	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	// Verify each transaction is in the block
	for _, tx := range txs {
		if !block.VerifyTransactionInBlock(tx.ID) {
			t.Errorf("Transaction %s should be verified in block", tx.ID[:8])
		}
	}

	// Verify non-existent transaction
	fakeTx := transaction.NewCoinbaseTransaction("fake", 9999, 99)
	if block.VerifyTransactionInBlock(fakeTx.ID) {
		t.Error("Non-existent transaction should not be verified")
	}
}

func TestSPVProofVerification(t *testing.T) {
	// Create a block with several transactions
	var txs []*transaction.Transaction
	txs = append(txs, transaction.NewCoinbaseTransaction("miner1", 5000000000, 1))
	for i := 0; i < 7; i++ {
		txs = append(txs, transaction.NewCoinbaseTransaction(
			"addr"+string(rune('a'+i)), int64(i*1000), int64(i)))
	}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")

	// Test SPV for each transaction
	for _, tx := range txs {
		proof, err := block.GenerateSPVProof(tx.ID)
		if err != nil {
			t.Errorf("Failed to generate proof for tx %s: %v", tx.ID[:8], err)
			continue
		}

		// Verify using SPV
		if !block.VerifyTransactionInBlock(tx.ID) {
			t.Errorf("SPV verification failed for tx %s", tx.ID[:8])
		}

		// Verify proof details
		if proof.MerkleRoot != block.MerkleRoot {
			t.Errorf("Proof root mismatch for tx %s", tx.ID[:8])
		}
	}
}

func TestEmptyBlockMerkleRoot(t *testing.T) {
	// Create block with no transactions
	block := &Block{
		Index:        1,
		Timestamp:    12345,
		Transactions: []*transaction.Transaction{},
		PrevHash:     "prev",
		Difficulty:   2,
		MinerID:      "miner",
	}

	// Empty block should have empty merkle root
	root := block.CalculateMerkleRoot()
	if root != "" {
		t.Error("Empty block should have empty merkle root")
	}
}

func TestMerkleRootInHash(t *testing.T) {
	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	hash1 := block.CalculateHash()

	// Change merkle root (simulate tampering)
	originalRoot := block.MerkleRoot
	block.MerkleRoot = "tampered"
	hash2 := block.CalculateHash()

	// Hashes should be different
	if hash1 == hash2 {
		t.Error("Changing merkle root should change block hash")
	}

	// Restore and verify
	block.MerkleRoot = originalRoot
	hash3 := block.CalculateHash()
	if hash1 != hash3 {
		t.Error("Restoring merkle root should restore original hash")
	}
}

func TestLegacyModeHashCalculation(t *testing.T) {
	// Save current config
	originalUseMerkle := config.UseMerkleTree()
	defer config.SetUseMerkleTree(originalUseMerkle)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	tx1 := transaction.NewCoinbaseTransaction("addr1", 1000, 1)
	txs := []*transaction.Transaction{coinbase, tx1}

	// Test with Merkle Tree mode
	config.SetUseMerkleTree(true)
	blockMerkle := NewBlock(1, txs, "prev_hash", 2, "miner1")
	blockMerkle.SetHash()

	if blockMerkle.MerkleRoot == "" {
		t.Error("MerkleRoot should be set in Merkle Tree mode")
	}

	// Test with Legacy mode (direct transaction serialization)
	config.SetUseMerkleTree(false)
	blockLegacy := NewBlock(1, txs, "prev_hash", 2, "miner1")
	blockLegacy.Timestamp = blockMerkle.Timestamp // Use same timestamp for comparison
	blockLegacy.SetHash()

	// In legacy mode, MerkleRoot should not be set during block creation
	if blockLegacy.MerkleRoot != "" {
		t.Error("MerkleRoot should NOT be set in Legacy mode during block creation")
	}

	// Hash should be different between modes (unless by coincidence)
	// Actually, they should definitely be different since one uses MerkleRoot string
	// and the other uses concatenated transaction IDs
	if blockMerkle.Hash == blockLegacy.Hash {
		t.Error("Hash should be different between Merkle and Legacy modes")
	}
}

func TestLegacyModeHashConsistency(t *testing.T) {
	// Save current config
	originalUseMerkle := config.UseMerkleTree()
	defer config.SetUseMerkleTree(originalUseMerkle)

	// Set to legacy mode
	config.SetUseMerkleTree(false)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.SetHash()
	hash1 := block.Hash

	// Hash should be deterministic
	hash2 := block.CalculateHash()
	if hash1 != hash2 {
		t.Error("Legacy mode hash should be deterministic")
	}
}

func TestModeSwitch(t *testing.T) {
	// Save current config
	originalUseMerkle := config.UseMerkleTree()
	defer config.SetUseMerkleTree(originalUseMerkle)

	coinbase := transaction.NewCoinbaseTransaction("miner1", 5000000000, 1)
	txs := []*transaction.Transaction{coinbase}

	// Create block in Merkle mode
	config.SetUseMerkleTree(true)
	block := NewBlock(1, txs, "prev_hash", 2, "miner1")
	block.SetHash()
	merkleHash := block.Hash

	// Same block recalculated in legacy mode should have different hash
	config.SetUseMerkleTree(false)
	legacyHash := block.CalculateHash()

	if merkleHash == legacyHash {
		t.Error("Switching modes should produce different hashes")
	}

	// Switch back to merkle mode
	config.SetUseMerkleTree(true)
	merkleHash2 := block.CalculateHash()

	if merkleHash != merkleHash2 {
		t.Error("Same mode should produce same hash")
	}
}
