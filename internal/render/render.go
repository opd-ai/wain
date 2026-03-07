// Package render provides Go bindings to the Rust render-sys static library.
//
// musl libc is required. The Rust library must be compiled for the musl
// target, and CC must be set to musl-gcc when building Go. Use `make build`
// which enforces both requirements and produces a fully static binary.
//
// Manual build steps:
//
//	# 1. Add the musl Rust target (once)
//	rustup target add x86_64-unknown-linux-musl
//
//	# 2. Build the Rust static library
//	cargo build --release --target x86_64-unknown-linux-musl \
//	  --manifest-path render-sys/Cargo.toml
//
//	# 3. Build the Go binary against musl
//	CC=musl-gcc CGO_ENABLED=1 \
//	  go build -ldflags "-extldflags '-static'" ./...
//
// # Installation of musl-gcc
//
// The build will fail if musl-gcc is not present. Install it with:
//
//	Ubuntu / Debian:  sudo apt-get install musl-tools
//	Fedora / RHEL:    sudo dnf install musl-gcc
//	Arch Linux:       sudo pacman -S musl
//	Alpine Linux:     apk add musl-dev
//	macOS (cross):    brew install FiloSottile/musl-cross/musl-cross
//	                  (sets up x86_64-linux-musl-gcc, not musl-gcc directly)
//
// # Cross-architecture builds
//
// The default LDFLAGS embed the x86_64 musl path. On other Linux
// architectures (e.g. aarch64) use make, which auto-detects the host arch:
//
//	make build   # detects RUST_MUSL_TARGET automatically via rustc -vV
//
// Or set CGO_LDFLAGS manually:
//
//	MUSL_LIB=render-sys/target/aarch64-unknown-linux-musl/release/librender.a
//	CGO_ENABLED=1 CGO_LDFLAGS="${MUSL_LIB} -ldl -lm -lpthread" \
//	  CGO_LDFLAGS_ALLOW=".*" CC=musl-gcc \
//	  go build -ldflags "-extldflags '-static'" ./...
package render

// #cgo LDFLAGS: ${SRCDIR}/../../render-sys/target/x86_64-unknown-linux-musl/release/librender.a -ldl -lm -lpthread
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
