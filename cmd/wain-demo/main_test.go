package main

import (
	"testing"
)

// TestWainDemoCompiles is a smoke test verifying the binary compiles.
// Full functional testing requires a display server and would require
// mocking the GUI system or running with a timeout.
func TestWainDemoCompiles(t *testing.T) {
	// This test just verifies the package compiles.
	// The main() function creates a full wain.App and runs until interrupted,
	// which is not suitable for automated testing without significant mocking.
	t.Log("wain-demo package compiles successfully")
}
