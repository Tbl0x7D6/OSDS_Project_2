package transaction

import (
	"testing"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("alice", "bob", 10.0)

	if tx.From != "alice" {
		t.Errorf("Expected from to be 'alice', got '%s'", tx.From)
	}
	if tx.To != "bob" {
		t.Errorf("Expected to to be 'bob', got '%s'", tx.To)
	}
	if tx.Amount != 10.0 {
		t.Errorf("Expected amount to be 10.0, got %f", tx.Amount)
	}
	if tx.ID == "" {
		t.Error("Expected transaction ID to be set")
	}
}

func TestTransactionHash(t *testing.T) {
	tx1 := NewTransaction("alice", "bob", 10.0)
	tx2 := NewTransaction("alice", "bob", 10.0)

	// Different transactions should have different hashes (due to timestamp)
	if tx1.ID == tx2.ID {
		t.Error("Expected different transaction IDs for different transactions")
	}
}

func TestTransactionSignAndVerify(t *testing.T) {
	tx := NewTransaction("alice", "bob", 10.0)

	// Unsigned transaction should not verify
	if tx.Verify() {
		t.Error("Unsigned transaction should not verify")
	}

	// Sign the transaction
	tx.Sign("alice_private_key")

	// Signed transaction should verify
	if !tx.Verify() {
		t.Error("Signed transaction should verify")
	}
}

func TestInvalidTransaction(t *testing.T) {
	// Transaction with zero amount
	tx := NewTransaction("alice", "bob", 0)
	tx.Sign("key")
	if tx.Verify() {
		t.Error("Transaction with zero amount should not verify")
	}

	// Transaction with negative amount
	tx2 := NewTransaction("alice", "bob", -10)
	tx2.Sign("key")
	if tx2.Verify() {
		t.Error("Transaction with negative amount should not verify")
	}

	// Transaction with empty sender
	tx3 := NewTransaction("", "bob", 10)
	tx3.Sign("key")
	if tx3.Verify() {
		t.Error("Transaction with empty sender should not verify")
	}

	// Transaction with empty receiver
	tx4 := NewTransaction("alice", "", 10)
	tx4.Sign("key")
	if tx4.Verify() {
		t.Error("Transaction with empty receiver should not verify")
	}
}

func TestTransactionSerialization(t *testing.T) {
	tx := NewTransaction("alice", "bob", 10.0)
	tx.Sign("alice_private_key")

	// Serialize
	data, err := tx.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize transaction: %v", err)
	}

	// Deserialize
	tx2, err := DeserializeTransaction(data)
	if err != nil {
		t.Fatalf("Failed to deserialize transaction: %v", err)
	}

	// Compare
	if tx.ID != tx2.ID {
		t.Error("Transaction ID mismatch after deserialization")
	}
	if tx.From != tx2.From {
		t.Error("From address mismatch after deserialization")
	}
	if tx.To != tx2.To {
		t.Error("To address mismatch after deserialization")
	}
	if tx.Amount != tx2.Amount {
		t.Error("Amount mismatch after deserialization")
	}
	if tx.Signature != tx2.Signature {
		t.Error("Signature mismatch after deserialization")
	}
}
