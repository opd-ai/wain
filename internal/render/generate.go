// Package render provides Go bindings to the Rust render-sys static library.
//
// Use `go generate ./...` followed by `go test ./...` or `go build` to build
// the full static binary. The go generate step builds the Rust rendering
// library, the musl compatibility stub, and writes a generated CGO LDFLAGS
// file (cgo_flags_generated.go) so that subsequent `go test ./...` invocations
// work without any additional CGO_LDFLAGS environment variable.
package render

//go:generate sh -c "cd ../../ && ./scripts/build-rust.sh"
