package config

import (
	"testing"
)

func TestMiningThreads(t *testing.T) {
	// Test default value
	SetMiningThreads(1) // Reset to default
	if MiningThreads() != 1 {
		t.Errorf("Default mining threads should be 1, got %d", MiningThreads())
	}

	// Test setting positive value
	SetMiningThreads(4)
	if MiningThreads() != 4 {
		t.Errorf("Mining threads should be 4, got %d", MiningThreads())
	}

	// Test setting zero defaults to 1
	SetMiningThreads(0)
	if MiningThreads() != 1 {
		t.Errorf("Mining threads with 0 should default to 1, got %d", MiningThreads())
	}

	// Test setting negative defaults to 1
	SetMiningThreads(-5)
	if MiningThreads() != 1 {
		t.Errorf("Mining threads with -5 should default to 1, got %d", MiningThreads())
	}

	// Test high value
	SetMiningThreads(16)
	if MiningThreads() != 16 {
		t.Errorf("Mining threads should be 16, got %d", MiningThreads())
	}

	// Clean up
	SetMiningThreads(1)
}

func TestMiningThreadsConcurrency(t *testing.T) {
	// Test thread-safety by concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(val int) {
			SetMiningThreads(val)
			_ = MiningThreads()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// If we reach here without deadlock/race, test passes
	SetMiningThreads(1) // Reset
}
