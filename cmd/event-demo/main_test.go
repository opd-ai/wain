package main

import (
	"testing"
)

// TestEventDemoCompiles is a smoke test verifying the binary compiles.
// Full functional testing requires a display server and would require
// mocking the event system or running with a timeout.
func TestEventDemoCompiles(t *testing.T) {
	// This test just verifies the package compiles.
	// The main() function creates a full GUI app and runs until interrupted,
	// which is not suitable for automated testing without significant mocking.
	t.Log("event-demo package compiles successfully")
}
