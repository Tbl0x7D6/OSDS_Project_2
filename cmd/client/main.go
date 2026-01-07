// Client is the main entry point for the blockchain client (wallet)
package main

import (
	"blockchain/pkg/network"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define commands
	submitCmd := flag.NewFlagSet("submit", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	chainCmd := flag.NewFlagSet("chain", flag.ExitOnError)

	// Submit transaction flags
	submitMiner := submitCmd.String("miner", "localhost:8001", "Miner address")
	submitFrom := submitCmd.String("from", "", "Sender address")
	submitTo := submitCmd.String("to", "", "Receiver address")
	submitAmount := submitCmd.Int64("amount", 0, "Amount to send (in satoshi)")

	// Status flags
	statusMiner := statusCmd.String("miner", "localhost:8001", "Miner address")

	// Chain flags
	chainMiner := chainCmd.String("miner", "localhost:8001", "Miner address")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "submit":
		submitCmd.Parse(os.Args[2:])
		if *submitFrom == "" || *submitTo == "" || *submitAmount <= 0 {
			fmt.Println("Error: from, to, and amount are required")
			submitCmd.PrintDefaults()
			os.Exit(1)
		}
		submitTransaction(*submitMiner, *submitFrom, *submitTo, *submitAmount)

	case "status":
		statusCmd.Parse(os.Args[2:])
		getMinerStatus(*statusMiner)

	case "chain":
		chainCmd.Parse(os.Args[2:])
		getChain(*chainMiner)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Blockchain Client")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  client submit -from <address> -to <address> -amount <amount> [-miner <address>]")
	fmt.Println("  client status [-miner <address>]")
	fmt.Println("  client chain [-miner <address>]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  submit   Submit a transaction to the blockchain")
	fmt.Println("  status   Get the status of a miner")
	fmt.Println("  chain    Get the current blockchain")
}

func submitTransaction(minerAddr, from, to string, amount int64) {
	client := network.NewClient(from, []network.PeerInfo{
		{ID: "miner", Address: minerAddr},
	})

	txID, err := client.SubmitTransaction(to, amount)
	if err != nil {
		fmt.Printf("Error submitting transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Transaction submitted successfully!\n")
	fmt.Printf("Transaction ID: %s\n", txID)
}

func getMinerStatus(minerAddr string) {
	client := network.NewClient("client", []network.PeerInfo{
		{ID: "miner", Address: minerAddr},
	})

	status, err := client.GetMinerStatus(minerAddr)
	if err != nil {
		fmt.Printf("Error getting miner status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Miner Status:\n")
	fmt.Printf("  ID:           %s\n", status.ID)
	fmt.Printf("  Chain Length: %d\n", status.ChainLength)
	fmt.Printf("  Pending TXs:  %d\n", status.PendingTxs)
	fmt.Printf("  Peers:        %d\n", status.Peers)
	fmt.Printf("  Mining:       %v\n", status.Mining)
}

func getChain(minerAddr string) {
	client := network.NewClient("client", []network.PeerInfo{
		{ID: "miner", Address: minerAddr},
	})

	blocks, err := client.GetChain(minerAddr)
	if err != nil {
		fmt.Printf("Error getting chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Blockchain (length: %d):\n", len(blocks))
	fmt.Println("========================================")
	for _, b := range blocks {
		fmt.Printf("Block #%d\n", b.Index)
		fmt.Printf("  Hash:      %s\n", b.Hash[:16]+"...")
		fmt.Printf("  PrevHash:  %s\n", b.PrevHash[:16]+"...")
		fmt.Printf("  Nonce:     %d\n", b.Nonce)
		fmt.Printf("  Miner:     %s\n", b.MinerID)
		fmt.Printf("  TXs:       %d\n", len(b.Transactions))

		// Show transaction details
		for i, tx := range b.Transactions {
			if tx.IsCoinbase() {
				fmt.Printf("    TX[%d]: Coinbase -> %s (%.2f BTC)\n",
					i, tx.Outputs[0].ScriptPubKey,
					float64(tx.Outputs[0].Value)/100000000.0)
			} else {
				inputSum := int64(0)
				outputSum := int64(0)
				for _, out := range tx.Outputs {
					outputSum += out.Value
				}
				fmt.Printf("    TX[%d]: %d inputs -> %d outputs (%.8f BTC)\n",
					i, len(tx.Inputs), len(tx.Outputs), float64(outputSum)/100000000.0)
				_ = inputSum
			}
		}
		fmt.Println("----------------------------------------")
	}
}
