// FakeMiner is a malicious miner for testing purposes
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
	peers := flag.String("peers", "", "Comma-separated list of peer addresses")
	difficulty := flag.Int("difficulty", 4, "Mining difficulty")
	maliciousType := flag.String("type", "invalid_pow", "Type of malicious behavior: invalid_pow, invalid_hash, invalid_prev_hash")

	flag.Parse()

	if *id == "" || *address == "" {
		fmt.Println("Usage: fakeminer -id <id> -address <address> -type <type> [-peers <peers>] [-difficulty <n>]")
		fmt.Println()
		fmt.Println("Malicious types:")
		fmt.Println("  invalid_pow       - Creates blocks that don't satisfy PoW")
		fmt.Println("  invalid_hash      - Creates blocks with incorrect hash")
		fmt.Println("  invalid_prev_hash - Creates blocks with wrong previous hash")
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

	// Create malicious miner
	miner := network.NewMaliciousMiner(*id, *address, *difficulty, peerList, *maliciousType)

	miner.SetBlockCallback(func(b *block.Block) {
		log.Printf("[MALICIOUS %s] Attempted to add block: #%d", *id, b.Index)
	})

	err := miner.Start()
	if err != nil {
		log.Fatalf("Failed to start malicious miner: %v", err)
	}

	// Sync with peers
	if len(peerList) > 0 {
		miner.SyncWithAllPeers()
	}

	// Start mining
	miner.StartMining()

	log.Printf("[MALICIOUS %s] Running with type: %s", *id, *maliciousType)

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	miner.Stop()
}
