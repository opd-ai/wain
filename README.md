# wain

[![CI](https://img.shields.io/github/actions/workflow/status/opd-ai/wain/ci.yml?branch=main&label=CI)](https://github.com/opd-ai/wain/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod-go-version/opd-ai/wain)](go.mod)

Wain is a statically-compiled Go UI toolkit that links a Rust rendering
library (via CGO and musl) for GPU-accelerated graphics on Linux. It
implements Wayland and X11 display protocols from scratch, provides a
software 2D rasterizer, and produces a single fully-static binary with
zero runtime dependencies.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [Demonstration Binaries](#demonstration-binaries)
- [Testing](#testing)
- [Font Atlas Generation](#font-atlas-generation)
- [Troubleshooting](#troubleshooting)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Go–Rust Static Linking** — CGO bridge to a Rust `staticlib`,
  producing a single binary with no dynamic dependencies
  (`render-sys/src/lib.rs`, `internal/render/binding.go`)
- **Wayland Client** — 9-package implementation covering wire format,
  SHM buffers, xdg-shell window management, pointer/keyboard input,
  DMA-BUF buffer sharing, data-device clipboard, and output handling
  (`internal/wayland/`)
- **X11 Client** — 9-package implementation covering connection setup,
  window operations, graphics context, MIT-SHM, DRI3 GPU buffer
  sharing, Present frame sync, DPI scaling, and selection/clipboard
  (`internal/x11/`)
- **Software 2D Rasterizer** — Filled and rounded rectangles,
  anti-aliased lines, quadratic/cubic Bézier curves, arc fills,
  SDF text rendering, box shadows, gradients, and Porter-Duff alpha
  compositing (`internal/raster/`)
- **UI Widget Layer** — Flexbox-like Row/Column layout engine,
  percentage-based sizing, Button/TextInput/ScrollContainer widgets,
  client-side window decorations, and DPI-aware scaling
  (`internal/ui/`)
- **GPU Buffer Infrastructure** — DRM/KMS ioctl wrappers for Intel
  i915 and Xe drivers, GPU buffer allocation with tiling, DMA-BUF
  export, and slab sub-allocation (`render-sys/src/allocator.rs`,
  `render-sys/src/slab.rs`)
- **GPU Command Submission** — Batch buffer construction, Intel 3D
  pipeline command encoding (MI commands, state, vertex, primitive),
  pipeline state objects, and surface/sampler state encoding
  (`render-sys/src/batch.rs`, `render-sys/src/cmd/`)
- **Shader Frontend** — WGSL and GLSL shader parsing and validation
  via naga 0.14; 7 WGSL shaders for common UI draw operations
  (`render-sys/src/shader.rs`, `render-sys/shaders/`)
- **Intel EU Backend** — Register allocator, instruction lowering,
  and 128-bit binary encoding for Gen9+ execution units
  (`render-sys/src/eu/`)
- **AMD RDNA Backend** — RDNA instruction set, register allocation,
  encoding, and PM4 command stream for AMD GPUs
  (`render-sys/src/rdna/`, `render-sys/src/amd.rs`,
  `render-sys/src/pm4.rs`)
- **Public API** — `App` type with display server auto-detection
  (Wayland preferred, X11 fallback), renderer auto-detection (Intel →
  AMD → software), window management, event dispatching, widget tree,
  canvas drawing, color, and resource management (`app.go`,
  `event.go`, `widget.go`, `publicwidget.go`, `resource.go`,
  `color.go`, `dispatcher.go`, `render.go`)
- **Frame Buffering** — Double/triple buffer ring with compositor
  synchronization (`internal/buffer/`)
- **Display List Rendering** — GPU backend with display list consumer,
  texture atlas management, and frame presentation
  (`internal/render/backend/`, `internal/render/atlas/`,
  `internal/render/display/`, `internal/render/present/`)

## Requirements

- **Go 1.24** or later
- **Rust (stable)** with the musl target for your architecture
- **musl C compiler** (`musl-gcc`)
- **Linux** (Wayland or X11 display server)

Install prerequisites:

```bash
# Go: https://go.dev/dl/
go version  # verify 1.24+

# Rust musl target
rustup target add x86_64-unknown-linux-musl
# For ARM64: rustup target add aarch64-unknown-linux-musl

# musl C compiler
sudo apt-get install musl-tools    # Ubuntu / Debian
sudo dnf install musl-gcc          # Fedora / RHEL
sudo pacman -S musl                # Arch Linux
apk add musl-dev                   # Alpine Linux
```

For macOS cross-compilation:

```bash
brew install FiloSottile/musl-cross/musl-cross
make build CC=x86_64-linux-musl-gcc
```

## Installation

### Build from Source

```bash
git clone https://github.com/opd-ai/wain.git
cd wain
make build
```

This checks dependencies, builds the Rust static library with the musl
target, compiles the GCC 14+ musl compatibility stub, and produces a
fully-static Go binary at `./bin/wain`. The Makefile auto-detects the
host architecture via `rustc -vV`.

### Use as a Go Dependency

```bash
go get github.com/opd-ai/wain
```

To rebuild the Rust backend from source, use the `wain-build` helper
(`cmd/wain-build/`):

```bash
go install github.com/opd-ai/wain/cmd/wain-build@latest
wain-build
go build .
```

### Go Generate Workflow

As an alternative to `make`, build using Go's native workflow:

```bash
# Ensure RUST_MUSL_TARGET matches the target used by scripts/build-rust.sh / the Makefile,
# for example: export RUST_MUSL_TARGET=x86_64-unknown-linux-musl
go generate ./...
CC=musl-gcc CGO_ENABLED=1 \
  CGO_LDFLAGS="$(pwd)/render-sys/target/${RUST_MUSL_TARGET}/release/librender_sys.a \
  $(pwd)/internal/render/dl_find_object_stub.o -ldl -lm -lpthread" \
  CGO_LDFLAGS_ALLOW=".*" \
  go build -ldflags "-extldflags '-static'" -o bin/wain ./cmd/wain
```

The `go generate` step (`internal/render/generate.go`) calls
`scripts/build-rust.sh`, which checks for required tools, detects the
host architecture, and builds both the Rust library and the musl stub.

## Usage

Run the Phase 0 validation binary to verify the Go–Rust link:

```bash
./bin/wain
# Output:
#   render.Add(6, 7) = 13
#   render library version: 0.1.0

./bin/wain --version
# Output: wain version: 0.1.0
```

Create a window using the public API (`app.go`):

```go
package main

import "github.com/opd-ai/wain"

func main() {
    app := wain.NewApp()
    app.Run() // blocks until app.Quit() is called
}
```

Build and run the Wayland or X11 demo (no CGO required):

```bash
make wayland-demo
./bin/wayland-demo

make x11-demo
./bin/x11-demo
```

Verify the binary is fully statically linked:

```bash
make check-static
# Expected output: "✓ Binary is fully statically linked."
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CC` | `musl-gcc` | musl C compiler; override for cross toolchains |
| `CARGO_FLAGS` | *(empty)* | Extra flags passed to `cargo` commands |
| `CGO_ENABLED` | `1` | Required for binaries linking the Rust library |
| `CGO_LDFLAGS` | *(set by Makefile)* | Paths to `librender_sys.a`, stub `.o`, and `-ldl -lm -lpthread` |
| `CGO_LDFLAGS_ALLOW` | `.*` | Regex allowing CGO linker flags |

Cross-compilation example:

```bash
make build CC=x86_64-linux-musl-gcc
```

The `.envrc` file auto-configures `CC`, `CGO_ENABLED`, `CGO_LDFLAGS`,
and `CGO_LDFLAGS_ALLOW` when used with
[direnv](https://direnv.net/). After installing direnv, hooking it into
your shell (see [direnv setup](https://direnv.net/docs/hook.html)), and
running `direnv allow`, the environment variables will auto-load when
you `cd` into this directory. Once loaded, standard `go test ./...` and
`go build` commands work without additional flags.

**Without direnv**, you must use `make test-go` and `make build` instead
of `go test ./...` and `go build`, as these Makefile targets set the
required `CGO_LDFLAGS` to link the Rust static library.

## Project Structure

```text
wain/
├── app.go                     # Public API: App, Window, AppConfig
├── event.go                   # Event types: Pointer, Key, Touch, Window, Custom
├── widget.go                  # Widget interface and BaseWidget
├── publicwidget.go            # PublicWidget, Container, Canvas drawing API
├── resource.go                # Font/Image resource management
├── color.go                   # Color type with RGB/RGBA constructors
├── dispatcher.go              # EventDispatcher and FocusManager
├── render.go                  # RenderBridge for display list rendering
├── doc.go                     # Package documentation
├── cmd/
│   ├── wain/                  # Phase 0 validation binary (Go → Rust linkage)
│   ├── wain-build/            # Helper tool to rebuild Rust backend from source
│   ├── wain-demo/             # Public API demo (display server auto-detection)
│   ├── event-demo/            # Event handling demonstration
│   ├── window-demo/           # Public Window API demonstration
│   ├── resource-demo/         # Resource management API demonstration
│   ├── wayland-demo/          # Wayland protocol + rasterizer + widgets demo
│   ├── x11-demo/              # X11 protocol + rasterizer + widgets demo
│   ├── widget-demo/           # Interactive widget demo (X11/Wayland)
│   ├── decorations-demo/      # Client-side window decorations demo
│   ├── clipboard-demo/        # Clipboard protocol demo (X11/Wayland)
│   ├── dmabuf-demo/           # Wayland DMA-BUF GPU buffer sharing demo
│   ├── x11-dmabuf-demo/       # X11 DRI3 GPU buffer sharing demo
│   ├── double-buffer-demo/    # Double/triple buffering with compositor sync
│   ├── gpu-triangle-demo/     # GPU command submission triangle demo
│   ├── gpu-display-demo/      # End-to-end GPU rendering to display pipeline
│   ├── amd-triangle-demo/     # AMD GPU detection and RDNA backend demo
│   ├── auto-render-demo/      # Automatic backend selection with fallback
│   ├── perf-demo/             # GPU performance profiling demo
│   ├── shader-test/           # Shader compilation test for all 7 UI shaders
│   └── gen-atlas/             # SDF font atlas generator tool
├── internal/
│   ├── render/                # Go CGO bindings to Rust rendering library
│   │   ├── atlas/             # Texture atlas management
│   │   ├── backend/           # GPU backend and display list consumer
│   │   ├── display/           # Framebuffer and display integration
│   │   └── present/           # Frame presentation
│   ├── wayland/               # Wayland protocol client (9 packages)
│   │   ├── wire/              # Binary marshaling and fd passing
│   │   ├── socket/            # Unix domain socket and SCM_RIGHTS
│   │   ├── client/            # Display, Registry, Compositor, Surface
│   │   ├── shm/               # Shared memory buffers (memfd)
│   │   ├── xdg/               # Window management (xdg-shell)
│   │   ├── input/             # Seat, Pointer, Keyboard
│   │   ├── dmabuf/            # DMA-BUF buffer sharing
│   │   ├── datadevice/        # Data device manager (clipboard/DnD)
│   │   └── output/            # Output configuration and mode handling
│   ├── x11/                   # X11 protocol client (9 packages)
│   │   ├── wire/              # Request/reply/event encoding
│   │   ├── client/            # Connection, window, extension support
│   │   ├── events/            # KeyPress, Button, Motion events
│   │   ├── gc/                # Graphics context, PutImage
│   │   ├── shm/               # MIT-SHM extension
│   │   ├── dri3/              # DRI3 extension (GPU buffer sharing)
│   │   ├── present/           # Present extension (frame sync)
│   │   ├── dpi/               # DPI detection and scaling
│   │   └── selection/         # Selection and clipboard handling
│   ├── raster/                # Software 2D rasterizer (7 packages)
│   │   ├── core/              # Rectangles, rounded rects, lines
│   │   ├── curves/            # Bézier curves, arc fills
│   │   ├── composite/         # Alpha blending, image filtering
│   │   ├── effects/           # Box shadow, gradients
│   │   ├── text/              # SDF-based text rendering
│   │   ├── displaylist/       # Display list construction
│   │   └── consumer/          # Display list execution
│   ├── ui/                    # UI widget framework (5 packages)
│   │   ├── layout/            # Flexbox-like Row/Column layout
│   │   ├── pctwidget/         # Percentage-based sizing
│   │   ├── widgets/           # Button, TextInput, ScrollContainer
│   │   ├── decorations/       # Client-side window decorations
│   │   └── scale/             # DPI-aware scaling
│   ├── buffer/                # Frame buffer ring (double/triple buffering)
│   ├── demo/                  # Shared utilities for demo binaries
│   └── integration/           # End-to-end integration tests
├── render-sys/                # Rust static library (C ABI exports)
│   ├── Cargo.toml             # Rust dependencies: nix 0.27, naga 0.14
│   ├── shaders/               # 7 WGSL shaders for UI rendering
│   └── src/                   # 32 Rust source files
│       ├── lib.rs             # C ABI exports
│       ├── drm.rs             # DRM device access
│       ├── i915.rs            # Intel i915 GPU driver
│       ├── xe.rs              # Intel Xe GPU driver
│       ├── detect.rs          # GPU generation detection
│       ├── allocator.rs       # GPU buffer allocation
│       ├── slab.rs            # Slab sub-allocator
│       ├── batch.rs           # Batch buffer builder
│       ├── pipeline.rs        # Pipeline state objects
│       ├── surface.rs         # Surface/sampler state encoding
│       ├── shader.rs          # WGSL/GLSL shader frontend (naga)
│       ├── shaders.rs         # Shader source embedding
│       ├── amd.rs             # AMD GPU driver wrappers
│       ├── pm4.rs             # AMD PM4 command stream
│       ├── cmd/               # Intel 3D pipeline commands (5 files)
│       ├── eu/                # Intel EU backend (6 files)
│       └── rdna/              # AMD RDNA backend (6 files)
├── scripts/                   # Build and analysis scripts
│   ├── build-rust.sh          # Rust library build (called by go generate)
│   ├── compute-coverage.sh    # Test coverage computation
│   └── analyze_godoc.go       # Documentation analysis
├── Makefile                   # Build automation (enforces static linking)
├── .github/workflows/ci.yml   # CI: build, test, and static linkage check
├── .envrc                     # direnv configuration for CGO flags
├── .golangci.yml              # Go linter configuration
├── .editorconfig              # Editor formatting rules
├── go.mod                     # Go module (github.com/opd-ai/wain, Go 1.24)
└── LICENSE                    # MIT License
```

## Architecture

The project is organized in five layers (bottom-up):

### Layer 1: Rust Rendering Library (`render-sys/`)

The `render-sys` crate (~14,400 lines total, ~9,900 lines of code)
compiles as a C-compatible static library (`staticlib`). It provides
DRM/KMS ioctl wrappers for Intel i915 and Xe GPU drivers, GPU buffer
allocation with tiling support, DMA-BUF export, batch buffer
construction with relocation, Intel 3D pipeline command encoding,
shader parsing via naga 0.14, Intel EU binary encoding for Gen9+,
and AMD RDNA instruction encoding.

**Rust dependencies:** nix 0.27 (ioctl), naga 0.14 (WGSL/GLSL parsing)

**C ABI exports:** `render_add`, `render_version`, `render_detect_gpu`,
`buffer_allocator_create`, `buffer_allocator_destroy`,
`buffer_allocate`, `buffer_export_dmabuf`, `buffer_get_info`,
`buffer_get_handle`, `buffer_destroy`, `render_submit_batch`,
`render_create_context`

### Layer 2: Go Render Bindings (`internal/render/`)

Go wrappers for all Rust C ABI exports, plus sub-packages for GPU
backend management (`backend/`), texture atlas management (`atlas/`),
display integration (`display/`), and frame presentation (`present/`).

### Layer 3: Protocol Layer (`internal/wayland/`, `internal/x11/`)

Two independent display-server implementations built from scratch
with no external Go dependencies. The Wayland client (9 packages)
covers wire format, shared memory, xdg-shell, input, DMA-BUF, data
device, and output handling. The X11 client (9 packages) covers
connection setup, window operations, graphics context, MIT-SHM, DRI3,
Present, DPI, and selection/clipboard.

### Layer 4: Software Rasterizer (`internal/raster/`)

A 7-package 2D rasterizer providing filled and rounded rectangles,
anti-aliased lines, Bézier curves, arc fills, SDF text rendering,
box shadows, gradients, Porter-Duff alpha compositing, display list
construction, and display list execution.

### Layer 5: UI Framework (`internal/ui/`)

A 5-package widget layer providing flexbox-like Row/Column layout,
percentage-based sizing, Button/TextInput/ScrollContainer widgets,
client-side window decorations (title bar, controls, resize handles),
and DPI-aware scaling.

**Static linking constraint:** The final binary must have zero dynamic
dependencies. This is enforced by compiling Rust with
`*-unknown-linux-musl` targets, building Go with `musl-gcc` and
`-extldflags '-static'`, and including a GCC 14+ compatibility stub
(`internal/render/dl_find_object_stub.c`). The CI pipeline verifies
static linkage via `ldd`.

## Demonstration Binaries

Binaries with a **Make Target** can be built using `make <target>`.
All other binaries can be built with `go build ./cmd/<name>` (with
appropriate CGO flags for those requiring CGO).

| Binary | Make Target | CGO | Description |
|--------|-------------|-----|-------------|
| `wain` | `make build` | Yes | Go → Rust linkage validation |
| `wayland-demo` | `make wayland-demo` | No | Wayland protocol + rasterizer + widgets |
| `x11-demo` | `make x11-demo` | No | X11 protocol + rasterizer + widgets |
| `widget-demo` | `make widget-demo` | Yes | Interactive widgets (auto-detects X11/Wayland) |
| `dmabuf-demo` | `make dmabuf-demo` | Yes | Wayland DMA-BUF GPU buffer sharing |
| `x11-dmabuf-demo` | `make x11-dmabuf-demo` | Yes | X11 DRI3 GPU buffer sharing |
| `double-buffer-demo` | `make double-buffer-demo` | Yes | Double/triple buffering with compositor sync |
| `gpu-triangle-demo` | `make gpu-triangle-demo` | Yes | GPU command submission triangle rendering |
| `gen-atlas` | `make gen-atlas` | Yes | SDF font atlas generator |
| `wain-demo` | `make wain-demo` | Yes | Public API demo (auto-detection) |
| `event-demo` | `make event-demo` | Yes | Event handling demonstration |
| `amd-triangle-demo` | — | Yes | AMD GPU detection and RDNA backend |
| `auto-render-demo` | — | Yes | Automatic backend selection with fallback |
| `clipboard-demo` | — | No | Clipboard protocol (X11/Wayland) |
| `decorations-demo` | — | No | Client-side window decorations |
| `gpu-display-demo` | — | Yes | End-to-end GPU rendering to display |
| `perf-demo` | — | Yes | GPU performance profiling |
| `resource-demo` | — | Yes | Resource management API |
| `shader-test` | — | Yes | Shader compilation for all 7 UI shaders |
| `window-demo` | — | Yes | Public Window API demonstration |
| `wain-build` | — | Yes | Rebuild Rust backend from source |

## Testing

### Quick Start with direnv

If you have [direnv](https://direnv.net/) installed and configured:

```bash
direnv allow          # one-time setup; allows .envrc to auto-load
cd ../ && cd -        # reload environment (or open a new shell in this directory)
go test ./...         # run all Go tests (works because CGO_LDFLAGS is set)
```

Note: direnv must be hooked into your shell for this to work. See
[direnv installation](https://direnv.net/docs/installation.html) for setup.

### Using Make Targets

```bash
make test             # run all tests (Rust + Go)
make test-rust        # run Rust tests only
make test-go          # run Go tests only
make test-visual      # run visual regression tests for rendering primitives
make coverage         # Go tests with per-package coverage summary
make coverage-html    # HTML coverage report at coverage/coverage.html
```

The project contains 61 test files across all packages.
Authoritative coverage is computed via `scripts/compute-coverage.sh`.

### Visual Regression Tests

The project includes automated screenshot comparison tests for all
rendering primitives (filled rectangles, rounded rectangles, lines,
text, gradients, and shadows). Visual tests generate reference images
on first run and compare subsequent renders against them with a
99.5% pixel match threshold:

```bash
make test-visual      # run visual regression tests
```

Reference images are stored in `internal/raster/testdata/`. If a test
fails, diff images are saved showing pixel-level differences (red =
different, green = matching).

**Without direnv** (or if direnv is not set up), you MUST use `make
test-go` instead of `go test ./...`. Direct `go test` requires
`CGO_LDFLAGS` to link the Rust library, which the Makefile sets
automatically but plain `go test` does not (see
[Troubleshooting](#troubleshooting)).

## Font Atlas Generation

The text rasterizer (`internal/raster/text/`) uses SDF (Signed
Distance Field) font rendering with a pre-baked glyph atlas. The
`gen-atlas` tool generates this atlas:

```bash
make gen-atlas
./bin/gen-atlas > atlas.bin
```

The output is a 256×256 grayscale bitmap (65,536 bytes) arranged as
a 16×16 glyph grid covering 95 printable ASCII characters (0x20–0x7E)
plus one replacement glyph. Regeneration is only needed when changing
the character set, glyph size, or atlas dimensions.

## Troubleshooting

### `make build` fails with "musl-gcc not found"

Install the musl C compiler for your platform
(see [Requirements](#requirements)).

### `go test ./...` fails with linker errors

**Cause:** Go tests require `CGO_LDFLAGS` to link the Rust static
library (`librender_sys.a`). Plain `go test ./...` does not set this
variable.

**Solution:**

```bash
# Option 1: Use the Makefile (always works)
make test-go

# Option 2: Use direnv (requires direnv installed and configured)
# Install direnv: https://direnv.net/docs/installation.html
# Hook into your shell: https://direnv.net/docs/hook.html
direnv allow                    # allow .envrc to load
cd ../ && cd -                  # reload environment in current shell
echo $CGO_LDFLAGS               # verify environment is loaded (should show paths)
go test ./...                   # now works

# Option 3: Set CGO_LDFLAGS manually (advanced)
# See internal/render/binding.go header comments for manual build steps
```

**Recommendation:** Use `make test-go` for reliability. The direnv
approach requires proper shell integration and may not work in CI/CD
environments or non-interactive shells.

### Binary has dynamic dependencies

Verify static linking prerequisites:

```bash
rustup show              # must list *-unknown-linux-musl target
which musl-gcc           # must return a valid path
make check-static        # verifies bin/wain is fully static
```

## Documentation

### Getting Started
- [GETTING_STARTED.md](GETTING_STARTED.md) — Step-by-step tutorial for
  building your first wain application
- [WIDGETS.md](WIDGETS.md) — Complete widget reference with examples
  and visual descriptions

### API & Architecture
- [API.md](API.md) — API reference for all internal packages
- [HARDWARE.md](HARDWARE.md) — Supported hardware matrix (Intel/AMD
  GPUs, kernel versions, display servers)
- [ROADMAP.md](ROADMAP.md) — 10-phase implementation plan

### Advanced Topics
- [ACCESSIBILITY.md](ACCESSIBILITY.md) — Accessibility support and
  AT-SPI2 implementation path
- [RECOMMENDED_LIBRARIES.md](RECOMMENDED_LIBRARIES.md) — Library
  selection rationale and static-compilation constraints
- [RELEASE.md](RELEASE.md) — Release process documentation
- [ANDROID_PORT_FEASIBILITY.md](ANDROID_PORT_FEASIBILITY.md) —
  Android porting analysis
- [render-sys/shaders/README.md](render-sys/shaders/README.md) —
  WGSL shader documentation with usage examples

## Contributing

See [ROADMAP.md](ROADMAP.md) for the complete 8-phase plan.

**Development commands:**

```bash
make build             # build fully static binary
make test              # run all tests (Rust + Go)
make test-rust         # run Rust tests only
make test-go           # run Go tests only
make coverage          # Go tests with coverage reporting
make coverage-html     # HTML coverage report
make check-static      # verify static linkage
make stats             # lines-of-code summary
make clean             # remove build artifacts
```

**Code style:** Go code uses tabs (see `.editorconfig`); Rust code uses
4-space indentation. The project lints Go with
[golangci-lint](https://golangci-lint.run/) (configuration in
`.golangci.yml`).

## License

[MIT](LICENSE) — Copyright (c) 2026 opdai
