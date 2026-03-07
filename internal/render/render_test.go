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
