#!/bin/bash
# Test script for parallel mining functionality
# This script tests the -threads flag with different thread counts

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_MINER="${SCRIPT_DIR}/bin/miner"
BIN_CLIENT="${SCRIPT_DIR}/bin/client"
LOG_DIR="${SCRIPT_DIR}/logs/test_parallel_mining"
DIFFICULTY=12
BASE_PORT=19000
TEST_DURATION=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Cleanup function
cleanup() {
    echo_info "Cleaning up..."
    pkill -f "miner.*-id test_parallel" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Build binaries
echo_info "Building binaries..."
cd "$SCRIPT_DIR"
go build -o bin/miner ./cmd/miner
go build -o bin/client ./cmd/client
echo_info "Build complete"

# Create log directory
mkdir -p "$LOG_DIR"

# Function to run a mining test with specific thread count
run_mining_test() {
    local threads=$1
    local port=$((BASE_PORT + threads))
    local miner_id="test_parallel_${threads}t"
    local log_file="${LOG_DIR}/miner_${threads}threads.log"
    
    echo_info "Testing with $threads thread(s) on port $port..."
    
    # Start miner
    "$BIN_MINER" \
        -id "$miner_id" \
        -address "localhost:$port" \
        -difficulty $DIFFICULTY \
        -threads $threads \
        -mine=true \
        > "$log_file" 2>&1 &
    
    local miner_pid=$!
    
    # Wait for miner to start
    sleep 2
    
    # Check if miner is running
    if ! kill -0 $miner_pid 2>/dev/null; then
        echo_error "Miner with $threads threads failed to start"
        cat "$log_file"
        return 1
    fi
    
    echo_info "Miner started with PID $miner_pid, mining for ${TEST_DURATION}s..."
    
    # Let it mine for the test duration
    sleep $TEST_DURATION
    
    # Get chain length
    local chain_length=0
    if command -v jq &> /dev/null; then
        chain_length=$("$BIN_CLIENT" blockchain -miner "localhost:$port" 2>/dev/null | jq -r '.chain_length // 0' || echo "0")
    else
        chain_length=$("$BIN_CLIENT" blockchain -miner "localhost:$port" 2>/dev/null | grep -o '"chain_length":[0-9]*' | cut -d: -f2 || echo "0")
    fi
    
    # Stop the miner
    kill $miner_pid 2>/dev/null || true
    wait $miner_pid 2>/dev/null || true
    
    echo_info "Threads: $threads, Blocks mined: $chain_length"
    echo "$threads,$chain_length" >> "${LOG_DIR}/results.csv"
    
    return 0
}

# Main test execution
echo_info "========================================="
echo_info "Parallel Mining Test"
echo_info "========================================="
echo_info "Difficulty: $DIFFICULTY"
echo_info "Test Duration: ${TEST_DURATION}s per test"
echo ""

# Initialize results file
echo "threads,blocks_mined" > "${LOG_DIR}/results.csv"

# Test with different thread counts
for threads in 1 2 4 8; do
    run_mining_test $threads
    sleep 2  # Brief pause between tests
done

echo ""
echo_info "========================================="
echo_info "Test Results Summary"
echo_info "========================================="
echo ""
cat "${LOG_DIR}/results.csv"
echo ""

# Validate results
echo_info "Validating results..."
all_passed=true

while IFS=',' read -r threads blocks; do
    if [[ "$threads" == "threads" ]]; then
        continue  # Skip header
    fi
    
    if [[ "$blocks" -lt 1 ]]; then
        echo_error "Test failed: $threads threads mined 0 blocks"
        all_passed=false
    else
        echo_info "PASS: $threads threads mined $blocks blocks"
    fi
done < "${LOG_DIR}/results.csv"

echo ""
if $all_passed; then
    echo_info "========================================="
    echo_info "All parallel mining tests PASSED!"
    echo_info "========================================="
    exit 0
else
    echo_error "========================================="
    echo_error "Some parallel mining tests FAILED!"
    echo_error "========================================="
    exit 1
fi
