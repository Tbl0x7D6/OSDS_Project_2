// Package block defines the block structure for the blockchain
package block

import (
	"blockchain/pkg/merkle"
	"blockchain/pkg/transaction"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Block represents a single block in the blockchain
type Block struct {
	Index        int64                      `json:"index"`
	Timestamp    int64                      `json:"timestamp"`
	Transactions []*transaction.Transaction `json:"transactions"`
	MerkleRoot   string                     `json:"merkle_root"`
	PrevHash     string                     `json:"prev_hash"`
	Hash         string                     `json:"hash"`
	Nonce        int64                      `json:"nonce"`
	Difficulty   int                        `json:"difficulty"`
	MinerID      string                     `json:"miner_id"`
}

// NewBlock creates a new block with the given transactions and previous hash
func NewBlock(index int64, transactions []*transaction.Transaction, prevHash string, difficulty int, minerID string) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().UnixNano(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Nonce:        0,
		Difficulty:   difficulty,
		MinerID:      minerID,
	}
	// Calculate Merkle Root
	block.MerkleRoot = block.CalculateMerkleRoot()
	return block
}

// NewGenesisBlock creates the genesis block (first block in the chain)
func NewGenesisBlock(difficulty int) *Block {
	// Genesis block uses a coinbase transaction
	genesisTransaction := transaction.NewCoinbaseTransaction("genesis", 0, 0)
	block := &Block{
		Index:        0,
		Timestamp:    time.Now().UnixNano(),
		Transactions: []*transaction.Transaction{genesisTransaction},
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
		Difficulty:   difficulty,
		MinerID:      "genesis",
	}
	block.MerkleRoot = block.CalculateMerkleRoot()
	block.Hash = block.CalculateHash()
	return block
}

// CalculateMerkleRoot computes the Merkle root of all transactions
func (b *Block) CalculateMerkleRoot() string {
	if len(b.Transactions) == 0 {
		return ""
	}

	txHashes := make([]string, len(b.Transactions))
	for i, tx := range b.Transactions {
		txHashes[i] = tx.ID
	}

	root, err := merkle.ComputeMerkleRoot(txHashes)
	if err != nil {
		return ""
	}
	return root
}

// CalculateHash computes the SHA256 hash of the block
func (b *Block) CalculateHash() string {
	// Use MerkleRoot instead of serializing all transactions
	data := fmt.Sprintf("%d%d%s%s%d%d%s",
		b.Index, b.Timestamp, b.MerkleRoot, b.PrevHash, b.Nonce, b.Difficulty, b.MinerID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SetHash calculates and sets the block's hash
func (b *Block) SetHash() {
	b.Hash = b.CalculateHash()
}

// Serialize converts the block to JSON bytes
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// DeserializeBlock converts JSON bytes to a Block
func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	return &block, err
}

// ValidateTransactions checks if all transactions in the block are valid
func (b *Block) ValidateTransactions() bool {
	for _, tx := range b.Transactions {
		if !tx.Verify() {
			return false
		}
	}
	return true
}

// HasValidHash checks if the block's hash matches its contents
func (b *Block) HasValidHash() bool {
	return b.Hash == b.CalculateHash()
}

var powNibbleLeadingZeros = [16]int{4, 3, 2, 2, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0}

func countLeadingZeroBits(hash string) (int, bool) {
	zeros := 0
	for i := 0; i < len(hash); i++ {
		var v int
		switch {
		case hash[i] >= '0' && hash[i] <= '9':
			v = int(hash[i] - '0')
		case hash[i] >= 'a' && hash[i] <= 'f':
			v = int(hash[i]-'a') + 10
		case hash[i] >= 'A' && hash[i] <= 'F':
			v = int(hash[i]-'A') + 10
		default:
			return 0, false
		}

		if v == 0 {
			zeros += 4
			continue
		}

		zeros += powNibbleLeadingZeros[v]
		return zeros, true
	}

	return zeros, true
}

// HasValidPoW checks if the block has a valid proof of work
func (b *Block) HasValidPoW() bool {
	leading, ok := countLeadingZeroBits(b.Hash)
	return ok && leading >= b.Difficulty
}

// Clone creates a deep copy of the block
func (b *Block) Clone() *Block {
	transactions := make([]*transaction.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		// Deep copy the transaction
		inputs := make([]transaction.TxInput, len(tx.Inputs))
		copy(inputs, tx.Inputs)
		outputs := make([]transaction.TxOutput, len(tx.Outputs))
		copy(outputs, tx.Outputs)

		transactions[i] = &transaction.Transaction{
			ID:      tx.ID,
			Inputs:  inputs,
			Outputs: outputs,
		}
	}

	return &Block{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: transactions,
		MerkleRoot:   b.MerkleRoot,
		PrevHash:     b.PrevHash,
		Hash:         b.Hash,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
		MinerID:      b.MinerID,
	}
}

// String returns a string representation of the block
func (b *Block) String() string {
	return fmt.Sprintf("Block{Index: %d, Hash: %s..., PrevHash: %s..., TxCount: %d, Nonce: %d, Miner: %s}",
		b.Index, b.Hash[:8], b.PrevHash[:8], len(b.Transactions), b.Nonce, b.MinerID)
}

// HasValidMerkleRoot checks if the block's Merkle root is correct
func (b *Block) HasValidMerkleRoot() bool {
	return b.MerkleRoot == b.CalculateMerkleRoot()
}

// GetMerkleTree builds and returns the Merkle Tree for this block
func (b *Block) GetMerkleTree() (*merkle.MerkleTree, error) {
	if len(b.Transactions) == 0 {
		return nil, merkle.ErrEmptyTree
	}

	txHashes := make([]string, len(b.Transactions))
	for i, tx := range b.Transactions {
		txHashes[i] = tx.ID
	}

	return merkle.NewMerkleTreeFromHashes(txHashes)
}

// GenerateSPVProof generates a SPV proof for a transaction in this block
func (b *Block) GenerateSPVProof(txID string) (*merkle.MerkleProof, error) {
	tree, err := b.GetMerkleTree()
	if err != nil {
		return nil, err
	}

	return tree.GenerateProof(txID)
}

// VerifyTransactionInBlock verifies that a transaction is included in this block using SPV
func (b *Block) VerifyTransactionInBlock(txID string) bool {
	proof, err := b.GenerateSPVProof(txID)
	if err != nil {
		return false
	}

	return merkle.VerifyProof(proof)
}
