// Package render provides Go bindings to the Rust render-sys static library.
//
// musl libc is required. The Rust library must be compiled for the musl
// target and CGO_LDFLAGS must point at the resulting archive. Use
// `make build` which enforces both requirements and produces a fully static
// binary.
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
//	#    CC defaults to musl-gcc; override for cross toolchains (see below).
//	MUSL_LIB=render-sys/target/x86_64-unknown-linux-musl/release/librender.a
//	CC=musl-gcc CGO_ENABLED=1 \
//	  CGO_LDFLAGS="${MUSL_LIB} -ldl -lm -lpthread" \
//	  CGO_LDFLAGS_ALLOW=".*" \
//	  go build -ldflags "-extldflags '-static'" ./...
//
// # Installation of the musl C compiler
//
// The build will fail if the musl C compiler is not present.
// The compiler is selected via the CC make variable (default: musl-gcc).
//
//	Ubuntu / Debian:  sudo apt-get install musl-tools          # provides musl-gcc
//	Fedora / RHEL:    sudo dnf install musl-gcc
//	Arch Linux:       sudo pacman -S musl
//	Alpine Linux:     apk add musl-dev
//	macOS (cross):    brew install FiloSottile/musl-cross/musl-cross
//	                  # provides x86_64-linux-musl-gcc; pass CC explicitly:
//	                  make build CC=x86_64-linux-musl-gcc
//
// # Cross-architecture builds
//
// The Rust musl target and the CGO_LDFLAGS archive path are auto-detected by
// `make build` via `rustc -vV`. To build manually on non-x86_64 hosts:
//
//	MUSL_LIB=render-sys/target/aarch64-unknown-linux-musl/release/librender.a
//	CGO_ENABLED=1 CGO_LDFLAGS="${MUSL_LIB} -ldl -lm -lpthread" \
//	  CGO_LDFLAGS_ALLOW=".*" CC=aarch64-linux-musl-gcc \
//	  go build -ldflags "-extldflags '-static'" ./...
package render

// The Rust static library path is NOT hardcoded here because the correct path
// depends on the host architecture and the musl Rust target. CGO_LDFLAGS must
// be set by the build system (Makefile / CI) to point at the correct archive:
//
//	CGO_LDFLAGS="<path>/librender.a -ldl -lm -lpthread" CGO_LDFLAGS_ALLOW=".*"
//
// Use `make build` or `make test-go` which set CGO_LDFLAGS automatically.

// #include <stdint.h>
//
// int32_t render_add(int32_t a, int32_t b);
// const char *render_version(void);
import "C"

// Add calls the Rust render_add function via the C ABI and returns a + b.
// This is the canonical smoke-test for the Go → Rust static-library link.
func Add(a, b int32) int32 {
	return int32(C.render_add(C.int32_t(a), C.int32_t(b)))
}

// Version returns the version string from the Rust render library.
func Version() string {
	return C.GoString(C.render_version())
}
