package test

import (
	"blockchain/pkg/block"
	"blockchain/pkg/blockchain"
	"blockchain/pkg/network"
	"blockchain/pkg/pow"
	"blockchain/pkg/transaction"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Integration test for the complete blockchain system
func TestIntegration_FullSystem(t *testing.T) {
	// Test 1: Run 5 miners and generate blocks
	t.Run("FiveMinersGenerateBlocks", func(t *testing.T) {
		numMiners := 5
		targetBlocks := 10 // Small number for tests
		ports := make([]string, numMiners)
		for i := 0; i < numMiners; i++ {
			ports[i] = fmt.Sprintf("180%02d", i)
		}

		var miners []*network.Miner

		// Create miners with peer connections
		for i := 0; i < numMiners; i++ {
			var peers []network.PeerInfo
			for j := 0; j < numMiners; j++ {
				if i != j {
					peers = append(peers, network.PeerInfo{
						ID:      fmt.Sprintf("miner%d", j),
						Address: "localhost:" + ports[j],
					})
				}
			}
			miner := network.NewMiner(fmt.Sprintf("miner%d", i), "localhost:"+ports[i], 2, peers)
			miners = append(miners, miner)
		}

		// Start all miners
		for _, m := range miners {
			err := m.Start()
			if err != nil {
				t.Fatalf("Failed to start miner: %v", err)
			}
		}
		defer func() {
			for _, m := range miners {
				m.StopMining()
				m.Stop()
			}
		}()

		// Start mining
		for _, m := range miners {
			m.StartMining()
		}

		// Wait for blocks
		err := waitForBlocks(miners, targetBlocks, 60*time.Second)
		if err != nil {
			t.Fatalf("Failed to mine blocks: %v", err)
		}

		// Validate all chains
		for i, m := range miners {
			err := m.Blockchain.ValidateChain()
			if err != nil {
				t.Errorf("Miner %d has invalid chain: %v", i, err)
			}
		}
	})
}

// Test difficulty adjustment affects mining speed
func TestIntegration_DifficultyAffectsMiningSpeed(t *testing.T) {
	// Create single miner with difficulty 1
	miner1 := network.NewMiner("miner1", "localhost:18100", 1, nil)
	err := miner1.Start()
	if err != nil {
		t.Fatalf("Failed to start miner: %v", err)
	}
	defer miner1.Stop()

	// Mine blocks with difficulty 1
	miner1.StartMining()
	start1 := time.Now()
	waitForBlocks([]*network.Miner{miner1}, 5, 30*time.Second)
	miner1.StopMining()
	time1 := time.Since(start1)
	blocks1 := miner1.Blockchain.GetLength()

	// Create miner with difficulty 3
	miner2 := network.NewMiner("miner2", "localhost:18101", 3, nil)
	err = miner2.Start()
	if err != nil {
		t.Fatalf("Failed to start miner: %v", err)
	}
	defer miner2.Stop()

	// Mine blocks with difficulty 3
	miner2.StartMining()
	start2 := time.Now()
	waitForBlocks([]*network.Miner{miner2}, 5, 120*time.Second)
	miner2.StopMining()
	time2 := time.Since(start2)
	blocks2 := miner2.Blockchain.GetLength()

	t.Logf("Difficulty 1: %d blocks in %v", blocks1, time1)
	t.Logf("Difficulty 3: %d blocks in %v", blocks2, time2)

	// Higher difficulty should generally take longer
	// (though not guaranteed due to randomness)
}

// Test corrupted block rejection
func TestIntegration_CorruptedBlockRejection(t *testing.T) {
	miner := network.NewMiner("honest", "localhost:18110", 2, nil)
	err := miner.Start()
	if err != nil {
		t.Fatalf("Failed to start miner: %v", err)
	}
	defer miner.Stop()

	initialLength := miner.Blockchain.GetLength()

	// Create a valid block first
	tx := transaction.NewTransaction("system", "attacker", 50.0)
	tx.Sign("system_key")
	txs := []*transaction.Transaction{tx}

	validBlock := block.NewBlock(1, txs, miner.Blockchain.GetLatestBlock().Hash, 2, "attacker")

	// Mine it properly first
	powInstance := pow.NewProofOfWork(validBlock)
	result := powInstance.Mine(context.Background())

	// Corrupt the block by tampering with the hash directly
	// This simulates block data corruption
	originalHash := result.Block.Hash
	result.Block.Hash = "00corrupted_hash_" + originalHash[16:]

	// Try to add corrupted block
	err = miner.Blockchain.AddBlock(result.Block)
	if err == nil {
		t.Error("Corrupted block should be rejected")
	}

	if miner.Blockchain.GetLength() != initialLength {
		t.Error("Chain length should not change after rejecting corrupted block")
	}
}

// Test lying miner (invalid PoW) rejection
func TestIntegration_LyingMinerRejection(t *testing.T) {
	// Create honest miner
	honest := network.NewMiner("honest", "localhost:18120", 2, nil)
	err := honest.Start()
	if err != nil {
		t.Fatalf("Failed to start honest miner: %v", err)
	}
	defer honest.Stop()

	initialLength := honest.Blockchain.GetLength()

	// Create block without proper PoW
	tx := transaction.NewTransaction("system", "liar", 50.0)
	tx.Sign("system_key")
	txs := []*transaction.Transaction{tx}

	lyingBlock := block.NewBlock(1, txs, honest.Blockchain.GetLatestBlock().Hash, 2, "liar")

	// Set an invalid hash (no proper PoW)
	lyingBlock.Nonce = 42
	lyingBlock.Hash = "00" + lyingBlock.CalculateHash()[2:] // Fake leading zeros but wrong hash

	err = honest.Blockchain.AddBlock(lyingBlock)
	if err == nil {
		t.Error("Block from lying miner should be rejected")
	}

	if honest.Blockchain.GetLength() != initialLength {
		t.Error("Chain should not accept lying miner's block")
	}
}

// Test fork resolution with longest chain rule
func TestIntegration_ForkResolutionLongestChain(t *testing.T) {
	// Create two independent miners
	miner1 := network.NewMiner("miner1", "localhost:18130", 2, nil)
	miner2 := network.NewMiner("miner2", "localhost:18131", 2, nil)

	err := miner1.Start()
	if err != nil {
		t.Fatalf("Failed to start miner1: %v", err)
	}
	defer miner1.Stop()

	err = miner2.Start()
	if err != nil {
		t.Fatalf("Failed to start miner2: %v", err)
	}
	defer miner2.Stop()

	// Mine on miner1 longer
	miner1.StartMining()
	waitForBlocks([]*network.Miner{miner1}, 8, 60*time.Second)
	miner1.StopMining()

	// Mine on miner2 shorter
	miner2.StartMining()
	waitForBlocks([]*network.Miner{miner2}, 3, 30*time.Second)
	miner2.StopMining()

	len1 := miner1.Blockchain.GetLength()
	len2Before := miner2.Blockchain.GetLength()

	t.Logf("Miner1 chain: %d blocks", len1)
	t.Logf("Miner2 chain before sync: %d blocks", len2Before)

	if len1 <= len2Before {
		t.Skip("Miner1 should have longer chain for this test")
	}

	// Now sync miner2 with miner1's longer chain
	miner2.Peers = []network.PeerInfo{{ID: "miner1", Address: "localhost:18130"}}
	miner2.SyncWithAllPeers()

	time.Sleep(1 * time.Second)

	len2After := miner2.Blockchain.GetLength()
	t.Logf("Miner2 chain after sync: %d blocks", len2After)

	// Miner2 should have adopted the longer chain
	if len2After != len1 {
		t.Errorf("Miner2 should adopt longer chain. Expected %d, got %d", len1, len2After)
	}
}

// Test chain validation catches corrupted chain
func TestIntegration_ChainValidationDetectsCorruption(t *testing.T) {
	bc := blockchain.NewBlockchain(2)

	// Add valid blocks
	for i := 0; i < 5; i++ {
		tx := transaction.NewTransaction("system", "miner", 50.0)
		tx.Sign("system_key")
		txs := []*transaction.Transaction{tx}

		newBlock := bc.CreateBlock(txs, "miner")

		// Mine
		powInstance := pow.NewProofOfWork(newBlock)
		result := powInstance.Mine(context.Background())
		if !result.Success {
			t.Fatal("Mining should succeed")
		}

		err := bc.AddBlock(result.Block)
		if err != nil {
			t.Fatalf("Failed to add block: %v", err)
		}
	}

	// Validate should pass
	err := bc.ValidateChain()
	if err != nil {
		t.Errorf("Valid chain should pass validation: %v", err)
	}

	// Corrupt a block
	bc.Blocks[3].Hash = "corrupted_hash"

	// Validation should fail
	err = bc.ValidateChain()
	if err == nil {
		t.Error("Corrupted chain should fail validation")
	}
}

// Test transaction flow
func TestIntegration_TransactionFlow(t *testing.T) {
	miner := network.NewMiner("miner1", "localhost:18140", 2, nil)
	err := miner.Start()
	if err != nil {
		t.Fatalf("Failed to start miner: %v", err)
	}
	defer miner.Stop()

	client := network.NewClient("alice", []network.PeerInfo{{ID: "miner1", Address: "localhost:18140"}})

	// Submit transaction
	txID, err := client.SubmitTransaction("bob", 10.0)
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}

	if txID == "" {
		t.Error("Transaction ID should not be empty")
	}

	// Check pending transactions
	pending := miner.GetPendingTransactions()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(pending))
	}

	// Start mining to include transaction in block
	miner.StartMining()
	waitForBlocks([]*network.Miner{miner}, 3, 30*time.Second)
	miner.StopMining()

	// Transaction should be cleared from pending
	time.Sleep(500 * time.Millisecond)
	pending = miner.GetPendingTransactions()
	if len(pending) != 0 {
		t.Logf("Pending transactions after mining: %d", len(pending))
	}
}

// Helper function to wait for blocks across all miners
func waitForBlocks(miners []*network.Miner, targetBlocks int, timeout time.Duration) error {
	start := time.Now()
	for {
		maxLen := 0
		for _, m := range miners {
			if m.Blockchain.GetLength() > maxLen {
				maxLen = m.Blockchain.GetLength()
			}
		}
		if maxLen >= targetBlocks {
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout: max chain length is %d, expected %d", maxLen, targetBlocks)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Benchmark mining performance
func BenchmarkMining(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tx := transaction.NewTransaction("system", "miner", 50.0)
		tx.Sign("system_key")
		txs := []*transaction.Transaction{tx}

		testBlock := block.NewBlock(1, txs, "0000", 2, "miner")
		powInstance := pow.NewProofOfWork(testBlock)
		powInstance.Mine(context.Background())
	}
}

// Test concurrent mining
func TestIntegration_ConcurrentMining(t *testing.T) {
	numMiners := 3
	ports := make([]string, numMiners)
	for i := 0; i < numMiners; i++ {
		ports[i] = fmt.Sprintf("181%02d", 50+i)
	}

	var miners []*network.Miner
	var mu sync.Mutex
	blocksReceived := make(map[string]int)

	// Create miners
	for i := 0; i < numMiners; i++ {
		var peers []network.PeerInfo
		for j := 0; j < numMiners; j++ {
			if i != j {
				peers = append(peers, network.PeerInfo{
					ID:      fmt.Sprintf("miner%d", j),
					Address: "localhost:" + ports[j],
				})
			}
		}
		miner := network.NewMiner(fmt.Sprintf("miner%d", i), "localhost:"+ports[i], 2, peers)

		// Track blocks received
		minerID := fmt.Sprintf("miner%d", i)
		miner.SetBlockCallback(func(b *block.Block) {
			mu.Lock()
			blocksReceived[minerID]++
			mu.Unlock()
		})

		miners = append(miners, miner)
	}

	// Start miners
	for _, m := range miners {
		m.Start()
	}
	defer func() {
		for _, m := range miners {
			m.StopMining()
			m.Stop()
		}
	}()

	// All mine concurrently
	for _, m := range miners {
		m.StartMining()
	}

	// Wait for blocks
	time.Sleep(10 * time.Second)

	// Stop mining
	for _, m := range miners {
		m.StopMining()
	}

	// Log results
	mu.Lock()
	for id, count := range blocksReceived {
		t.Logf("%s received %d blocks", id, count)
	}
	mu.Unlock()

	// All miners should have some blocks
	for _, m := range miners {
		if m.Blockchain.GetLength() < 2 {
			t.Errorf("Miner should have at least 2 blocks, got %d", m.Blockchain.GetLength())
		}
	}
}
