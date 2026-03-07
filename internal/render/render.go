// Package render provides Go bindings to the Rust render-sys static library.
//
// Build the Rust library first:
//
//	cd render-sys && cargo build --release
//
// Then build the Go package with CGO enabled (the default):
//
//	CGO_ENABLED=1 go build ./...
//
// # Musl / fully-static builds
//
// To link against a musl-target Rust library (required for a fully static
// binary), override the library path via CGO_LDFLAGS at build time:
//
//	cargo build --release --target x86_64-unknown-linux-musl \
//	  --manifest-path render-sys/Cargo.toml
//
//	MUSL_LIB=render-sys/target/x86_64-unknown-linux-musl/release/librender.a
//	CGO_ENABLED=1 \
//	  CGO_LDFLAGS="${MUSL_LIB} -ldl -lm -lpthread" \
//	  CGO_LDFLAGS_ALLOW=".*" \
//	  CC=musl-gcc \
//	  go build -ldflags "-extldflags '-static'" ./...
package render

// #cgo LDFLAGS: ${SRCDIR}/../../render-sys/target/release/librender.a -ldl -lm -lpthread
//
// #include <stdint.h>
//
// int32_t render_add(int32_t a, int32_t b);
// const char *render_version(void);
import "C"
import "unsafe"

// Add calls the Rust render_add function via the C ABI and returns a + b.
// This is the canonical smoke-test for the Go → Rust static-library link.
func Add(a, b int32) int32 {
	return int32(C.render_add(C.int32_t(a), C.int32_t(b)))
}

// Version returns the version string from the Rust render library.
func Version() string {
	return C.GoString((*C.char)(unsafe.Pointer(C.render_version())))
}
