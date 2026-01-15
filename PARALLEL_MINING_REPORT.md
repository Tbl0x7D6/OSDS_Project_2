# Parallel Mining Implementation Report

## Overview

This document describes the implementation of parallel mining functionality in the blockchain project. The feature allows miners to use multiple CPU threads to increase mining throughput.

## Changes Made

### 1. Configuration Module (`pkg/config/config.go`)

Added new configuration for mining threads:

```go
// miningThreads controls the number of parallel threads for mining
// Default is 1 (sequential mining, no parallelism)
miningThreads = 1

// MiningThreads returns the number of parallel threads for mining
func MiningThreads() int

// SetMiningThreads sets the number of parallel threads for mining
// If threads <= 0, it defaults to 1 (sequential mining)
func SetMiningThreads(threads int)
```

### 2. Command Line Interface (`cmd/miner/main.go`)

Added the `-threads` flag to the miner command:

```go
threads := flag.Int("threads", 1, "Number of parallel mining threads (default: 1, no parallelism)")
```

**Usage:**
```bash
./bin/miner -id <miner_id> -address <address> -threads <n>
```

**Examples:**
- Single-threaded (default): `./bin/miner -id miner1 -address localhost:8001`
- 4 parallel threads: `./bin/miner -id miner1 -address localhost:8001 -threads 4`
- 8 parallel threads: `./bin/miner -id miner1 -address localhost:8001 -threads 8`

### 3. Network Module (`pkg/network/network.go`)

Modified the `mineBlock()` function to use parallel mining when threads > 1:

```go
go func() {
    // Use parallel mining if threads > 1, otherwise use sequential mining
    threads := config.MiningThreads()
    if threads > 1 {
        result = powInstance.MineParallel(context.TODO(), threads)
    } else {
        // Sequential mining
        result = powInstance.Mine(context.TODO(), callback)
    }
    close(done)
}()
```

### 4. Proof of Work Module (`pkg/pow/pow.go`)

The `MineParallel` function was already implemented with the following features:
- Each worker starts from a random nonce + worker offset to avoid duplication
- Workers explore different nonce spaces to maximize coverage
- Uses atomic operations for thread-safe result coordination
- Uses context cancellation for clean shutdown

```go
func (pow *ProofOfWork) MineParallel(ctx context.Context, workers int) *MiningResult
```

## Test Files

### Go Unit Tests

**`pkg/pow/pow_test.go`** - Added tests:
- `TestMineParallelWithDifferentWorkerCounts`: Tests mining with 1, 2, 4, and 8 workers
- `TestMineParallelConsistency`: Verifies consistent valid blocks across multiple iterations
- `BenchmarkMineParallelWorkers`: Benchmarks performance with different worker counts

**`pkg/config/config_test.go`** - Added tests:
- `TestMiningThreads`: Tests getting/setting thread configuration
- `TestMiningThreadsConcurrency`: Tests thread-safety of configuration access

### Bash Integration Test

**`test_parallel_mining.sh`** - Comprehensive integration test that:
- Starts miners with different thread counts (1, 2, 4, 8)
- Mines for a fixed duration and counts blocks mined
- Validates that all configurations can mine successfully
- Reports performance comparison

## Performance Results

Testing with difficulty 12, 10 seconds per test:

| Threads | Blocks Mined | Improvement |
|---------|--------------|-------------|
| 1       | 857          | baseline    |
| 2       | 1134         | +32%        |
| 4       | 1520         | +77%        |
| 8       | 1534         | +79%        |

**Note:** Performance improvements depend on:
- Number of available CPU cores
- Mining difficulty
- System load
- Random nonce distribution

## How It Works

### Sequential Mining (threads=1)
```
Worker -> Try nonce 0, 1, 2, 3, ... -> Find valid hash
```

### Parallel Mining (threads=4)
```
Worker 0 -> Try nonce 0, 4, 8, 12, ...
Worker 1 -> Try nonce 1, 5, 9, 13, ...  } -> First valid hash wins
Worker 2 -> Try nonce 2, 6, 10, 14, ...
Worker 3 -> Try nonce 3, 7, 11, 15, ...
```

Each worker explores a different subset of the nonce space, effectively multiplying the hash rate by the number of workers (up to CPU core limits).

## Running Tests

### Run Go Unit Tests
```bash
cd OSDS_Project_2
go test -v ./pkg/pow/...
go test -v ./pkg/config/...
```

### Run All Tests
```bash
go test ./...
```

### Run Integration Test
```bash
./test_parallel_mining.sh
```

## Files Modified

1. `pkg/config/config.go` - Added MiningThreads configuration
2. `pkg/config/config_test.go` - New file for config tests
3. `cmd/miner/main.go` - Added -threads flag
4. `pkg/network/network.go` - Integrated parallel mining
5. `pkg/pow/pow_test.go` - Added parallel mining tests
6. `test_parallel_mining.sh` - New integration test script

## Backward Compatibility

- Default behavior (threads=1) matches original sequential mining
- Existing scripts and commands continue to work without modification
- The -threads flag is optional

## Recommendations

- For optimal performance, set threads equal to or less than CPU cores
- Higher thread counts may not always improve performance due to:
  - CPU core limits
  - Memory bandwidth
  - Context switching overhead
- Difficulty level affects the benefit of parallelism (higher difficulty = more benefit)
