# wain

**A statically-compiled Go UI toolkit with GPU rendering via Rust**

## Table of Contents

- [Status](#status)
- [Current Functionality](#current-functionality)
- [Known Limitations](#known-limitations)
- [Documentation](#documentation)
- [Prerequisites](#prerequisites)
- [Build](#build)
- [Test](#test)
- [Verify Static Linking](#verify-static-linking)
- [Run](#run)
- [Demonstration Binaries](#demonstration-binaries)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Manual Build](#manual-build-without-makefile)
- [Font Atlas Generation](#font-atlas-generation)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Status

**Phases 0–2** complete, **Phase 3** (GPU Command Submission) and **Phase 4.1** (Shader Frontend IR) partially implemented.
See [ROADMAP.md](ROADMAP.md) for the full 8-phase implementation plan.

## Current Functionality

### Foundation (Phase 0) — ✅ Complete
- ✅ Go → Rust static library linking (CGO + musl)
- ✅ C ABI boundary validation (`render_add`, `render_version`)
- ✅ Fully static binary output (no dynamic dependencies)

### Protocol Layer (Phase 1.1–1.2) — ✅ Complete
**Wayland Client** (7 packages, ~4,325 LOC):
- ✅ Wire format: binary protocol marshaling, fd passing via SCM_RIGHTS
- ✅ Core objects: wl_display, wl_registry, wl_compositor, wl_surface
- ✅ Shared memory: wl_shm, wl_shm_pool, wl_buffer (memfd_create)
- ✅ Window management: xdg_wm_base, xdg_surface, xdg_toplevel
- ✅ Input handling: wl_seat, wl_pointer, wl_keyboard with basic keycode-to-keysym translation (hardcoded QWERTY layout)
- ✅ DMA-BUF: zwp_linux_dmabuf_v1 protocol for GPU buffer sharing

**X11 Client** (7 packages, ~3,288 LOC):
- ✅ Connection setup: authentication, XID allocation, extension queries
- ✅ Window operations: CreateWindow, MapWindow, ConfigureWindow
- ✅ Graphics context: CreateGC, PutImage, CreatePixmap
- ✅ Event handling: KeyPress, ButtonPress, MotionNotify, Expose
- ✅ MIT-SHM extension: zero-copy shared memory image transfers
- ✅ DRI3 extension: GPU buffer sharing via DMA-BUF file descriptors
- ✅ Present extension: frame synchronization and swap control

### Buffer Infrastructure (Phase 2) — ✅ Complete
**Rust DRM/KMS Integration** (~13,885 LOC total in render-sys):
- ✅ Kernel ioctl wrappers: i915 and Xe GPU drivers (`i915.rs`, `xe.rs`)
- ✅ DRM device access and GPU generation detection (`drm.rs`, `detect.rs`)
- ✅ Buffer allocation: GPU-visible buffers with tiling support (`allocator.rs`)
- ✅ DMA-BUF export: fd-based buffer sharing across processes
- ✅ Slab allocator: efficient sub-allocation from large GPU buffers (`slab.rs`)

### GPU Command Submission (Phase 3) — 🔧 In Progress
**Intel GPU Command Infrastructure** (render-sys):
- ✅ GPU generation detection via Go bindings (`render.DetectGPU`)
- ✅ GPU context creation for i915/Xe (`render.CreateContext`)
- ✅ Batch buffer construction with relocation support (`batch.rs`)
- ✅ Intel 3D pipeline command encoding: MI commands, pipeline select, state base address, viewport, clip, SF, WM, PS, vertex buffers/elements, 3DPRIMITIVE (`cmd/`)
- ✅ Pipeline state objects for common UI draw operations (`pipeline.rs`)
- ✅ Surface state and sampler state encoding (`surface.rs`)
- ✅ Batch submission via Go bindings (`render.SubmitBatch`)
- ✅ GPU triangle demonstration binary (`cmd/gpu-triangle-demo/`)

### Shader Frontend (Phase 4.1) — ✅ Complete
- ✅ WGSL and GLSL shader parsing via naga (`shader.rs`)
- ✅ Shader validation pipeline
- ✅ naga 0.14 integrated as Rust dependency

### UI Shaders (Phase 4.2) — ✅ Complete
- ✅ 7 WGSL shaders authored in `render-sys/shaders/`
- ✅ All shaders validated with naga (22 shader tests passing, 7 GPU tests ignored)
- ✅ Comprehensive shader documentation (478-line README)
- ✅ Shaders: solid_fill, textured_quad, sdf_text, box_shadow, rounded_rect, linear_gradient, radial_gradient

### Intel EU Backend (Phase 4.3) — 🔧 In Progress
**Core Compilation Pipeline** (~4,090 LOC):
- ✅ Main compile() function implemented
- ✅ Register allocator (linear-scan, r0-r127 GRF mapping)
- ✅ Instruction lowering (binary/unary ops, math functions, control flow)
- ✅ Binary encoding (128-bit EU instruction format for Gen9+)
- ✅ Basic shader compilation validated (vertex shaders compile to EU binary)
- ⚠️ Advanced features deferred: URB I/O, texture SEND instructions, optimizations

### Rendering Layer (Phase 1.4) — ✅ Complete
**Software 2D Rasterizer** (5 packages, ~2,458 LOC):
- ✅ Primitives: filled rectangles, rounded rectangles, anti-aliased lines
- ✅ Curves: quadratic/cubic Bezier, arc fills
- ✅ Text: SDF-based rendering with embedded glyph atlas
- ✅ Effects: box shadow (Gaussian blur), linear/radial gradients
- ✅ Compositing: alpha blending (Porter-Duff), bilinear image filtering

### UI Framework (Phase 1.5) — ✅ Complete
**Widget Layer** (3 packages, ~2,500 LOC):
- ✅ Layout system: flexbox-like Row/Column with flex-grow/shrink, gaps, padding
- ✅ Widgets: Button, TextInput, ScrollContainer with event handlers
- ✅ Sizing: percentage-based dimensions with auto-layout

### Integration Status
- ✅ Demonstration binaries: `demo`, `wayland-demo`, `x11-demo`, `widget-demo`, `x11-dmabuf-demo`, `dmabuf-demo`, `gpu-triangle-demo`, `double-buffer-demo`
- ✅ Full protocol → rasterizer → display pipeline verified with integration tests
- ✅ GPU buffer sharing tested on both X11 (DRI3) and Wayland (dmabuf)
- ✅ Frame buffer ring management for double/triple buffering with compositor synchronization (`internal/buffer/`, `internal/integration/`)
- ⚠️ All packages marked `internal/` (public API surface planned for later)

**Not yet implemented:** Full GPU rendering pipeline integration (Phase 5+), advanced EU backend features (URB I/O, texture sampling, shader optimizations - Phase 4.3 cont'd), AMD GPU support (Phase 6). The project currently uses CPU-based software rendering. GPU buffers are allocated and shared, GPU command submission infrastructure exists, and basic shader compilation to EU binary is functional, but GPU rendering is not yet wired into the display pipeline.

## Known Limitations

**Integration status:**
- ✅ Demonstration binaries showing protocol → rasterizer → display pipeline working
- ✅ End-to-end integration tests verify full stack functionality
- ⚠️ All packages marked `internal/` — no public API for external users yet
- ⚠️ No platform abstraction layer (users must choose Wayland or X11 explicitly)
- ⚠️ No production-ready event loop (demos have basic event handling only)

**Rendering:**
- ⚠️ CPU-only software rendering (GPU rendering pipeline not yet connected)
- ⚠️ Single-threaded rasterizer (no tile-based threading)

**Testing:**
- ✅ Unit tests for all packages (57 test files)
- ✅ End-to-end integration tests for DRI3, GPU, and Wayland subsystems
- ✅ Fuzz tests for Wayland and X11 wire protocol encoding/decoding
- ⚠️ No automated screenshot comparison tests

**Future work (Phase 5+):**
See [ROADMAP.md](ROADMAP.md) for planned GPU rendering backends, AMD support, and polish features.

## Documentation

**Comprehensive documentation is available in the following files:**

- **[API.md](API.md)** — Complete API reference for all internal packages (rendering, protocols, UI widgets, integration)
- **[HARDWARE.md](HARDWARE.md)** — Supported hardware matrix (Intel/AMD GPUs, kernel versions, display servers, testing matrix)
- **[ROADMAP.md](ROADMAP.md)** — 8-phase implementation plan with detailed milestones (Phases 0-8)
- **[ACCESSIBILITY.md](ACCESSIBILITY.md)** — Accessibility support documentation and future AT-SPI2 implementation path
- **[RECOMMENDED_LIBRARIES.md](RECOMMENDED_LIBRARIES.md)** — Library selection rationale and design constraints

**Quick links:**
- GPU Support: See [HARDWARE.md § GPU Support](HARDWARE.md#gpu-support)
- Build Instructions: See [Build](#build) section below
- Architecture Overview: See [Architecture](#architecture) section below
- Code Examples: See [Demonstration Binaries](#demonstration-binaries) and `cmd/` directory

## Prerequisites

### Required Tools

1. **Go 1.24+**
   ```bash
   go version  # should report 1.24 or later
   ```

2. **Rust (stable) with musl target**
   ```bash
   rustup target add x86_64-unknown-linux-musl
   # For ARM: rustup target add aarch64-unknown-linux-musl
   ```

3. **musl C compiler**
   ```bash
   # Ubuntu / Debian
   sudo apt-get install musl-tools

   # Fedora / RHEL
   sudo dnf install musl-gcc

   # Arch Linux
   sudo pacman -S musl

   # Alpine Linux
   apk add musl-dev

   # macOS (cross-compilation)
   brew install FiloSottile/musl-cross/musl-cross
   # Then pass CC=x86_64-linux-musl-gcc to make
   ```

## Build

### Quick Build (Recommended)
```bash
# Build the static binary (checks deps, builds Rust library, builds Go binary)
make build

# Output: ./bin/wain (fully static executable)
```

The Makefile auto-detects the host architecture via `rustc -vV` and selects the appropriate musl target (e.g., `x86_64-unknown-linux-musl` or `aarch64-unknown-linux-musl`).

### Alternative: Go Generate Workflow

You can also build using Go's native workflow with `go generate`:

```bash
# Step 1: Generate build artifacts (Rust library + musl stub)
go generate ./...

# Step 2: Build the Go binary
CC=musl-gcc CGO_ENABLED=1 \
  CGO_LDFLAGS="$(pwd)/render-sys/target/$(uname -m)-unknown-linux-musl/release/librender.a $(pwd)/internal/render/dl_find_object_stub.o -ldl -lm -lpthread" \
  CGO_LDFLAGS_ALLOW=".*" \
  go build -ldflags "-extldflags '-static'" -o bin/wain ./cmd/wain
```

The `go generate` step:
1. Checks for required tools (musl-gcc, cargo, rustup)
2. Auto-detects host architecture
3. Installs musl Rust target if missing
4. Builds the Rust static library (`librender.a`)
5. Compiles the musl compatibility stub

**Recommendation:** Use `make build` for simplicity. The `go generate` workflow is provided for integration with Go-native build systems and CI pipelines that prefer Go tooling over Make.

### Configurable Variables

| Variable      | Default     | Description                                    |
|---------------|-------------|------------------------------------------------|
| `CC`          | `musl-gcc`  | musl C compiler; override for cross toolchains |
| `CARGO_FLAGS` | (empty)     | Extra flags passed to `cargo` commands          |

Example cross-compilation:
```bash
make build CC=x86_64-linux-musl-gcc
```

## Test

### Quick Start (direnv)

For the best developer experience, use [direnv](https://direnv.net/) to auto-configure CGO flags:

```bash
# One-time setup:
direnv allow

# Now standard Go commands work:
go test ./...
go test ./internal/raster/...
go test -v ./internal/wayland/wire
```

The `.envrc` file automatically sets `CGO_LDFLAGS` when entering the project directory.

### Using Make Targets

```bash
# Run all tests (Rust + Go)
make test

# Run only Rust tests
make test-rust

# Run only Go tests
make test-go

# Run Go tests with coverage reporting
make coverage
# Shows per-package coverage (average: ~70%)

# Generate HTML coverage report
make coverage-html
# HTML report: coverage/coverage.html
```

**Test Suite Coverage:**
- **Rust:** 252 tests total (244 passing, 8 GPU tests ignored)
  - Includes comprehensive unit tests for shader compilation, EU backend, batch processing, and pipeline management
  - 22 shader validation tests (naga integration tests for all 7 WGSL shaders)
- **Go:** 57 test files covering all 40 packages
  - Protocol implementations (Wayland wire format, X11 protocol)
  - 2D rasterization (curves, text, effects, compositing)
  - UI framework (layout, widgets, event handling)
  - Integration tests (full protocol → rasterizer → display pipeline)
  - **Code coverage:** ~70% average across 34 library packages (range: 8.9% to 100%)
    - High coverage (>90%): buffer, raster, UI layout, X11 present/gc/dpi
    - Moderate coverage (50-90%): Wayland/X11 protocols, UI widgets, effects
    - Lower coverage (<50%): render backend, integration tests (hardware-dependent)

**Without direnv:** Use `make test-go` instead of `go test ./...`. Direct `go test` requires `CGO_LDFLAGS` to link the Rust library (see [Troubleshooting](#troubleshooting)).

## Verify Static Linking

```bash
# Verify the binary has no dynamic dependencies
make check-static

# Expected output: "✓ Binary is fully statically linked."
```

## Run

```bash
./bin/wain
# Output:
#   render.Add(6, 7) = 13
#   render library version: 0.1.0

./bin/wain --version
# Output: wain version: 0.1.0

./bin/wain --help
# Shows usage information
```

## Demonstration Binaries

The project includes several demonstration binaries that exercise different subsystems:

| Binary              | Make Target          | CGO Required | Description                                             |
|---------------------|----------------------|--------------|---------------------------------------------------------|
| `bin/wain`          | `make build`         | Yes          | Phase 0 validation (Go → Rust linkage)                  |
| `bin/wayland-demo`  | `make wayland-demo`  | No           | Wayland protocol + rasterizer + widgets demo             |
| `bin/x11-demo`      | `make x11-demo`      | No           | X11 protocol + rasterizer + widgets demo                 |
| `bin/widget-demo`   | `make widget-demo`   | Yes          | Interactive widget demo (auto-detects X11/Wayland)       |
| `bin/x11-dmabuf-demo` | `make x11-dmabuf-demo` | Yes      | X11 DRI3/Present GPU buffer sharing demo                 |
| `bin/dmabuf-demo`   | `make dmabuf-demo`   | Yes          | Wayland DMA-BUF GPU buffer sharing demo                  |
| `bin/double-buffer-demo` | `make double-buffer-demo` | Yes  | Phase 5.3 double/triple buffering with compositor sync  |
| `bin/gpu-triangle-demo` | `make gpu-triangle-demo` | Yes  | GPU command submission triangle rendering demo           |
| `bin/amd-triangle-demo` | `make amd-triangle-demo` | Yes  | Phase 6.4 AMD GPU detection and RDNA backend demo        |
| `bin/auto-render-demo` | `make auto-render-demo` | Yes   | Phase 7.1 automatic backend selection with fallback      |
| `bin/clipboard-demo` | `make clipboard-demo` | No         | Phase 8.2 clipboard protocol demo (X11/Wayland)          |
| `bin/decorations-demo` | `make decorations-demo` | Yes    | Phase 8.3 client-side window decorations demo            |
| `bin/perf-demo`     | `make perf-demo`     | Yes          | GPU performance profiling with frame time measurements   |
| `bin/shader-test`   | `make shader-test`   | Yes          | Phase 4.6 shader compilation test for all 7 UI shaders   |
| `bin/gen-atlas`     | `make gen-atlas`     | Yes          | SDF font atlas generator tool                            |

## Architecture

The project consists of five layers (bottom-up):

### 1. Rust Rendering Library (`render-sys/`, ~5,372 LOC)

```
render-sys/src/
├── lib.rs          → C ABI exports (render_add, render_version, buffer_*, render_*)
├── drm.rs          → DRM device access (open, ioctl wrappers)
├── i915.rs         → i915 GPU driver (GEM create/close, exec, context, mmap)
├── xe.rs           → Xe GPU driver (VM create, exec queue, exec, mmap)
├── detect.rs       → GPU generation detection (Gen9/Gen11/Gen12/Xe)
├── allocator.rs    → Buffer allocation with tiling (None/X/Y)
├── slab.rs         → Slab sub-allocator for GPU buffers
├── batch.rs        → Batch buffer builder with relocation support
├── cmd/            → Intel 3D pipeline command encoding (MI, state, primitive)
├── pipeline.rs     → Pre-baked pipeline state configurations
├── surface.rs      → Surface state and sampler state encoding
└── shader.rs       → WGSL/GLSL shader frontend via naga
```

- **Dependencies:** nix 0.27 (ioctl), naga 0.14 (WGSL/GLSL parsing)
- **Build:** Compiled as `staticlib` with musl target for static linking
- **C ABI exports:** `render_add`, `render_version`, `render_detect_gpu`, `buffer_allocator_create`, `buffer_allocator_destroy`, `buffer_allocate`, `buffer_export_dmabuf`, `buffer_get_info`, `buffer_get_handle`, `buffer_destroy`, `render_submit_batch`, `render_create_context`

### 2. Go Bindings (`internal/render/`)

Provides Go wrappers for all Rust C ABI exports:
- `render.Add`, `render.Version` — ABI smoke tests
- `render.DetectGPU` — GPU generation detection
- `render.NewAllocator`, `Allocator.Allocate`, `Allocator.ExportDmabuf` — GPU buffer management
- `render.CreateContext`, `render.SubmitBatch` — GPU command submission

### 3. Protocol Layer (`internal/wayland/`, `internal/x11/`)

```
Protocol Implementations (~6,280 LOC)
├── Wayland Client (7 packages, ~3,392 LOC)
│   ├── wire/        → Binary marshaling + fd passing
│   ├── socket/      → Unix domain socket + SCM_RIGHTS
│   ├── client/      → Display, Registry, Compositor, Surface
│   ├── shm/         → Shared memory buffers (memfd)
│   ├── xdg/         → Window management (xdg-shell)
│   ├── input/       → Seat, Pointer, Keyboard (hardcoded QWERTY)
│   └── dmabuf/      → DMA-BUF buffer sharing (linux-dmabuf protocol)
└── X11 Client (7 packages, ~2,888 LOC)
    ├── wire/        → Request/reply/event encoding, extension queries
    ├── client/      → Connection, CreateWindow, MapWindow, extension support
    ├── events/      → KeyPress, Button, Motion events
    ├── gc/          → Graphics context, PutImage
    ├── shm/         → MIT-SHM extension (zero-copy image transfers)
    ├── dri3/        → DRI3 extension (GPU buffer sharing via DMA-BUF)
    └── present/     → Present extension (frame synchronization)
```

### 4. Rendering Layer (`internal/raster/`)

```
Software 2D Rasterizer (~1,877 LOC)
├── core/        → Rectangles, rounded rects, lines
├── curves/      → Quadratic/cubic Bezier, arc fills
├── composite/   → Alpha blending, image filtering
├── effects/     → Box shadow, gradients
└── text/        → SDF-based text rendering
```

### 5. UI Framework (`internal/ui/`)

```
Widget Layer (~1,503 LOC)
├── layout/      → Flexbox-like Row/Column layout
├── pctwidget/   → Percentage-based sizing
└── widgets/     → Button, TextInput, ScrollContainer
```

**Key constraint:** The final binary must be fully static (no libc dependency) to support deployment without system dependencies. This is enforced via:
- Rust compiled with `x86_64-unknown-linux-musl` (or `aarch64-unknown-linux-musl`) target
- Go compiled with `musl-gcc` and `-extldflags '-static'`
- GCC 14+ compatibility stub (`internal/render/dl_find_object_stub.c`) for musl builds
- Verification: `ldd bin/wain` reports "not a dynamic executable"

## Project Structure

```
wain/
├── cmd/
│   ├── wain/              # Phase 0 validation binary
│   ├── demo/              # Phase 1 X11 rendering pipeline demo
│   ├── wayland-demo/      # Wayland protocol demo
│   ├── x11-demo/          # X11 protocol demo
│   ├── widget-demo/       # Interactive widget demo (X11/Wayland)
│   ├── x11-dmabuf-demo/   # X11 DRI3 GPU buffer sharing demo
│   ├── dmabuf-demo/       # Wayland DMA-BUF GPU buffer sharing demo
│   ├── gpu-triangle-demo/ # GPU command submission triangle demo
│   └── gen-atlas/         # SDF font atlas generator tool
├── internal/
│   ├── render/            # Go CGO bindings to Rust (binding.go, dmabuf.go)
│   ├── wayland/           # Wayland protocol client (7 packages)
│   ├── x11/               # X11 protocol client (7 packages)
│   ├── raster/            # Software 2D rasterizer (5 packages)
│   ├── ui/                # Widget layer + layout (3 packages)
│   ├── buffer/            # Frame buffer ring for double/triple buffering
│   ├── demo/              # Shared utilities for demo binaries
│   └── integration/       # End-to-end integration tests
├── render-sys/            # Rust static library (C ABI exports)
│   ├── Cargo.toml         # Rust package definition (staticlib, nix, naga)
│   └── src/               # Rust source (16 files, ~5,372 LOC)
├── Makefile               # Build automation (enforces static linking)
├── ROADMAP.md             # 8-phase implementation plan
├── RECOMMENDED_LIBRARIES.md # Approved dependencies reference
├── go.mod                 # Go module: github.com/opd-ai/wain (Go 1.24)
├── .github/workflows/ci.yml # CI: build + test + static linkage verification
└── LICENSE                # MIT License
```

## Manual Build (without Makefile)

If you need to build manually (e.g., for debugging the build process):

```bash
# 1. Build the Rust static library
cargo build --release \
  --target x86_64-unknown-linux-musl \
  --manifest-path render-sys/Cargo.toml

# 2. Compile the GCC 14+ / musl compatibility stub
musl-gcc -c -o internal/render/dl_find_object_stub.o \
  internal/render/dl_find_object_stub.c

# 3. Build the Go binary
RUST_LIB="render-sys/target/x86_64-unknown-linux-musl/release/librender.a"
DL_STUB="internal/render/dl_find_object_stub.o"
CC=musl-gcc CGO_ENABLED=1 \
  CGO_LDFLAGS="${PWD}/${RUST_LIB} ${PWD}/${DL_STUB} -ldl -lm -lpthread" \
  CGO_LDFLAGS_ALLOW=".*" \
  go build -ldflags "-extldflags '-static'" -o bin/wain ./cmd/wain

# 4. Verify
ldd bin/wain  # should print "not a dynamic executable"
```

## Font Atlas Generation

The text rasterizer (`internal/raster/text/`) uses SDF (Signed Distance Field) font rendering with a pre-baked atlas embedded in the binary. The `gen-atlas` tool generates this atlas.

**Building the tool:**
```bash
make gen-atlas
# Output: ./bin/gen-atlas
```

**Running the generator:**
```bash
./bin/gen-atlas > atlas.bin
# Generates: 256x256 SDF atlas covering ASCII printable chars (0x20-0x7E)
```

**Atlas format:**
- 256×256 grayscale bitmap (65,536 bytes)
- 16×16 glyph grid, each cell is 16×16 pixels
- Contains 95 printable ASCII characters plus 1 replacement glyph (□)
- Binary format: raw uint8 array + metadata (rune, position, metrics)

You only need to regenerate the atlas if you change the supported character set, glyph size, atlas dimensions, or font rendering algorithm.

## Troubleshooting

### `make build` fails with "musl-gcc not found"

Install the musl C compiler (see [Prerequisites](#prerequisites) section above).

### `go test ./...` fails with linker errors

Go tests require `CGO_LDFLAGS` to be set. You have two options:

**Option 1: Use direnv (recommended for daily development)**
```bash
# Install direnv: https://direnv.net/
# Then allow the project's .envrc file:
direnv allow

# Now go test works directly:
go test ./...
```

The `.envrc` file automatically configures `CGO_LDFLAGS` when entering the project directory.

**Option 2: Use make wrapper**
```bash
make test-go
```

The Makefile sets the required CGO flags and ensures dependencies are built.

### Binary is not static (has dynamic dependencies)

Verify you are using:
- Rust musl target: `rustup show` should list `x86_64-unknown-linux-musl`
- musl-gcc: `which musl-gcc` should return a path
- Static ldflags: Check `go build -x` output for `-extldflags '-static'`

Run `make check-static` to verify the binary is fully static.

## Contributing

See [ROADMAP.md](ROADMAP.md) for the complete 8-phase plan toward full GPU rendering.

**Development commands:**
```bash
make build         # Build fully static binary
make test          # Run all tests (Rust + Go)
make test-rust     # Run Rust tests only
make test-go       # Run Go tests only
make coverage      # Run Go tests with coverage reporting
make coverage-html # Generate HTML coverage report
make check-static  # Verify static linkage
make clean         # Remove build artifacts
```

**Priority contributions (Phase 4.2+):**
1. **Shader lowering** — Compile naga IR to Intel EU machine code
2. **UI shader authoring** — Write WGSL vertex/fragment shaders for common draw types
3. **GPU rendering backend** — Wire GPU command submission into the display pipeline
4. **AMD GPU support** — RDNA ISA backend for AMD GPUs

## License

MIT License — see [LICENSE](LICENSE) file.
