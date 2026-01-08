#!/bin/bash
# Demo script for the blockchain system

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
BIN_MINER="$ROOT_DIR/bin/miner"
BIN_CLIENT="$ROOT_DIR/bin/client"
BIN_FAKEMINER="$ROOT_DIR/bin/fakeminer"
LOG_DIR="$ROOT_DIR/logs/demo"

mkdir -p "$LOG_DIR"

PIDS=()
declare -a DEMO1_ADDRS SPEED_ADDRS SPEED2_ADDRS CORR_ADDRS POW_ADDRS FORKA_ADDRS FORKB_ADDRS BRIDGE_ADDRS

cleanup() {
    if [ ${#PIDS[@]} -gt 0 ]; then
        kill "${PIDS[@]}" 2>/dev/null || true
    fi
    pkill -f "$BIN_MINER" 2>/dev/null || true
    pkill -f "$BIN_FAKEMINER" 2>/dev/null || true
}
trap cleanup EXIT

log_section() {
    echo "=============================================="
    echo "$1"
    echo "=============================================="
}

comma_join() {
    local IFS=','
    echo "$*"
}

get_chain_value() {
    local miner_input=$1
    # Use the first address if a list is passed (space- or comma-separated)
    local miner=${miner_input%% *}
    miner=${miner%%,*}
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
    echo "  waiting for $target blocks from $miner (timeout ${timeout}s)"
    while [ $elapsed -lt $timeout ]; do
        latest=$(get_chain_value "$miner" "chain_length" 0)
        if [ $((elapsed % 5)) -eq 0 ]; then
            echo "    progress: $latest blocks"
        fi
        if [ "$latest" -ge "$target" ]; then
            echo "    target reached: $latest blocks"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    echo "    timeout after $timeout seconds (length $latest)"
    return 1
}

print_chain_tail() {
    local miner=$1
    local payload length difficulty
    payload=$($BIN_CLIENT blockchain -miner "$miner" -detail 2>/dev/null | tr -d '\n' || true)
    length=$(echo "$payload" | sed -n 's/.*"chain_length"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p')
    difficulty=$(echo "$payload" | sed -n 's/.*"difficulty"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p')
    echo "Chain length: ${length:-0}, difficulty: ${difficulty:-0}"
    local lines
    lines=$(echo "$payload" | tr '{' '\n' | grep -E '"index"|"hash"|"prev_hash"|"miner_id"' 2>/dev/null || true)
    echo "$lines" | tail -n 20
}

start_miner_group() {
    local prefix=$1
    local count=$2
    local base_port=$3
    local difficulty=$4
    local mine_flag=$5
    local out_var=$6
    local extra_peers=${7:-}
    local addresses=()

    for i in $(seq 0 $((count - 1))); do
        addresses+=("localhost:$((base_port + i))")
    done

    for i in "${!addresses[@]}"; do
        local addr=${addresses[$i]}
        local id="${prefix}$((i + 1))"
        local peers=""
        for peer in "${addresses[@]}"; do
            if [ "$peer" != "$addr" ]; then
                peers="${peers:+$peers,}$peer"
            fi
        done
        if [ -n "$extra_peers" ]; then
            peers="${peers:+$peers,}$extra_peers"
        fi
        local log_file="$LOG_DIR/${id}.log"
        echo "  starting $id at $addr (difficulty $difficulty, mine=$mine_flag)"
        local args=(-id "$id" -address "$addr" -difficulty "$difficulty" -mine="$mine_flag")
        if [ -n "$peers" ]; then
            args+=(-peers "$peers")
        fi
        "$BIN_MINER" "${args[@]}" >"$log_file" 2>&1 &
        PIDS+=($!)
        sleep 0.3
    done

    eval "$out_var=(\"${addresses[@]}\")"
}

start_fakeminer_node() {
    local id=$1
    local address=$2
    local difficulty=$3
    local peers=$4
    local behavior=$5
    local log_file="$LOG_DIR/${id}.log"
    echo "  starting malicious $id at $address (type=$behavior)"
    local args=(-id "$id" -address "$address" -difficulty "$difficulty" -type "$behavior")
    if [ -n "$peers" ]; then
        args+=(-peers "$peers")
    fi
    "$BIN_FAKEMINER" "${args[@]}" >"$log_file" 2>&1 &
    PIDS+=($!)
    sleep 0.3
}

show_rejection_logs() {
    local logfile=$1
    local label=$2
    echo "  $label"
    if [ -f "$logfile" ]; then
        grep -E "Rejected block" "$logfile" | tail -n 5 || echo "    (no rejection logs yet)"
    else
        echo "    (log not found)"
    fi
}

show_sync_logs() {
    local logfile=$1
    local label=$2
    echo "  $label"
    if [ -f "$logfile" ]; then
        grep -E "Synchronized chain" "$logfile" | tail -n 5 || echo "    (no sync logs yet)"
    else
        echo "    (log not found)"
    fi
}

stop_started_processes() {
    if [ ${#PIDS[@]} -gt 0 ]; then
        kill "${PIDS[@]}" 2>/dev/null || true
        sleep 1
        PIDS=()
    fi
    pkill -f "$BIN_MINER" 2>/dev/null || true
    pkill -f "$BIN_FAKEMINER" 2>/dev/null || true
}

log_section "Building project binaries"
go build -o "$BIN_MINER" ./cmd/miner
go build -o "$BIN_CLIENT" ./cmd/client
go build -o "$BIN_FAKEMINER" ./cmd/fakeminer
echo "Build complete. Logs at $LOG_DIR"
echo

# Demo 1: Run N miners and build a 100+ block chain
log_section "Demo 1: N miners generate 100+ blocks"
MINER_COUNT=${MINER_COUNT:-5}
DEMO1_DIFFICULTY=${DEMO1_DIFFICULTY:-3}
TARGET_BLOCKS=100
start_miner_group "miner" "$MINER_COUNT" 9101 "$DEMO1_DIFFICULTY" true DEMO1_ADDRS
sleep 2
wait_for_blocks "${DEMO1_ADDRS[0]}" "$TARGET_BLOCKS" 120 || true
print_chain_tail "${DEMO1_ADDRS[0]}"
echo
stop_started_processes

# Demo 2: Higher difficulty slows block production
log_section "Demo 2: Difficulty affects block speed"
SPEED_COUNT=${SPEED_COUNT:-3}
LOW_DIFF=${LOW_DIFF:-2}
HIGH_DIFF=${HIGH_DIFF:-4}
RUN_SECONDS=${RUN_SECONDS:-10}

start_miner_group "spdL" "$SPEED_COUNT" 9201 "$LOW_DIFF" true SPEED_ADDRS
sleep "$RUN_SECONDS"
LEN_LOW=$(get_chain_value "${SPEED_ADDRS[0]}" "chain_length" 0)
BLOCKS_LOW=$((LEN_LOW - 1))
stop_started_processes

start_miner_group "spdH" "$SPEED_COUNT" 9301 "$HIGH_DIFF" true SPEED2_ADDRS
sleep "$RUN_SECONDS"
LEN_HIGH=$(get_chain_value "${SPEED2_ADDRS[0]}" "chain_length" 0)
BLOCKS_HIGH=$((LEN_HIGH - 1))
stop_started_processes

echo "  Blocks in ${RUN_SECONDS}s @ difficulty $LOW_DIFF: $BLOCKS_LOW"
echo "  Blocks in ${RUN_SECONDS}s @ difficulty $HIGH_DIFF: $BLOCKS_HIGH"
echo

# Demo 3: Corrupted blocks (invalid hash) are rejected
log_section "Demo 3: Corrupted block rejection"
MAL_ADDR="localhost:9403"
start_miner_group "goodC" 2 9401 3 false CORR_ADDRS "$MAL_ADDR"
CORR_PEERS=$(comma_join "${CORR_ADDRS[@]}")
start_fakeminer_node "corruptor" "$MAL_ADDR" 1 "$CORR_PEERS" "invalid_hash"
sleep 10
LEN_CORR=$(get_chain_value "${CORR_ADDRS[0]}" "chain_length" 0)
echo "  Honest chain length (should remain at genesis-level): $LEN_CORR"
show_rejection_logs "$LOG_DIR/goodC1.log" "Honest miner rejection log"
stop_started_processes
echo

# Demo 4: Lying miner with invalid PoW is rejected
log_section "Demo 4: Invalid PoW rejection"
MAL_POW_ADDR="localhost:9453"
start_miner_group "goodP" 2 9451 3 false POW_ADDRS "$MAL_POW_ADDR"
POW_PEERS=$(comma_join "${POW_ADDRS[@]}")
start_fakeminer_node "liar" "$MAL_POW_ADDR" 1 "$POW_PEERS" "invalid_pow"
sleep 10
LEN_POW=$(get_chain_value "${POW_ADDRS[0]}" "chain_length" 0)
echo "  Honest chain length (invalid PoW blocks ignored): $LEN_POW"
show_rejection_logs "$LOG_DIR/goodP1.log" "Honest miner rejection log"
stop_started_processes
echo

# Demo 5: Fork resolution via longest chain rule
log_section "Demo 5: Fork resolved by longest chain"
BRIDGE_ADDR="localhost:9620"
start_miner_group "forkA" 2 9601 2 true FORKA_ADDRS "$BRIDGE_ADDR"
start_miner_group "forkB" 2 9651 2 true FORKB_ADDRS "$BRIDGE_ADDR"
sleep 8
LEN_A_PRE=$(get_chain_value "${FORKA_ADDRS[0]}" "chain_length" 0)
LEN_B_PRE=$(get_chain_value "${FORKB_ADDRS[0]}" "chain_length" 0)
echo "  Pre-merge lengths -> group A: $LEN_A_PRE, group B: $LEN_B_PRE"

BRIDGE_PEERS=$(comma_join "${FORKA_ADDRS[@]}" "${FORKB_ADDRS[@]}")
start_miner_group "bridge" 1 9620 2 true BRIDGE_ADDRS "$BRIDGE_PEERS"
sleep 8
LEN_BRIDGE=$(get_chain_value "${BRIDGE_ADDRS[0]}" "chain_length" 0)
LEN_B_POST=$(get_chain_value "${FORKB_ADDRS[0]}" "chain_length" 0)
echo "  After reconnect -> bridge: $LEN_BRIDGE, group B adopts: $LEN_B_POST"
show_sync_logs "$LOG_DIR/forkB1.log" "Fork group B sync log"
print_chain_tail "${FORKB_ADDRS[0]}"

stop_started_processes
echo "Demo complete. Review logs under $LOG_DIR for detailed traces."
