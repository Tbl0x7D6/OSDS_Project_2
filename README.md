# Bitcoin-like Blockchain Implementation (Nakamoto Protocol)

A Go implementation of a Bitcoin-like blockchain system featuring Proof of Work consensus, peer-to-peer networking, and transaction handling.

## Features

### Core Components

1. **Block Structure**
   - Index, Timestamp, Transactions list
   - Previous block hash (hash pointer)
   - Current block hash
   - Nonce for PoW
   - Difficulty level
   - Miner ID

2. **Blockchain & UTXO**
   - Genesis block creation
   - UTXO set management
   - Transaction validation (Satoshi units, multiple inputs/outputs)
   - Chain validation
   - Longest chain rule for fork resolution
   - Block verification (hash, PoW, transactions)

3. **Proof of Work (PoW)**
   - Adjustable difficulty (number of leading zeros)
   - Single-threaded and parallel mining
   - Mining cancellation support

4. **Network Layer**
   - RPC-based communication
   - Transaction broadcasting
   - Block broadcasting
   - Chain synchronization

5. **Client/Wallet**
   - Transaction submission
   - Chain querying
   - Miner status checking

## Project Structure

```
blockchain/
├── cmd/
│   ├── client/         # Client CLI application
│   ├── miner/          # Miner node application
│   └── fakeminer/      # Malicious miner for testing
├── pkg/
│   ├── block/          # Block data structure
│   ├── blockchain/     # Blockchain implementation
│   ├── network/        # P2P networking and RPC
│   ├── pow/            # Proof of Work algorithm
│   └── transaction/    # Transaction handling
├── test/               # Integration tests
├── demo.sh             # Demo script
├── go.mod
└── README.md
```

## Building

```bash
go build ./...
```

Build individual binaries:
```bash
go build -o bin/miner ./cmd/miner
go build -o bin/client ./cmd/client
go build -o bin/fakeminer ./cmd/fakeminer
```

## Running

### Start Miners

Start multiple miners (recommended 5 for demo):

```bash
# Miner 1
./bin/miner -id miner1 -address localhost:8001 -difficulty 4 \
    -peers localhost:8002,localhost:8003,localhost:8004,localhost:8005

# Miner 2
./bin/miner -id miner2 -address localhost:8002 -difficulty 4 \
    -peers localhost:8001,localhost:8003,localhost:8004,localhost:8005

# ... repeat for miners 3-5
```

### Use the Client

Submit a transaction (amount in satoshi, 1 BTC = 100,000,000 satoshi):
```bash
./bin/client submit -miner localhost:8001 -from alice -to bob -amount 1000000000
```

Check miner status:
```bash
./bin/client status -miner localhost:8001
```

View the blockchain:
```bash
./bin/client chain -miner localhost:8001
```

### Run the Demo

```bash
chmod +x demo.sh
./demo.sh
```

## Testing

Run all tests:
```bash
go test ./... -v
```

Run specific test suites:
```bash
# Transaction tests
go test ./pkg/transaction/... -v

# Block tests
go test ./pkg/block/... -v

# Blockchain tests
go test ./pkg/blockchain/... -v

# PoW tests
go test ./pkg/pow/... -v

# Network tests
go test ./pkg/network/... -v

# Integration tests
go test ./test/... -v
```

## Demo Requirements Checklist

### 1. Run 5+ miners and generate 100+ blocks ✓
- Multiple miners can be started with different addresses
- Each miner broadcasts blocks to peers
- Integration test `TestFiveMinersGenerateBlocks` demonstrates this

### 2. Difficulty adjustment affects mining speed ✓
- `-difficulty` flag controls the number of leading zeros required
- Higher difficulty = exponentially more hash attempts needed
- Test `TestDifficultyAffectsMiningSpeed` demonstrates this

### 3. Corrupted blocks are rejected ✓
- `HasValidHash()` checks block hash integrity
- `HasValidPoW()` verifies PoW requirement
- `ValidateTransactions()` validates all transactions
- Test `TestIntegration_CorruptedBlockRejection` demonstrates this

### 4. Lying miners (invalid PoW) are rejected ✓
- Blocks without valid PoW are rejected
- `Validate()` function checks both hash and PoW
- Test `TestIntegration_LyingMinerRejection` demonstrates this
- `fakeminer` tool can simulate malicious behavior

### 5. Longest chain rule for forks ✓
- `ReplaceChain()` implements longest chain selection
- Shorter chains are rejected
- Tests `TestLongestChainRule` and `TestIntegration_ForkResolutionLongestChain` demonstrate this

## Performance Metrics

| Metric | Description | How to Measure |
|--------|-------------|----------------|
| Block time | Average time between blocks | Observe mining logs |
| Transactions per block | Number of TXs included | Check block data |
| Network latency | Time for block propagation | Observe sync logs |
| Hash rate | Hashes per second | PoW benchmark tests |

Run benchmarks:
```bash
go test ./pkg/pow/... -bench=. -benchtime=10s
go test ./test/... -bench=. -benchtime=10s
```

## Architecture

### Consensus
- Proof of Work (PoW) with adjustable difficulty
- Longest chain wins in case of forks
- All blocks are validated before acceptance

### Networking
- TCP-based RPC communication
- Gossip protocol for transaction dissemination
- Pull-based chain synchronization

### Transaction Model
- Simple transfer model (from, to, amount)
- Coinbase transactions for mining rewards
- Basic signature verification

## Security Features

1. **Hash Integrity**: Each block contains the hash of the previous block
2. **PoW Verification**: All blocks must meet difficulty requirement
3. **Transaction Validation**: Invalid transactions are rejected
4. **Chain Validation**: Full chain validation on sync

## Limitations & Future Improvements

Current limitations (can be addressed in Part II):
- No persistent storage (in-memory only)
- Simplified transaction/signature model
- No Merkle tree for transactions
- Static peer list
- No dynamic difficulty adjustment

## License

This project is for educational purposes as part of a distributed systems course.
