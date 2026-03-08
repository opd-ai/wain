// Package render provides Go bindings to the Rust render-sys static library.
//
// Use `go generate ./...` followed by `go build` to build the full static binary.
// The go generate step builds the Rust rendering library and the musl compatibility stub.
package render

//go:generate sh -c "cd ../../ && ./scripts/build-rust.sh"
