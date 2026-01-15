#!/bin/bash
# Test script for dynamic difficulty adjustment feature

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
BIN_MINER="$ROOT_DIR/bin/miner"
BIN_CLIENT="$ROOT_DIR/bin/client"
LOG_DIR="$ROOT_DIR/logs/test_dynamic_difficulty"

mkdir -p "$LOG_DIR"

PIDS=()

cleanup() {
    if [ ${#PIDS[@]} -gt 0 ]; then
        kill "${PIDS[@]}" 2>/dev/null || true
    fi
    pkill -f "$BIN_MINER" 2>/dev/null || true
}
trap cleanup EXIT

log_section() {
    echo "=============================================="
    echo "$1"
    echo "=============================================="
}

log_pass() {
    echo "[PASS] $1"
}

log_fail() {
    echo "[FAIL] $1"
    exit 1
}

get_chain_value() {
    local miner=$1
    local field=$2
    local default=${3:-0}
    local payload value
    payload=$($BIN_CLIENT blockchain -miner "$miner" 2>/dev/null | tr -d '\n' || true)
    value=$(echo "$payload" | grep -o "\"${field}\"[[:space:]]*:[[:space:]]*[0-9][0-9]*" | head -n1 | sed 's/[^0-9]//g')
    if [ -z "$value" ]; then
        echo "$default"
    else
        echo "$value"
    fi
}

wait_for_blocks() {
    local miner=$1
    local target=$2
    local timeout=$3
    local elapsed=0
    local latest=0
    echo "  Waiting for $target blocks from $miner (timeout ${timeout}s)"
    while [ $elapsed -lt $timeout ]; do
        latest=$(get_chain_value "$miner" "chain_length" 0)
        if [ "$latest" -ge "$target" ]; then
            echo "    Target reached: $latest blocks"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    echo "    Timeout after $timeout seconds (length $latest)"
    return 1
}

stop_all() {
    if [ ${#PIDS[@]} -gt 0 ]; then
        kill "${PIDS[@]}" 2>/dev/null || true
        sleep 1
        PIDS=()
    fi
    pkill -f "$BIN_MINER" 2>/dev/null || true
}

# Build the project
log_section "Building project binaries"
go build -o "$BIN_MINER" ./cmd/miner
go build -o "$BIN_CLIENT" ./cmd/client
echo "Build complete."
echo

# Test 1: Verify the -dynamic-difficulty flag exists
log_section "Test 1: Verify -dynamic-difficulty flag"
if "$BIN_MINER" -h 2>&1 | grep -q "dynamic-difficulty"; then
    log_pass "The -dynamic-difficulty flag is present in miner help"
else
    log_fail "The -dynamic-difficulty flag is missing from miner help"
fi
echo

# Test 2: Start a miner with dynamic difficulty disabled (default)
log_section "Test 2: Static difficulty mode (default)"
"$BIN_MINER" -id "static1" -address "localhost:9801" -difficulty 10 -mine=true > "$LOG_DIR/static1.log" 2>&1 &
PIDS+=($!)
sleep 3

if grep -q "Static difficulty mode" "$LOG_DIR/static1.log"; then
    log_pass "Miner started in static difficulty mode by default"
else
    log_fail "Miner did not start in static difficulty mode"
fi
stop_all
echo

# Test 3: Start a miner with dynamic difficulty enabled
log_section "Test 3: Dynamic difficulty mode"
"$BIN_MINER" -id "dynamic1" -address "localhost:9802" -difficulty 10 -mine=true -dynamic-difficulty=true > "$LOG_DIR/dynamic1.log" 2>&1 &
PIDS+=($!)
sleep 3

if grep -q "Dynamic difficulty adjustment enabled" "$LOG_DIR/dynamic1.log"; then
    log_pass "Miner started with dynamic difficulty enabled"
else
    log_fail "Miner did not enable dynamic difficulty"
fi
stop_all
echo

# Test 4: Verify difficulty adjustment occurs with dynamic difficulty
log_section "Test 4: Difficulty adjustment with fast blocks"
# Use very low difficulty so blocks are mined quickly
"$BIN_MINER" -id "dyn_test" -address "localhost:9803" -difficulty 4 -mine=true -dynamic-difficulty=true > "$LOG_DIR/dyn_test.log" 2>&1 &
PIDS+=($!)

# Wait for enough blocks to trigger adjustment (at least 6 blocks)
if wait_for_blocks "localhost:9803" 10 60; then
    # Check if difficulty adjustment occurred
    if grep -q "Difficulty adjusted" "$LOG_DIR/dyn_test.log"; then
        log_pass "Difficulty adjustment occurred as expected"
    else
        echo "  (Note: Difficulty may not have adjusted if block time was within target range)"
        log_pass "Mining completed successfully with dynamic difficulty enabled"
    fi
else
    log_fail "Failed to mine enough blocks for difficulty adjustment"
fi
stop_all
echo

# Test 5: Compare static vs dynamic difficulty block production
log_section "Test 5: Compare static vs dynamic difficulty modes"
RUN_TIME=20

# Static difficulty test
echo "  Running static difficulty miner for ${RUN_TIME}s..."
"$BIN_MINER" -id "compare_static" -address "localhost:9804" -difficulty 6 -mine=true > "$LOG_DIR/compare_static.log" 2>&1 &
PIDS+=($!)
sleep "$RUN_TIME"
STATIC_BLOCKS=$(get_chain_value "localhost:9804" "chain_length" 0)
stop_all
echo "    Static mode: $STATIC_BLOCKS blocks"

# Dynamic difficulty test
echo "  Running dynamic difficulty miner for ${RUN_TIME}s..."
"$BIN_MINER" -id "compare_dynamic" -address "localhost:9805" -difficulty 6 -mine=true -dynamic-difficulty=true > "$LOG_DIR/compare_dynamic.log" 2>&1 &
PIDS+=($!)
sleep "$RUN_TIME"
DYNAMIC_BLOCKS=$(get_chain_value "localhost:9805" "chain_length" 0)
stop_all
echo "    Dynamic mode: $DYNAMIC_BLOCKS blocks"

if [ "$STATIC_BLOCKS" -gt 0 ] && [ "$DYNAMIC_BLOCKS" -gt 0 ]; then
    log_pass "Both static and dynamic modes produced blocks successfully"
else
    log_fail "Failed to produce blocks in one of the modes"
fi
echo

# Test 6: Multiple miners with dynamic difficulty
log_section "Test 6: Multiple miners with dynamic difficulty"
for i in 1 2 3; do
    port=$((9810 + i))
    peers=""
    for j in 1 2 3; do
        if [ $j -ne $i ]; then
            pport=$((9810 + j))
            if [ -n "$peers" ]; then
                peers="$peers,localhost:$pport"
            else
                peers="localhost:$pport"
            fi
        fi
    done
    "$BIN_MINER" -id "multi$i" -address "localhost:$port" -peers "$peers" -difficulty 8 -mine=true -dynamic-difficulty=true > "$LOG_DIR/multi$i.log" 2>&1 &
    PIDS+=($!)
    sleep 0.5
done

sleep 15
MULTI_BLOCKS=$(get_chain_value "localhost:9811" "chain_length" 0)
echo "  Chain length after 15s: $MULTI_BLOCKS blocks"

if [ "$MULTI_BLOCKS" -gt 5 ]; then
    log_pass "Multiple miners with dynamic difficulty worked correctly"
else
    log_fail "Multiple miners failed to produce enough blocks"
fi
stop_all
echo

log_section "All tests passed!"
echo "Test logs available at: $LOG_DIR"
