// Package network implements the P2P network layer for miners
package network

import (
	"blockchain/pkg/block"
	"blockchain/pkg/blockchain"
	"blockchain/pkg/pow"
	"blockchain/pkg/transaction"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// MessageType represents the type of network message
type MessageType int

const (
	MsgTransaction MessageType = iota
	MsgBlock
	MsgRequestChain
	MsgResponseChain
	MsgPing
	MsgPong
)

// Message represents a network message
type Message struct {
	Type    MessageType
	Payload []byte
}

// PeerInfo contains information about a peer
type PeerInfo struct {
	ID      string
	Address string
}

// Miner represents a mining node in the network
type Miner struct {
	ID            string
	Address       string
	Blockchain    *blockchain.Blockchain
	PendingTxs    []*transaction.Transaction
	Peers         []PeerInfo
	txMutex       sync.RWMutex
	listener      net.Listener
	rpcServer     *rpc.Server
	blockCallback func(*block.Block)
	miningEnabled bool
	miningMutex   sync.RWMutex
	stopMining    chan struct{}
	isMalicious   bool // For testing: if true, creates invalid blocks
	maliciousType string
	stopped       bool
	stoppedMutex  sync.RWMutex
}

// RPCService provides RPC methods for the miner
type RPCService struct {
	miner *Miner
}

// TransactionArgs represents arguments for submitting a transaction
type TransactionArgs struct {
	InputSpecs []struct {
		TxID     string
		OutIndex int
	} // UTXOs to spend
	Outputs     []transaction.TxOutput // Transaction outputs
	PrivateKeys map[string]string      // Map of public key hex -> private key hex
}

// TransactionReply represents the reply after submitting a transaction
type TransactionReply struct {
	Success bool
	TxID    string
	Error   string
}

// BlockArgs represents arguments for receiving a block
type BlockArgs struct {
	BlockData []byte
}

// BlockReply represents the reply after receiving a block
type BlockReply struct {
	Success bool
	Error   string
}

// ChainArgs represents arguments for chain synchronization
type ChainArgs struct {
	StartIndex int64
}

// ChainReply represents the reply with chain data
type ChainReply struct {
	Blocks [][]byte
	Length int
}

// StatusReply represents the miner status
type StatusReply struct {
	ID          string
	ChainLength int
	PendingTxs  int
	Peers       int
	Mining      bool
}

// NewMiner creates a new mining node
func NewMiner(id, address string, difficulty int, peers []PeerInfo) *Miner {
	return &Miner{
		ID:            id,
		Address:       address,
		Blockchain:    blockchain.NewBlockchain(difficulty),
		PendingTxs:    make([]*transaction.Transaction, 0),
		Peers:         peers,
		miningEnabled: false,
		stopMining:    make(chan struct{}),
		isMalicious:   false,
	}
}

// NewMaliciousMiner creates a miner that generates invalid blocks for testing
func NewMaliciousMiner(id, address string, difficulty int, peers []PeerInfo, maliciousType string) *Miner {
	miner := NewMiner(id, address, difficulty, peers)
	miner.isMalicious = true
	miner.maliciousType = maliciousType
	return miner
}

// Start starts the miner's RPC server
func (m *Miner) Start() error {
	m.rpcServer = rpc.NewServer()
	service := &RPCService{miner: m}
	err := m.rpcServer.Register(service)
	if err != nil {
		return fmt.Errorf("failed to register RPC service: %v", err)
	}

	listener, err := net.Listen("tcp", m.Address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	m.listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Listener was closed
				return
			}
			go m.rpcServer.ServeConn(conn)
		}
	}()

	log.Printf("[%s] Miner started on %s", m.ID, m.Address)
	return nil
}

// Stop stops the miner
func (m *Miner) Stop() {
	m.stoppedMutex.Lock()
	m.stopped = true
	m.stoppedMutex.Unlock()

	m.StopMining()
	if m.listener != nil {
		m.listener.Close()
	}
	log.Printf("[%s] Miner stopped", m.ID)
}

// IsStopped returns true if the miner has been stopped
func (m *Miner) IsStopped() bool {
	m.stoppedMutex.RLock()
	defer m.stoppedMutex.RUnlock()
	return m.stopped
}

// SubmitTransaction RPC method to receive a transaction from a client
func (s *RPCService) SubmitTransaction(args *TransactionArgs, reply *TransactionReply) error {
	// Create a transaction using the provided UTXO inputs and outputs
	utxoSet := s.miner.Blockchain.GetUTXOSet()

	// Use CreateTransaction with the new signature
	tx, err := utxoSet.CreateTransaction(args.InputSpecs, args.Outputs, args.PrivateKeys)
	if err != nil {
		reply.Success = false
		reply.Error = fmt.Sprintf("failed to create transaction: %v", err)
		return nil
	}

	if !tx.Verify() {
		reply.Success = false
		reply.Error = "invalid transaction"
		return nil
	}

	// Validate against UTXO set (includes signature verification)
	if err := s.miner.Blockchain.ValidateTransaction(tx); err != nil {
		reply.Success = false
		reply.Error = fmt.Sprintf("transaction validation failed: %v", err)
		return nil
	}

	s.miner.AddTransaction(tx)
	reply.Success = true
	reply.TxID = tx.ID

	// Broadcast transaction to peers
	go s.miner.BroadcastTransaction(tx)

	log.Printf("[%s] Received transaction: %s", s.miner.ID, tx.String())
	return nil
}

// ReceiveTransaction RPC method to receive a transaction from another miner
func (s *RPCService) ReceiveTransaction(args *BlockArgs, reply *TransactionReply) error {
	tx, err := transaction.DeserializeTransaction(args.BlockData)
	if err != nil {
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}

	if !tx.Verify() {
		reply.Success = false
		reply.Error = "invalid transaction"
		return nil
	}

	// Check if we already have this transaction
	s.miner.txMutex.RLock()
	for _, existingTx := range s.miner.PendingTxs {
		if existingTx.ID == tx.ID {
			s.miner.txMutex.RUnlock()
			reply.Success = true
			reply.TxID = tx.ID
			return nil
		}
	}
	s.miner.txMutex.RUnlock()

	// Validate against UTXO set
	if err := s.miner.Blockchain.ValidateTransaction(tx); err != nil {
		reply.Success = false
		reply.Error = fmt.Sprintf("transaction validation failed: %v", err)
		return nil
	}

	s.miner.AddTransaction(tx)
	reply.Success = true
	reply.TxID = tx.ID

	log.Printf("[%s] Received transaction from peer: %s", s.miner.ID, tx.String())
	return nil
}

// ReceiveBlock RPC method to receive a block from another miner
func (s *RPCService) ReceiveBlock(args *BlockArgs, reply *BlockReply) error {
	newBlock, err := block.DeserializeBlock(args.BlockData)
	if err != nil {
		reply.Success = false
		reply.Error = fmt.Sprintf("failed to deserialize block: %v", err)
		return nil
	}

	// Validate the block
	if !newBlock.HasValidHash() {
		reply.Success = false
		reply.Error = "invalid block hash"
		log.Printf("[%s] Rejected block with invalid hash from miner %s", s.miner.ID, newBlock.MinerID)
		return nil
	}

	if !newBlock.HasValidPoW() {
		reply.Success = false
		reply.Error = "invalid proof of work"
		log.Printf("[%s] Rejected block with invalid PoW from miner %s", s.miner.ID, newBlock.MinerID)
		return nil
	}

	if !pow.Validate(newBlock) {
		reply.Success = false
		reply.Error = "PoW validation failed"
		log.Printf("[%s] Rejected block - PoW validation failed from miner %s", s.miner.ID, newBlock.MinerID)
		return nil
	}

	// Try to add the block
	err = s.miner.Blockchain.AddBlock(newBlock)
	if err != nil {
		// If block doesn't fit, might need chain sync
		if errors.Is(err, blockchain.ErrInvalidPrevHash) || errors.Is(err, blockchain.ErrInvalidIndex) {
			// Check if their chain might be longer
			if newBlock.Index > s.miner.Blockchain.GetLatestBlock().Index {
				// Try to sync with the sender (async to not block RPC)
				go s.miner.SyncWithAllPeers()
			}
		}
		reply.Success = false
		reply.Error = err.Error()
		return nil
	}

	log.Printf("[%s] Accepted block #%d from miner %s", s.miner.ID, newBlock.Index, newBlock.MinerID)

	// Remove transactions that are now in the block
	s.miner.RemoveTransactions(newBlock.Transactions)

	// Notify callback if set
	if s.miner.blockCallback != nil {
		s.miner.blockCallback(newBlock)
	}

	reply.Success = true
	return nil
}

// GetChain RPC method to get the blockchain
func (s *RPCService) GetChain(args *ChainArgs, reply *ChainReply) error {
	blocks := s.miner.Blockchain.GetBlocksFrom(args.StartIndex)
	reply.Blocks = make([][]byte, len(blocks))
	for i, b := range blocks {
		data, err := b.Serialize()
		if err != nil {
			return err
		}
		reply.Blocks[i] = data
	}
	reply.Length = s.miner.Blockchain.GetLength()
	return nil
}

// GetStatus RPC method to get miner status
func (s *RPCService) GetStatus(args *struct{}, reply *StatusReply) error {
	s.miner.txMutex.RLock()
	pendingCount := len(s.miner.PendingTxs)
	s.miner.txMutex.RUnlock()

	s.miner.miningMutex.RLock()
	mining := s.miner.miningEnabled
	s.miner.miningMutex.RUnlock()

	reply.ID = s.miner.ID
	reply.ChainLength = s.miner.Blockchain.GetLength()
	reply.PendingTxs = pendingCount
	reply.Peers = len(s.miner.Peers)
	reply.Mining = mining
	return nil
}

// AddTransaction adds a transaction to the pending pool
func (m *Miner) AddTransaction(tx *transaction.Transaction) {
	m.txMutex.Lock()
	defer m.txMutex.Unlock()

	// Check for duplicates
	for _, existingTx := range m.PendingTxs {
		if existingTx.ID == tx.ID {
			return
		}
	}
	m.PendingTxs = append(m.PendingTxs, tx)
}

// RemoveTransactions removes transactions from the pending pool
func (m *Miner) RemoveTransactions(txs []*transaction.Transaction) {
	m.txMutex.Lock()
	defer m.txMutex.Unlock()

	txMap := make(map[string]bool)
	for _, tx := range txs {
		txMap[tx.ID] = true
	}

	newPending := make([]*transaction.Transaction, 0)
	for _, tx := range m.PendingTxs {
		if !txMap[tx.ID] {
			newPending = append(newPending, tx)
		}
	}
	m.PendingTxs = newPending
}

// GetPendingTransactions returns a copy of pending transactions
func (m *Miner) GetPendingTransactions() []*transaction.Transaction {
	m.txMutex.RLock()
	defer m.txMutex.RUnlock()

	txs := make([]*transaction.Transaction, len(m.PendingTxs))
	copy(txs, m.PendingTxs)
	return txs
}

// BroadcastTransaction broadcasts a transaction to all peers
func (m *Miner) BroadcastTransaction(tx *transaction.Transaction) {
	data, err := tx.Serialize()
	if err != nil {
		log.Printf("[%s] Failed to serialize transaction: %v", m.ID, err)
		return
	}

	for _, peer := range m.Peers {
		go func(p PeerInfo) {
			client, err := rpc.Dial("tcp", p.Address)
			if err != nil {
				return
			}
			defer client.Close()

			args := &BlockArgs{BlockData: data}
			var reply TransactionReply
			client.Call("RPCService.ReceiveTransaction", args, &reply)
		}(peer)
	}
}

// filterValidTransactions filters transactions that are valid against current UTXO set
func (m *Miner) filterValidTransactions(txs []*transaction.Transaction) []*transaction.Transaction {
	validTxs := make([]*transaction.Transaction, 0)

	// Create a temporary UTXO set to track spending within this batch
	tempUTXO := m.Blockchain.GetUTXOSet()

	for _, tx := range txs {
		// Skip coinbase transactions (they shouldn't be in pending)
		if tx.IsCoinbase() {
			continue
		}

		// Validate against temp UTXO set
		if err := tempUTXO.ValidateTransaction(tx); err != nil {
			// Invalid transaction, skip it
			continue
		}

		// Process transaction to update temp UTXO (prevent double-spend in same block)
		tempUTXO.ProcessTransaction(tx)
		validTxs = append(validTxs, tx)
	}

	return validTxs
}

// BroadcastBlock broadcasts a block to all peers
func (m *Miner) BroadcastBlock(b *block.Block) {
	// Don't broadcast if miner is stopped
	if m.IsStopped() {
		return
	}

	// Validate all transactions before broadcasting
	if !b.ValidateTransactions() {
		log.Printf("[%s] Block has invalid transactions, not broadcasting", m.ID)
		return
	}

	data, err := b.Serialize()
	if err != nil {
		log.Printf("[%s] Failed to serialize block: %v", m.ID, err)
		return
	}

	for _, peer := range m.Peers {
		go func(p PeerInfo) {
			// Check again before connecting
			if m.IsStopped() {
				return
			}
			client, err := rpc.Dial("tcp", p.Address)
			if err != nil {
				// Silently ignore connection errors (peer may be down)
				return
			}
			defer client.Close()

			args := &BlockArgs{BlockData: data}
			var reply BlockReply
			client.Call("RPCService.ReceiveBlock", args, &reply)
			// Ignore errors - peer may have stopped
		}(peer)
	}
}

// StartMining starts the mining process
func (m *Miner) StartMining() {
	m.miningMutex.Lock()
	if m.miningEnabled {
		m.miningMutex.Unlock()
		return
	}
	m.miningEnabled = true
	m.stopMining = make(chan struct{})
	m.miningMutex.Unlock()

	go m.miningLoop()
	log.Printf("[%s] Mining started", m.ID)
}

// StopMining stops the mining process
func (m *Miner) StopMining() {
	m.miningMutex.Lock()
	if !m.miningEnabled {
		m.miningMutex.Unlock()
		return
	}
	m.miningEnabled = false
	close(m.stopMining)
	m.miningMutex.Unlock()
	log.Printf("[%s] Mining stopped", m.ID)
}

// miningLoop is the main mining loop
func (m *Miner) miningLoop() {
	for {
		select {
		case <-m.stopMining:
			return
		default:
			m.mineBlock()
		}
	}
}

// mineBlock attempts to mine a new block
func (m *Miner) mineBlock() {
	// Get pending transactions (limit to 10 per block for simplicity)
	pendingTxs := m.GetPendingTransactions()

	// Filter and validate pending transactions against current UTXO set
	validTxs := m.filterValidTransactions(pendingTxs)
	if len(validTxs) > 10 {
		validTxs = validTxs[:10]
	}

	// Calculate total fees from transactions
	var totalFees int64
	utxoSet := m.Blockchain.GetUTXOSet()
	for _, tx := range validTxs {
		totalFees += tx.GetFee(utxoSet)
	}

	// Add coinbase transaction (mining reward + fees)
	// 50 BTC = 5,000,000,000 satoshi
	reward := int64(5000000000) + totalFees
	coinbase := transaction.NewCoinbaseTransaction(m.ID, reward, m.Blockchain.GetLatestBlock().Index+1)
	txs := append([]*transaction.Transaction{coinbase}, validTxs...)

	// Create new block
	newBlock := m.Blockchain.CreateBlock(txs, m.ID)

	// Mine the block
	powInstance := pow.NewProofOfWork(newBlock)

	m.miningMutex.RLock()
	stopChan := m.stopMining
	m.miningMutex.RUnlock()

	// Use context for cancellation
	done := make(chan struct{})
	var result *pow.MiningResult

	go func() {
		// Replace nil with context.TODO() to avoid passing a nil context
		result = powInstance.Mine(context.TODO(), func(nonce int64) {
			select {
			case <-stopChan:
				return
			default:
			}
		})
		close(done)
	}()

	// Wait for mining to complete or be cancelled
	select {
	case <-stopChan:
		return
	case <-done:
	}

	if result == nil || !result.Success {
		return
	}

	// For malicious miner testing
	if m.isMalicious {
		switch m.maliciousType {
		case "invalid_hash":
			// Corrupt the hash
			result.Block.Hash = "0000000000000000000000000000000000000000000000000000000000000000"
		case "invalid_pow":
			// Set invalid nonce (hash won't match PoW requirement)
			result.Block.Hash = result.Block.CalculateHash()
			result.Block.Hash = "ffff" + result.Block.Hash[4:]
		case "invalid_prev_hash":
			// Corrupt the previous hash
			result.Block.PrevHash = "0000000000000000000000000000000000000000000000000000000000000001"
			result.Block.Hash = result.Block.CalculateHash()
		}
	}

	// Check if block is still valid (chain may have changed during mining)
	err := m.Blockchain.AddBlock(result.Block)
	if err != nil {
		// This is normal during blockchain competition, another miner beat us
		// No need to log this as it's expected behavior
		return
	}

	log.Printf("[%s] Mined block #%d with %d transactions, nonce: %d",
		m.ID, result.Block.Index, len(result.Block.Transactions), result.Nonce)

	// Remove included transactions from pending pool
	m.RemoveTransactions(txs)

	// Broadcast the block
	m.BroadcastBlock(result.Block)

	// Notify callback
	if m.blockCallback != nil {
		m.blockCallback(result.Block)
	}
}

// SyncWithPeer synchronizes the blockchain with a peer
func (m *Miner) SyncWithPeer(peer PeerInfo) error {
	client, err := rpc.Dial("tcp", peer.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}
	defer client.Close()

	args := &ChainArgs{StartIndex: 0}
	var reply ChainReply
	err = client.Call("RPCService.GetChain", args, &reply)
	if err != nil {
		return fmt.Errorf("failed to get chain: %v", err)
	}

	if reply.Length <= m.Blockchain.GetLength() {
		return nil // Our chain is longer or equal
	}

	// Deserialize blocks
	blocks := make([]*block.Block, len(reply.Blocks))
	for i, data := range reply.Blocks {
		b, err := block.DeserializeBlock(data)
		if err != nil {
			return fmt.Errorf("failed to deserialize block: %v", err)
		}
		blocks[i] = b
	}

	// Replace chain if valid and longer
	err = m.Blockchain.ReplaceChain(blocks)
	if err != nil {
		return fmt.Errorf("failed to replace chain: %v", err)
	}

	log.Printf("[%s] Synchronized chain with peer %s, new length: %d", m.ID, peer.ID, len(blocks))
	return nil
}

// SyncWithAllPeers synchronizes with all peers
func (m *Miner) SyncWithAllPeers() {
	if m.IsStopped() {
		return
	}
	for _, peer := range m.Peers {
		if m.IsStopped() {
			return
		}
		m.SyncWithPeer(peer)
		// Ignore sync errors silently
	}
}

// SetBlockCallback sets a callback function called when a new block is added
func (m *Miner) SetBlockCallback(callback func(*block.Block)) {
	m.blockCallback = callback
}

// SetDifficulty updates the mining difficulty
func (m *Miner) SetDifficulty(difficulty int) {
	m.Blockchain.SetDifficulty(difficulty)
	log.Printf("[%s] Difficulty set to %d", m.ID, difficulty)
}

// Client represents a blockchain client (wallet)
type Client struct {
	ID     string
	Miners []PeerInfo
}

// NewClient creates a new client
func NewClient(id string, miners []PeerInfo) *Client {
	return &Client{
		ID:     id,
		Miners: miners,
	}
}

// SubmitTransaction submits a transaction to a miner
// inputSpecs: UTXOs to spend
// outputs: transaction outputs
// privateKeys: map of public key -> private key for signing
func (c *Client) SubmitTransaction(
	inputSpecs []struct {
		TxID     string
		OutIndex int
	},
	outputs []transaction.TxOutput,
	privateKeys map[string]string,
) (string, error) {
	if len(c.Miners) == 0 {
		return "", errors.New("no miners available")
	}

	// Connect to first available miner
	for _, miner := range c.Miners {
		client, err := rpc.Dial("tcp", miner.Address)
		if err != nil {
			continue
		}
		defer client.Close()

		args := &TransactionArgs{
			InputSpecs:  inputSpecs,
			Outputs:     outputs,
			PrivateKeys: privateKeys,
		}
		var reply TransactionReply
		err = client.Call("RPCService.SubmitTransaction", args, &reply)
		if err != nil {
			continue
		}

		if reply.Success {
			return reply.TxID, nil
		}
		return "", errors.New(reply.Error)
	}

	return "", errors.New("failed to connect to any miner")
}

// GetMinerStatus gets the status of a miner
func (c *Client) GetMinerStatus(minerAddress string) (*StatusReply, error) {
	client, err := rpc.Dial("tcp", minerAddress)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var reply StatusReply
	err = client.Call("RPCService.GetStatus", &struct{}{}, &reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

// GetChain gets the blockchain from a miner
func (c *Client) GetChain(minerAddress string) ([]*block.Block, error) {
	client, err := rpc.Dial("tcp", minerAddress)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	args := &ChainArgs{StartIndex: 0}
	var reply ChainReply
	err = client.Call("RPCService.GetChain", args, &reply)
	if err != nil {
		return nil, err
	}

	blocks := make([]*block.Block, len(reply.Blocks))
	for i, data := range reply.Blocks {
		b, err := block.DeserializeBlock(data)
		if err != nil {
			return nil, err
		}
		blocks[i] = b
	}

	return blocks, nil
}

// SerializeBlocks serializes a slice of blocks
func SerializeBlocks(blocks []*block.Block) ([]byte, error) {
	return json.Marshal(blocks)
}

// DeserializeBlocks deserializes a slice of blocks
func DeserializeBlocks(data []byte) ([]*block.Block, error) {
	var blocks []*block.Block
	err := json.Unmarshal(data, &blocks)
	return blocks, err
}

// WaitForBlocks waits for the specified number of blocks to be mined
func WaitForBlocks(miner *Miner, targetBlocks int, timeout time.Duration) error {
	start := time.Now()
	for {
		if miner.Blockchain.GetLength() >= targetBlocks {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for %d blocks", targetBlocks)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
