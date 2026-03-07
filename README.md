# wain

**A statically-compiled Go UI toolkit with GPU rendering via Rust**

## Status

**Phase 0** (Foundation & Toolchain Setup) — Build toolchain validation  
See [ROADMAP.md](ROADMAP.md) for the full 8-phase implementation plan.

## Current Functionality

Phase 0 implements the core build infrastructure:
- ✅ Go → Rust static library linking (CGO + musl)
- ✅ C ABI boundary validation (`render_add`, `render_version`)
- ✅ Fully static binary output (no dynamic dependencies)

**Not yet implemented:** GPU rendering, Mesa/Vulkan integration, X11/Wayland protocol support, or UI toolkit APIs are planned for future phases (see [ROADMAP.md](ROADMAP.md)).

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

```
┌─────────────┐
│ cmd/wain    │  Go binary (main package)
│ (main.go)   │
└──────┬──────┘
       │ imports
       ▼
┌─────────────────┐
│ internal/render │  Go CGO bindings
│ (render.go)     │
└──────┬──────────┘
       │ CGO → C ABI
       ▼
┌─────────────────┐
│ render-sys      │  Rust static library (.a)
│ (lib.rs)        │  Compiled with: staticlib, musl target
└─────────────────┘
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
├── cmd/wain/              # Go binary entry point
├── internal/render/       # Go CGO bindings to Rust
├── render-sys/            # Rust static library (C ABI exports)
├── Makefile               # Build automation (enforces static linking)
├── ROADMAP.md             # 8-phase implementation plan
└── go.mod                 # Go module definition
```

### Code Conventions

- **Error handling:** Not yet applicable (Phase 0 has no error-prone operations)
- **Testing:** Table-driven tests for Go; unit tests for Rust
- **Documentation:** All exported functions must have godoc comments
- **Naming:** Follow Go conventions; avoid package/file stuttering

### Adding New Functionality

See [ROADMAP.md](ROADMAP.md) for planned phases:
- **Phase 1:** DRM/KMS device enumeration
- **Phase 2:** Intel GPU initialization (i915)
- **Phase 3:** Vulkan context & minimal triangle
- **Phase 4:** X11 window protocol
- **Phase 5:** Basic UI primitives
- **Phase 6:** Text rendering
- **Phase 7:** Input handling
- **Phase 8:** AMD GPU support

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

This project is in Phase 0 (foundation). Contributions are welcome once Phase 0 is complete and CI is established. See [ROADMAP.md](ROADMAP.md) for planned work.
