// Client is the main entry point for the blockchain client (wallet)
package main

import (
	"blockchain/pkg/block"
	"blockchain/pkg/network"
	"blockchain/pkg/transaction"
	"encoding/json"
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"time"
)

// WalletOutput represents a wallet in JSON format
type WalletOutput struct {
	Address    string `json:"address"`     // Public key (hex)
	PrivateKey string `json:"private_key"` // Private key (hex)
	CreatedAt  string `json:"created_at"`  // Timestamp
}

// BlockchainStatusOutput represents blockchain status in JSON format
type BlockchainStatusOutput struct {
	ChainLength       int                  `json:"chain_length"`
	Difficulty        int                  `json:"difficulty"`
	LatestBlockHash   string               `json:"latest_block_hash"`
	LatestBlockIndex  int64                `json:"latest_block_index"`
	LatestBlockMiner  string               `json:"latest_block_miner"`
	LatestBlockTime   int64                `json:"latest_block_time"`
	TotalTransactions int                  `json:"total_transactions"`
	MinerStatus       *network.StatusReply `json:"miner_status,omitempty"`
	Blocks            []BlockOutput        `json:"blocks,omitempty"`
}

// BlockOutput represents a block in JSON format
type BlockOutput struct {
	Index        int64               `json:"index"`
	Hash         string              `json:"hash"`
	PrevHash     string              `json:"prev_hash"`
	Timestamp    int64               `json:"timestamp"`
	Nonce        int64               `json:"nonce"`
	Difficulty   int                 `json:"difficulty"`
	MinerID      string              `json:"miner_id"`
	Transactions []TransactionOutput `json:"transactions"`
}

// TransactionOutput represents a transaction in JSON format
type TransactionOutput struct {
	ID         string                 `json:"id"`
	Inputs     []transaction.TxInput  `json:"inputs"`
	Outputs    []transaction.TxOutput `json:"outputs"`
	IsCoinbase bool                   `json:"is_coinbase"`
}

// WalletStatusOutput represents wallet status in JSON format
type WalletStatusOutput struct {
	Address    string       `json:"address"`
	Balance    int64        `json:"balance"`
	BalanceBTC float64      `json:"balance_btc"`
	UTXOs      []UTXOOutput `json:"utxos"`
	UTXOCount  int          `json:"utxo_count"`
}

// UTXOOutput represents a UTXO in JSON format
type UTXOOutput struct {
	TxID         string  `json:"txid"`
	OutIndex     int     `json:"out_index"`
	Value        int64   `json:"value"`
	ValueBTC     float64 `json:"value_btc"`
	ScriptPubKey string  `json:"scriptpubkey"`
}

// ErrorOutput represents an error in JSON format
type ErrorOutput struct {
	Error string `json:"error"`
}

// TransferOutput represents a transfer result in JSON format
type TransferOutput struct {
	Success bool   `json:"success"`
	TxID    string `json:"txid"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	// Define commands
	walletCmd := flag.NewFlagSet("wallet", flag.ExitOnError)
	blockchainCmd := flag.NewFlagSet("blockchain", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	transferCmd := flag.NewFlagSet("transfer", flag.ExitOnError)

	// Wallet command flags (no flags needed for generation)

	// Blockchain command flags
	blockchainMiner := blockchainCmd.String("miner", "localhost:8001", "Miner address")
	blockchainDetail := blockchainCmd.Bool("detail", false, "Include detailed block information")

	// Balance command flags
	balanceMiner := balanceCmd.String("miner", "localhost:8001", "Miner address")
	balanceAddress := balanceCmd.String("address", "", "Wallet address (public key)")

	// Transfer command flags
	transferMiner := transferCmd.String("miner", "localhost:8001", "Miner address")
	transferFrom := transferCmd.String("from", "", "Sender's public key (address)")
	transferPrivateKey := transferCmd.String("privkey", "", "Sender's private key")
	transferInputs := transferCmd.String("inputs", "", "Comma-separated list of UTXOs to spend (format: txid:outindex,txid:outindex)")
	transferOutputs := transferCmd.String("outputs", "", "Comma-separated list of outputs (format: address:amount,address:amount)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "wallet":
		walletCmd.Parse(os.Args[2:])
		generateWallet()

	case "blockchain":
		blockchainCmd.Parse(os.Args[2:])
		getBlockchainStatus(*blockchainMiner, *blockchainDetail)

	case "balance":
		balanceCmd.Parse(os.Args[2:])
		if *balanceAddress == "" {
			outputError("address is required")
			os.Exit(1)
		}
		getWalletStatus(*balanceMiner, *balanceAddress)

	case "transfer":
		transferCmd.Parse(os.Args[2:])
		if *transferFrom == "" || *transferPrivateKey == "" || *transferInputs == "" || *transferOutputs == "" {
			outputError("from, privkey, inputs, and outputs are required")
			os.Exit(1)
		}
		sendTransfer(*transferMiner, *transferFrom, *transferPrivateKey, *transferInputs, *transferOutputs)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `Blockchain Client - JSON CLI Tool

Usage:
  client wallet                                    Generate a new wallet (keypair)
  client blockchain [-miner <address>] [-detail]  Get blockchain status and parameters
  client balance -address <address> [-miner <address>]  Get wallet balance and UTXOs
  client transfer -from <address> -privkey <key> -inputs <utxos> -outputs <outputs> [-miner <address>]

Commands:
  wallet       Generate a new wallet keypair (outputs JSON)
  blockchain   Get current blockchain status (outputs JSON)
  balance      Get wallet balance and all UTXOs (outputs JSON)
  transfer     Send a transaction with multiple outputs (outputs JSON)

Options:
  -miner <address>    Miner node address (default: localhost:8001)
  -address <address>  Wallet address (public key in hex)
  -detail             Include detailed block information in blockchain command
  -from <address>     Sender's public key (address)
  -privkey <key>      Sender's private key (hex)
  -inputs <utxos>     Comma-separated list of UTXOs to spend (format: txid:outindex,txid:outindex)
  -outputs <outputs>  Comma-separated list of outputs (format: address:amount,address:amount)
                      Amount in satoshi. Excess will be miner fee.

All output is in JSON format for frontend integration.
`
	fmt.Println(usage)
}

func outputJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		outputError(fmt.Sprintf("failed to marshal JSON: %v", err))
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func outputError(message string) {
	outputJSON(ErrorOutput{Error: message})
}

// generateWallet creates a new wallet (keypair) and outputs it as JSON
func generateWallet() {
	kp, err := transaction.GenerateKeyPair()
	if err != nil {
		outputError(fmt.Sprintf("failed to generate wallet: %v", err))
		os.Exit(1)
	}

	wallet := WalletOutput{
		Address:    kp.GetPublicKeyHex(),
		PrivateKey: kp.GetPrivateKeyHex(),
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	outputJSON(wallet)
}

// getBlockchainStatus retrieves and outputs blockchain status as JSON
func getBlockchainStatus(minerAddr string, includeDetail bool) {
	client, err := rpc.Dial("tcp", minerAddr)
	if err != nil {
		outputError(fmt.Sprintf("failed to connect to miner: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Get miner status
	var statusReply network.StatusReply
	err = client.Call("RPCService.GetStatus", &struct{}{}, &statusReply)
	if err != nil {
		outputError(fmt.Sprintf("failed to get miner status: %v", err))
		os.Exit(1)
	}

	// Get blockchain
	chainArgs := &network.ChainArgs{StartIndex: 0}
	var chainReply network.ChainReply
	err = client.Call("RPCService.GetChain", chainArgs, &chainReply)
	if err != nil {
		outputError(fmt.Sprintf("failed to get blockchain: %v", err))
		os.Exit(1)
	}

	// Deserialize blocks
	blocks := make([]*block.Block, len(chainReply.Blocks))
	for i, data := range chainReply.Blocks {
		b, err := block.DeserializeBlock(data)
		if err != nil {
			outputError(fmt.Sprintf("failed to deserialize block: %v", err))
			os.Exit(1)
		}
		blocks[i] = b
	}

	// Build output
	output := BlockchainStatusOutput{
		ChainLength: len(blocks),
		MinerStatus: &statusReply,
	}

	// Calculate total transactions
	totalTxs := 0
	for _, b := range blocks {
		totalTxs += len(b.Transactions)
	}
	output.TotalTransactions = totalTxs

	if len(blocks) > 0 {
		latest := blocks[len(blocks)-1]
		output.Difficulty = latest.Difficulty
		output.LatestBlockHash = latest.Hash
		output.LatestBlockIndex = latest.Index
		output.LatestBlockMiner = latest.MinerID
		output.LatestBlockTime = latest.Timestamp
	}

	// Include detailed block information if requested
	if includeDetail {
		output.Blocks = make([]BlockOutput, len(blocks))
		for i, b := range blocks {
			output.Blocks[i] = convertBlockToOutput(b)
		}
	}

	outputJSON(output)
}

// getWalletStatus retrieves and outputs wallet balance and UTXOs as JSON
func getWalletStatus(minerAddr, address string) {
	client, err := rpc.Dial("tcp", minerAddr)
	if err != nil {
		outputError(fmt.Sprintf("failed to connect to miner: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Get blockchain to access UTXO set
	chainArgs := &network.ChainArgs{StartIndex: 0}
	var chainReply network.ChainReply
	err = client.Call("RPCService.GetChain", chainArgs, &chainReply)
	if err != nil {
		outputError(fmt.Sprintf("failed to get blockchain: %v", err))
		os.Exit(1)
	}

	// Deserialize blocks and rebuild UTXO set
	blocks := make([]*block.Block, len(chainReply.Blocks))
	for i, data := range chainReply.Blocks {
		b, err := block.DeserializeBlock(data)
		if err != nil {
			outputError(fmt.Sprintf("failed to deserialize block: %v", err))
			os.Exit(1)
		}
		blocks[i] = b
	}

	// Build UTXO set from blocks
	utxoSet := transaction.NewUTXOSet()
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			utxoSet.ProcessTransaction(tx)
		}
	}

	// Get balance and UTXOs for the address
	balance := utxoSet.GetBalance(address)
	utxos := utxoSet.FindUTXOsForAddress(address)

	// Convert UTXOs to output format
	utxoOutputs := make([]UTXOOutput, len(utxos))
	for i, utxo := range utxos {
		utxoOutputs[i] = UTXOOutput{
			TxID:         utxo.TxID,
			OutIndex:     utxo.OutIndex,
			Value:        utxo.Value,
			ValueBTC:     float64(utxo.Value) / transaction.SatoshiPerBTC,
			ScriptPubKey: utxo.ScriptPubKey,
		}
	}

	output := WalletStatusOutput{
		Address:    address,
		Balance:    balance,
		BalanceBTC: float64(balance) / transaction.SatoshiPerBTC,
		UTXOs:      utxoOutputs,
		UTXOCount:  len(utxos),
	}

	outputJSON(output)
}

// convertBlockToOutput converts a block to output format
func convertBlockToOutput(b *block.Block) BlockOutput {
	txs := make([]TransactionOutput, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = TransactionOutput{
			ID:         tx.ID,
			Inputs:     tx.Inputs,
			Outputs:    tx.Outputs,
			IsCoinbase: tx.IsCoinbase(),
		}
	}

	return BlockOutput{
		Index:        b.Index,
		Hash:         b.Hash,
		PrevHash:     b.PrevHash,
		Timestamp:    b.Timestamp,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
		MinerID:      b.MinerID,
		Transactions: txs,
	}
}

// sendTransfer creates and sends a transfer transaction with multiple outputs
func sendTransfer(minerAddr, from, privateKey, inputs, outputs string) {
	// Parse UTXO inputs
	inputSpecs, err := parseUTXOInputs(inputs)
	if err != nil {
		outputError(fmt.Sprintf("failed to parse inputs: %v", err))
		os.Exit(1)
	}

	// Parse outputs
	outputSpecs, err := parseOutputs(outputs)
	if err != nil {
		outputError(fmt.Sprintf("failed to parse outputs: %v", err))
		os.Exit(1)
	}

	// Connect to miner
	client, err := rpc.Dial("tcp", minerAddr)
	if err != nil {
		outputError(fmt.Sprintf("failed to connect to miner: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Get blockchain to validate UTXO ownership
	chainArgs := &network.ChainArgs{StartIndex: 0}
	var chainReply network.ChainReply
	err = client.Call("RPCService.GetChain", chainArgs, &chainReply)
	if err != nil {
		outputError(fmt.Sprintf("failed to get blockchain: %v", err))
		os.Exit(1)
	}

	// Deserialize blocks and rebuild UTXO set
	blocks := make([]*block.Block, len(chainReply.Blocks))
	for i, data := range chainReply.Blocks {
		b, err := block.DeserializeBlock(data)
		if err != nil {
			outputError(fmt.Sprintf("failed to deserialize block: %v", err))
			os.Exit(1)
		}
		blocks[i] = b
	}

	// Build UTXO set from blocks
	utxoSet := transaction.NewUTXOSet()
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			utxoSet.ProcessTransaction(tx)
		}
	}

	// Calculate total input value and validate ownership
	var totalInput int64
	for _, spec := range inputSpecs {
		utxo := utxoSet.FindUTXO(spec.TxID, spec.OutIndex)
		if utxo == nil {
			outputError(fmt.Sprintf("UTXO not found: %s:%d", spec.TxID, spec.OutIndex))
			os.Exit(1)
		}
		if utxo.ScriptPubKey != from {
			outputError(fmt.Sprintf("UTXO %s:%d does not belong to address %s", spec.TxID, spec.OutIndex, from))
			os.Exit(1)
		}
		totalInput += utxo.Value
	}

	// Calculate total output value
	var totalOutput int64
	for _, out := range outputSpecs {
		totalOutput += out.Value
	}

	// Calculate miner fee (can be 0 or positive, but not negative)
	minerFee := totalInput - totalOutput
	if minerFee < 0 {
		outputError(fmt.Sprintf("insufficient funds: input=%d satoshi, output=%d satoshi, deficit=%d satoshi", totalInput, totalOutput, -minerFee))
		os.Exit(1)
	}

	// Create transaction args for RPC
	txArgs := &network.TransactionArgs{
		InputSpecs:  inputSpecs,
		Outputs:     outputSpecs,
		PrivateKeys: map[string]string{from: privateKey},
	}

	// Submit transaction via RPC
	var txReply network.TransactionReply
	err = client.Call("RPCService.SubmitTransaction", txArgs, &txReply)
	if err != nil {
		outputError(fmt.Sprintf("RPC call failed: %v", err))
		os.Exit(1)
	}

	// Output result
	output := TransferOutput{
		Success: txReply.Success,
		TxID:    txReply.TxID,
	}

	if txReply.Success {
		output.Message = fmt.Sprintf("Transfer successful! %d outputs, total: %d satoshi (%.8f BTC)", len(outputSpecs), totalOutput, float64(totalOutput)/transaction.SatoshiPerBTC)
		if minerFee > 0 {
			output.Message += fmt.Sprintf(". Miner fee: %d satoshi (%.8f BTC)", minerFee, float64(minerFee)/transaction.SatoshiPerBTC)
		}
	} else {
		output.Error = txReply.Error
	}

	outputJSON(output)
}

// parseUTXOInputs parses comma-separated UTXO inputs (format: txid:outindex,txid:outindex)
func parseUTXOInputs(inputs string) ([]struct {
	TxID     string
	OutIndex int
}, error) {
	var specs []struct {
		TxID     string
		OutIndex int
	}

	parts := splitAndTrim(inputs, ",")
	for _, part := range parts {
		pair := splitAndTrim(part, ":")
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid UTXO format: %s (expected txid:outindex)", part)
		}

		var outIndex int
		_, err := fmt.Sscanf(pair[1], "%d", &outIndex)
		if err != nil {
			return nil, fmt.Errorf("invalid output index: %s", pair[1])
		}

		specs = append(specs, struct {
			TxID     string
			OutIndex int
		}{
			TxID:     pair[0],
			OutIndex: outIndex,
		})
	}

	return specs, nil
}

// parseOutputs parses comma-separated outputs (format: address:amount,address:amount)
func parseOutputs(outputs string) ([]transaction.TxOutput, error) {
	var txOutputs []transaction.TxOutput

	parts := splitAndTrim(outputs, ",")
	for _, part := range parts {
		pair := splitAndTrim(part, ":")
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid output format: %s (expected address:amount)", part)
		}

		var amount int64
		_, err := fmt.Sscanf(pair[1], "%d", &amount)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %s", pair[1])
		}

		if amount <= 0 {
			return nil, fmt.Errorf("amount must be positive: %d", amount)
		}

		txOutputs = append(txOutputs, transaction.TxOutput{
			Value:        amount,
			ScriptPubKey: pair[0],
		})
	}

	if len(txOutputs) == 0 {
		return nil, fmt.Errorf("no valid outputs")
	}

	return txOutputs, nil
}

// splitAndTrim splits a string by delimiter and trims whitespace
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by separator
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	var parts []string
	var current string
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, current)
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	parts = append(parts, current)
	return parts
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
