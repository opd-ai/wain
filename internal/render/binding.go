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
//
// // Relocation entry for i915 batch submission
// typedef struct {
//     uint32_t target_handle;
//     uint32_t delta;
//     uint64_t offset;
//     uint64_t presumed_offset;
//     uint32_t read_domains;
//     uint32_t write_domain;
// } RelocationEntry;
//
// int32_t render_submit_batch(
//     const char *path,
//     uint32_t batch_handle,
//     uint32_t batch_len_bytes,
//     const RelocationEntry *relocs,
//     size_t relocs_count,
//     uint32_t context_id
// );
//
// int32_t render_create_context(
//     const char *path,
//     uint32_t *out_context_id,
//     uint32_t *out_vm_id
// );
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

// Relocation represents a GPU address relocation entry for batch submission.
//
// When a batch buffer references another GPU buffer (e.g., a render target),
// the kernel needs to patch the actual GPU virtual address at submission time.
// Relocations tell the kernel where to patch these addresses.
type Relocation struct {
	TargetHandle   uint32 // GEM handle of the buffer being referenced
	Delta          uint32 // Offset within the target buffer
	Offset         uint64 // Offset in the batch buffer (bytes)
	PresumedOffset uint64 // Expected GPU address (0 = unknown)
	ReadDomains    uint32 // Cache read domains (I915_GEM_DOMAIN_*)
	WriteDomain    uint32 // Cache write domain (I915_GEM_DOMAIN_*)
}

// Cache domain constants for relocations.
const (
	GemDomainRender      uint32 = 0x00000002
	GemDomainInstruction uint32 = 0x00000010
)

// SubmitBatch submits a GPU batch buffer and waits for completion.
//
// This function submits a command stream to the GPU and blocks until execution
// completes. It automatically detects the GPU type and uses the appropriate
// driver interface (i915 or Xe).
//
// # Arguments
//   - path: Path to the DRM device (e.g., "/dev/dri/renderD128")
//   - batchHandle: GEM buffer handle containing the command stream
//   - batchLenBytes: Length of the command stream in bytes
//   - relocs: Slice of relocation entries (can be nil if no relocations needed)
//   - contextID: GPU context ID (use 0 for default context)
//
// Returns an error if submission fails.
//
// Example:
//
//	err := render.SubmitBatch("/dev/dri/renderD128", batchHandle, 1024, nil, 0)
//	if err != nil {
//	    log.Fatalf("Batch submission failed: %v", err)
//	}
func SubmitBatch(path string, batchHandle uint32, batchLenBytes uint32, relocs []Relocation, contextID uint32) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var crelocs *C.RelocationEntry
	var relocsCount C.size_t

	if len(relocs) > 0 {
		// Convert Go slice to C array
		cRelocs := make([]C.RelocationEntry, len(relocs))
		for i, r := range relocs {
			cRelocs[i].target_handle = C.uint32_t(r.TargetHandle)
			cRelocs[i].delta = C.uint32_t(r.Delta)
			cRelocs[i].offset = C.uint64_t(r.Offset)
			cRelocs[i].presumed_offset = C.uint64_t(r.PresumedOffset)
			cRelocs[i].read_domains = C.uint32_t(r.ReadDomains)
			cRelocs[i].write_domain = C.uint32_t(r.WriteDomain)
		}
		crelocs = &cRelocs[0]
		relocsCount = C.size_t(len(relocs))
	}

	result := C.render_submit_batch(
		cpath,
		C.uint32_t(batchHandle),
		C.uint32_t(batchLenBytes),
		crelocs,
		relocsCount,
		C.uint32_t(contextID),
	)

	if result < 0 {
		return &SubmitError{path: path}
	}
	return nil
}

// SubmitError represents an error during batch submission.
type SubmitError struct {
	path string
}

func (e *SubmitError) Error() string {
	return "batch submission failed for device " + e.path
}

// GpuContext represents a GPU context handle.
//
// Contexts isolate GPU state and allow multiple independent workloads.
// For i915, only ContextID is used. For Xe, both ContextID (exec queue) and
// VmID are used.
type GpuContext struct {
	ContextID uint32 // Context/exec queue ID
	VmID      uint32 // VM ID (Xe only, 0 for i915)
}

// CreateContext creates a GPU context for command submission.
//
// A context isolates GPU state and allows multiple independent workloads to
// execute concurrently. Most applications should create one context per
// rendering thread or workload.
//
// For i915 GPUs, only the ContextID field is populated. For Xe GPUs, both
// ContextID (exec queue) and VmID are populated.
//
// Returns a GpuContext on success, or an error if context creation fails.
//
// Example:
//
//	ctx, err := render.CreateContext("/dev/dri/renderD128")
//	if err != nil {
//	    log.Fatalf("Context creation failed: %v", err)
//	}
//	log.Printf("Created context ID: %d", ctx.ContextID)
func CreateContext(path string) (*GpuContext, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var contextID C.uint32_t
	var vmID C.uint32_t

	result := C.render_create_context(cpath, &contextID, &vmID)

	if result < 0 {
		return nil, &ContextCreateError{path: path}
	}

	return &GpuContext{
		ContextID: uint32(contextID),
		VmID:      uint32(vmID),
	}, nil
}

// ContextCreateError represents an error during context creation.
type ContextCreateError struct {
	path string
}

func (e *ContextCreateError) Error() string {
	return "context creation failed for device " + e.path
}
