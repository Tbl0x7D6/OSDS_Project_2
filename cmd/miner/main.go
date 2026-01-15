// Miner is the main entry point for the mining node
package main

import (
	"blockchain/pkg/block"
	"blockchain/pkg/config"
	"blockchain/pkg/network"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// shortID returns the first 6 characters of an ID for logging
func shortID(id string) string {
	if len(id) <= 6 {
		return id
	}
	return id[:6]
}

func main() {
	// Parse command line arguments
	id := flag.String("id", "", "Miner ID")
	address := flag.String("address", "0.0.0.0:8001", "Listen address (default: 0.0.0.0:8001)")
	peers := flag.String("peers", "", "Comma-separated list of peer addresses (e.g., localhost:8002,localhost:8003)")
	difficulty := flag.Int("difficulty", 4, "Mining difficulty (number of leading zeros)")
	autoMine := flag.Bool("mine", true, "Start mining automatically")
	useMerkle := flag.Bool("merkle", true, "Use Merkle Tree for block hash calculation (default: true)")
	dynamicDiff := flag.Bool("dynamic-difficulty", false, "Enable dynamic difficulty adjustment (default: false)")
	threads := flag.Int("threads", 1, "Number of parallel mining threads (default: 1, no parallelism)")

	flag.Parse()

	if *id == "" {
		fmt.Println("Usage: miner -id <id> -address <address> [-peers <peers>] [-difficulty <n>] [-mine] [-merkle] [-threads <n>]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -id        Miner ID (required)")
		fmt.Println("  -address   Listen address (default: 0.0.0.0:8001)")
		fmt.Println("  -peers     Comma-separated peer addresses")
		fmt.Println("  -difficulty Mining difficulty (default: 4)")
		fmt.Println("  -mine      Start mining automatically (default: true)")
		fmt.Println("  -merkle    Use Merkle Tree for block hash (default: true)")
		fmt.Println("  -dynamic-difficulty  Enable dynamic difficulty adjustment (default: false)")
		fmt.Println("  -threads   Number of parallel mining threads (default: 1)")
		os.Exit(1)
	}

	// Set global Merkle Tree configuration
	config.SetUseMerkleTree(*useMerkle)
	if *useMerkle {
		log.Printf("[%s] Using Merkle Tree for block hash calculation", shortID(*id))
	} else {
		log.Printf("[%s] Using direct transaction serialization for block hash calculation (legacy mode)", shortID(*id))
	}

	// Set dynamic difficulty configuration
	config.SetUseDynamicDifficulty(*dynamicDiff)
	if *dynamicDiff {
		log.Printf("[%s] Dynamic difficulty adjustment enabled (target: 1 block per 10 seconds)", shortID(*id))
	} else {
		log.Printf("[%s] Static difficulty mode (difficulty: %d)", shortID(*id), *difficulty)
	}

	// Set parallel mining threads configuration
	config.SetMiningThreads(*threads)
	if *threads > 1 {
		log.Printf("[%s] Parallel mining enabled with %d threads", shortID(*id), *threads)
	} else {
		log.Printf("[%s] Sequential mining (single thread)", shortID(*id))
	}

	// Parse peers
	var peerList []network.PeerInfo
	if *peers != "" {
		peerAddrs := strings.Split(*peers, ",")
		for i, addr := range peerAddrs {
			peerList = append(peerList, network.PeerInfo{
				ID:      fmt.Sprintf("peer%d", i),
				Address: strings.TrimSpace(addr),
			})
		}
	}

	// Create and start miner
	miner := network.NewMiner(*id, *address, *difficulty, peerList)

	// Set up logging callback
	miner.SetBlockCallback(func(b *block.Block) {
		log.Printf("[%s] New block added: #%d", shortID(*id), b.Index)
	})

	// Start the miner server
	err := miner.Start()
	if err != nil {
		log.Fatalf("Failed to start miner: %v", err)
	}

	// Sync with peers
	if len(peerList) > 0 {
		log.Printf("[%s] Syncing with %d peers...", shortID(*id), len(peerList))
		miner.SyncWithAllPeers()
	}

	// Start mining if enabled
	if *autoMine {
		miner.StartMining()
	}

	log.Printf("[%s] Miner is running. Chain length: %d", shortID(*id), miner.Blockchain.GetLength())

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[%s] Shutting down...", shortID(*id))
	miner.Stop()
}
