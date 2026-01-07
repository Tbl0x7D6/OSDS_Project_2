// Package blockchain implements the blockchain data structure and operations
package blockchain

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"errors"
	"sync"
)

var (
	ErrInvalidBlock    = errors.New("invalid block")
	ErrInvalidChain    = errors.New("invalid chain")
	ErrInvalidPrevHash = errors.New("invalid previous hash")
	ErrInvalidPoW      = errors.New("invalid proof of work")
	ErrInvalidIndex    = errors.New("invalid block index")
	ErrBlockExists     = errors.New("block already exists")
	ErrInvalidGenesis  = errors.New("invalid genesis block")
	ErrChainTooShort   = errors.New("chain too short to replace")
)

// Blockchain represents the entire blockchain
type Blockchain struct {
	Blocks     []*block.Block
	Difficulty int
	mu         sync.RWMutex
}

// NewBlockchain creates a new blockchain with a genesis block
func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:     make([]*block.Block, 0),
		Difficulty: difficulty,
	}
	// Create genesis block
	genesis := block.NewGenesisBlock(difficulty)
	bc.Blocks = append(bc.Blocks, genesis)
	return bc
}

// NewBlockchainFromBlocks creates a blockchain from existing blocks
func NewBlockchainFromBlocks(blocks []*block.Block, difficulty int) *Blockchain {
	return &Blockchain{
		Blocks:     blocks,
		Difficulty: difficulty,
	}
}

// GetLatestBlock returns the most recent block in the chain
func (bc *Blockchain) GetLatestBlock() *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlockByIndex returns the block at the specified index
func (bc *Blockchain) GetBlockByIndex(index int64) *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if index < 0 || index >= int64(len(bc.Blocks)) {
		return nil
	}
	return bc.Blocks[index]
}

// GetBlockByHash returns the block with the specified hash
func (bc *Blockchain) GetBlockByHash(hash string) *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for _, b := range bc.Blocks {
		if b.Hash == hash {
			return b
		}
	}
	return nil
}

// GetLength returns the number of blocks in the chain
func (bc *Blockchain) GetLength() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks)
}

// AddBlock adds a validated block to the blockchain
func (bc *Blockchain) AddBlock(newBlock *block.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Validate the block
	if err := bc.validateBlockUnlocked(newBlock); err != nil {
		return err
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
}

// ValidateBlock validates a single block against the chain
func (bc *Blockchain) ValidateBlock(newBlock *block.Block) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.validateBlockUnlocked(newBlock)
}

// validateBlockUnlocked validates a block without acquiring the lock
func (bc *Blockchain) validateBlockUnlocked(newBlock *block.Block) error {
	latestBlock := bc.Blocks[len(bc.Blocks)-1]

	// Check if the index is correct
	if newBlock.Index != latestBlock.Index+1 {
		return ErrInvalidIndex
	}

	// Check if the previous hash is correct
	if newBlock.PrevHash != latestBlock.Hash {
		return ErrInvalidPrevHash
	}

	// Check if the hash is valid
	if !newBlock.HasValidHash() {
		return ErrInvalidBlock
	}

	// Check if PoW is valid
	if !newBlock.HasValidPoW() {
		return ErrInvalidPoW
	}

	// Validate all transactions
	if !newBlock.ValidateTransactions() {
		return ErrInvalidBlock
	}

	return nil
}

// ValidateChain validates the entire blockchain
func (bc *Blockchain) ValidateChain() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return ErrInvalidChain
	}

	// Validate genesis block
	genesis := bc.Blocks[0]
	if genesis.Index != 0 {
		return ErrInvalidGenesis
	}
	if genesis.PrevHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return ErrInvalidGenesis
	}
	if !genesis.HasValidHash() {
		return ErrInvalidGenesis
	}

	// Validate each subsequent block
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		// Check index
		if currentBlock.Index != prevBlock.Index+1 {
			return ErrInvalidIndex
		}

		// Check previous hash pointer
		if currentBlock.PrevHash != prevBlock.Hash {
			return ErrInvalidPrevHash
		}

		// Check hash is valid
		if !currentBlock.HasValidHash() {
			return ErrInvalidBlock
		}

		// Check PoW is valid
		if !currentBlock.HasValidPoW() {
			return ErrInvalidPoW
		}

		// Check transactions are valid
		if !currentBlock.ValidateTransactions() {
			return ErrInvalidBlock
		}
	}

	return nil
}

// ReplaceChain replaces the current chain with a new one if it's longer and valid
// This implements the longest chain rule
func (bc *Blockchain) ReplaceChain(newBlocks []*block.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if new chain is longer
	if len(newBlocks) <= len(bc.Blocks) {
		return ErrChainTooShort
	}

	// Validate the new chain
	newChain := NewBlockchainFromBlocks(newBlocks, bc.Difficulty)
	if err := newChain.ValidateChain(); err != nil {
		return err
	}

	// Replace the chain
	bc.Blocks = newBlocks
	return nil
}

// CreateBlock creates a new block with pending transactions
func (bc *Blockchain) CreateBlock(transactions []*transaction.Transaction, minerID string) *block.Block {
	bc.mu.RLock()
	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	bc.mu.RUnlock()

	newBlock := block.NewBlock(
		latestBlock.Index+1,
		transactions,
		latestBlock.Hash,
		bc.Difficulty,
		minerID,
	)
	return newBlock
}

// GetBlocks returns a copy of all blocks in the chain
func (bc *Blockchain) GetBlocks() []*block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	blocks := make([]*block.Block, len(bc.Blocks))
	for i, b := range bc.Blocks {
		blocks[i] = b.Clone()
	}
	return blocks
}

// GetBlocksFrom returns blocks starting from a specific index
func (bc *Blockchain) GetBlocksFrom(startIndex int64) []*block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if startIndex < 0 || startIndex >= int64(len(bc.Blocks)) {
		return nil
	}

	blocks := make([]*block.Block, len(bc.Blocks)-int(startIndex))
	for i := int(startIndex); i < len(bc.Blocks); i++ {
		blocks[i-int(startIndex)] = bc.Blocks[i].Clone()
	}
	return blocks
}

// SetDifficulty updates the mining difficulty
func (bc *Blockchain) SetDifficulty(difficulty int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Difficulty = difficulty
}

// GetDifficulty returns the current mining difficulty
func (bc *Blockchain) GetDifficulty() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Difficulty
}
