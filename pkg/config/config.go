// Package config provides global configuration for the blockchain
package config

import "sync"

var (
	// useMerkleTree controls whether to use Merkle Tree for block hash calculation
	// Default is true (use Merkle Tree)
	useMerkleTree = true

	// useDynamicDifficulty controls whether to use dynamic difficulty adjustment
	// Default is false (use static difficulty)
	useDynamicDifficulty = false

	// miningThreads controls the number of parallel threads for mining
	// Default is 1 (sequential mining, no parallelism)
	miningThreads = 1

	mu sync.RWMutex
)

// UseMerkleTree returns whether Merkle Tree should be used for block hash calculation
func UseMerkleTree() bool {
	mu.RLock()
	defer mu.RUnlock()
	return useMerkleTree
}

// SetUseMerkleTree sets whether to use Merkle Tree for block hash calculation
func SetUseMerkleTree(use bool) {
	mu.Lock()
	defer mu.Unlock()
	useMerkleTree = use
}

// UseDynamicDifficulty returns whether dynamic difficulty adjustment is enabled
func UseDynamicDifficulty() bool {
	mu.RLock()
	defer mu.RUnlock()
	return useDynamicDifficulty
}

// SetUseDynamicDifficulty sets whether to use dynamic difficulty adjustment
func SetUseDynamicDifficulty(use bool) {
	mu.Lock()
	defer mu.Unlock()
	useDynamicDifficulty = use
}

// MiningThreads returns the number of parallel threads for mining
func MiningThreads() int {
	mu.RLock()
	defer mu.RUnlock()
	return miningThreads
}

// SetMiningThreads sets the number of parallel threads for mining
// If threads <= 0, it defaults to 1 (sequential mining)
func SetMiningThreads(threads int) {
	mu.Lock()
	defer mu.Unlock()
	if threads <= 0 {
		miningThreads = 1
	} else {
		miningThreads = threads
	}
}
