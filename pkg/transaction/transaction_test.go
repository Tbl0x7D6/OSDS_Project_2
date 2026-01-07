package transaction

import (
	"testing"
)

// Helper function to create a key pair for testing
func mustGenerateKeyPair(t *testing.T) *KeyPair {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	return kp
}

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

func TestKeyPairGeneration(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if kp.PrivateKey == nil {
		t.Error("Private key should not be nil")
	}
	if kp.PublicKey == nil {
		t.Error("Public key should not be nil")
	}

	// Test hex conversion
	pubHex := kp.GetPublicKeyHex()
	privHex := kp.GetPrivateKeyHex()

	if pubHex == "" {
		t.Error("Public key hex should not be empty")
	}
	if privHex == "" {
		t.Error("Private key hex should not be empty")
	}

	// Test round-trip conversion
	pubKey, err := HexToPublicKey(pubHex)
	if err != nil {
		t.Fatalf("Failed to convert hex to public key: %v", err)
	}
	if pubKey.X.Cmp(kp.PublicKey.X) != 0 || pubKey.Y.Cmp(kp.PublicKey.Y) != 0 {
		t.Error("Public key round-trip failed")
	}

	privKey, err := HexToPrivateKey(privHex)
	if err != nil {
		t.Fatalf("Failed to convert hex to private key: %v", err)
	}
	if privKey.D.Cmp(kp.PrivateKey.D) != 0 {
		t.Error("Private key round-trip failed")
	}
}

func TestECDSASignAndVerify(t *testing.T) {
	kp := mustGenerateKeyPair(t)
	dataToSign := "test data to sign"

	// Sign with private key
	signature, err := SignECDSA(dataToSign, kp.GetPrivateKeyHex())
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Verify with public key
	if !VerifyECDSA(dataToSign, signature, kp.GetPublicKeyHex()) {
		t.Error("Signature verification failed")
	}

	// Verify with wrong data should fail
	if VerifyECDSA("wrong data", signature, kp.GetPublicKeyHex()) {
		t.Error("Verification should fail with wrong data")
	}

	// Verify with wrong public key should fail
	kp2 := mustGenerateKeyPair(t)
	if VerifyECDSA(dataToSign, signature, kp2.GetPublicKeyHex()) {
		t.Error("Verification should fail with wrong public key")
	}
}

func TestTransactionSignAndVerify(t *testing.T) {
	// Create key pairs for alice and bob
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePubHex := aliceKP.GetPublicKeyHex()
	bobPubHex := bobKP.GetPublicKeyHex()

	// Create a UTXO set with some funds for alice
	utxoSet := NewUTXOSet()
	coinbase := NewCoinbaseTransaction(alicePubHex, 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Create a transaction from alice to bob
	tx, err := utxoSet.CreateTransaction(alicePubHex, bobPubHex, 1000000000, aliceKP.GetPrivateKeyHex())
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// Verify the transaction structure
	if !tx.Verify() {
		t.Error("Signed transaction should verify structure")
	}

	// Validate against UTXO set (includes signature verification)
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
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePubHex := aliceKP.GetPublicKeyHex()
	bobPubHex := bobKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Process coinbase transaction
	coinbase := NewCoinbaseTransaction(alicePubHex, 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Check UTXO was created
	aliceBalance := utxoSet.GetBalance(alicePubHex)
	if aliceBalance != 5000000000 {
		t.Errorf("Expected balance 5000000000, got %d", aliceBalance)
	}

	// Create and process a transaction from alice to bob
	tx, err := utxoSet.CreateTransaction(alicePubHex, bobPubHex, 1000000000, aliceKP.GetPrivateKeyHex())
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	utxoSet.ProcessTransaction(tx)

	// Check balances after transaction
	bobBalance := utxoSet.GetBalance(bobPubHex)
	if bobBalance != 1000000000 {
		t.Errorf("Expected bob's balance 1000000000, got %d", bobBalance)
	}

	aliceBalance = utxoSet.GetBalance(alicePubHex)
	// Alice should have change (5000000000 - 1000000000 = 4000000000)
	if aliceBalance != 4000000000 {
		t.Errorf("Expected alice's balance 4000000000, got %d", aliceBalance)
	}
}

func TestUTXOSetValidateTransaction(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePubHex := aliceKP.GetPublicKeyHex()
	bobPubHex := bobKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction(alicePubHex, 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Valid transaction with correct signature
	tx, _ := utxoSet.CreateTransaction(alicePubHex, bobPubHex, 1000000000, aliceKP.GetPrivateKeyHex())
	err := utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("Valid transaction should pass validation: %v", err)
	}

	// Transaction spending non-existent UTXO
	badTx := &Transaction{
		Inputs:  []TxInput{{TxID: "nonexistent", OutIndex: 0, ScriptSig: "sig"}},
		Outputs: []TxOutput{{Value: 1000, ScriptPubKey: bobPubHex}},
	}
	badTx.ID = badTx.CalculateHash()

	err = utxoSet.ValidateTransaction(badTx)
	if err == nil {
		t.Error("Transaction with non-existent UTXO should fail validation")
	}
}

func TestCreateTransactionInsufficientFunds(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePubHex := aliceKP.GetPublicKeyHex()
	bobPubHex := bobKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction(alicePubHex, 1000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Try to spend more than available
	_, err := utxoSet.CreateTransaction(alicePubHex, bobPubHex, 2000000, aliceKP.GetPrivateKeyHex())
	if err == nil {
		t.Error("Should fail with insufficient funds")
	}
}

func TestTransactionFee(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePubHex := aliceKP.GetPublicKeyHex()
	bobPubHex := bobKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice some funds
	coinbase := NewCoinbaseTransaction(alicePubHex, 5000000000, 0)
	utxoSet.ProcessTransaction(coinbase)

	// Create transaction manually with fee
	inputs := []TxInput{{TxID: coinbase.ID, OutIndex: 0}}
	outputs := []TxOutput{
		{Value: 3000000000, ScriptPubKey: bobPubHex},   // Send 30 BTC
		{Value: 1000000000, ScriptPubKey: alicePubHex}, // Change 10 BTC
		// Missing 10 BTC becomes fee
	}
	tx := NewUTXOTransaction(inputs, outputs)
	tx.Sign(aliceKP.GetPrivateKeyHex())

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

	// Test regular transaction string
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	inputs := []TxInput{{TxID: "abc123", OutIndex: 0}}
	outputs := []TxOutput{{Value: 1000000, ScriptPubKey: bobKP.GetPublicKeyHex()}}
	tx := NewUTXOTransaction(inputs, outputs)
	tx.Sign(aliceKP.GetPrivateKeyHex())
	str = tx.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}
}

// ============== Multi-signature Tests ==============

func TestMultiInputTransaction(t *testing.T) {
	// Create key pairs for alice, bob and charlie
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	charlieKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()
	charliePub := charlieKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice and bob some funds via coinbase
	coinbaseAlice := NewCoinbaseTransaction(alicePub, 3000000000, 0) // 30 BTC
	coinbaseBob := NewCoinbaseTransaction(bobPub, 2000000000, 1)     // 20 BTC
	utxoSet.ProcessTransaction(coinbaseAlice)
	utxoSet.ProcessTransaction(coinbaseBob)

	// Create a transaction spending both alice's and bob's UTXOs
	// Total: 50 BTC -> 45 BTC to charlie, rest is fee
	inputSpecs := []struct {
		TxID     string
		OutIndex int
	}{
		{TxID: coinbaseAlice.ID, OutIndex: 0},
		{TxID: coinbaseBob.ID, OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 4500000000, ScriptPubKey: charliePub}, // 45 BTC to charlie
	}

	privateKeys := map[string]string{
		alicePub: aliceKP.GetPrivateKeyHex(),
		bobPub:   bobKP.GetPrivateKeyHex(),
	}

	tx, err := utxoSet.CreateMultiInputTransaction(inputSpecs, outputs, privateKeys)
	if err != nil {
		t.Fatalf("Failed to create multi-input transaction: %v", err)
	}

	// Verify transaction structure
	if !tx.Verify() {
		t.Error("Multi-input transaction should have valid structure")
	}

	// Validate against UTXO set (includes signature verification)
	err = utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("Multi-input transaction should be valid: %v", err)
	}

	// Process the transaction
	utxoSet.ProcessTransaction(tx)

	// Verify balances
	charlieBalance := utxoSet.GetBalance(charliePub)
	if charlieBalance != 4500000000 {
		t.Errorf("Expected charlie's balance 4500000000, got %d", charlieBalance)
	}

	aliceBalance := utxoSet.GetBalance(alicePub)
	if aliceBalance != 0 {
		t.Errorf("Expected alice's balance 0, got %d", aliceBalance)
	}

	bobBalance := utxoSet.GetBalance(bobPub)
	if bobBalance != 0 {
		t.Errorf("Expected bob's balance 0, got %d", bobBalance)
	}
}

func TestMultiInputTransactionWithChange(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	charlieKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()
	charliePub := charlieKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice and bob some funds
	coinbaseAlice := NewCoinbaseTransaction(alicePub, 3000000000, 0)
	coinbaseBob := NewCoinbaseTransaction(bobPub, 2000000000, 1)
	utxoSet.ProcessTransaction(coinbaseAlice)
	utxoSet.ProcessTransaction(coinbaseBob)

	// Create transaction: alice + bob -> charlie (40 BTC) + alice change (9 BTC) + fee (1 BTC)
	inputSpecs := []struct {
		TxID     string
		OutIndex int
	}{
		{TxID: coinbaseAlice.ID, OutIndex: 0},
		{TxID: coinbaseBob.ID, OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 4000000000, ScriptPubKey: charliePub}, // 40 BTC to charlie
		{Value: 900000000, ScriptPubKey: alicePub},    // 9 BTC change to alice
	}

	privateKeys := map[string]string{
		alicePub: aliceKP.GetPrivateKeyHex(),
		bobPub:   bobKP.GetPrivateKeyHex(),
	}

	tx, err := utxoSet.CreateMultiInputTransaction(inputSpecs, outputs, privateKeys)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// Check fee
	fee := tx.GetFee(utxoSet)
	expectedFee := int64(100000000) // 1 BTC
	if fee != expectedFee {
		t.Errorf("Expected fee %d, got %d", expectedFee, fee)
	}

	// Validate and process
	err = utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("Transaction should be valid: %v", err)
	}

	utxoSet.ProcessTransaction(tx)

	// Verify balances
	if utxoSet.GetBalance(charliePub) != 4000000000 {
		t.Errorf("Charlie's balance incorrect")
	}
	if utxoSet.GetBalance(alicePub) != 900000000 {
		t.Errorf("Alice's change incorrect")
	}
}

func TestMultiInputTransactionWrongSignature(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	charlieKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()
	charliePub := charlieKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give alice and bob some funds
	coinbaseAlice := NewCoinbaseTransaction(alicePub, 3000000000, 0)
	coinbaseBob := NewCoinbaseTransaction(bobPub, 2000000000, 1)
	utxoSet.ProcessTransaction(coinbaseAlice)
	utxoSet.ProcessTransaction(coinbaseBob)

	// Create transaction inputs
	inputs := []TxInput{
		{TxID: coinbaseAlice.ID, OutIndex: 0},
		{TxID: coinbaseBob.ID, OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 4500000000, ScriptPubKey: charliePub},
	}

	tx := NewUTXOTransaction(inputs, outputs)

	// Sign both inputs with alice's key only (bob's input should fail verification)
	dataToSign := tx.GetDataToSign()
	aliceSig, _ := SignECDSA(dataToSign, aliceKP.GetPrivateKeyHex())
	tx.Inputs[0].ScriptSig = aliceSig
	tx.Inputs[1].ScriptSig = aliceSig // Wrong! Should be bob's signature

	// Validation should fail
	err := utxoSet.ValidateTransaction(tx)
	if err == nil {
		t.Error("Transaction with wrong signature should fail validation")
	}
}

func TestSignWithPrivateKeys(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	charlieKP := mustGenerateKeyPair(t)
	recipientKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()
	charliePub := charlieKP.GetPublicKeyHex()

	// Create a transaction with 3 inputs from different owners
	inputs := []TxInput{
		{TxID: "tx1", OutIndex: 0},
		{TxID: "tx2", OutIndex: 0},
		{TxID: "tx3", OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 10000000, ScriptPubKey: recipientKP.GetPublicKeyHex()},
	}

	tx := NewUTXOTransaction(inputs, outputs)

	// Map input index to owner (public key hex)
	utxoOwners := map[int]string{
		0: alicePub,
		1: bobPub,
		2: charliePub,
	}

	// Provide private keys
	privateKeys := map[string]string{
		alicePub:   aliceKP.GetPrivateKeyHex(),
		bobPub:     bobKP.GetPrivateKeyHex(),
		charliePub: charlieKP.GetPrivateKeyHex(),
	}

	err := tx.SignWithPrivateKeys(utxoOwners, privateKeys)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify each input has a signature
	for i, in := range tx.Inputs {
		if in.ScriptSig == "" {
			t.Errorf("Input %d should have a signature", i)
		}
	}

	// Verify signatures
	if !tx.VerifySignatures(utxoOwners) {
		t.Error("All signatures should verify")
	}
}

func TestSignWithPrivateKeysMissingKey(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()

	inputs := []TxInput{
		{TxID: "tx1", OutIndex: 0},
		{TxID: "tx2", OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 10000000, ScriptPubKey: "recipient"},
	}

	tx := NewUTXOTransaction(inputs, outputs)

	utxoOwners := map[int]string{
		0: alicePub,
		1: bobPub,
	}

	// Missing bob's private key
	privateKeys := map[string]string{
		alicePub: aliceKP.GetPrivateKeyHex(),
	}

	err := tx.SignWithPrivateKeys(utxoOwners, privateKeys)
	if err == nil {
		t.Error("Should fail when missing private key")
	}
}

func TestVerifySignaturesPartiallyValid(t *testing.T) {
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()

	inputs := []TxInput{
		{TxID: "tx1", OutIndex: 0},
		{TxID: "tx2", OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 10000000, ScriptPubKey: "recipient"},
	}

	tx := NewUTXOTransaction(inputs, outputs)

	// Sign only the first input correctly
	dataToSign := tx.GetDataToSign()
	aliceSig, _ := SignECDSA(dataToSign, aliceKP.GetPrivateKeyHex())
	tx.Inputs[0].ScriptSig = aliceSig
	tx.Inputs[1].ScriptSig = "invalid_signature"

	utxoOwners := map[int]string{
		0: alicePub,
		1: bobPub,
	}

	// Should fail because second signature is invalid
	if tx.VerifySignatures(utxoOwners) {
		t.Error("Verification should fail with invalid signature")
	}
}

func TestThreePartyTransaction(t *testing.T) {
	// Scenario: Alice, Bob, and Charlie each contribute UTXOs to pay Dave
	aliceKP := mustGenerateKeyPair(t)
	bobKP := mustGenerateKeyPair(t)
	charlieKP := mustGenerateKeyPair(t)
	daveKP := mustGenerateKeyPair(t)
	alicePub := aliceKP.GetPublicKeyHex()
	bobPub := bobKP.GetPublicKeyHex()
	charliePub := charlieKP.GetPublicKeyHex()
	davePub := daveKP.GetPublicKeyHex()

	utxoSet := NewUTXOSet()

	// Give each person some funds
	coinbaseAlice := NewCoinbaseTransaction(alicePub, 1000000000, 0)     // 10 BTC
	coinbaseBob := NewCoinbaseTransaction(bobPub, 2000000000, 1)         // 20 BTC
	coinbaseCharlie := NewCoinbaseTransaction(charliePub, 1500000000, 2) // 15 BTC
	utxoSet.ProcessTransaction(coinbaseAlice)
	utxoSet.ProcessTransaction(coinbaseBob)
	utxoSet.ProcessTransaction(coinbaseCharlie)

	// Create transaction: all three -> dave (40 BTC) + bob change (4 BTC) + fee (1 BTC)
	inputSpecs := []struct {
		TxID     string
		OutIndex int
	}{
		{TxID: coinbaseAlice.ID, OutIndex: 0},
		{TxID: coinbaseBob.ID, OutIndex: 0},
		{TxID: coinbaseCharlie.ID, OutIndex: 0},
	}

	outputs := []TxOutput{
		{Value: 4000000000, ScriptPubKey: davePub}, // 40 BTC to dave
		{Value: 400000000, ScriptPubKey: bobPub},   // 4 BTC change to bob
	}

	privateKeys := map[string]string{
		alicePub:   aliceKP.GetPrivateKeyHex(),
		bobPub:     bobKP.GetPrivateKeyHex(),
		charliePub: charlieKP.GetPrivateKeyHex(),
	}

	tx, err := utxoSet.CreateMultiInputTransaction(inputSpecs, outputs, privateKeys)
	if err != nil {
		t.Fatalf("Failed to create 3-party transaction: %v", err)
	}

	// Validate
	err = utxoSet.ValidateTransaction(tx)
	if err != nil {
		t.Errorf("3-party transaction should be valid: %v", err)
	}

	// Verify fee
	fee := tx.GetFee(utxoSet)
	expectedFee := int64(100000000) // 1 BTC
	if fee != expectedFee {
		t.Errorf("Expected fee %d, got %d", expectedFee, fee)
	}

	// Process
	utxoSet.ProcessTransaction(tx)

	// Verify final balances
	if utxoSet.GetBalance(davePub) != 4000000000 {
		t.Errorf("Dave's balance incorrect: %d", utxoSet.GetBalance(davePub))
	}
	if utxoSet.GetBalance(bobPub) != 400000000 {
		t.Errorf("Bob's change incorrect: %d", utxoSet.GetBalance(bobPub))
	}
	if utxoSet.GetBalance(alicePub) != 0 {
		t.Errorf("Alice should have 0 balance")
	}
	if utxoSet.GetBalance(charliePub) != 0 {
		t.Errorf("Charlie should have 0 balance")
	}
}

func TestCoinbaseNoSignatureRequired(t *testing.T) {
	coinbase := NewCoinbaseTransaction("miner", 5000000000, 0)

	// Coinbase should verify without signature verification
	utxoOwners := map[int]string{}
	if !coinbase.VerifySignatures(utxoOwners) {
		t.Error("Coinbase should pass signature verification")
	}
}

// ============== ECDSA Key Management Tests ==============

func TestInvalidPrivateKeyHex(t *testing.T) {
	_, err := HexToPrivateKey("invalid_hex")
	if err == nil {
		t.Error("Should fail with invalid hex")
	}
}

func TestInvalidPublicKeyHex(t *testing.T) {
	_, err := HexToPublicKey("invalid_hex")
	if err == nil {
		t.Error("Should fail with invalid hex")
	}

	// Valid hex but invalid public key encoding
	_, err = HexToPublicKey("0102030405")
	if err == nil {
		t.Error("Should fail with invalid public key encoding")
	}
}

func TestSignWithInvalidPrivateKey(t *testing.T) {
	_, err := SignECDSA("data", "invalid_key")
	if err == nil {
		t.Error("Should fail with invalid private key")
	}
}

func TestVerifyWithInvalidSignature(t *testing.T) {
	kp := mustGenerateKeyPair(t)

	// Invalid signature format
	if VerifyECDSA("data", "invalid", kp.GetPublicKeyHex()) {
		t.Error("Should fail with invalid signature")
	}

	// Wrong length signature
	if VerifyECDSA("data", "0102030405", kp.GetPublicKeyHex()) {
		t.Error("Should fail with wrong length signature")
	}
}

func TestMultipleKeyPairsUniqueness(t *testing.T) {
	kp1 := mustGenerateKeyPair(t)
	kp2 := mustGenerateKeyPair(t)

	if kp1.GetPublicKeyHex() == kp2.GetPublicKeyHex() {
		t.Error("Two key pairs should have different public keys")
	}

	if kp1.GetPrivateKeyHex() == kp2.GetPrivateKeyHex() {
		t.Error("Two key pairs should have different private keys")
	}
}
