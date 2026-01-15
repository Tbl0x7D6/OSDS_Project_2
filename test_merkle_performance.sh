#!/bin/bash
# Test script to compare block generation speed with and without Merkle tree
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
BIN_MINER="$ROOT_DIR/bin/miner"
BIN_CLIENT="$ROOT_DIR/bin/client"
LOG_DIR="$ROOT_DIR/logs/test_merkle_performance"
MINER_IP_FILE="$ROOT_DIR/minerip.txt"

mkdir -p "$LOG_DIR"

NUM_CLIENTS=10
NUM_MINERS=5
DIFFICULTY=18
TEST_DURATION=60
TX_RATE=100
MIN_CHAIN_LENGTH=5

PIDS=()
declare -a CLIENT_ADDRESSES
declare -a CLIENT_PRIVKEYS
declare -a MINER_PUBKEYS
declare -a MINER_PRIVKEYS
declare -a MINER_ADDRESSES

cleanup() {
    echo "Cleaning up..."
    [ ${#PIDS[@]} -gt 0 ] && kill "${PIDS[@]}" 2>/dev/null || true
    pkill -f "$BIN_MINER" 2>/dev/null || true
    jobs -p | xargs -r kill 2>/dev/null || true
    sleep 2
}
trap cleanup EXIT

log_section() {
    echo ""
    echo "=============================================="
    echo "$1"
    echo "=============================================="
}

json_get() {
    echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$2',''))" 2>/dev/null || echo ""
}

# Read miner IPs from external file
read_miner_ips() {
    if [[ ! -f "$MINER_IP_FILE" ]]; then
        echo "  Warning: Miner IP file not found: $MINER_IP_FILE"
        echo "  Using default localhost entries..."
        EXTERNAL_MINER_IPS=("127.0.0.1" "127.0.0.1" "127.0.0.1" "127.0.0.1" "127.0.0.1")
        return
    fi
    
    EXTERNAL_MINER_IPS=()
    while IFS= read -r line || [[ -n "$line" ]]; do
        line=$(echo "$line" | xargs)
        if [[ -n "$line" && ! "$line" =~ ^# ]]; then
            EXTERNAL_MINER_IPS+=("$line")
        fi
    done < "$MINER_IP_FILE"
    
    echo "  Loaded ${#EXTERNAL_MINER_IPS[@]} miner IP(s) from $MINER_IP_FILE"
    for i in "${!EXTERNAL_MINER_IPS[@]}"; do
        echo "    Miner $((i+1)) IP: ${EXTERNAL_MINER_IPS[$i]}"
    done
}

# Get miner IP for a given index (cycles through available IPs if needed)
get_miner_ip() {
    local index=$1
    local ip_count=${#EXTERNAL_MINER_IPS[@]}
    if [[ $ip_count -eq 0 ]]; then
        echo "127.0.0.1"
    else
        local ip_index=$((index % ip_count))
        echo "${EXTERNAL_MINER_IPS[$ip_index]}"
    fi
}


get_first_utxo() {
    local json_input="$1"
    echo "$json_input" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    utxos = d.get('utxos', [])
    if utxos:
        u = utxos[0]
        print(u['txid'], u['out_index'], u['value'])
    else:
        print('', '', '')
except:
    print('', '', '')
" 2>/dev/null
}

get_chain_length() {
    local payload
    payload=$($BIN_CLIENT blockchain -miner "$1" 2>/dev/null || echo '{"chain_length":0}')
    json_get "$payload" "chain_length"
}

wait_for_chain_length() {
    local miner=$1 target=$2 timeout=$3 elapsed=0
    echo "  Waiting for chain length >= $target..."
    while [ $elapsed -lt $timeout ]; do
        local len=$(get_chain_length "$miner")
        if [ -n "$len" ] && [ "$len" -ge "$target" ] 2>/dev/null; then
            echo "    Chain length reached: $len"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    echo "    Timeout waiting for chain length"
    return 1
}

generate_wallets() {
    echo "  Generating $NUM_CLIENTS client wallets..."
    CLIENT_ADDRESSES=()
    CLIENT_PRIVKEYS=()
    for i in $(seq 1 $NUM_CLIENTS); do
        local wallet_json=$($BIN_CLIENT wallet 2>/dev/null)
        CLIENT_ADDRESSES+=("$(json_get "$wallet_json" "address")")
        CLIENT_PRIVKEYS+=("$(json_get "$wallet_json" "private_key")")
        echo "    Client $i: ${CLIENT_ADDRESSES[-1]:0:16}..."
    done
}

generate_miner_keys() {
    echo "  Generating $NUM_MINERS miner keypairs..."
    MINER_PUBKEYS=()
    MINER_PRIVKEYS=()
    for i in $(seq 1 $NUM_MINERS); do
        local wallet_json=$($BIN_CLIENT wallet 2>/dev/null)
        MINER_PUBKEYS+=("$(json_get "$wallet_json" "address")")
        MINER_PRIVKEYS+=("$(json_get "$wallet_json" "private_key")")
        echo "    Miner $i pubkey: ${MINER_PUBKEYS[-1]:0:16}..."
    done
}

start_miners() {
    local merkle_flag=$1 base_port=$2 prefix=$3
    MINER_ADDRESSES=()
    
    # Build miner addresses using external IPs
    for i in $(seq 1 $NUM_MINERS); do
        local miner_ip=$(get_miner_ip $((i - 1)))
        local port=$((base_port + i - 1))
        MINER_ADDRESSES+=("${miner_ip}:${port}")
    done
    
    for i in $(seq 1 $NUM_MINERS); do
        local miner_ip=$(get_miner_ip $((i - 1)))
        local port=$((base_port + i - 1))
        local addr="${miner_ip}:${port}"
        local id="${MINER_PUBKEYS[$((i-1))]}"
        local peers=""
        
        # Build peer list using external IPs
        for j in $(seq 1 $NUM_MINERS); do
            if [ $j -ne $i ]; then
                local peer_ip=$(get_miner_ip $((j - 1)))
                local peer_port=$((base_port + j - 1))
                [ -n "$peers" ] && peers="$peers," || true
                peers="${peers}${peer_ip}:${peer_port}"
            fi
        done
        
        echo "  Starting ${prefix}miner$i at $addr (merkle=$merkle_flag)"
        "$BIN_MINER" -id "$id" -address "$addr" -difficulty $DIFFICULTY -mine=true -merkle=$merkle_flag -peers "$peers" > "$LOG_DIR/${prefix}miner$i.log" 2>&1 &
        PIDS+=($!)
        sleep 0.3
    done
}

fund_client() {
    local miner_addr=$1 miner_pub=$2 miner_priv=$3 client_addr=$4 amount=$5
    local balance_json=$($BIN_CLIENT balance -miner "$miner_addr" -address "$miner_pub" 2>/dev/null || echo '{}')
    local utxo_info=$(get_first_utxo "$balance_json")
    read utxo_txid utxo_index utxo_value <<< "$utxo_info"
    
    if [ -n "$utxo_txid" ] && [ -n "$utxo_value" ] && [ "$utxo_value" -gt $((amount + 1)) ] 2>/dev/null; then
        local change=$((utxo_value - amount - 1))
        local result=$($BIN_CLIENT transfer -miner "$miner_addr" -from "$miner_pub" -privkey "$miner_priv" -inputs "${utxo_txid}:${utxo_index}" -outputs "${client_addr}:${amount},${miner_pub}:${change}" 2>/dev/null || echo '{}')
        local success=$(json_get "$result" "success")
        [ "$success" = "True" ] || [ "$success" = "true" ] && return 0
    fi
    return 1
}

fund_all_clients() {
    echo "  Funding clients from miner coinbase rewards..."
    local funded=0 amount=100000000
    
    for idx in $(seq 0 $((NUM_CLIENTS - 1))); do
        local client_addr="${CLIENT_ADDRESSES[$idx]}"
        for mi in $(seq 0 $((NUM_MINERS - 1))); do
            if fund_client "${MINER_ADDRESSES[$mi]}" "${MINER_PUBKEYS[$mi]}" "${MINER_PRIVKEYS[$mi]}" "$client_addr" "$amount"; then
                funded=$((funded + 1))
                break
            fi
        done
    done
    echo "    Funded $funded clients"
    sleep 2
}

send_transactions() {
    local miner=$1 duration=$2 rate=$3
    local end_time=$((SECONDS + duration)) tx_count=0 success_count=0
    local interval=$(python3 -c "print(1.0 / $rate)")
    
    echo "  Starting transaction flood: $rate tx/s for ${duration}s..."
    
    while [ $SECONDS -lt $end_time ]; do
        local sender_idx=$((RANDOM % NUM_CLIENTS))
        local receiver_idx=$(((sender_idx + 1 + RANDOM % (NUM_CLIENTS - 1)) % NUM_CLIENTS))
        local sender_addr="${CLIENT_ADDRESSES[$sender_idx]}"
        local sender_privkey="${CLIENT_PRIVKEYS[$sender_idx]}"
        local receiver_addr="${CLIENT_ADDRESSES[$receiver_idx]}"
        
        local balance_json=$($BIN_CLIENT balance -miner "$miner" -address "$sender_addr" 2>/dev/null || echo '{}')
        local utxo_info=$(get_first_utxo "$balance_json")
        read utxo_txid utxo_index utxo_value <<< "$utxo_info"
        
        if [ -n "$utxo_txid" ] && [ -n "$utxo_value" ] && [ "$utxo_value" -gt 2 ] 2>/dev/null; then
            local send_amount=1 change=$((utxo_value - 2))
            if [ $change -gt 0 ]; then
                local result=$($BIN_CLIENT transfer -miner "$miner" -from "$sender_addr" -privkey "$sender_privkey" -inputs "${utxo_txid}:${utxo_index}" -outputs "${receiver_addr}:${send_amount},${sender_addr}:${change}" 2>/dev/null || echo '{}')
                local success=$(json_get "$result" "success")
                [ "$success" = "True" ] || [ "$success" = "true" ] && success_count=$((success_count + 1))
            fi
        fi
        
        tx_count=$((tx_count + 1))
        sleep $interval
    done
    
    echo "    Attempted: $tx_count, Successful: $success_count"
}

run_test() {
    local merkle_flag=$1 test_name=$2 base_port=$3
    
    log_section "Running test: $test_name (merkle=$merkle_flag)"
    PIDS=()
    
    start_miners "$merkle_flag" $base_port "$test_name"
    local primary_miner="${MINER_ADDRESSES[0]}"
    sleep 3
    
    echo "  Waiting for initial chain to build..."
    wait_for_chain_length "$primary_miner" $MIN_CHAIN_LENGTH 120 || { echo "  Failed"; return 1; }
    
    fund_all_clients
    
    local start_length=$(get_chain_length "$primary_miner")
    local start_time=$SECONDS
    
    echo "  Starting chain length: $start_length"
    echo "  Running test for ${TEST_DURATION}s with transaction load..."
    
    send_transactions "$primary_miner" $TEST_DURATION $TX_RATE &
    local tx_pid=$!
    
    sleep $TEST_DURATION
    wait $tx_pid 2>/dev/null || true
    
    local end_length=$(get_chain_length "$primary_miner")
    local elapsed=$((SECONDS - start_time))
    local blocks_mined=$((end_length - start_length))
    local blocks_per_second=$(echo "scale=4; $blocks_mined / $elapsed" | bc)
    local avg_block_time=$(echo "scale=4; $elapsed / $blocks_mined" | bc 2>/dev/null || echo "N/A")
    
    local multi_tx_blocks=0
    for i in $(seq 1 $NUM_MINERS); do
        local count=$(grep -E -c "with [2-9] transactions|with [1-9][0-9] transactions" "$LOG_DIR/${test_name}miner${i}.log" 2>/dev/null | tr -d "\n" || echo "0")
        multi_tx_blocks=$((multi_tx_blocks + count))
    done
    
    echo ""
    echo "  Results for $test_name:"
    echo "    Blocks mined: $blocks_mined"
    echo "    Blocks with user transactions: $multi_tx_blocks"
    echo "    Elapsed time: ${elapsed}s"
    echo "    Blocks per second: $blocks_per_second"
    echo "    Average block time: ${avg_block_time}s"
    
    echo "  Stopping miners..."
    [ ${#PIDS[@]} -gt 0 ] && kill "${PIDS[@]}" 2>/dev/null || true
    sleep 2
    pkill -f "$BIN_MINER" 2>/dev/null || true
    sleep 1
    
    echo "$blocks_mined" > "$LOG_DIR/${test_name}_blocks.txt"
    echo "$blocks_per_second" > "$LOG_DIR/${test_name}_rate.txt"
    echo "$multi_tx_blocks" > "$LOG_DIR/${test_name}_multi_tx.txt"
}

log_section "Merkle Tree Performance Test"
echo "Configuration: $NUM_CLIENTS clients, $NUM_MINERS miners, difficulty $DIFFICULTY, ${TEST_DURATION}s duration, $TX_RATE tx/s"

log_section "Building binaries"
cd "$ROOT_DIR"
go build -o "$BIN_MINER" ./cmd/miner
go build -o "$BIN_CLIENT" ./cmd/client
echo "Build complete."

log_section "Generating client wallets"
generate_wallets

log_section "Generating miner keypairs"
generate_miner_keys

log_section "Loading miner IPs"
read_miner_ips

run_test "false" "no_merkle" 9900
run_test "true" "with_merkle" 9950

log_section "Performance Comparison"

NO_MERKLE_BLOCKS=$(cat "$LOG_DIR/no_merkle_blocks.txt" 2>/dev/null || echo "0")
WITH_MERKLE_BLOCKS=$(cat "$LOG_DIR/with_merkle_blocks.txt" 2>/dev/null || echo "0")
NO_MERKLE_RATE=$(cat "$LOG_DIR/no_merkle_rate.txt" 2>/dev/null || echo "0")
WITH_MERKLE_RATE=$(cat "$LOG_DIR/with_merkle_rate.txt" 2>/dev/null || echo "0")
NO_MERKLE_MULTI_TX=$(cat "$LOG_DIR/no_merkle_multi_tx.txt" 2>/dev/null || echo "0")
WITH_MERKLE_MULTI_TX=$(cat "$LOG_DIR/with_merkle_multi_tx.txt" 2>/dev/null || echo "0")

echo ""
echo "  WITHOUT Merkle Tree:"
echo "    Blocks mined: $NO_MERKLE_BLOCKS"
echo "    Rate: $NO_MERKLE_RATE blocks/s"
echo "    Blocks with user transactions: $NO_MERKLE_MULTI_TX"
echo ""
echo "  WITH Merkle Tree:"
echo "    Blocks mined: $WITH_MERKLE_BLOCKS"
echo "    Rate: $WITH_MERKLE_RATE blocks/s"
echo "    Blocks with user transactions: $WITH_MERKLE_MULTI_TX"
echo ""

if [ "$NO_MERKLE_BLOCKS" -gt 0 ] && [ "$WITH_MERKLE_BLOCKS" -gt 0 ]; then
    DIFF=$((NO_MERKLE_BLOCKS - WITH_MERKLE_BLOCKS))
    if [ $DIFF -gt 0 ]; then
        PERCENT=$(echo "scale=2; ($DIFF * 100) / $NO_MERKLE_BLOCKS" | bc)
        echo "  Merkle tree mode is ${PERCENT}% slower ($DIFF fewer blocks)"
    elif [ $DIFF -lt 0 ]; then
        DIFF=$((-DIFF))
        PERCENT=$(echo "scale=2; ($DIFF * 100) / $WITH_MERKLE_BLOCKS" | bc)
        echo "  Non-Merkle mode is ${PERCENT}% slower ($DIFF fewer blocks)"
    else
        echo "  Both modes performed equally"
    fi
fi

echo ""
echo "Test logs available at: $LOG_DIR"
log_section "Test Complete"
