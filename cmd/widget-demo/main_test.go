package main

import (
	"testing"
)

// TestWidgetDemoCompiles is a smoke test verifying the binary compiles.
// Full functional testing requires a display server (X11 or Wayland).
func TestWidgetDemoCompiles(t *testing.T) {
	// Verify platform detection logic
	platform := detectPlatform()
	if platform == "" {
		t.Error("detectPlatform() returned empty string")
	}

	validPlatforms := map[string]bool{
		"x11":     true,
		"wayland": true,
	}
	if !validPlatforms[platform] {
		t.Errorf("detectPlatform() returned invalid platform: %q", platform)
	}
}

// TestDetectPlatform verifies platform detection returns valid values.
func TestDetectPlatform(t *testing.T) {
	platform := detectPlatform()

	validPlatforms := []string{"x11", "wayland"}
	valid := false
	for _, p := range validPlatforms {
		if platform == p {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("detectPlatform() = %q, want one of %v", platform, validPlatforms)
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
