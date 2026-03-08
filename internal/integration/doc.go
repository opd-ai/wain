// Package integration provides cross-layer integration tests that verify
// protocol handling, buffer management, and GPU rendering functionality.
//
// # Test Requirements
//
// Tests in this package require CGO linking against the Rust render-sys library.
// Running tests with `go test` directly will fail with linker errors like:
//
//	undefined reference to `render_add'
//	undefined reference to `render_detect_gpu'
//
// To run integration tests, use ONE of these methods:
//
//	1. make test-go              (recommended for CI)
//	2. direnv allow && go test   (recommended for development)
//	3. See internal/integration/README.md for manual CGO setup
//
// For details on test environment setup, see the project README.md Test section.
package integration
