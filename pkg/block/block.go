// Package block defines the block structure for the blockchain
package block

import (
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
	return block
}

// NewGenesisBlock creates the genesis block (first block in the chain)
func NewGenesisBlock(difficulty int) *Block {
	genesisTransaction := transaction.NewTransaction("system", "genesis", 0)
	genesisTransaction.Sign("genesis_key")
	block := &Block{
		Index:        0,
		Timestamp:    time.Now().UnixNano(),
		Transactions: []*transaction.Transaction{genesisTransaction},
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
		Difficulty:   difficulty,
		MinerID:      "genesis",
	}
	block.Hash = block.CalculateHash()
	return block
}

// CalculateHash computes the SHA256 hash of the block
func (b *Block) CalculateHash() string {
	// Serialize transactions
	txData := ""
	for _, tx := range b.Transactions {
		txData += tx.ID
	}

	data := fmt.Sprintf("%d%d%s%s%d%d%s",
		b.Index, b.Timestamp, txData, b.PrevHash, b.Nonce, b.Difficulty, b.MinerID)
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

// HasValidPoW checks if the block has a valid proof of work
func (b *Block) HasValidPoW() bool {
	// Check that hash has required number of leading zeros
	prefix := ""
	for i := 0; i < b.Difficulty; i++ {
		prefix += "0"
	}
	return len(b.Hash) >= b.Difficulty && b.Hash[:b.Difficulty] == prefix
}

// Clone creates a deep copy of the block
func (b *Block) Clone() *Block {
	transactions := make([]*transaction.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		txCopy := *tx
		transactions[i] = &txCopy
	}

	return &Block{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: transactions,
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
