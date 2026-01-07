// Package transaction defines the UTXO-based transaction structure for the blockchain
package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Satoshi constants
const (
	SatoshiPerBTC = 100_000_000 // 1 BTC = 100,000,000 satoshi
)

// TxInput represents a transaction input (reference to a previous output)
type TxInput struct {
	TxID      string `json:"txid"`      // Previous transaction ID
	OutIndex  int    `json:"out_index"` // Index of the output in the previous transaction
	ScriptSig string `json:"scriptsig"` // Simplified: signature proving ownership
}

// TxOutput represents a transaction output
type TxOutput struct {
	Value        int64  `json:"value"`        // Amount in satoshi
	ScriptPubKey string `json:"scriptpubkey"` // Simplified: public key (wallet address)
}

// Transaction represents a UTXO-based transaction
type Transaction struct {
	ID      string     `json:"id"`
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
}

// IsCoinbase checks if this is a coinbase transaction (mining reward)
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && tx.Inputs[0].TxID == "" && tx.Inputs[0].OutIndex == -1
}

// NewCoinbaseTransaction creates a new coinbase transaction (mining reward)
func NewCoinbaseTransaction(to string, reward int64, blockHeight int64) *Transaction {
	// Coinbase input has no previous transaction
	input := TxInput{
		TxID:      "",
		OutIndex:  -1,
		ScriptSig: fmt.Sprintf("coinbase:%d", blockHeight), // Block height in scriptsig
	}

	output := TxOutput{
		Value:        reward,
		ScriptPubKey: to,
	}

	tx := &Transaction{
		Inputs:  []TxInput{input},
		Outputs: []TxOutput{output},
	}
	tx.ID = tx.CalculateHash()
	return tx
}

// NewTransactionSatoshi creates a new transaction with amount in satoshi
func NewTransactionSatoshi(from, to string, amount int64) *Transaction {
	// This is a simplified transaction creator
	// In a real implementation, you'd need to:
	// 1. Find UTXOs belonging to 'from'
	// 2. Select enough UTXOs to cover the amount
	// 3. Create change output if needed

	input := TxInput{
		TxID:      "placeholder", // Will be set when actually spending UTXOs
		OutIndex:  0,
		ScriptSig: "", // Will be signed later
	}

	output := TxOutput{
		Value:        amount,
		ScriptPubKey: to,
	}

	tx := &Transaction{
		Inputs:  []TxInput{input},
		Outputs: []TxOutput{output},
	}
	tx.ID = tx.CalculateHash()
	return tx
}

// NewUTXOTransaction creates a transaction spending specific UTXOs
func NewUTXOTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}
	tx.ID = tx.CalculateHash()
	return tx
}

// CalculateHash computes the hash of the transaction
func (tx *Transaction) CalculateHash() string {
	var buf bytes.Buffer

	for _, in := range tx.Inputs {
		buf.WriteString(in.TxID)
		buf.WriteString(fmt.Sprintf("%d", in.OutIndex))
		buf.WriteString(in.ScriptSig) // Include scriptsig for coinbase uniqueness
	}

	for _, out := range tx.Outputs {
		buf.WriteString(fmt.Sprintf("%d", out.Value))
		buf.WriteString(out.ScriptPubKey)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:])
}

// Sign signs all inputs of the transaction with the given private key
// Simplified: In real Bitcoin, each input would be signed separately
func (tx *Transaction) Sign(privateKey string) {
	// Create signature for the transaction
	dataToSign := tx.getDataToSign()
	signatureData := dataToSign + privateKey
	hash := sha256.Sum256([]byte(signatureData))
	signature := hex.EncodeToString(hash[:])

	// Apply signature to all inputs
	for i := range tx.Inputs {
		tx.Inputs[i].ScriptSig = signature
	}

	// Recalculate ID after signing (scriptsig not included in hash, but for consistency)
	tx.ID = tx.CalculateHash()
}

// SignInput signs a specific input
func (tx *Transaction) SignInput(index int, privateKey string) {
	if index < 0 || index >= len(tx.Inputs) {
		return
	}

	dataToSign := tx.getDataToSign()
	signatureData := dataToSign + privateKey
	hash := sha256.Sum256([]byte(signatureData))
	tx.Inputs[index].ScriptSig = hex.EncodeToString(hash[:])
}

// getDataToSign returns the data that should be signed
func (tx *Transaction) getDataToSign() string {
	var buf bytes.Buffer

	for _, in := range tx.Inputs {
		buf.WriteString(in.TxID)
		buf.WriteString(fmt.Sprintf("%d", in.OutIndex))
	}

	for _, out := range tx.Outputs {
		buf.WriteString(fmt.Sprintf("%d", out.Value))
		buf.WriteString(out.ScriptPubKey)
	}

	return buf.String()
}

// Verify verifies the transaction's basic validity
// Note: Full UTXO verification requires access to the UTXO set
func (tx *Transaction) Verify() bool {
	// Coinbase transactions have special rules
	if tx.IsCoinbase() {
		return tx.verifyCoinbase()
	}

	// Must have at least one input and one output
	if len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return false
	}

	// All inputs must be signed
	for _, in := range tx.Inputs {
		if in.ScriptSig == "" {
			return false
		}
		if in.TxID == "" {
			return false
		}
	}

	// All outputs must have positive value
	for _, out := range tx.Outputs {
		if out.Value <= 0 {
			return false
		}
		if out.ScriptPubKey == "" {
			return false
		}
	}

	return true
}

// verifyCoinbase verifies a coinbase transaction
func (tx *Transaction) verifyCoinbase() bool {
	if len(tx.Inputs) != 1 {
		return false
	}
	if len(tx.Outputs) == 0 {
		return false
	}
	// Coinbase output must have positive value
	for _, out := range tx.Outputs {
		if out.Value < 0 {
			return false
		}
		if out.ScriptPubKey == "" {
			return false
		}
	}
	return true
}

// TotalOutputValue calculates total output value
func (tx *Transaction) TotalOutputValue() int64 {
	var total int64
	for _, out := range tx.Outputs {
		total += out.Value
	}
	return total
}

// GetFee calculates the transaction fee (input total - output total)
// Returns the fee if known, or 0 for coinbase transactions
func (tx *Transaction) GetFee(utxoSet *UTXOSet) int64 {
	if tx.IsCoinbase() {
		return 0
	}

	if utxoSet == nil {
		return 0
	}

	var inputTotal int64
	for _, in := range tx.Inputs {
		utxo := utxoSet.FindUTXO(in.TxID, in.OutIndex)
		if utxo != nil {
			inputTotal += utxo.Value
		}
	}

	outputTotal := tx.TotalOutputValue()

	if inputTotal > outputTotal {
		return inputTotal - outputTotal
	}
	return 0
}

// Serialize converts the transaction to JSON bytes
func (tx *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(tx)
}

// DeserializeTransaction converts JSON bytes to a Transaction
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	err := json.Unmarshal(data, &tx)
	return &tx, err
}

// String returns a string representation of the transaction
func (tx *Transaction) String() string {
	if tx.IsCoinbase() {
		return fmt.Sprintf("TX{ID: %s..., Coinbase, Outputs: %d}",
			tx.ID[:8], len(tx.Outputs))
	}
	return fmt.Sprintf("TX{ID: %s..., Inputs: %d, Outputs: %d}",
		tx.ID[:8], len(tx.Inputs), len(tx.Outputs))
}

// UTXO represents an unspent transaction output
type UTXO struct {
	TxID         string `json:"txid"`
	OutIndex     int    `json:"out_index"`
	Value        int64  `json:"value"`
	ScriptPubKey string `json:"scriptpubkey"`
}

// UTXOSet manages the set of unspent transaction outputs
type UTXOSet struct {
	UTXOs map[string]map[int]*UTXO // txid -> outIndex -> UTXO
}

// NewUTXOSet creates a new UTXO set
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{
		UTXOs: make(map[string]map[int]*UTXO),
	}
}

// AddUTXO adds a UTXO to the set
func (us *UTXOSet) AddUTXO(txID string, outIndex int, value int64, scriptPubKey string) {
	if us.UTXOs[txID] == nil {
		us.UTXOs[txID] = make(map[int]*UTXO)
	}
	us.UTXOs[txID][outIndex] = &UTXO{
		TxID:         txID,
		OutIndex:     outIndex,
		Value:        value,
		ScriptPubKey: scriptPubKey,
	}
}

// RemoveUTXO removes a UTXO from the set (when it's spent)
func (us *UTXOSet) RemoveUTXO(txID string, outIndex int) {
	if us.UTXOs[txID] != nil {
		delete(us.UTXOs[txID], outIndex)
		if len(us.UTXOs[txID]) == 0 {
			delete(us.UTXOs, txID)
		}
	}
}

// FindUTXO finds a specific UTXO
func (us *UTXOSet) FindUTXO(txID string, outIndex int) *UTXO {
	if us.UTXOs[txID] != nil {
		return us.UTXOs[txID][outIndex]
	}
	return nil
}

// FindUTXOsForAddress finds all UTXOs belonging to an address
func (us *UTXOSet) FindUTXOsForAddress(address string) []*UTXO {
	var utxos []*UTXO
	for _, outputs := range us.UTXOs {
		for _, utxo := range outputs {
			if utxo.ScriptPubKey == address {
				utxos = append(utxos, utxo)
			}
		}
	}
	return utxos
}

// GetBalance returns the total balance for an address
func (us *UTXOSet) GetBalance(address string) int64 {
	var balance int64
	utxos := us.FindUTXOsForAddress(address)
	for _, utxo := range utxos {
		balance += utxo.Value
	}
	return balance
}

// HasUTXO checks if a specific UTXO exists
func (us *UTXOSet) HasUTXO(txID string, outIndex int) bool {
	return us.FindUTXO(txID, outIndex) != nil
}

// ProcessTransaction updates the UTXO set based on a transaction
func (us *UTXOSet) ProcessTransaction(tx *Transaction) {
	// Remove spent UTXOs (inputs)
	if !tx.IsCoinbase() {
		for _, in := range tx.Inputs {
			us.RemoveUTXO(in.TxID, in.OutIndex)
		}
	}

	// Add new UTXOs (outputs)
	for i, out := range tx.Outputs {
		us.AddUTXO(tx.ID, i, out.Value, out.ScriptPubKey)
	}
}

// ValidateTransaction validates a transaction against the UTXO set
func (us *UTXOSet) ValidateTransaction(tx *Transaction) error {
	// Coinbase transactions don't spend UTXOs
	if tx.IsCoinbase() {
		return nil
	}

	var inputTotal int64

	for _, in := range tx.Inputs {
		// Check if UTXO exists
		utxo := us.FindUTXO(in.TxID, in.OutIndex)
		if utxo == nil {
			return fmt.Errorf("UTXO not found: %s:%d", in.TxID, in.OutIndex)
		}

		// Simplified signature verification:
		// In real Bitcoin, we'd verify the scriptSig against the scriptPubKey
		if in.ScriptSig == "" {
			return fmt.Errorf("missing signature for input %s:%d", in.TxID, in.OutIndex)
		}

		inputTotal += utxo.Value
	}

	outputTotal := tx.TotalOutputValue()

	// Input total must be >= output total (difference is fee)
	if inputTotal < outputTotal {
		return fmt.Errorf("insufficient funds: input=%d, output=%d", inputTotal, outputTotal)
	}

	return nil
}

// Copy creates a deep copy of the UTXO set
func (us *UTXOSet) Copy() *UTXOSet {
	newSet := NewUTXOSet()
	for txID, outputs := range us.UTXOs {
		for outIndex, utxo := range outputs {
			newSet.AddUTXO(txID, outIndex, utxo.Value, utxo.ScriptPubKey)
		}
	}
	return newSet
}

// CreateTransaction creates a transaction from one address to another
// Automatically selects UTXOs and creates change output
func (us *UTXOSet) CreateTransaction(from, to string, amount int64, privateKey string) (*Transaction, error) {
	utxos := us.FindUTXOsForAddress(from)

	var selectedUTXOs []*UTXO
	var totalInput int64

	// Select UTXOs until we have enough
	for _, utxo := range utxos {
		selectedUTXOs = append(selectedUTXOs, utxo)
		totalInput += utxo.Value
		if totalInput >= amount {
			break
		}
	}

	if totalInput < amount {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", totalInput, amount)
	}

	// Create inputs
	var inputs []TxInput
	for _, utxo := range selectedUTXOs {
		inputs = append(inputs, TxInput{
			TxID:     utxo.TxID,
			OutIndex: utxo.OutIndex,
		})
	}

	// Create outputs
	outputs := []TxOutput{
		{Value: amount, ScriptPubKey: to},
	}

	// Create change output if needed (excess goes to miner as fee if no change output)
	change := totalInput - amount
	if change > 0 {
		outputs = append(outputs, TxOutput{Value: change, ScriptPubKey: from})
	}

	tx := NewUTXOTransaction(inputs, outputs)
	tx.Sign(privateKey)

	return tx, nil
}
