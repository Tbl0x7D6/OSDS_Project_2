// Package transaction defines the UTXO-based transaction structure for the blockchain
package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
)

// Satoshi constants
const (
	SatoshiPerBTC = 100_000_000 // 1 BTC = 100,000,000 satoshi
)

// TxInput represents a transaction input (reference to a previous output)
type TxInput struct {
	TxID      string `json:"txid"`      // Previous transaction ID
	OutIndex  int    `json:"out_index"` // Index of the output in the previous transaction
	ScriptSig string `json:"scriptsig"` // Signature proving ownership (signed by private key of UTXO owner)
}

// TxOutput represents a transaction output
type TxOutput struct {
	Value        int64  `json:"value"`        // Amount in satoshi
	ScriptPubKey string `json:"scriptpubkey"` // Public key (account address)
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

// NewCoinbaseTransaction creates a new coinbase transaction (mining reward + fees)
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

// NewUTXOTransaction creates a transaction spending specific UTXOs
func NewUTXOTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}
	return tx
}

// CalculateHash computes the hash of the transaction (excludes scriptSig for signing)
func (tx *Transaction) CalculateHash() string {
	var buf bytes.Buffer

	for _, in := range tx.Inputs {
		buf.WriteString(in.TxID)
		buf.WriteString(fmt.Sprintf("%d", in.OutIndex))
		// Note: ScriptSig is NOT included in the hash for regular transactions
		// This allows the hash to be stable before and after signing
		// For coinbase, we include it for uniqueness
		if tx.IsCoinbase() {
			buf.WriteString(in.ScriptSig)
		}
	}

	for _, out := range tx.Outputs {
		buf.WriteString(fmt.Sprintf("%d", out.Value))
		buf.WriteString(out.ScriptPubKey)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:])
}

// GetDataToSign returns the canonical data to be signed (all scriptSigs cleared)
func (tx *Transaction) GetDataToSign() string {
	var buf bytes.Buffer

	for _, in := range tx.Inputs {
		buf.WriteString(in.TxID)
		buf.WriteString(fmt.Sprintf("%d", in.OutIndex))
		// ScriptSig is NOT included - it's cleared before signing
	}

	for _, out := range tx.Outputs {
		buf.WriteString(fmt.Sprintf("%d", out.Value))
		buf.WriteString(out.ScriptPubKey)
	}

	return buf.String()
}

// KeyPair represents an ECDSA key pair for signing transactions
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// GenerateKeyPair generates a new ECDSA key pair using P-256 curve
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}
	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// PublicKeyToHex converts a public key to hex string for storage
func PublicKeyToHex(pubKey *ecdsa.PublicKey) string {
	// Encode as uncompressed point: 04 || X || Y
	pubBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	return hex.EncodeToString(pubBytes)
}

// HexToPublicKey converts a hex string back to a public key
func HexToPublicKey(hexStr string) (*ecdsa.PublicKey, error) {
	pubBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %v", err)
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), pubBytes)
	if x == nil {
		return nil, fmt.Errorf("invalid public key encoding")
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}, nil
}

// PrivateKeyToHex converts a private key to hex string for storage
func PrivateKeyToHex(privKey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(privKey.D.Bytes())
}

// HexToPrivateKey converts a hex string back to a private key
func HexToPrivateKey(hexStr string) (*ecdsa.PrivateKey, error) {
	privBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %v", err)
	}

	privKey := new(ecdsa.PrivateKey)
	privKey.PublicKey.Curve = elliptic.P256()
	privKey.D = new(big.Int).SetBytes(privBytes)
	privKey.PublicKey.X, privKey.PublicKey.Y = privKey.PublicKey.Curve.ScalarBaseMult(privBytes)

	return privKey, nil
}

// GetPublicKeyHex returns the hex-encoded public key from a KeyPair
func (kp *KeyPair) GetPublicKeyHex() string {
	return PublicKeyToHex(kp.PublicKey)
}

// GetPrivateKeyHex returns the hex-encoded private key from a KeyPair
func (kp *KeyPair) GetPrivateKeyHex() string {
	return PrivateKeyToHex(kp.PrivateKey)
}

// SignECDSA signs data using ECDSA and returns the signature as hex string
// The signature is ASN.1 DER encoded
func SignECDSA(dataToSign string, privateKeyHex string) (string, error) {
	privateKey, err := HexToPrivateKey(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	// Hash the data first
	hash := sha256.Sum256([]byte(dataToSign))

	// Sign the hash using ASN.1 DER encoding
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %v", err)
	}

	return hex.EncodeToString(signature), nil
}

// VerifyECDSA verifies an ECDSA signature
// The signature is expected to be ASN.1 DER encoded
func VerifyECDSA(dataToSign, signatureHex, publicKeyHex string) bool {
	publicKey, err := HexToPublicKey(publicKeyHex)
	if err != nil {
		return false
	}

	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	// Hash the data
	hash := sha256.Sum256([]byte(dataToSign))

	// Verify the ASN.1 DER encoded signature
	return ecdsa.VerifyASN1(publicKey, hash[:], signatureBytes)
}

// SignWithPrivateKeys signs the transaction with multiple private keys (ECDSA)
// Each input must be signed by the owner of the referenced UTXO
// utxoOwners maps input index -> public key hex
// privateKeys maps public key hex -> private key hex
func (tx *Transaction) SignWithPrivateKeys(utxoOwners map[int]string, privateKeys map[string]string) error {
	if tx.IsCoinbase() {
		return nil // Coinbase transactions don't need signing
	}

	// Get the data to sign (with all scriptSigs cleared)
	dataToSign := tx.GetDataToSign()

	// Sign each input with the corresponding owner's private key
	for i := range tx.Inputs {
		owner, ok := utxoOwners[i]
		if !ok {
			return fmt.Errorf("no owner specified for input %d", i)
		}

		privateKey, ok := privateKeys[owner]
		if !ok {
			return fmt.Errorf("no private key for owner %s of input %d", owner, i)
		}

		// Generate ECDSA signature for this input
		signature, err := SignECDSA(dataToSign, privateKey)
		if err != nil {
			return fmt.Errorf("failed to sign input %d: %v", i, err)
		}
		tx.Inputs[i].ScriptSig = signature
	}

	// Recalculate ID
	tx.ID = tx.CalculateHash()
	return nil
}

// Verify verifies the transaction's basic structural validity
// Note: Full signature verification requires access to the UTXO set
func (tx *Transaction) Verify() bool {
	// Coinbase transactions have special rules
	if tx.IsCoinbase() {
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

	// Must have at least one input and one output
	if len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return false
	}

	// All inputs must have non-empty values
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

// VerifySignatures verifies all input signatures against their corresponding UTXO public keys
// utxoPublicKeys maps input index -> public key hex (scriptPubKey from the referenced UTXO)
func (tx *Transaction) VerifySignatures(utxoPublicKeys map[int]string) bool {
	if tx.IsCoinbase() {
		return true // Coinbase doesn't need signature verification
	}

	dataToSign := tx.GetDataToSign()

	for i, in := range tx.Inputs {
		publicKey, ok := utxoPublicKeys[i]
		if !ok {
			return false // No public key provided for this input
		}

		if !VerifyECDSA(dataToSign, in.ScriptSig, publicKey) {
			return false // Signature verification failed
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
		value, ok := us.UTXOs[txID][outIndex]
		if ok {
			return value
		}
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
// This includes checking UTXO existence, balance, and signature verification
func (us *UTXOSet) ValidateTransaction(tx *Transaction) error {
	// Coinbase transactions don't spend UTXOs
	if tx.IsCoinbase() {
		return nil
	}

	var inputTotal int64
	utxoPublicKeys := make(map[int]string)

	for i, in := range tx.Inputs {
		// Check if UTXO exists
		utxo := us.FindUTXO(in.TxID, in.OutIndex)
		if utxo == nil {
			return fmt.Errorf("UTXO not found: %s:%d", in.TxID, in.OutIndex)
		}

		// Check for empty signature
		if in.ScriptSig == "" {
			return fmt.Errorf("missing signature for input %s:%d", in.TxID, in.OutIndex)
		}

		// Store the public key for signature verification
		utxoPublicKeys[i] = utxo.ScriptPubKey
		inputTotal += utxo.Value
	}

	// Verify all signatures
	if !tx.VerifySignatures(utxoPublicKeys) {
		return fmt.Errorf("signature verification failed")
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

// CreateTransaction creates a transaction with inputs from one or multiple owners
// inputSpecs: list of UTXOs to spend (txID and output index)
// outputs: list of transaction outputs (recipients and amounts)
// privateKeys: map of public key hex -> private key hex for signing
// This function automatically finds UTXO owners and signs with the provided private keys
func (us *UTXOSet) CreateTransaction(
	inputSpecs []struct {
		TxID     string
		OutIndex int
	},
	outputs []TxOutput,
	privateKeys map[string]string,
) (*Transaction, error) {
	// Create inputs and collect owners
	var inputs []TxInput
	utxoOwners := make(map[int]string)
	var totalInput int64

	for i, spec := range inputSpecs {
		utxo := us.FindUTXO(spec.TxID, spec.OutIndex)
		if utxo == nil {
			return nil, fmt.Errorf("UTXO not found: %s:%d", spec.TxID, spec.OutIndex)
		}

		inputs = append(inputs, TxInput{
			TxID:     spec.TxID,
			OutIndex: spec.OutIndex,
		})
		utxoOwners[i] = utxo.ScriptPubKey
		totalInput += utxo.Value
	}

	// Verify we have private keys for all owners
	for i, owner := range utxoOwners {
		if _, ok := privateKeys[owner]; !ok {
			return nil, fmt.Errorf("missing private key for owner %s of input %d", owner, i)
		}
	}

	// Calculate total output value
	var totalOutput int64
	for _, out := range outputs {
		totalOutput += out.Value
	}

	// Verify sufficient funds
	if totalInput < totalOutput {
		return nil, fmt.Errorf("insufficient funds: input=%d, output=%d", totalInput, totalOutput)
	}

	tx := NewUTXOTransaction(inputs, outputs)

	// Sign with multiple private keys
	err := tx.SignWithPrivateKeys(utxoOwners, privateKeys)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// GetAllUTXOs returns all UTXOs in the set (for debugging/testing)
func (us *UTXOSet) GetAllUTXOs() []*UTXO {
	var all []*UTXO
	for _, outputs := range us.UTXOs {
		for _, utxo := range outputs {
			all = append(all, utxo)
		}
	}
	// Sort for deterministic output
	sort.Slice(all, func(i, j int) bool {
		if all[i].TxID != all[j].TxID {
			return all[i].TxID < all[j].TxID
		}
		return all[i].OutIndex < all[j].OutIndex
	})
	return all
}
