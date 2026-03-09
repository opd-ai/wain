package main

import (
	"testing"
)

// TestGpuTriangleDemoCompiles is a smoke test verifying the binary compiles.
// Full functional testing requires GPU hardware and display server.
func TestGpuTriangleDemoCompiles(t *testing.T) {
	// Verify constants are defined
	if windowWidth <= 0 {
		t.Errorf("windowWidth must be positive, got %d", windowWidth)
	}
	if windowHeight <= 0 {
		t.Errorf("windowHeight must be positive, got %d", windowHeight)
	}
}

// TestConstants verifies window size constants are reasonable.
func TestConstants(t *testing.T) {
	if windowWidth <= 0 {
		t.Errorf("windowWidth must be positive, got %d", windowWidth)
	}
	if windowHeight <= 0 {
		t.Errorf("windowHeight must be positive, got %d", windowHeight)
	}
	if windowWidth > 10000 || windowHeight > 10000 {
		t.Errorf("window dimensions suspiciously large: %dx%d", windowWidth, windowHeight)
	}
}
