// Miner is the main entry point for the mining node
package main

import (
	"blockchain/pkg/block"
	"blockchain/pkg/network"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// Parse command line arguments
	id := flag.String("id", "", "Miner ID")
	address := flag.String("address", "", "Listen address (e.g., localhost:8001)")
	peers := flag.String("peers", "", "Comma-separated list of peer addresses (e.g., localhost:8002,localhost:8003)")
	difficulty := flag.Int("difficulty", 4, "Mining difficulty (number of leading zeros)")
	autoMine := flag.Bool("mine", true, "Start mining automatically")

	flag.Parse()

	if *id == "" || *address == "" {
		fmt.Println("Usage: miner -id <id> -address <address> [-peers <peers>] [-difficulty <n>] [-mine]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -id        Miner ID (required)")
		fmt.Println("  -address   Listen address (required)")
		fmt.Println("  -peers     Comma-separated peer addresses")
		fmt.Println("  -difficulty Mining difficulty (default: 4)")
		fmt.Println("  -mine      Start mining automatically (default: true)")
		os.Exit(1)
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
		log.Printf("[%s] New block added: #%d", *id, b.Index)
	})

	// Start the miner server
	err := miner.Start()
	if err != nil {
		log.Fatalf("Failed to start miner: %v", err)
	}

	// Sync with peers
	if len(peerList) > 0 {
		log.Printf("[%s] Syncing with %d peers...", *id, len(peerList))
		miner.SyncWithAllPeers()
	}

	// Start mining if enabled
	if *autoMine {
		miner.StartMining()
	}

	log.Printf("[%s] Miner is running. Chain length: %d", *id, miner.Blockchain.GetLength())

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[%s] Shutting down...", *id)
	miner.Stop()
}
