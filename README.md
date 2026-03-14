# Wain

[![CI](https://github.com/opd-ai/wain/actions/workflows/ci.yml/badge.svg)](https://github.com/opd-ai/wain/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod-go-version/opd-ai/wain)](go.mod)

Wain is a statically-compiled Go UI toolkit for Linux that renders via a
Rust GPU backend with automatic software fallback. It implements the
Wayland and X11 display protocols directly — producing fully static,
zero-dependency binaries that run on any Linux distribution.

## Table of Contents

- [Why Wain?](#why-wain)
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Examples](#examples)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Why Wain?

Most Go UI toolkits — including Fyne and Gio — require CGO and depend on
system shared libraries (e.g., OpenGL, X11, Wayland client libs) that
**cannot** be statically linked on Linux. Wain takes a different approach:

| Toolkit | Static Linking | Zero Deps | Native Look | Cross-Platform | Pure Go |
|---------|---------------|-----------|-------------|----------------|---------|
| **Wain** | ✅ Yes (musl) | ✅ Yes | No (custom) | Linux only | Go + Rust |
| **Fyne** | ❌ No | ❌ No | No (custom) | Yes | No (CGO) |
| **Gio** | ❌ No | ❌ No | No (custom) | Yes | No (CGO) |
| **GTK** | ❌ No | ❌ No | Yes (Linux) | Partial | No (CGO) |

Wain implements Wayland and X11 display protocols directly and submits
GPU commands to kernel DRM interfaces — no OpenGL, Vulkan, or system
graphics libraries required. The result is a single, fully static binary
with zero runtime dependencies.

## Features

- **Display Server Auto-Detection** — connects to Wayland when available,
  falls back to X11 (`app.go`)
- **GPU Renderer Auto-Detection** — probes Intel (Gen9–Xe) and AMD (RDNA 1–3)
  GPUs, falls back to software rasterization (`internal/render/backend/`)
- **Fully Static Binaries** — links against musl libc and a Rust static
  library; output binaries have zero runtime dependencies (`Makefile`)
- **Widget System** — Button, Label, TextInput, ScrollView, ImageWidget,
  Spacer with percentage-based sizing (`concretewidgets.go`, `layout.go`)
- **Layout Containers** — Row, Column, Stack, Grid, and Panel with
  flexbox-style alignment, padding, and gap (`layout.go`)
- **Software Rasterizer** — rectangles, rounded rectangles, anti-aliased
  lines, Bézier curves, gradients, shadows, and SDF text
  (`internal/raster/`)
- **GPU Command Submission** — Intel i915/Xe and AMD RDNA batch command
  generation with DMA-BUF export (`render-sys/src/`)
- **Shader Compilation** — WGSL shaders compiled to Intel EU and AMD RDNA
  native ISA via naga (`render-sys/src/eu/`, `render-sys/src/rdna/`)
- **Wayland Protocol** — compositor connection, `wl_shm`, `xdg_shell`,
  input, clipboard, DMA-BUF, and output handling
  (`internal/wayland/`)
- **X11 Protocol** — server connection, windows, DRI3, Present, MIT-SHM,
  clipboard, drag-and-drop, and HiDPI detection (`internal/x11/`)
- **AT-SPI2 Accessibility** — D-Bus screen reader integration with
  Accessible, Component, Action, and Text interfaces (`internal/a11y/`);
  requires `-tags=atspi` (see [ACCESSIBILITY.md](./ACCESSIBILITY.md))
- **Theming** — DefaultDark, DefaultLight, and HighContrast built-in themes
  (`theme.go`)
- **Clipboard** — read/write clipboard on both Wayland and X11
  (`clipboard.go`)
- **Animations** — keyframe animation system with easing functions
  (`animate.go`)
- **Client-Side Decorations** — title bar and resize handles
  (`internal/ui/decorations/`)
- **HiDPI Support** — automatic scale factor detection on both Wayland and
  X11 (`internal/ui/scale/`, `internal/x11/dpi/`)
- **Double/Triple Buffering** — frame synchronization with compositor
  (`internal/buffer/`)

## Requirements

| Requirement | Minimum Version | Notes |
|-------------|-----------------|-------|
| Linux | Kernel 4.17+ | Wayland or X11 display server |
| Go | 1.24 | Set in `go.mod` |
| Rust (stable) | — | For building `render-sys` from source |
| Cargo | — | Installed with Rust |
| musl-gcc | — | Static linking toolchain |
| musl Rust target | — | `rustup target add x86_64-unknown-linux-musl` |

### Go Dependencies

| Module | Version | Purpose |
|--------|---------|---------|
| `golang.org/x/sys` | v0.27.0 | Linux system call access |
| `github.com/godbus/dbus/v5` | v5.2.2 | AT-SPI2 accessibility over D-Bus |

### Rust Dependencies (render-sys)

| Crate | Version | Purpose |
|-------|---------|---------|
| `nix` | 0.27 | DRM `ioctl` interface |
| `naga` | 0.14 | WGSL/GLSL shader parsing and validation |

## Installation

1. Install Go 1.24+ from <https://go.dev/dl/>

2. Install the Rust toolchain:

   ```bash
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   ```

3. Add the musl Rust target for your architecture:

   ```bash
   rustup target add x86_64-unknown-linux-musl     # x86_64
   rustup target add aarch64-unknown-linux-musl     # ARM64
   ```

4. Install musl-gcc:

   ```bash
   sudo apt-get install musl-tools    # Ubuntu / Debian
   sudo dnf install musl-gcc          # Fedora
   sudo pacman -S musl                # Arch Linux
   ```

5. Add wain to your Go project:

   ```bash
   go get github.com/opd-ai/wain
   ```

6. Generate the Rust static library and CGO linker flags:

   ```bash
   go generate ./...
   ```

After `go generate` completes, standard `go build` and `go test` commands
work without additional flags.

## Usage

### Minimal Application

Create a window with a button and a label:

```go
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

func main() {
	app := wain.NewApp()

	app.Notify(func() {
		win, err := app.NewWindow(wain.WindowConfig{
			Title:       "Hello, wain!",
			Width:       400,
			Height:      200,
			Decorations: true,
		})
		if err != nil {
			log.Fatal(err)
		}
		win.OnClose(func() { app.Quit() })

		col := wain.NewColumn()
		col.SetPadding(20)
		col.SetGap(10)

		label := wain.NewLabel("Press the button.", wain.Size{Width: 100, Height: 30})
		col.Add(label)

		btn := wain.NewButton("Click Me", wain.Size{Width: 50, Height: 20})
		btn.OnClick(func() {
			fmt.Println("Button clicked!")
			label.SetText("Hello, wain!")
		})
		col.Add(btn)

		win.SetLayout(col)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Custom Configuration

Override default settings with `AppConfig`:

```go
app := wain.NewAppWithConfig(wain.AppConfig{
	Width:         1024,
	Height:        768,
	ForceSoftware: true,  // skip GPU detection
	Verbose:       true,  // log backend selection
})
```

### Grid Layout

Arrange widgets in a grid:

```go
grid := wain.NewGrid(3) // 3 columns
for i := range 6 {
	grid.Add(wain.NewLabel(
		fmt.Sprintf("Cell %d", i+1),
		wain.Size{Width: 33, Height: 50},
	))
}
win.SetLayout(grid)
```

## Configuration

### AppConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Width` | `int` | `800` | Initial window width in pixels |
| `Height` | `int` | `600` | Initial window height in pixels |
| `ForceSoftware` | `bool` | `false` | Force software rendering, skip GPU detection |
| `ForceX11` | `bool` | `false` | Force X11, skip Wayland detection |
| `DRMPath` | `string` | `"/dev/dri/renderD128"` | DRM device path for GPU detection |
| `Verbose` | `bool` | `false` | Log backend and display server selection |

### WindowConfig Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Title` | `string` | `""` | Window title bar text |
| `Width` | `int` | `800` | Initial window width in pixels |
| `Height` | `int` | `600` | Initial window height in pixels |
| `MinWidth` | `int` | `0` | Minimum window width (0 = no minimum) |
| `MinHeight` | `int` | `0` | Minimum window height (0 = no minimum) |
| `MaxWidth` | `int` | `0` | Maximum window width (0 = no maximum) |
| `MaxHeight` | `int` | `0` | Maximum window height (0 = no maximum) |
| `Fullscreen` | `bool` | `false` | Start in fullscreen mode |
| `Decorations` | `bool` | `true` | Show window decorations (title bar, borders) |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WAYLAND_DISPLAY` | (system) | Wayland compositor socket; presence triggers Wayland mode |
| `DISPLAY` | (system) | X11 display; used when Wayland is unavailable |

### Makefile Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CC` | `musl-gcc` | musl C compiler for CGO static linking |
| `CARGO_FLAGS` | (empty) | Extra flags passed to `cargo build` and `cargo test` |

## Project Structure

```
wain/
├── app.go                  # App, Window, event loop, display/renderer auto-detection
├── concretewidgets.go      # Button, Label, TextInput, ScrollView, ImageWidget, Spacer
├── layout.go               # Panel, Row, Column, Stack, Grid, Size, Align
├── event.go                # PointerEvent, KeyEvent, TouchEvent, WindowEvent, DragEvent
├── theme.go                # Theme, DefaultDark, DefaultLight, HighContrast
├── render.go               # Rendering bridge to internal raster/GPU backends
├── clipboard.go            # Clipboard read/write (Wayland and X11)
├── accessibility.go        # AT-SPI2 screen reader integration
├── animate.go              # Animation system with easing functions
├── resource.go             # Image and Font resource types
├── color.go                # Color type with RGB constructor
├── dispatcher.go           # EventDispatcher and FocusManager
├── publicwidget.go         # PublicWidget and Container interfaces
├── cmd/                    # Executable binaries (wain, demos, tools)
│   ├── wain/               # Main entry point binary
│   ├── wain-build/         # Build tool for Rust library
│   ├── example-app/        # Complete reference application
│   ├── widget-demo/        # Interactive widget demonstration
│   ├── bench/              # Software rendering benchmarks
│   └── ...                 # 18 additional demo and tool binaries
├── example/                # Standalone examples using only the public API
│   ├── hello/              # Minimal hello-world application
│   └── multi-window/       # Multiple windows demonstration
├── internal/
│   ├── a11y/               # AT-SPI2 accessibility (D-Bus interfaces)
│   ├── buffer/             # Double/triple buffer ring and synchronization
│   ├── demo/               # Shared demo setup utilities
│   ├── integration/        # Cross-layer integration tests
│   ├── raster/             # Software 2D rasterizer
│   │   ├── composite/      # Image compositing (Blit, SrcOver alpha)
│   │   ├── consumer/       # Display list consumers (Software, GPU)
│   │   ├── curves/         # Bézier curves and elliptical arcs
│   │   ├── displaylist/    # Display list abstraction for draw commands
│   │   ├── effects/        # Gradients and Gaussian blur shadows
│   │   ├── primitives/     # Rectangles, rounded rects, anti-aliased lines
│   │   └── text/           # SDF font atlas text rendering
│   ├── render/             # GPU abstraction layer
│   │   ├── atlas/          # GPU texture atlases (font SDF, image LRU)
│   │   ├── backend/        # Unified renderer with GPU detection and fallback
│   │   ├── display/        # GPU-to-display pipeline (DMA-BUF, Wayland, X11)
│   │   └── present/        # Frame presentation abstractions
│   ├── ui/                 # Widget and layout internals
│   │   ├── animation/      # Animation system internals
│   │   ├── decorations/    # Client-side window decorations
│   │   ├── layout/         # Flexbox-like Row/Column containers
│   │   ├── pctwidget/      # Percentage-based responsive layout
│   │   ├── scale/          # HiDPI scale factor management
│   │   └── widgets/        # Internal widget implementations
│   ├── wayland/            # Wayland protocol implementation
│   │   ├── client/         # Compositor connection and protocol sync
│   │   ├── datadevice/     # Clipboard (wl_data_device)
│   │   ├── dmabuf/         # DMA-BUF buffer creation (zwp_linux_dmabuf_v1)
│   │   ├── input/          # Pointer, keyboard, touch input
│   │   ├── output/         # HiDPI scale detection (wl_output)
│   │   ├── shm/            # Shared memory buffer pools (wl_shm)
│   │   ├── socket/         # Unix domain sockets with fd passing
│   │   ├── wire/           # Wayland wire protocol marshaling
│   │   └── xdg/            # XDG shell (toplevel windows)
│   └── x11/                # X11 protocol implementation
│       ├── client/         # Server connection and window operations
│       ├── dnd/            # Drag-and-drop (XDND)
│       ├── dpi/            # HiDPI detection
│       ├── dri3/           # DRI3 GPU buffer sharing
│       ├── events/         # X11 event structures
│       ├── gc/             # Graphics context (CreateGC, PutImage)
│       ├── present/        # Present extension (frame sync)
│       ├── selection/      # Clipboard (CLIPBOARD/PRIMARY)
│       ├── shm/            # MIT-SHM extension
│       └── wire/           # X11 wire protocol
├── render-sys/             # Rust static rendering library
│   ├── src/
│   │   ├── lib.rs          # C-ABI entry points for Go FFI
│   │   ├── detect.rs       # GPU generation detection (Intel, AMD)
│   │   ├── allocator.rs    # GPU buffer allocation (GEM, DMA-BUF)
│   │   ├── batch.rs        # GPU command batch submission
│   │   ├── pipeline.rs     # Rendering pipeline state
│   │   ├── submit.rs       # Command submission and sync
│   │   ├── shader.rs       # WGSL shader handling
│   │   ├── drm.rs          # DRM device operations
│   │   ├── eu/             # Intel EU shader compiler (6 files)
│   │   └── rdna/           # AMD RDNA shader compiler (6 files)
│   ├── shaders/            # WGSL shader source files
│   └── Cargo.toml          # Rust crate manifest
├── scripts/                # Build and verification scripts
│   ├── build-rust.sh       # Builds Rust library (called by go generate)
│   ├── verify-build.sh     # End-to-end build verification
│   └── compute-coverage.sh # Test coverage calculation
├── Makefile                # Build, test, and demo targets
├── .envrc                  # direnv CGO environment setup
└── .golangci.yml           # Linter configuration
```

## Building from Source

### Quick Build

```bash
make build
```

This checks all prerequisites, builds the Rust static library for your
architecture, compiles the musl compatibility stub, builds the Go binary,
and verifies static linkage.

### Individual Targets

```bash
make rust           # Build Rust static library only
make go             # Build Go binary (requires Rust library)
make check-deps     # Verify all build prerequisites
make check-static   # Assert binary is fully statically linked
make clean          # Remove all build artifacts
```

### Demo Binaries

```bash
make wayland-demo        # Wayland protocol demo (pure Go, no CGO)
make x11-demo            # X11 protocol demo (pure Go, no CGO)
make widget-demo         # Interactive widget demo (requires Rust library)
make gpu-triangle-demo   # GPU command submission demo
make wain-demo           # Public API lifecycle demo
make event-demo          # Event handling demo
make example-app         # Full reference application
make bench               # Software renderer benchmarks
```

### Using direnv

The `.envrc` file auto-configures CGO environment variables when entering
the project directory:

```bash
direnv allow    # one-time setup
go test ./...   # works without make or manual flag setup
```

## Testing

### Run All Tests

```bash
make test           # Rust tests + Go tests
make test-go        # Go tests only
make test-rust      # Rust tests only
```

### Visual Regression Tests

```bash
make test-visual    # Run visual tests against reference images
```

### Coverage

```bash
make coverage       # Run tests with coverage summary
make coverage-html  # Generate HTML report at coverage/coverage.html
```

### CI Workflow

The CI pipeline (`.github/workflows/ci.yml`) runs three jobs:

1. **Build & Test** — Rust tests, Go tests, golangci-lint, integration
   tests, static binary verification
2. **GPU Integration Tests** — conditional GPU tests when
   `/dev/dri/renderD128` is detected
3. **Benchmarks** — software rasterizer frame timing (target: ≤16.7 ms at
   1920×1080 for 60 FPS)

## Examples

Working examples that use only the public `wain` package:

- **[example/hello/](./example/hello/)** — minimal application with a
  button and label
- **[example/multi-window/](./example/multi-window/)** — opening and
  managing multiple windows

Build and run an example:

```bash
go generate ./...
go build -o hello ./example/hello
./hello
```

## Documentation

| Document | Description |
|----------|-------------|
| [GETTING_STARTED.md](./GETTING_STARTED.md) | Step-by-step first application guide |
| [API.md](./API.md) | Public and internal API reference |
| [TUTORIAL.md](./TUTORIAL.md) | Build a contact-form application |
| [WIDGETS.md](./WIDGETS.md) | Widget reference with all properties and methods |
| [HARDWARE.md](./HARDWARE.md) | Supported GPU hardware matrix (Intel Gen9–Xe, AMD RDNA 1–3) |
| [ACCESSIBILITY.md](./ACCESSIBILITY.md) | AT-SPI2 screen reader integration guide |
| [STABILITY.md](./STABILITY.md) | API stability policy and deprecation process |
| [ROADMAP.md](./ROADMAP.md) | Development phases and status |
| [CHANGELOG.md](./CHANGELOG.md) | Release history (Keep a Changelog format) |
| [RELEASE.md](./RELEASE.md) | Release workflow and pre-built library distribution |

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for development setup, coding
standards, and submission guidelines.

Quick start for contributors:

```bash
git clone https://github.com/opd-ai/wain.git
cd wain
make build         # build everything
make test          # run all tests
```

Pre-commit checklist:

- Tests pass: `make test-go`
- No vet warnings: `go vet ./...`
- Static linkage verified: `make check-static`
- Exported identifiers documented with godoc comments
- TODOs tracked in `TECHNICAL_DEBT.md`

## License

[MIT](./LICENSE) — Copyright (c) 2026 opdai
