# wain

**A statically-compiled Go UI toolkit with GPU rendering via Rust**

## Status

**Phase 1** (Software Rendering Path) — 85% Complete  
See [ROADMAP.md](ROADMAP.md) for the full 8-phase implementation plan.

## Current Functionality

### Foundation (Phase 0) — ✅ Complete
- ✅ Go → Rust static library linking (CGO + musl)
- ✅ C ABI boundary validation (`render_add`, `render_version`)
- ✅ Fully static binary output (no dynamic dependencies)

### Protocol Layer (Phase 1.1-1.2) — ✅ Complete
**Wayland Client** (6 packages, ~2,100 LOC):
- ✅ Wire format: binary protocol marshaling, fd passing via SCM_RIGHTS
- ✅ Core objects: wl_display, wl_registry, wl_compositor, wl_surface
- ✅ Shared memory: wl_shm, wl_shm_pool, wl_buffer (memfd_create)
- ✅ Window management: xdg_wm_base, xdg_surface, xdg_toplevel
- ✅ Input handling: wl_seat, wl_pointer, wl_keyboard with xkbcommon keymap

**X11 Client** (4 packages, ~1,150 LOC):
- ✅ Connection setup: authentication, XID allocation
- ✅ Window operations: CreateWindow, MapWindow, ConfigureWindow
- ✅ Graphics context: CreateGC, PutImage, CreatePixmap
- ✅ Event handling: KeyPress, ButtonPress, MotionNotify, Expose

### Rendering Layer (Phase 1.4) — ✅ Complete
**Software 2D Rasterizer** (5 packages, ~1,550 LOC):
- ✅ Primitives: filled rectangles, rounded rectangles, anti-aliased lines
- ✅ Curves: quadratic/cubic Bezier, arc fills
- ✅ Text: SDF-based rendering with embedded glyph atlas
- ✅ Effects: box shadow (Gaussian blur), linear/radial gradients
- ✅ Compositing: alpha blending (Porter-Duff), bilinear image filtering

### UI Framework (Phase 1.5) — ✅ Complete
**Widget Layer** (3 packages, ~700 LOC):
- ✅ Layout system: flexbox-like Row/Column with flex-grow/shrink, gaps, padding
- ✅ Widgets: Button, TextInput, ScrollContainer with event handlers
- ✅ Sizing: percentage-based dimensions with auto-layout

### Integration Status — ⚠️ In Progress
- ❌ No demonstration binaries (components exist but not wired together)
- ❌ All packages marked `internal/` (no public API surface yet)

**Not yet implemented:** GPU rendering (Phase 2+), DRM/KMS buffer infrastructure, Intel/AMD GPU command submission, shader compiler pipeline. The project currently uses CPU-based software rendering only.

## Prerequisites

### Required Tools

1. **Go 1.24+**
   ```bash
   go version  # should report 1.24 or later
   ```

2. **Rust 1.70+ with musl target**
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

```bash
# Build the static binary
make build

# Output: ./bin/wain (fully static executable)
```

## Test

```bash
# Run all tests (Rust + Go)
make test

# Run only Rust tests
make test-rust

# Run only Go tests
make test-go
```

**Note:** Do NOT use `go test ./...` directly. Go tests require CGO_LDFLAGS to be set to link the Rust static library, which is architecture-dependent. The `make test-go` target handles this automatically. Direct `go test` will fail with linker errors (`undefined reference to render_add`).

**Why:** The Rust library path is architecture-dependent and auto-detected by the Makefile. Direct `go test` doesn't have this information.

## Verify Static Linking

```bash
# Verify the binary has no dynamic dependencies
make check-static

# Expected output: "not a dynamic executable"
```

## Run

```bash
./bin/wain
# Output:
#   render.Add(6, 7) = 13
#   render library version: 0.1.0
```

This demonstrates the Go → Rust static library linkage is working correctly.

## Architecture

The project consists of four main layers (bottom-up):

### 1. Rust Rendering Library (render-sys/)
```
render-sys/src/lib.rs  →  librender.a (static library)
```
- **Current scope:** C ABI test functions (`render_add`, `render_version`)
- **Future scope:** GPU command submission (Phase 2+)
- **Build:** Compiled with musl target for static linking

### 2. Protocol Layer (internal/wayland/, internal/x11/)
```
Protocol Implementations (~3,250 LOC)
├── Wayland Client (6 packages)
│   ├── wire/        → Binary marshaling + fd passing
│   ├── socket/      → Unix domain socket + SCM_RIGHTS
│   ├── client/      → Display, Registry, Compositor, Surface
│   ├── shm/         → Shared memory buffers (memfd)
│   ├── xdg/         → Window management (xdg-shell)
│   └── input/       → Seat, Pointer, Keyboard, xkbcommon
└── X11 Client (4 packages)
    ├── wire/        → Request/reply/event encoding
    ├── client/      → Connection, CreateWindow, MapWindow
    ├── events/      → KeyPress, Button, Motion events
    └── gc/          → Graphics context, PutImage
```

### 3. Rendering Layer (internal/raster/)
```
Software 2D Rasterizer (~1,550 LOC)
├── core/        → Rectangles, rounded rects, lines
├── curves/      → Quadratic/cubic Bezier, arc fills
├── composite/   → Alpha blending, image filtering
├── effects/     → Box shadow, gradients
└── text/        → SDF-based text rendering
```

### 4. UI Framework (internal/ui/)
```
Widget Layer (~700 LOC)
├── layout/      → Flexbox-like Row/Column layout
├── pctwidget/   → Percentage-based sizing
└── widgets/     → Button, TextInput, ScrollContainer
```

### 5. Application Layer (cmd/)
```
┌─────────────┐
│ cmd/wain    │  Demo binary (Phase 0 validation only)
│ (main.go)   │  Calls render.Add, render.Version
└─────────────┘

Future: cmd/wayland-demo, cmd/x11-demo, cmd/widget-demo
```

**Key constraint:** The final binary must be fully static (no libc dependency) to support deployment without system dependencies. This is enforced via:
- Rust compiled with `x86_64-unknown-linux-musl` target
- Go compiled with `musl-gcc` and `-extldflags '-static'`
- Verification: `ldd bin/wain` reports "not a dynamic executable"

## Manual Build (without Makefile)

If you need to build manually (e.g., for debugging the build process):

```bash
# 1. Build the Rust static library
cargo build --release \
  --target x86_64-unknown-linux-musl \
  --manifest-path render-sys/Cargo.toml

# 2. Build the Go binary
MUSL_LIB="render-sys/target/x86_64-unknown-linux-musl/release/librender.a"
CC=musl-gcc CGO_ENABLED=1 \
  CGO_LDFLAGS="${MUSL_LIB} -ldl -lm -lpthread" \
  CGO_LDFLAGS_ALLOW=".*" \
  go build -ldflags "-extldflags '-static'" -o bin/wain ./cmd/wain

# 3. Verify
ldd bin/wain  # should print "not a dynamic executable"
```

## Project Goals

From [ROADMAP.md](ROADMAP.md):

> "A single static Go binary that speaks X11/Wayland natively and renders UI via GPU using a custom minimal Rust driver (Intel first, then AMD)."

**Target audience:** Developers building hardware-accelerated UI applications who need:
- Single-binary deployment (no runtime dependencies)
- Direct GPU access without heavyweight frameworks
- Native X11/Wayland protocol support
- Cross-platform Linux support (x86_64, ARM64)

## Development

### Project Structure

```
wain/
├── cmd/
│   ├── wain/              # Phase 0 validation binary
│   └── gen-atlas/         # SDF font atlas generator (internal tool)
├── internal/
│   ├── render/            # Go CGO bindings to Rust
│   ├── wayland/           # Wayland protocol client (6 packages)
│   ├── x11/               # X11 protocol client (4 packages)
│   ├── raster/            # Software 2D rasterizer (5 packages)
│   └── ui/                # Widget layer + layout (3 packages)
├── render-sys/            # Rust static library (C ABI exports)
├── Makefile               # Build automation (enforces static linking)
├── ROADMAP.md             # 8-phase implementation plan
└── go.mod                 # Go module definition
```

### Code Conventions

- **Error handling:** Not yet standardized (Phase 1 focus was implementation breadth)
- **Testing:** Table-driven tests for Go; unit tests for Rust
- **Documentation:** All exported functions should have godoc comments (96.9% current coverage)
- **Naming:** Follow Go conventions; avoid package/file stuttering
- **Complexity targets:** Cyclomatic ≤10, function length ≤50 lines (some Phase 1 functions exceed this)

### Completing Phase 1

The protocol and rendering implementations are complete but not integrated. To finish Phase 1:

1. **Create demonstration binaries:**
   - `cmd/wayland-demo/` — Open a Wayland window, render solid color
   - `cmd/x11-demo/` — Open an X11 window, render solid color
   - `cmd/widget-demo/` — Interactive UI with buttons, text input, scrolling

2. **Refactor complexity hotspots:**
   - `layoutRow`/`layoutColumn` (internal/ui/layout/) — Reduce 40-line duplication
   - `EncodeMessage` (internal/wayland/wire/) — Extract helper functions

3. **Create public API surface:**
   - Move protocol clients from `internal/` to public packages
   - Add platform abstraction layer to hide Wayland/X11 differences

4. **Add integration tests:**
   - End-to-end tests covering protocol → rasterizer → display

See [PLAN.md](PLAN.md) for detailed implementation steps.

### Adding New Functionality

See [ROADMAP.md](ROADMAP.md) for planned phases:
- **Phase 1:** Software rendering path (85% complete — needs integration demos)
- **Phase 2:** DRM/KMS buffer infrastructure
- **Phase 3:** Intel GPU command submission
- **Phase 4:** Shader compiler pipeline (GLSL/WGSL → Intel EU binary)
- **Phase 5:** GPU rendering backend integration
- **Phase 6:** AMD GPU support (RDNA ISA backend)
- **Phase 7:** Hardening & fallback (auto-detection, error recovery)
- **Phase 8:** Polish (HiDPI, clipboard, window decorations, accessibility)

## Known Limitations

### Phase 1 (Current)

**Integration gaps:**
- No demonstration binaries showing protocol → rasterizer → display pipeline
- All packages marked `internal/` — no public API for external users
- No platform abstraction layer (users must choose Wayland or X11 explicitly)
- No event loop implementation (components exist but not wired together)

**Missing optimizations:**
- X11: No MIT-SHM extension (uses slower PutImage fallback)
- Rasterizer: No tile-based threading (single-threaded CPU rendering)
- Layout: High complexity in `layoutRow`/`layoutColumn` (needs refactoring)

**Testing:**
- Unit tests exist for individual packages (all passing)
- No end-to-end integration tests
- No automated screenshot comparison tests

### Phase 2+ (Future)

See [ROADMAP.md](ROADMAP.md) for planned GPU rendering features.

## Troubleshooting

### `make build` fails with "musl-gcc not found"

Install the musl C compiler (see Prerequisites section above).

### `go test ./...` fails with linker errors

Go tests require CGO_LDFLAGS to be set. Use `make test-go` instead of running `go test` directly.

**Why:** The Rust library path is architecture-dependent and auto-detected by the Makefile. Direct `go test` doesn't have this information.

### Binary is not static (has dynamic dependencies)

Verify you're using:
- Rust musl target: `rustup show` should list `x86_64-unknown-linux-musl`
- musl-gcc: `which musl-gcc` should return a path
- Static ldflags: Check `go build -x` output for `-extldflags '-static'`

Run `make check-static` to verify the binary is fully static.

## License

See [LICENSE](LICENSE) file.

## Contributing

This project is in **Phase 1** (Software Rendering Path — 85% complete). 

**Priority contributions:**
1. **Integration demos** — Create working demonstration binaries (see [PLAN.md](PLAN.md) Step 1-2)
2. **Complexity refactoring** — Reduce duplication in layout system (see [PLAN.md](PLAN.md) Step 3-4)
3. **Public API design** — Move packages from `internal/` to public exports
4. **MIT-SHM extension** — Optimize X11 rendering performance
5. **Integration tests** — End-to-end tests covering full protocol stack

See [ROADMAP.md](ROADMAP.md) for the complete 8-phase plan and [AUDIT-2026-03-07.md](AUDIT-2026-03-07.md) for detailed findings from the recent code audit.
