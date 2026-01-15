#!/bin/bash
# Test script for parallel mining functionality
# This script tests the -threads flag with different thread counts
# Reads miner IPs from minerip.txt

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_MINER="${SCRIPT_DIR}/bin/miner"
BIN_CLIENT="${SCRIPT_DIR}/bin/client"
LOG_DIR="${SCRIPT_DIR}/logs/test_parallel_mining"
MINER_IP_FILE="${SCRIPT_DIR}/minerip.txt"
REMOTE_DIR="/osds_project2"
DIFFICULTY=20
BASE_PORT=19000
TEST_DURATION=60

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -o ConnectTimeout=5"
SCP_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -o ConnectTimeout=5"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Read miner IPs from file
read_miner_ips() {
    if [[ ! -f "$MINER_IP_FILE" ]]; then
        echo_error "Miner IP file not found: $MINER_IP_FILE"
        echo_info "Creating default minerip.txt with localhost entries..."
        echo -e "127.0.0.1\n127.0.0.2\n127.0.0.3\n127.0.0.4" > "$MINER_IP_FILE"
    fi
    
    # Read IPs into array (skip empty lines and comments)
    MINER_IPS=()
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines and comments
        line=$(echo "$line" | xargs)  # Trim whitespace
        if [[ -n "$line" && ! "$line" =~ ^# ]]; then
            MINER_IPS+=("$line")
        fi
    done < "$MINER_IP_FILE"
    
    echo_info "Loaded ${#MINER_IPS[@]} miner IP(s) from $MINER_IP_FILE"
    for i in "${!MINER_IPS[@]}"; do
        echo_info "  Miner $((i+1)): ${MINER_IPS[$i]}"
    done
}

# Cleanup function
cleanup() {
    echo_info "Cleaning up..."
    # Stop remote miners
    if [[ ${#MINER_IPS[@]} -gt 0 ]]; then
        for ip in "${MINER_IPS[@]}"; do
            ssh -n $SSH_OPTS root@"$ip" "pkill -f ${REMOTE_DIR}/miner_parallel || pkill -f 'miner.*test_parallel' || true" >/dev/null 2>&1 || true
        done
    fi
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

# Read miner IPs
read_miner_ips

# Function to get miner IP for a given index (cycles through available IPs)
get_miner_ip() {
    local index=$1
    local ip_count=${#MINER_IPS[@]}
    if [[ $ip_count -eq 0 ]]; then
        echo "127.0.0.1"  # Fallback
    else
        local ip_index=$((index % ip_count))
        echo "${MINER_IPS[$ip_index]}"
    fi
}

# Function to run a mining test with specific thread count
run_mining_test() {
    local threads=$1
    local test_index=$2
    local miner_ip=$(get_miner_ip $test_index)
    local port=$((BASE_PORT + threads))
    local miner_id="test_parallel_${threads}t"
    local log_file="${LOG_DIR}/miner_${threads}threads.log"
    local remote_log="${REMOTE_DIR}/miner_parallel_${threads}t.log"
    local full_address="${miner_ip}:${port}"
    
    echo_info "Testing with $threads thread(s) on $full_address..."
    
    # Create remote directory
    ssh -n $SSH_OPTS root@"$miner_ip" "mkdir -p $REMOTE_DIR" >/dev/null 2>&1 || {
        echo_error "Failed to create remote directory on $miner_ip"
        return 1
    }
    
    # Stop any existing miner
    ssh -n $SSH_OPTS root@"$miner_ip" "pkill -f ${REMOTE_DIR}/miner_parallel || pkill -f 'miner.*test_parallel' || true; rm -f ${REMOTE_DIR}/miner_parallel" >/dev/null 2>&1 || true
    
    # Copy miner binary
    scp $SCP_OPTS "$BIN_MINER" root@"$miner_ip":"${REMOTE_DIR}/miner_parallel.new" >/dev/null 2>&1 < /dev/null || {
        echo_error "Failed to copy binary to $miner_ip"
        return 1
    }
    
    # Install binary
    ssh -n $SSH_OPTS root@"$miner_ip" "mv -f ${REMOTE_DIR}/miner_parallel.new ${REMOTE_DIR}/miner_parallel && chmod +x ${REMOTE_DIR}/miner_parallel" >/dev/null 2>&1 || {
        echo_error "Failed to install binary on $miner_ip"
        return 1
    }
    
    # Start miner remotely
    ssh -n $SSH_OPTS root@"$miner_ip" "nohup ${REMOTE_DIR}/miner_parallel -id '$miner_id' -address '0.0.0.0:${port}' -difficulty $DIFFICULTY -threads $threads -mine=true > $remote_log 2>&1 &" >/dev/null 2>&1 || {
        echo_error "Failed to start miner on $miner_ip"
        return 1
    }
    
    # Wait for miner to start
    sleep 3
    
    echo_info "Miner started on $full_address, mining for ${TEST_DURATION}s..."
    
    # Check if miner is actually running on remote
    local miner_running=$(ssh -n $SSH_OPTS root@"$miner_ip" "pgrep -f ${REMOTE_DIR}/miner_parallel | wc -l" 2>/dev/null || echo "0")
    echo_info "DEBUG: Miner processes running on $miner_ip: $miner_running"
    
    # Let it mine for the test duration
    sleep $TEST_DURATION
    
    # Get chain length with debug info
    echo_info "DEBUG: Querying blockchain status from $full_address..."
    local blockchain_response=$("$BIN_CLIENT" blockchain -miner "$full_address" 2>&1)
    local client_exit_code=$?
    
    echo_info "DEBUG: Client exit code: $client_exit_code"
    echo_info "DEBUG: Raw response: $blockchain_response"
    
    local chain_length=0
    if [ $client_exit_code -eq 0 ]; then
        if command -v jq &> /dev/null; then
            chain_length=$(echo "$blockchain_response" | jq -r '.chain_length // 0' 2>/dev/null || echo "0")
        else
            # Handle optional whitespace after colon in JSON
            chain_length=$(echo "$blockchain_response" | grep -o '"chain_length"[[:space:]]*:[[:space:]]*[0-9]\+' | grep -o '[0-9]\+' || echo "0")
        fi
    else
        echo_error "Failed to query blockchain from $full_address"
    fi
    
    echo_info "DEBUG: Parsed chain_length: $chain_length"
    
    # Download log for inspection
    scp $SCP_OPTS root@"$miner_ip":"$remote_log" "$log_file" >/dev/null 2>&1 || {
        echo_warn "Could not download log from $miner_ip"
    }
    
    if [ -f "$log_file" ]; then
        echo_info "DEBUG: Last 10 lines of miner log:"
        tail -10 "$log_file" | while IFS= read -r line; do
            echo_info "  $line"
        done
    fi
    
    # Stop the miner
    ssh -n $SSH_OPTS root@"$miner_ip" "pkill -f ${REMOTE_DIR}/miner_parallel || true" >/dev/null 2>&1 || true
    sleep 1
    
    echo_info "Threads: $threads, Address: $full_address, Blocks mined: $chain_length"
    echo "$threads,$full_address,$chain_length" >> "${LOG_DIR}/results.csv"
    
    return 0
}

# Main test execution
echo_info "========================================="
echo_info "Parallel Mining Test"
echo_info "========================================="
echo_info "Difficulty: $DIFFICULTY"
echo_info "Test Duration: ${TEST_DURATION}s per test"
echo_info "Miner IP File: $MINER_IP_FILE"
echo ""

# Initialize results file
echo "threads,address,blocks_mined" > "${LOG_DIR}/results.csv"

# Test with different thread counts
test_index=0
for threads in 1 2 4 8; do
    run_mining_test $threads $test_index
    test_index=$((test_index + 1))
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

while IFS=',' read -r threads address blocks; do
    if [[ "$threads" == "threads" ]]; then
        continue  # Skip header
    fi
    
    if [[ "$blocks" -lt 1 ]]; then
        echo_error "Test failed: $threads threads on $address mined 0 blocks"
        all_passed=false
    else
        echo_info "PASS: $threads threads on $address mined $blocks blocks"
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
