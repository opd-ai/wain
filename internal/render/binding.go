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
// #include <stdlib.h>
//
// int32_t render_add(int32_t a, int32_t b);
// const char *render_version(void);
// int32_t render_detect_gpu(const char *path);
import "C"

import "unsafe"

// Add calls the Rust render_add function via the C ABI and returns a + b.
// This is the canonical smoke-test for the Go → Rust static-library link.
func Add(a, b int32) int32 {
	return int32(C.render_add(C.int32_t(a), C.int32_t(b)))
}

// Version returns the version string from the Rust render library.
func Version() string {
	return C.GoString(C.render_version())
}

// GpuGeneration represents a detected Intel GPU generation.
type GpuGeneration int

const (
	// GpuUnknown represents an unknown or unsupported GPU.
	GpuUnknown GpuGeneration = 0
	// GpuGen9 represents Gen9 (Skylake, Kaby Lake, Coffee Lake).
	GpuGen9 GpuGeneration = 9
	// GpuGen11 represents Gen11 (Ice Lake).
	GpuGen11 GpuGeneration = 11
	// GpuGen12 represents Gen12 (Tiger Lake, Rocket Lake, Alder Lake).
	GpuGen12 GpuGeneration = 12
	// GpuXe represents Xe (Meteor Lake and later).
	GpuXe GpuGeneration = 13
)

// String returns a human-readable name for the GPU generation.
func (g GpuGeneration) String() string {
	switch g {
	case GpuGen9:
		return "Gen9 (Skylake/Kaby Lake/Coffee Lake)"
	case GpuGen11:
		return "Gen11 (Ice Lake)"
	case GpuGen12:
		return "Gen12 (Tiger Lake/Rocket Lake/Alder Lake)"
	case GpuXe:
		return "Xe (Meteor Lake+)"
	case GpuUnknown:
		return "Unknown"
	default:
		return "Invalid"
	}
}

// DetectGPU queries the GPU generation from the DRM device at the given path.
//
// Returns GpuUnknown on error or if the GPU is not recognized.
//
// Example:
//
//	gen := render.DetectGPU("/dev/dri/renderD128")
//	if gen == render.GpuUnknown {
//	    log.Println("Unknown or unsupported GPU")
//	} else {
//	    log.Printf("Detected: %s", gen)
//	}
func DetectGPU(path string) GpuGeneration {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	result := C.render_detect_gpu(cpath)
	if result < 0 {
		return GpuUnknown
	}
	return GpuGeneration(result)
}
