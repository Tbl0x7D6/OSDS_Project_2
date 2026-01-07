package transaction

import (
	"testing"
)

func TestNewCoinbaseTransaction(t *testing.T) {
	tx := NewCoinbaseTransaction("miner1", 5000000000, 1) // 50 BTC reward

	if !tx.IsCoinbase() {
		t.Error("Expected coinbase transaction")
	}
	if len(tx.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(tx.Inputs))
	}
	if tx.Inputs[0].TxID != "" {
		t.Error("Coinbase input should have empty TxID")
	}
	if tx.Inputs[0].OutIndex != -1 {
		t.Errorf("Coinbase input should have OutIndex -1, got %d", tx.Inputs[0].OutIndex)
	}
	if len(tx.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Outputs))
	}
	if tx.Outputs[0].Value != 5000000000 {
		t.Errorf("Expected value 5000000000, got %d", tx.Outputs[0].Value)
	}
	if tx.Outputs[0].ScriptPubKey != "miner1" {
		t.Errorf("Expected scriptPubKey 'miner1', got '%s'", tx.Outputs[0].ScriptPubKey)
	}
}

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("alice", "bob", 10.0)

	if tx.IsCoinbase() {
		t.Error("Regular transaction should not be coinbase")
	}
	if len(tx.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(tx.Outputs))
	}
	// 10 BTC = 1,000,000,000 satoshi
	if tx.Outputs[0].Value != 1000000000 {
		t.Errorf("Expected value 1000000000 satoshi, got %d", tx.Outputs[0].Value)
	}
	if tx.Outputs[0].ScriptPubKey != "bob" {
		t.Errorf("Expected scriptPubKey 'bob', got '%s'", tx.Outputs[0].ScriptPubKey)
	}
}

func TestNewTransactionSatoshi(t *testing.T) {
	tx := NewTransactionSatoshi("alice", "bob", 50000)

	if tx.Outputs[0].Value != 50000 {
		t.Errorf("Expected value 50000, got %d", tx.Outputs[0].Value)
	}
}

func TestTransactionSignAndVerify(t *testing.T) {
	// Create a UTXO set with some funds
	utxoSet := NewUTXOSet()

	// First create a coinbase transaction to give alice some funds
	coinbase := NewCoinbaseTransaction("alice", 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Create a transaction from alice to bob using the UTXO
	tx, err := utxoSet.CreateTransaction("alice", "bob", 1000000000, "alice_private_key")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// Verify the transaction
	if !tx.Verify() {
		t.Error("Signed transaction should verify")
	}

	// Validate against UTXO set
	err = utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("Transaction should be valid against UTXO set: %v", err)
	}
}

func TestUnsignedTransaction(t *testing.T) {
	// Create unsigned transaction
	inputs := []TxInput{{TxID: "abc123", OutIndex: 0}}
	outputs := []TxOutput{{Value: 1000, ScriptPubKey: "bob"}}
	tx := NewUTXOTransaction(inputs, outputs)

	// Unsigned transaction should not verify
	if tx.Verify() {
		t.Error("Unsigned transaction should not verify")
	}
}

func TestCoinbaseVerify(t *testing.T) {
	coinbase := NewCoinbaseTransaction("miner", 5000000000, 1)

	if !coinbase.Verify() {
		t.Error("Valid coinbase transaction should verify")
	}
}

func TestInvalidCoinbase(t *testing.T) {
	// Coinbase with invalid output (negative value)
	tx := &Transaction{
		Inputs:  []TxInput{{TxID: "", OutIndex: -1, ScriptSig: "coinbase:1"}},
		Outputs: []TxOutput{{Value: -100, ScriptPubKey: "miner"}},
	}
	tx.ID = tx.CalculateHash()

	if tx.Verify() {
		t.Error("Coinbase with negative output should not verify")
	}

	// Coinbase with empty scriptPubKey
	tx2 := &Transaction{
		Inputs:  []TxInput{{TxID: "", OutIndex: -1, ScriptSig: "coinbase:1"}},
		Outputs: []TxOutput{{Value: 100, ScriptPubKey: ""}},
	}
	tx2.ID = tx2.CalculateHash()

	if tx2.Verify() {
		t.Error("Coinbase with empty scriptPubKey should not verify")
	}
}

func TestTransactionHash(t *testing.T) {
	tx1 := NewCoinbaseTransaction("miner", 5000000000, 1)
	tx2 := NewCoinbaseTransaction("miner", 5000000000, 2) // Different block height

	if tx1.ID == tx2.ID {
		t.Error("Transactions with different data should have different IDs")
	}
}

func TestTransactionSerialization(t *testing.T) {
	coinbase := NewCoinbaseTransaction("miner", 5000000000, 1)

	// Serialize
	data, err := coinbase.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize transaction: %v", err)
	}

	// Deserialize
	tx2, err := DeserializeTransaction(data)
	if err != nil {
		t.Fatalf("Failed to deserialize transaction: %v", err)
	}

	// Compare
	if coinbase.ID != tx2.ID {
		t.Error("Transaction ID mismatch after deserialization")
	}
	if len(tx2.Inputs) != len(coinbase.Inputs) {
		t.Error("Input count mismatch after deserialization")
	}
	if len(tx2.Outputs) != len(coinbase.Outputs) {
		t.Error("Output count mismatch after deserialization")
	}
}

func TestUTXOSet(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Add some UTXOs
	utxoSet.AddUTXO("tx1", 0, 1000000, "alice")
	utxoSet.AddUTXO("tx1", 1, 2000000, "bob")
	utxoSet.AddUTXO("tx2", 0, 500000, "alice")

	// Test FindUTXO
	utxo := utxoSet.FindUTXO("tx1", 0)
	if utxo == nil {
		t.Fatal("UTXO not found")
	}
	if utxo.Value != 1000000 {
		t.Errorf("Expected value 1000000, got %d", utxo.Value)
	}

	// Test FindUTXOsForAddress
	aliceUTXOs := utxoSet.FindUTXOsForAddress("alice")
	if len(aliceUTXOs) != 2 {
		t.Errorf("Expected 2 UTXOs for alice, got %d", len(aliceUTXOs))
	}

	// Test GetBalance
	aliceBalance := utxoSet.GetBalance("alice")
	if aliceBalance != 1500000 {
		t.Errorf("Expected balance 1500000, got %d", aliceBalance)
	}

	// Test RemoveUTXO
	utxoSet.RemoveUTXO("tx1", 0)
	if utxoSet.HasUTXO("tx1", 0) {
		t.Error("UTXO should have been removed")
	}

	aliceBalance = utxoSet.GetBalance("alice")
	if aliceBalance != 500000 {
		t.Errorf("Expected balance 500000 after removal, got %d", aliceBalance)
	}
}

func TestUTXOSetProcessTransaction(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Process coinbase transaction
	coinbase := NewCoinbaseTransaction("alice", 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Check UTXO was created
	aliceBalance := utxoSet.GetBalance("alice")
	if aliceBalance != 5000000000 {
		t.Errorf("Expected balance 5000000000, got %d", aliceBalance)
	}

	// Create and process a transaction from alice to bob
	tx, err := utxoSet.CreateTransaction("alice", "bob", 1000000000, "alice_key")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	utxoSet.ProcessTransaction(tx)

	// Check balances after transaction
	bobBalance := utxoSet.GetBalance("bob")
	if bobBalance != 1000000000 {
		t.Errorf("Expected bob's balance 1000000000, got %d", bobBalance)
	}

	aliceBalance = utxoSet.GetBalance("alice")
	// Alice should have change (5000000000 - 1000000000 = 4000000000)
	if aliceBalance != 4000000000 {
		t.Errorf("Expected alice's balance 4000000000, got %d", aliceBalance)
	}
}

func TestUTXOSetValidateTransaction(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction("alice", 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Valid transaction
	tx, _ := utxoSet.CreateTransaction("alice", "bob", 1000000000, "alice_key")
	err := utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("Valid transaction should pass validation: %v", err)
	}

	// Transaction spending non-existent UTXO
	badTx := &Transaction{
		Inputs:  []TxInput{{TxID: "nonexistent", OutIndex: 0, ScriptSig: "sig"}},
		Outputs: []TxOutput{{Value: 1000, ScriptPubKey: "bob"}},
	}
	badTx.ID = badTx.CalculateHash()

	err = utxoSet.ValidateTransaction(badTx)
	if err == nil {
		t.Error("Transaction with non-existent UTXO should fail validation")
	}
}

func TestCreateTransactionInsufficientFunds(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction("alice", 1000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Try to spend more than available
	_, err := utxoSet.CreateTransaction("alice", "bob", 2000000, "alice_key")
	if err == nil {
		t.Error("Should fail with insufficient funds")
	}
}

func TestTransactionFee(t *testing.T) {
	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction("alice", 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Create transaction manually with fee
	inputs := []TxInput{{TxID: coinbase.ID, OutIndex: 0}}
	outputs := []TxOutput{
		{Value: 3000000000, ScriptPubKey: "bob"},   // Send 30 BTC
		{Value: 1000000000, ScriptPubKey: "alice"}, // Change 10 BTC
		// Missing 10 BTC becomes fee
	}
	tx := NewUTXOTransaction(inputs, outputs)
	tx.Sign("alice_key")

	fee := tx.GetFee(utxoSet)
	// Fee should be 5000000000 - 3000000000 - 1000000000 = 1000000000
	if fee != 1000000000 {
		t.Errorf("Expected fee 1000000000, got %d", fee)
	}
}

func TestCoinbaseFee(t *testing.T) {
	utxoSet := NewUTXOSet()
	coinbase := NewCoinbaseTransaction("miner", 5000000000, 0)

	fee := coinbase.GetFee(utxoSet)
	if fee != 0 {
		t.Errorf("Coinbase fee should be 0, got %d", fee)
	}
}

func TestTotalOutputValue(t *testing.T) {
	tx := &Transaction{
		Outputs: []TxOutput{
			{Value: 1000000, ScriptPubKey: "alice"},
			{Value: 2000000, ScriptPubKey: "bob"},
			{Value: 500000, ScriptPubKey: "charlie"},
		},
	}

	total := tx.TotalOutputValue()
	if total != 3500000 {
		t.Errorf("Expected total 3500000, got %d", total)
	}
}

func TestUTXOSetCopy(t *testing.T) {
	utxoSet := NewUTXOSet()
	utxoSet.AddUTXO("tx1", 0, 1000000, "alice")
	utxoSet.AddUTXO("tx2", 0, 2000000, "bob")

	// Create copy
	copy := utxoSet.Copy()

	// Modify original
	utxoSet.RemoveUTXO("tx1", 0)

	// Copy should still have the UTXO
	if !copy.HasUTXO("tx1", 0) {
		t.Error("Copy should still have tx1:0")
	}

	// Original should not have it
	if utxoSet.HasUTXO("tx1", 0) {
		t.Error("Original should not have tx1:0")
	}
}

func TestTransactionString(t *testing.T) {
	coinbase := NewCoinbaseTransaction("miner", 5000000000, 0)
	str := coinbase.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}

	tx := NewTransactionSatoshi("alice", "bob", 1000000)
	tx.Sign("key")
	str = tx.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}
}
