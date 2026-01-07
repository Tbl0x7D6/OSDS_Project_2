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

# Start 5 miners
DIFFICULTY=4  # Higher difficulty for better observation

echo "Starting 5 miners with difficulty $DIFFICULTY..."
./bin/miner -id miner1 -address localhost:9001 -difficulty $DIFFICULTY \
    -peers localhost:9002,localhost:9003,localhost:9004,localhost:9005 &
sleep 1

./bin/miner -id miner2 -address localhost:9002 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9003,localhost:9004,localhost:9005 &
sleep 1

./bin/miner -id miner3 -address localhost:9003 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9004,localhost:9005 &
sleep 1

./bin/miner -id miner4 -address localhost:9004 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9003,localhost:9005 &
sleep 1

./bin/miner -id miner5 -address localhost:9005 -difficulty $DIFFICULTY \
    -peers localhost:9001,localhost:9002,localhost:9003,localhost:9004 &

sleep 2  # Wait for all miners to start and sync
echo "All miners started. Mining for blocks..."
echo ""

# Wait for ~100 blocks
TARGET_BLOCKS=100
echo "Waiting for $TARGET_BLOCKS blocks to be mined..."
echo "This may take a minute with difficulty $DIFFICULTY..."
echo ""

while true; do
    CHAIN_LENGTH=$(./bin/client chain -miner localhost:9001 2>/dev/null | grep "length" | awk '{print $NF}' | tr -d ')' || echo "0")
    if [ "$CHAIN_LENGTH" -ge "$TARGET_BLOCKS" ] 2>/dev/null; then
        echo "Target reached! Chain length: $CHAIN_LENGTH"
        break
    fi
    sleep 2
done

echo ""
echo "Getting miner status..."
./bin/client status -miner localhost:9001
echo ""

# Stop miners
cleanup

echo "=============================================="
echo "Demo 2: Difficulty adjustment affects mining speed"
echo "=============================================="
echo ""

echo "Testing with difficulty 1..."
./bin/miner -id speed_test1 -address localhost:9011 -difficulty 1 &
MINER_PID=$!
sleep 5
BLOCKS_D1=$(./bin/client chain -miner localhost:9011 2>/dev/null | grep "length" | awk '{print $NF}' | tr -d ')' || echo "0")
kill $MINER_PID 2>/dev/null || true
sleep 1

echo "Testing with difficulty 3..."
./bin/miner -id speed_test2 -address localhost:9012 -difficulty 3 &
MINER_PID=$!
sleep 5
BLOCKS_D3=$(./bin/client chain -miner localhost:9012 2>/dev/null | grep "length" | awk '{print $NF}' | tr -d ')' || echo "0")
kill $MINER_PID 2>/dev/null || true
sleep 1

echo ""
echo "Results:"
echo "  Difficulty 1: $BLOCKS_D1 blocks in 5 seconds"
echo "  Difficulty 3: $BLOCKS_D3 blocks in 5 seconds"
echo "Higher difficulty = slower block generation"
echo ""

echo "=============================================="
echo "Demo 3: Submit transactions via client"
echo "=============================================="
echo ""

./bin/miner -id tx_test -address localhost:9021 -difficulty 2 &
sleep 2

echo "Submitting transactions..."
./bin/client submit -miner localhost:9021 -from alice -to bob -amount 10.5
./bin/client submit -miner localhost:9021 -from bob -to charlie -amount 5.25
./bin/client submit -miner localhost:9021 -from charlie -to alice -amount 2.0

echo ""
echo "Getting miner status after transactions..."
./bin/client status -miner localhost:9021
sleep 3
echo ""
echo "After mining a few blocks..."
./bin/client status -miner localhost:9021

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
