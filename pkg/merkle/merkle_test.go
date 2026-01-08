package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNewMerkleTreeSingleElement(t *testing.T) {
	data := [][]byte{[]byte("tx1")}
	tree, err := NewMerkleTree(data)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	if tree.Root == nil {
		t.Fatal("Tree root should not be nil")
	}

	// Single element: root hash = hash(tx1)
	expectedHash := sha256.Sum256([]byte("tx1"))
	if tree.GetRootHash() != hex.EncodeToString(expectedHash[:]) {
		t.Errorf("Root hash mismatch for single element tree")
	}
}

func TestNewMerkleTreeTwoElements(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2")}
	tree, err := NewMerkleTree(data)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	if tree.Root == nil {
		t.Fatal("Tree root should not be nil")
	}

	// Two elements: root = hash(hash(tx1) + hash(tx2))
	h1 := sha256.Sum256([]byte("tx1"))
	h2 := sha256.Sum256([]byte("tx2"))
	combined := append(h1[:], h2[:]...)
	expectedRoot := sha256.Sum256(combined)

	if tree.GetRootHash() != hex.EncodeToString(expectedRoot[:]) {
		t.Errorf("Root hash mismatch for two element tree")
	}
}

func TestNewMerkleTreeThreeElements(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}
	tree, err := NewMerkleTree(data)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	if tree.Root == nil {
		t.Fatal("Tree root should not be nil")
	}

	// Three elements:
	// Level 0: H(tx1), H(tx2), H(tx3)
	// Level 1: H(H(tx1)+H(tx2)), H(H(tx3)+H(tx3)) (duplicate for odd)
	// Level 2: H(Level1[0] + Level1[1])
	h1 := sha256.Sum256([]byte("tx1"))
	h2 := sha256.Sum256([]byte("tx2"))
	h3 := sha256.Sum256([]byte("tx3"))

	combined12 := append(h1[:], h2[:]...)
	h12 := sha256.Sum256(combined12)

	combined33 := append(h3[:], h3[:]...)
	h33 := sha256.Sum256(combined33)

	combinedRoot := append(h12[:], h33[:]...)
	expectedRoot := sha256.Sum256(combinedRoot)

	if tree.GetRootHash() != hex.EncodeToString(expectedRoot[:]) {
		t.Errorf("Root hash mismatch for three element tree")
	}
}

func TestNewMerkleTreeFourElements(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3"), []byte("tx4")}
	tree, err := NewMerkleTree(data)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Four elements (perfect binary tree):
	// Level 0: H(tx1), H(tx2), H(tx3), H(tx4)
	// Level 1: H(H(tx1)+H(tx2)), H(H(tx3)+H(tx4))
	// Level 2: H(Level1[0] + Level1[1])
	h1 := sha256.Sum256([]byte("tx1"))
	h2 := sha256.Sum256([]byte("tx2"))
	h3 := sha256.Sum256([]byte("tx3"))
	h4 := sha256.Sum256([]byte("tx4"))

	combined12 := append(h1[:], h2[:]...)
	h12 := sha256.Sum256(combined12)

	combined34 := append(h3[:], h4[:]...)
	h34 := sha256.Sum256(combined34)

	combinedRoot := append(h12[:], h34[:]...)
	expectedRoot := sha256.Sum256(combinedRoot)

	if tree.GetRootHash() != hex.EncodeToString(expectedRoot[:]) {
		t.Errorf("Root hash mismatch for four element tree")
	}
}

func TestNewMerkleTreeEmpty(t *testing.T) {
	data := [][]byte{}
	_, err := NewMerkleTree(data)
	if err != ErrEmptyTree {
		t.Errorf("Expected ErrEmptyTree, got %v", err)
	}
}

func TestNewMerkleTreeFromHashes(t *testing.T) {
	hashes := []string{"abc123", "def456", "ghi789"}
	tree, err := NewMerkleTreeFromHashes(hashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree from hashes: %v", err)
	}

	if tree.Root == nil {
		t.Fatal("Tree root should not be nil")
	}

	rootHash := tree.GetRootHash()
	if len(rootHash) != 64 { // SHA256 hex = 64 chars
		t.Errorf("Expected 64 char hash, got %d", len(rootHash))
	}
}

func TestGenerateAndVerifyProofSingleElement(t *testing.T) {
	txHashes := []string{"tx1"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Generate proof for tx1
	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify the proof
	if !VerifyProof(proof) {
		t.Error("Valid proof should verify successfully")
	}
}

func TestGenerateAndVerifyProofTwoElements(t *testing.T) {
	txHashes := []string{"tx1", "tx2"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Test proof for tx1
	proof1, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof for tx1: %v", err)
	}

	if !VerifyProof(proof1) {
		t.Error("Valid proof for tx1 should verify successfully")
	}

	// Test proof for tx2
	proof2, err := tree.GenerateProof("tx2")
	if err != nil {
		t.Fatalf("Failed to generate proof for tx2: %v", err)
	}

	if !VerifyProof(proof2) {
		t.Error("Valid proof for tx2 should verify successfully")
	}
}

func TestGenerateAndVerifyProofFourElements(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Test proof for each transaction
	for _, txHash := range txHashes {
		proof, err := tree.GenerateProof(txHash)
		if err != nil {
			t.Fatalf("Failed to generate proof for %s: %v", txHash, err)
		}

		if !VerifyProof(proof) {
			t.Errorf("Valid proof for %s should verify successfully", txHash)
		}

		// Check proof structure
		if len(proof.Siblings) != 2 { // log2(4) = 2 levels
			t.Errorf("Expected 2 siblings for 4 elements, got %d", len(proof.Siblings))
		}
	}
}

func TestGenerateAndVerifyProofOddElements(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4", "tx5"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Test proof for each transaction
	for _, txHash := range txHashes {
		proof, err := tree.GenerateProof(txHash)
		if err != nil {
			t.Fatalf("Failed to generate proof for %s: %v", txHash, err)
		}

		if !VerifyProof(proof) {
			t.Errorf("Valid proof for %s should verify successfully", txHash)
		}
	}
}

func TestInvalidProof(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Tamper with the proof
	proof.Siblings[0] = "0000000000000000000000000000000000000000000000000000000000000000"

	if VerifyProof(proof) {
		t.Error("Tampered proof should not verify")
	}
}

func TestInvalidTxHash(t *testing.T) {
	txHashes := []string{"tx1", "tx2"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	// Try to generate proof for non-existent transaction
	_, err = tree.GenerateProof("tx_not_exist")
	if err != ErrTransactionNotFound {
		t.Errorf("Expected ErrTransactionNotFound, got %v", err)
	}
}

func TestProofWithWrongRoot(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Change the merkle root
	proof.MerkleRoot = "0000000000000000000000000000000000000000000000000000000000000000"

	if VerifyProof(proof) {
		t.Error("Proof with wrong root should not verify")
	}
}

func TestVerifyProofWithRoot(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create merkle tree: %v", err)
	}

	proof, err := tree.GenerateProof("tx1")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify using the helper function
	result := VerifyProofWithRoot(proof.TxHash, proof.MerkleRoot, proof.Siblings, proof.Directions)
	if !result {
		t.Error("VerifyProofWithRoot should return true for valid proof")
	}
}

func TestComputeMerkleRoot(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}

	root1, err := ComputeMerkleRoot(txHashes)
	if err != nil {
		t.Fatalf("Failed to compute merkle root: %v", err)
	}

	// Create tree and compare
	tree, _ := NewMerkleTreeFromHashes(txHashes)
	root2 := tree.GetRootHash()

	if root1 != root2 {
		t.Error("ComputeMerkleRoot should match tree root")
	}
}

func TestComputeMerkleRootEmpty(t *testing.T) {
	_, err := ComputeMerkleRoot([]string{})
	if err != ErrEmptyTree {
		t.Errorf("Expected ErrEmptyTree, got %v", err)
	}
}

func TestMerkleTreeDeterministic(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}

	tree1, _ := NewMerkleTreeFromHashes(txHashes)
	tree2, _ := NewMerkleTreeFromHashes(txHashes)

	if tree1.GetRootHash() != tree2.GetRootHash() {
		t.Error("Merkle tree should be deterministic")
	}
}

func TestMerkleTreeOrderMatters(t *testing.T) {
	tree1, _ := NewMerkleTreeFromHashes([]string{"tx1", "tx2"})
	tree2, _ := NewMerkleTreeFromHashes([]string{"tx2", "tx1"})

	if tree1.GetRootHash() == tree2.GetRootHash() {
		t.Error("Transaction order should affect merkle root")
	}
}

func TestLargeMerkleTree(t *testing.T) {
	// Test with 100 transactions
	var txHashes []string
	for i := 0; i < 100; i++ {
		txHashes = append(txHashes, hex.EncodeToString([]byte{byte(i)}))
	}

	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		t.Fatalf("Failed to create large merkle tree: %v", err)
	}

	// Verify proofs for a few transactions
	for _, i := range []int{0, 49, 99} {
		proof, err := tree.GenerateProof(txHashes[i])
		if err != nil {
			t.Fatalf("Failed to generate proof for tx %d: %v", i, err)
		}

		if !VerifyProof(proof) {
			t.Errorf("Proof for tx %d should verify", i)
		}
	}
}

func TestNilProof(t *testing.T) {
	if VerifyProof(nil) {
		t.Error("Nil proof should not verify")
	}
}

func TestProofMismatchedLengths(t *testing.T) {
	proof := &MerkleProof{
		TxHash:     "tx1",
		MerkleRoot: "abc",
		Siblings:   []string{"a", "b"},
		Directions: []bool{true}, // Mismatched length
	}

	if VerifyProof(proof) {
		t.Error("Proof with mismatched lengths should not verify")
	}
}
