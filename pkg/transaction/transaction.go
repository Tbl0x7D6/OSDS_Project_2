// Package transaction defines the transaction structure for the blockchain
package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Transaction represents a single transaction in the blockchain
type Transaction struct {
	ID        string  `json:"id"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
	Signature string  `json:"signature"` // Simplified signature for demo
}

// NewTransaction creates a new transaction
func NewTransaction(from, to string, amount float64) *Transaction {
	tx := &Transaction{
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now().UnixNano(),
	}
	tx.ID = tx.CalculateHash()
	return tx
}

// CalculateHash computes the hash of the transaction
func (tx *Transaction) CalculateHash() string {
	data := fmt.Sprintf("%s%s%f%d", tx.From, tx.To, tx.Amount, tx.Timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Sign signs the transaction (simplified for demo - in real system would use private key)
func (tx *Transaction) Sign(privateKey string) {
	data := fmt.Sprintf("%s%s%f%d%s", tx.From, tx.To, tx.Amount, tx.Timestamp, privateKey)
	hash := sha256.Sum256([]byte(data))
	tx.Signature = hex.EncodeToString(hash[:])
}

// Verify verifies the transaction signature (simplified)
func (tx *Transaction) Verify() bool {
	// For demo purposes, we just check that signature is not empty
	// In a real implementation, this would verify against public key
	if tx.Signature == "" {
		return false
	}
	// Basic validation
	if tx.Amount <= 0 {
		return false
	}
	if tx.From == "" || tx.To == "" {
		return false
	}
	return true
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
	return fmt.Sprintf("TX{ID: %s..., From: %s, To: %s, Amount: %.2f}",
		tx.ID[:8], tx.From, tx.To, tx.Amount)
}
