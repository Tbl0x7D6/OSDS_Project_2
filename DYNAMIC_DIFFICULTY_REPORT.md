# Dynamic Difficulty Adjustment Implementation Report

## Overview

This document describes the implementation of dynamic difficulty adjustment for the blockchain project. The feature automatically adjusts the mining difficulty to maintain a target block production rate of **1 block per 10 seconds**.

## Goal

The goal is to implement a Bitcoin-style difficulty adjustment mechanism that:
1. Adjusts the difficulty (number of leading zero bits) dynamically
2. Targets 1 block per 10 seconds (6 blocks per minute)
3. Provides a command-line switch to enable/disable the feature
4. Includes comprehensive tests

## Implementation Details

### 1. New Package: `pkg/difficulty`

Created a new package `pkg/difficulty` with the following components:

#### Constants
- `TargetBlockTime = 10 * time.Second` - Target time between blocks
- `AdjustmentInterval = 6` - Number of blocks between difficulty adjustments (approximately 1 minute at target rate)
- `MinDifficulty = 1` - Minimum allowed difficulty
- `MaxDifficulty = 32` - Maximum allowed difficulty
- `MaxAdjustmentFactor = 2.0` - Limits how much difficulty can change at once

#### DifficultyAdjuster Struct
The `DifficultyAdjuster` struct manages the dynamic difficulty state:
- `enabled` - Whether dynamic difficulty is active
- `currentDifficulty` - The current difficulty level
- Thread-safe with mutex protection

#### Key Functions

| Function | Description |
|----------|-------------|
| `NewDifficultyAdjuster(initialDifficulty, enabled)` | Creates a new difficulty adjuster |
| `ShouldAdjust(blockIndex)` | Returns true if it is time to adjust difficulty |
| `CalculateNewDifficulty(blocks, currentDifficulty)` | Calculates new difficulty based on recent block times |
| `CalculateAverageBlockTime(blocks)` | Calculates average time between blocks |
| `CalculateAdjustment(blocks, currentDifficulty)` | Returns detailed adjustment info |
| `GetBlocksPerMinute(blocks)` | Calculates blocks per minute rate |

#### Difficulty Adjustment Algorithm

```
1. Get the last 6 blocks (AdjustmentInterval)
2. Calculate actual time taken for these blocks
3. Calculate expected time (6 blocks * 10 seconds = 60 seconds)
4. Calculate ratio = expected_time / actual_time
5. If ratio > 1.2: blocks too fast, increase difficulty by 1
6. If ratio < 0.8: blocks too slow, decrease difficulty by 1
7. Otherwise: keep difficulty unchanged
8. Clamp difficulty between MinDifficulty and MaxDifficulty
```

### 2. Configuration Updates (`pkg/config`)

Added new global configuration:
- `useDynamicDifficulty` - Boolean flag to enable/disable dynamic difficulty
- `SetUseDynamicDifficulty(use bool)` - Setter function
- `UseDynamicDifficulty() bool` - Getter function

### 3. Blockchain Updates (`pkg/blockchain`)

Added new method:
- `GetRecentBlocks(n int) []*block.Block` - Returns the most recent n blocks for difficulty calculation

### 4. Network/Miner Updates (`pkg/network`)

#### Miner Struct Changes
Added `DifficultyAdjuster` field to the `Miner` struct.

#### New Methods
- `maybeAdjustDifficulty(newBlock)` - Called after each block is mined, adjusts difficulty if needed
- `SetDynamicDifficulty(enabled)` - Enables/disables dynamic difficulty at runtime
- `IsDynamicDifficultyEnabled()` - Returns current dynamic difficulty state

#### Mining Loop Integration
The difficulty adjustment is triggered after successfully mining a block:
```go
// In mineBlock() function, after block is added:
m.maybeAdjustDifficulty(result.Block)
```

### 5. CLI Updates (`cmd/miner`)

Added new command-line flag:
```
-dynamic-difficulty  Enable dynamic difficulty adjustment (default: false)
```

Usage example:
```bash
# Start miner with dynamic difficulty enabled
./bin/miner -id miner1 -address localhost:8001 -difficulty 10 -dynamic-difficulty=true

# Start miner with static difficulty (default)
./bin/miner -id miner1 -address localhost:8001 -difficulty 10
```

## Files Modified

| File | Changes |
|------|---------|
| `pkg/difficulty/difficulty.go` | **NEW** - Dynamic difficulty adjustment package |
| `pkg/difficulty/difficulty_test.go` | **NEW** - Unit tests for difficulty package |
| `pkg/config/config.go` | Added `useDynamicDifficulty` configuration |
| `pkg/blockchain/blockchain.go` | Added `GetRecentBlocks()` method |
| `pkg/network/network.go` | Added `DifficultyAdjuster`, `maybeAdjustDifficulty()`, and related methods |
| `cmd/miner/main.go` | Added `-dynamic-difficulty` flag |
| `test_dynamic_difficulty.sh` | **NEW** - Bash test script |

## Testing

### Unit Tests (`pkg/difficulty/difficulty_test.go`)

15 test cases covering:
- DifficultyAdjuster creation and configuration
- Enable/disable functionality
- `ShouldAdjust()` at various block indices
- Difficulty calculation when blocks are too fast
- Difficulty calculation when blocks are too slow
- Difficulty calculation when blocks are on target
- Minimum and maximum difficulty bounds
- Insufficient blocks handling
- Average block time calculation
- Blocks per minute calculation
- Difficulty clamping

Run tests:
```bash
go test ./pkg/difficulty/... -v
```

### Integration Tests (`test_dynamic_difficulty.sh`)

6 test cases covering:
1. Verify `-dynamic-difficulty` flag exists
2. Static difficulty mode (default behavior)
3. Dynamic difficulty mode activation
4. Difficulty adjustment with fast blocks
5. Comparison between static and dynamic modes
6. Multiple miners with dynamic difficulty

Run tests:
```bash
./test_dynamic_difficulty.sh
```

## Logging

When dynamic difficulty is enabled, the miner logs:
- At startup: `Dynamic difficulty adjustment enabled (target: 1 block per 10 seconds)`
- On adjustment: `Difficulty adjusted: X -> Y (avg block time: Zs, target: 10s)`

When static difficulty is used:
- At startup: `Static difficulty mode (difficulty: X)`

## Future Improvements

1. **Configurable target block time** - Allow users to specify custom target block time
2. **Smoother adjustment** - Implement weighted moving average for more stable adjustments
3. **Peer synchronization** - Coordinate difficulty across network peers
4. **Difficulty history** - Track and display difficulty changes over time
5. **API endpoint** - Add RPC endpoint to query current difficulty and adjustment info

## Conclusion

The dynamic difficulty adjustment feature has been successfully implemented and tested. It allows the blockchain to automatically adjust mining difficulty to maintain a consistent block production rate, regardless of changes in mining power. The feature is disabled by default to maintain backward compatibility and can be enabled via the `-dynamic-difficulty` command-line flag.
