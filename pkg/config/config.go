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
