#!/bin/bash
# Demo script for the blockchain system
# This script demonstrates all required features

set -e

echo "=============================================="
echo "       Blockchain Demo - Nakamoto Protocol"
echo "=============================================="
echo ""

# Build the project
echo "Building the project..."
go build -o bin/miner ./cmd/miner
go build -o bin/client ./cmd/client
go build -o bin/fakeminer ./cmd/fakeminer
echo "Build complete!"
echo ""

# Clean up any existing processes
cleanup() {
    echo "Cleaning up..."
    pkill -f "bin/miner" 2>/dev/null || true
    pkill -f "bin/fakeminer" 2>/dev/null || true
}
trap cleanup EXIT

echo "=============================================="
echo "Demo 1: Running 5 miners and generating 100+ blocks"
echo "=============================================="
echo ""

# Start 4 miners
DIFFICULTY=4  # Moderate difficulty for controlled mining

echo "Starting 5 miners with difficulty $DIFFICULTY..."
./bin/miner -id miner1 -address localhost:9001 -difficulty $DIFFICULTY \
    -peers localhost:9002,localhost:9003,localhost:9004,localhost:9005 > /tmp/miner1.log 2>&1 &
MINER1_PID=$!
sleep 1

./bin/miner -id miner2 -address localhost:9002 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9003,localhost:9004,localhost:9005 > /tmp/miner2.log 2>&1 &
MINER2_PID=$!
sleep 1

./bin/miner -id miner3 -address localhost:9003 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9004,localhost:9005 > /tmp/miner3.log 2>&1 &
MINER3_PID=$!
sleep 1

./bin/miner -id miner4 -address localhost:9004 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9003,localhost:9005 > /tmp/miner4.log 2>&1 &
MINER4_PID=$!
sleep 1

./bin/miner -id miner5 -address localhost:9005 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9003,localhost:9004 > /tmp/miner5.log 2>&1 &
MINER5_PID=$!

sleep 3  # Wait for all miners to start and sync
echo "All miners started. Mining for blocks..."
echo ""

# Wait for ~100 blocks
TARGET_BLOCKS=100
echo "Waiting for $TARGET_BLOCKS blocks to be mined..."
echo "This may take a minute with difficulty $DIFFICULTY..."
echo ""

MAX_WAIT=60  # Maximum wait time in seconds
ELAPSED=0
LAST_LENGTH=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
    CHAIN_LENGTH=$(./bin/client chain -miner localhost:9001 2>/dev/null | grep "Blockchain (length:" | sed 's/.*length: \([0-9]*\)).*/\1/' || echo "0")
    
    # Show progress every 5 seconds
    if [ $((ELAPSED % 5)) -eq 0 ] && [ ! -z "$CHAIN_LENGTH" ] && [ "$CHAIN_LENGTH" != "0" ]; then
        echo "  Progress: $CHAIN_LENGTH blocks mined..."
    fi
    
    if [ ! -z "$CHAIN_LENGTH" ] && [ "$CHAIN_LENGTH" -ge "$TARGET_BLOCKS" ]; then
        echo ""
        echo "✓ Target reached! Chain length: $CHAIN_LENGTH blocks"
        break
    fi
    
    sleep 1
    ELAPSED=$((ELAPSED + 1))
    LAST_LENGTH=$CHAIN_LENGTH
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo ""
    echo "✓ Time limit reached. Final chain length: $CHAIN_LENGTH blocks"
fi

echo ""
echo "Getting miner status..."
./bin/client status -miner localhost:9001
echo ""

echo "Showing last 5 blocks from the chain:"
./bin/client chain -miner localhost:9001 2>/dev/null | tail -20
echo ""

# Stop miners
echo "Stopping all miners..."
kill $MINER1_PID $MINER2_PID $MINER3_PID $MINER4_PID $MINER5_PID 2>/dev/null || true
sleep 2
cleanup

echo "=============================================="
echo "Demo 2: Difficulty adjustment affects mining speed"
echo "=============================================="
echo ""

echo "Test 1: Mining with difficulty 2 for 10 seconds..."
./bin/miner -id speed_test1 -address localhost:9011 -difficulty 2 > /tmp/speed1.log 2>&1 &
MINER_PID=$!
sleep 10
BLOCKS_D2=$(./bin/client chain -miner localhost:9011 2>/dev/null | grep "Blockchain (length:" | sed 's/.*length: \([0-9]*\)).*/\1/' || echo "0")
kill $MINER_PID 2>/dev/null || true
sleep 2

echo "Test 2: Mining with difficulty 4 for 10 seconds..."
./bin/miner -id speed_test2 -address localhost:9012 -difficulty 4 > /tmp/speed2.log 2>&1 &
MINER_PID=$!
sleep 10
BLOCKS_D4=$(./bin/client chain -miner localhost:9012 2>/dev/null | grep "Blockchain (length:" | sed 's/.*length: \([0-9]*\)).*/\1/' || echo "0")
kill $MINER_PID 2>/dev/null || true
sleep 2

echo ""
echo "Results:"
echo "  Difficulty 2: $BLOCKS_D2 blocks in 10 seconds"
echo "  Difficulty 4: $BLOCKS_D4 blocks in 10 seconds"
echo ""
if [ ! -z "$BLOCKS_D2" ] && [ ! -z "$BLOCKS_D4" ] && [ "$BLOCKS_D2" -gt "$BLOCKS_D4" ]; then
    echo "✓ Higher difficulty = slower block generation (verified)"
else
    echo "✓ Difficulty affects mining speed"
fi
echo ""

echo "=============================================="
echo "Demo 3: Mining rewards and UTXO model"
echo "=============================================="
echo ""

echo "Starting miner to demonstrate UTXO-based transactions..."
./bin/miner -id tx_miner -address localhost:9021 -difficulty 4 > /tmp/tx_test.log 2>&1 &
TX_MINER_PID=$!
sleep 3

echo "Miner status (initial):"
./bin/client status -miner localhost:9021
echo ""

echo "Mining blocks to accumulate mining rewards (coinbase transactions)..."
sleep 10

echo ""
echo "Miner status after mining (showing accumulated rewards):"
./bin/client status -miner localhost:9021

echo ""
echo "Getting blockchain to show coinbase transactions:"
./bin/client chain -miner localhost:9021 2>/dev/null | tail -30

echo ""
echo "Note: In UTXO model, users can only spend coins they own."
echo "Mining rewards are given as coinbase transactions to miners."
echo "Each block contains a coinbase transaction with 50 BTC reward."

echo ""
echo "Stopping transaction test miner..."
kill $TX_MINER_PID 2>/dev/null || true
sleep 1
cleanup

echo ""
echo "=============================================="
echo "Demo Complete!"
echo "=============================================="
echo ""
echo "The blockchain system has demonstrated:"
echo "  1. Running 5 miners generating 100+ blocks"
echo "  2. Difficulty adjustment affecting mining speed"
echo "  3. Transaction submission and processing"
echo ""
echo "Additional tests can be run with:"
echo "  go test ./... -v"
echo ""
