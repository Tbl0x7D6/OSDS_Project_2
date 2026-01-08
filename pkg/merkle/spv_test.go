package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestSPVRealWorldScenario tests a realistic SPV scenario
func TestSPVRealWorldScenario(t *testing.T) {
	// Simulate real transaction IDs (hex-encoded SHA256 hashes)
	txHashes := make([]string, 0)
	for i := 0; i < 16; i++ {
		hash := sha256.Sum256([]byte{byte(i), byte(i + 1), byte(i + 2)})
		txHashes = append(txHashes, hex.EncodeToString(hash[:]))
	}

	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	// Store the root (this would be in block header)
	merkleRoot := tree.GetRootHash()

	// Later, a light client receives a transaction and wants to verify it
	for i, txHash := range txHashes {
		proof, err := tree.GenerateProof(txHash)
		if err != nil {
			t.Errorf("Failed to generate proof for tx %d: %v", i, err)
			continue
		}

		// Light client verifies with just:
		// - transaction hash
		// - merkle root from block header
		// - merkle proof (siblings + directions)
		verified := VerifyProofWithRoot(txHash, merkleRoot, proof.Siblings, proof.Directions)
		if !verified {
			t.Errorf("SPV verification failed for tx %d", i)
		}
	}
}

// TestSPVProofSize verifies proof size is logarithmic
func TestSPVProofSize(t *testing.T) {
	testCases := []struct {
		numTx         int
		expectedDepth int
	}{
		{1, 0},  // 2^0 = 1
		{2, 1},  // 2^1 = 2
		{4, 2},  // 2^2 = 4
		{8, 3},  // 2^3 = 8
		{16, 4}, // 2^4 = 16
		{32, 5}, // 2^5 = 32
	}

	for _, tc := range testCases {
		txHashes := make([]string, tc.numTx)
		for i := 0; i < tc.numTx; i++ {
			txHashes[i] = hex.EncodeToString([]byte{byte(i)})
		}

		tree, err := NewMerkleTreeFromHashes(txHashes)
		if err != nil {
			t.Fatalf("Failed to create tree with %d elements: %v", tc.numTx, err)
		}

		proof, err := tree.GenerateProof(txHashes[0])
		if err != nil {
			t.Fatalf("Failed to generate proof: %v", err)
		}

		if len(proof.Siblings) != tc.expectedDepth {
			t.Errorf("For %d transactions, expected proof depth %d, got %d",
				tc.numTx, tc.expectedDepth, len(proof.Siblings))
		}
	}
}

// TestSPVTamperedTransaction ensures tampered transactions fail verification
func TestSPVTamperedTransaction(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	// Get valid proof for tx1
	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Original should verify
	if !VerifyProof(proof) {
		t.Error("Original proof should verify")
	}

	// Try to verify tampered transaction with original proof
	tamperedProof := &MerkleProof{
		TxHash:     "tx1_tampered",
		MerkleRoot: proof.MerkleRoot,
		Siblings:   proof.Siblings,
		Directions: proof.Directions,
	}

	if VerifyProof(tamperedProof) {
		t.Error("Tampered transaction should not verify")
	}
}

// TestSPVPartialProofAttack tests against partial proof attacks
func TestSPVPartialProofAttack(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4", "tx5", "tx6", "tx7", "tx8"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Try with incomplete proof (missing siblings)
	if len(proof.Siblings) > 1 {
		incompleteProof := &MerkleProof{
			TxHash:     proof.TxHash,
			MerkleRoot: proof.MerkleRoot,
			Siblings:   proof.Siblings[:len(proof.Siblings)-1],
			Directions: proof.Directions[:len(proof.Directions)-1],
		}

		if VerifyProof(incompleteProof) {
			t.Error("Incomplete proof should not verify")
		}
	}
}

// TestSPVWrongOrderSiblings tests that sibling order matters
func TestSPVWrongOrderSiblings(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify original works
	if !VerifyProof(proof) {
		t.Fatal("Original proof should verify")
	}

	// Flip all directions
	if len(proof.Siblings) > 0 {
		flippedProof := &MerkleProof{
			TxHash:     proof.TxHash,
			MerkleRoot: proof.MerkleRoot,
			Siblings:   proof.Siblings,
			Directions: make([]bool, len(proof.Directions)),
		}
		for i, d := range proof.Directions {
			flippedProof.Directions[i] = !d
		}

		if VerifyProof(flippedProof) {
			t.Error("Proof with flipped directions should not verify (unless by coincidence)")
		}
	}
}

// TestMerkleTreeConsistency verifies tree consistency
func TestMerkleTreeConsistency(t *testing.T) {
	// Test that adding transactions changes the root
	txHashes1 := []string{"tx1", "tx2"}
	txHashes2 := []string{"tx1", "tx2", "tx3"}

	tree1, _ := NewMerkleTreeFromHashes(txHashes1)
	tree2, _ := NewMerkleTreeFromHashes(txHashes2)

	if tree1.GetRootHash() == tree2.GetRootHash() {
		t.Error("Different transaction sets should have different roots")
	}
}

// TestSPVWithIdenticalTransactions tests behavior with duplicate transaction hashes
func TestSPVWithIdenticalTransactions(t *testing.T) {
	// In real blockchain, tx IDs are unique, but let's test edge case
	txHashes := []string{"same_tx", "same_tx"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	// Should still be able to generate and verify proof
	proof, err := tree.GenerateProof("same_tx")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	if !VerifyProof(proof) {
		t.Error("Proof for duplicate transaction should still verify")
	}
}

// TestSPVProofForEachPosition tests proof generation for each tree position
func TestSPVProofForEachPosition(t *testing.T) {
	sizes := []int{1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17}

	for _, size := range sizes {
		txHashes := make([]string, size)
		for i := 0; i < size; i++ {
			txHashes[i] = hex.EncodeToString([]byte{byte(i), byte(size)})
		}

		tree, err := NewMerkleTreeFromHashes(txHashes)
		if err != nil {
			t.Fatalf("Failed to create tree of size %d: %v", size, err)
		}

		for i, txHash := range txHashes {
			proof, err := tree.GenerateProof(txHash)
			if err != nil {
				t.Errorf("Size %d, position %d: failed to generate proof: %v", size, i, err)
				continue
			}

			if !VerifyProof(proof) {
				t.Errorf("Size %d, position %d: proof verification failed", size, i)
			}
		}
	}
}

// BenchmarkMerkleTreeCreation benchmarks tree creation
func BenchmarkMerkleTreeCreation(b *testing.B) {
	txHashes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		hash := sha256.Sum256([]byte{byte(i), byte(i >> 8)})
		txHashes[i] = hex.EncodeToString(hash[:])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMerkleTreeFromHashes(txHashes)
	}
}

// BenchmarkProofGeneration benchmarks proof generation
func BenchmarkProofGeneration(b *testing.B) {
	txHashes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		hash := sha256.Sum256([]byte{byte(i), byte(i >> 8)})
		txHashes[i] = hex.EncodeToString(hash[:])
	}

	tree, _ := NewMerkleTreeFromHashes(txHashes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GenerateProof(txHashes[i%1000])
	}
}

// BenchmarkProofVerification benchmarks proof verification
func BenchmarkProofVerification(b *testing.B) {
	txHashes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		hash := sha256.Sum256([]byte{byte(i), byte(i >> 8)})
		txHashes[i] = hex.EncodeToString(hash[:])
	}

	tree, _ := NewMerkleTreeFromHashes(txHashes)
	proofs := make([]*MerkleProof, 100)
	for i := 0; i < 100; i++ {
		proofs[i], _ = tree.GenerateProof(txHashes[i*10])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyProof(proofs[i%100])
	}
}
