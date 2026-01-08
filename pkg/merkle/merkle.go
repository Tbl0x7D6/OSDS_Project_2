// Package merkle implements the Merkle Tree data structure for transaction verification
package merkle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrEmptyTree           = errors.New("cannot create merkle tree from empty data")
	ErrInvalidProof        = errors.New("invalid merkle proof")
	ErrTransactionNotFound = errors.New("transaction not found in tree")
)

// MerkleNode represents a node in the Merkle Tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Hash  []byte
}

// MerkleTree represents a Merkle Tree
type MerkleTree struct {
	Root       *MerkleNode
	LeafHashes [][]byte // Original leaf hashes for proof generation
}

// MerkleProof represents a proof that a transaction is included in the Merkle Tree
type MerkleProof struct {
	TxHash     string   `json:"tx_hash"`     // The transaction hash being proven
	MerkleRoot string   `json:"merkle_root"` // Expected Merkle root
	Siblings   []string `json:"siblings"`    // Sibling hashes on the path to root
	Directions []bool   `json:"directions"`  // true = sibling is on the right, false = sibling is on the left
}

// NewMerkleNode creates a new Merkle Tree node
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := &MerkleNode{}

	if left == nil && right == nil {
		// Leaf node - hash the data
		hash := sha256.Sum256(data)
		node.Hash = hash[:]
	} else {
		// Internal node - hash the concatenation of children
		var combined []byte
		combined = append(combined, left.Hash...)
		if right != nil {
			combined = append(combined, right.Hash...)
		} else {
			// If there's no right child, duplicate the left
			combined = append(combined, left.Hash...)
		}
		hash := sha256.Sum256(combined)
		node.Hash = hash[:]
	}

	node.Left = left
	node.Right = right
	return node
}

// NewMerkleTree creates a new Merkle Tree from a list of data (transaction hashes)
func NewMerkleTree(data [][]byte) (*MerkleTree, error) {
	if len(data) == 0 {
		return nil, ErrEmptyTree
	}

	// Store leaf hashes for proof generation
	leafHashes := make([][]byte, len(data))
	for i, d := range data {
		hash := sha256.Sum256(d)
		leafHashes[i] = hash[:]
	}

	// Create leaf nodes
	var nodes []*MerkleNode
	for _, d := range data {
		node := NewMerkleNode(nil, nil, d)
		nodes = append(nodes, node)
	}

	// Build the tree bottom-up
	for len(nodes) > 1 {
		var level []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				// Pair exists
				node := NewMerkleNode(nodes[i], nodes[i+1], nil)
				level = append(level, node)
			} else {
				// Odd node - duplicate it
				node := NewMerkleNode(nodes[i], nodes[i], nil)
				level = append(level, node)
			}
		}

		nodes = level
	}

	return &MerkleTree{
		Root:       nodes[0],
		LeafHashes: leafHashes,
	}, nil
}

// NewMerkleTreeFromHashes creates a Merkle Tree from hex-encoded transaction hashes
func NewMerkleTreeFromHashes(txHashes []string) (*MerkleTree, error) {
	if len(txHashes) == 0 {
		return nil, ErrEmptyTree
	}

	data := make([][]byte, len(txHashes))
	for i, h := range txHashes {
		hashBytes, err := hex.DecodeString(h)
		if err != nil {
			// If it's not a valid hex, use the string bytes directly
			data[i] = []byte(h)
		} else {
			data[i] = hashBytes
		}
	}

	return NewMerkleTree(data)
}

// GetRootHash returns the root hash as a hex string
func (mt *MerkleTree) GetRootHash() string {
	if mt.Root == nil {
		return ""
	}
	return hex.EncodeToString(mt.Root.Hash)
}

// GetRootHashBytes returns the root hash as bytes
func (mt *MerkleTree) GetRootHashBytes() []byte {
	if mt.Root == nil {
		return nil
	}
	return mt.Root.Hash
}

// GenerateProof generates a Merkle proof for a given transaction hash
func (mt *MerkleTree) GenerateProof(txHash string) (*MerkleProof, error) {
	if mt.Root == nil {
		return nil, ErrEmptyTree
	}

	// Convert txHash to bytes
	txBytes, err := hex.DecodeString(txHash)
	if err != nil {
		txBytes = []byte(txHash)
	}

	// Hash the transaction data to get the leaf hash
	leafHash := sha256.Sum256(txBytes)

	// Find the leaf index
	leafIndex := -1
	for i, h := range mt.LeafHashes {
		if bytes.Equal(h, leafHash[:]) {
			leafIndex = i
			break
		}
	}

	if leafIndex == -1 {
		return nil, ErrTransactionNotFound
	}

	// Generate the proof path
	siblings, directions := mt.generateProofPath(leafIndex)

	return &MerkleProof{
		TxHash:     txHash,
		MerkleRoot: mt.GetRootHash(),
		Siblings:   siblings,
		Directions: directions,
	}, nil
}

// generateProofPath generates the sibling hashes and directions for a proof
func (mt *MerkleTree) generateProofPath(leafIndex int) ([]string, []bool) {
	var siblings []string
	var directions []bool

	// Rebuild the tree level by level to collect siblings
	// Start with leaf hashes
	currentLevel := make([][]byte, len(mt.LeafHashes))
	copy(currentLevel, mt.LeafHashes)

	index := leafIndex

	for len(currentLevel) > 1 {
		var nextLevel [][]byte
		siblingIndex := -1

		// Determine sibling index
		if index%2 == 0 {
			// Current is on the left, sibling is on the right
			if index+1 < len(currentLevel) {
				siblingIndex = index + 1
				directions = append(directions, true) // Sibling is on the right
			} else {
				// No sibling, use self (duplicate)
				siblingIndex = index
				directions = append(directions, true)
			}
		} else {
			// Current is on the right, sibling is on the left
			siblingIndex = index - 1
			directions = append(directions, false) // Sibling is on the left
		}

		siblings = append(siblings, hex.EncodeToString(currentLevel[siblingIndex]))

		// Build next level
		for i := 0; i < len(currentLevel); i += 2 {
			var combined []byte
			combined = append(combined, currentLevel[i]...)
			if i+1 < len(currentLevel) {
				combined = append(combined, currentLevel[i+1]...)
			} else {
				combined = append(combined, currentLevel[i]...)
			}
			hash := sha256.Sum256(combined)
			nextLevel = append(nextLevel, hash[:])
		}

		currentLevel = nextLevel
		index = index / 2
	}

	return siblings, directions
}

// VerifyProof verifies a Merkle proof
// Returns true if the proof is valid, false otherwise
func VerifyProof(proof *MerkleProof) bool {
	if proof == nil || len(proof.Siblings) != len(proof.Directions) {
		return false
	}

	// Convert txHash to bytes and hash it to get the leaf hash
	txBytes, err := hex.DecodeString(proof.TxHash)
	if err != nil {
		txBytes = []byte(proof.TxHash)
	}
	currentHash := sha256.Sum256(txBytes)
	current := currentHash[:]

	// Walk up the tree using the proof
	for i, siblingHex := range proof.Siblings {
		sibling, err := hex.DecodeString(siblingHex)
		if err != nil {
			return false
		}

		var combined []byte
		if proof.Directions[i] {
			// Sibling is on the right: current || sibling
			combined = append(combined, current...)
			combined = append(combined, sibling...)
		} else {
			// Sibling is on the left: sibling || current
			combined = append(combined, sibling...)
			combined = append(combined, current...)
		}

		hash := sha256.Sum256(combined)
		current = hash[:]
	}

	// Compare with the expected root
	computedRoot := hex.EncodeToString(current)
	return computedRoot == proof.MerkleRoot
}

// VerifyProofWithRoot verifies a Merkle proof against a given root
func VerifyProofWithRoot(txHash string, merkleRoot string, siblings []string, directions []bool) bool {
	proof := &MerkleProof{
		TxHash:     txHash,
		MerkleRoot: merkleRoot,
		Siblings:   siblings,
		Directions: directions,
	}
	return VerifyProof(proof)
}

// ComputeMerkleRoot computes the Merkle root from transaction hashes
// This is a convenience function for creating blocks
func ComputeMerkleRoot(txHashes []string) (string, error) {
	tree, err := NewMerkleTreeFromHashes(txHashes)
	if err != nil {
		return "", err
	}
	return tree.GetRootHash(), nil
}
