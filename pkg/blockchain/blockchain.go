// Package blockchain implements the blockchain data structure and operations
package blockchain

import (
	"blockchain/pkg/block"
	"blockchain/pkg/transaction"
	"errors"
	"sync"
)

var (
	ErrInvalidBlock       = errors.New("invalid block")
	ErrInvalidChain       = errors.New("invalid chain")
	ErrInvalidPrevHash    = errors.New("invalid previous hash")
	ErrInvalidPoW         = errors.New("invalid proof of work")
	ErrInvalidIndex       = errors.New("invalid block index")
	ErrBlockExists        = errors.New("block already exists")
	ErrInvalidGenesis     = errors.New("invalid genesis block")
	ErrChainTooShort      = errors.New("chain too short to replace")
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrDoubleSpend        = errors.New("double spend detected")
)

const (
	// BaseSubsidy is the fixed block subsidy used for miner rewards (in satoshi)
	BaseSubsidy int64 = 5000000000
)

// Blockchain represents the entire blockchain
type Blockchain struct {
	Blocks     []*block.Block
	Difficulty int
	UTXOSet    *transaction.UTXOSet
	mu         sync.RWMutex
}

// NewBlockchain creates a new blockchain with a genesis block
func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:     make([]*block.Block, 0),
		Difficulty: difficulty,
		UTXOSet:    transaction.NewUTXOSet(),
	}
	// Create genesis block
	genesis := block.NewGenesisBlock(difficulty)
	bc.Blocks = append(bc.Blocks, genesis)
	// Process genesis block transactions
	for _, tx := range genesis.Transactions {
		bc.UTXOSet.ProcessTransaction(tx)
	}
	return bc
}

// NewBlockchainFromBlocks creates a blockchain from existing blocks
func NewBlockchainFromBlocks(blocks []*block.Block, difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:     blocks,
		Difficulty: difficulty,
		UTXOSet:    transaction.NewUTXOSet(),
	}
	// Rebuild UTXO set from blocks
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			bc.UTXOSet.ProcessTransaction(tx)
		}
	}
	return bc
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

	// Update UTXO set with transactions from the new block
	for _, tx := range newBlock.Transactions {
		bc.UTXOSet.ProcessTransaction(tx)
	}

	return nil
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

	// Validate all transactions (basic validation)
	if !newBlock.ValidateTransactions() {
		return ErrInvalidBlock
	}

	// Validate transactions against UTXO set
	if err := bc.ValidateBlockTransactions(newBlock); err != nil {
		return err
	}

	return nil
}

// ValidateBlockTransactions validates all transactions in a block against the UTXO set
func (bc *Blockchain) ValidateBlockTransactions(newBlock *block.Block) error {
	// Create a temporary UTXO set copy to track spent outputs within this block
	tempUTXO := bc.UTXOSet.Copy()

	var totalFees int64
	var coinbaseValue int64
	coinbaseCount := 0

	for i, tx := range newBlock.Transactions {
		if tx.IsCoinbase() {
			coinbaseCount++
			if coinbaseCount > 1 {
				return ErrInvalidTransaction
			}
			// Enforce coinbase placement at the start of the block
			if i != 0 {
				return ErrInvalidTransaction
			}
			coinbaseValue = tx.TotalOutputValue()
			// Process immediately so any (optional) spends within the same block still see the outputs
			tempUTXO.ProcessTransaction(tx)
			continue
		}

		// Validate against current UTXO set
		if err := tempUTXO.ValidateTransaction(tx); err != nil {
			return ErrInvalidTransaction
		}

		// Accumulate fees before mutating the UTXO set
		totalFees += tx.GetFee(tempUTXO)

		// Process the transaction (remove spent, add new)
		tempUTXO.ProcessTransaction(tx)
	}

	// Require exactly one coinbase transaction
	if coinbaseCount != 1 {
		return ErrInvalidTransaction
	}

	expectedReward := BaseSubsidy + totalFees
	if coinbaseValue > expectedReward {
		return ErrInvalidTransaction
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

	// Replace the chain and UTXO set
	bc.Blocks = newBlocks
	bc.UTXOSet = newChain.UTXOSet
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

// ValidateTransaction validates a single transaction against the UTXO set
func (bc *Blockchain) ValidateTransaction(tx *transaction.Transaction) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Basic validation
	if !tx.Verify() {
		return ErrInvalidTransaction
	}

	// UTXO validation (skip for coinbase)
	if !tx.IsCoinbase() {
		if err := bc.UTXOSet.ValidateTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

// GetUTXOSet returns a copy of the current UTXO set
func (bc *Blockchain) GetUTXOSet() *transaction.UTXOSet {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.UTXOSet.Copy()
}

// GetBalance returns the balance for an address
func (bc *Blockchain) GetBalance(address string) int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.UTXOSet.GetBalance(address)
}

// GetRecentBlocks returns the most recent n blocks for difficulty calculation
func (bc *Blockchain) GetRecentBlocks(n int) []*block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	length := len(bc.Blocks)
	if length == 0 {
		return nil
	}

	if n > length {
		n = length
	}

	start := length - n
	blocks := make([]*block.Block, n)
	for i := start; i < length; i++ {
		blocks[i-start] = bc.Blocks[i].Clone()
	}
	return blocks
}
