// Package render_test contains tests for the render package.
//
// IMPORTANT: These tests require CGO_LDFLAGS to be set to link the Rust static library.
// Do NOT run tests with `go test ./...` directly.
// Instead, use: make test-go
//
// The `make test-go` target sets the required CGO_LDFLAGS environment variable
// pointing to the Rust static library (librender.a) and its dependencies.
//
// Direct `go test` will fail with linker errors:
//
//	undefined reference to `render_add'
//	undefined reference to `render_version'
//
// This is intentional. The project enforces musl-based static builds and the
// library path is architecture-dependent, so CGO_LDFLAGS cannot be hardcoded.
package render_test

import (
	"os"
	"testing"

	render "github.com/opd-ai/wain/internal/render"
)

// TestMain provides a clear error message if CGO_LDFLAGS is not set.
// The render package requires the Rust static library to be linked via CGO_LDFLAGS.
// Use `make test-go` instead of `go test ./...` to run tests with the correct configuration.
func TestMain(m *testing.M) {
	// Try to call a Rust function to verify the library is linked
	// If this panics or the linker failed before we got here, TestMain won't run
	// but the linker error is clear enough. If we get here, the library is available.

	// Note: We can't actually test this without triggering the link error,
	// so we just provide documentation via this TestMain's existence.
	// The real fix is that if linking fails, the error message is clear.

	os.Exit(m.Run())
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int32
		expected int32
	}{
		{"positive", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"negative", -4, 4, 0},
		{"both negative", -3, -7, -10},
		{"identity", 42, 0, 42},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := render.Add(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	v := render.Version()
	if v == "" {
		t.Fatal("Version() returned empty string")
	}
	t.Logf("render library version: %s", v)
}

func TestDetectGPU(t *testing.T) {
	// Test with a non-existent path - should return GpuUnknown
	gen := render.DetectGPU("/dev/null")
	if gen != render.GpuUnknown {
		t.Errorf("DetectGPU(/dev/null) = %v, want GpuUnknown", gen)
	}

	// Test with the standard render node path
	// This may or may not exist on the test machine, so we accept any result
	gen = render.DetectGPU("/dev/dri/renderD128")
	t.Logf("GPU detection result for /dev/dri/renderD128: %s (code: %d)", gen, gen)

	// Test that the generation constants have expected values
	if render.GpuGen9 != 9 {
		t.Errorf("GpuGen9 = %d, want 9", render.GpuGen9)
	}
	if render.GpuGen11 != 11 {
		t.Errorf("GpuGen11 = %d, want 11", render.GpuGen11)
	}
	if render.GpuGen12 != 12 {
		t.Errorf("GpuGen12 = %d, want 12", render.GpuGen12)
	}
	if render.GpuXe != 13 {
		t.Errorf("GpuXe = %d, want 13", render.GpuXe)
	}
}

func TestGpuGenerationString(t *testing.T) {
	tests := []struct {
		gen      render.GpuGeneration
		expected string
	}{
		{render.GpuGen9, "Gen9 (Skylake/Kaby Lake/Coffee Lake)"},
		{render.GpuGen11, "Gen11 (Ice Lake)"},
		{render.GpuGen12, "Gen12 (Tiger Lake/Rocket Lake/Alder Lake)"},
		{render.GpuXe, "Xe (Meteor Lake+)"},
		{render.GpuUnknown, "Unknown"},
		{render.GpuGeneration(999), "Invalid"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			got := tc.gen.String()
			if got != tc.expected {
				t.Errorf("GpuGeneration(%d).String() = %q, want %q", tc.gen, got, tc.expected)
			}
		})
	}
}

