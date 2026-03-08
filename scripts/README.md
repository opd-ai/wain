# Build Scripts

This directory contains build automation scripts for the wain project.

## build-rust.sh

Builds the Rust rendering library and musl compatibility stub. Called by `go generate` in `internal/render/generate.go`.

### What it does

1. **Dependency checks**: Verifies musl-gcc, cargo, and rustup are installed
2. **Architecture detection**: Auto-detects host architecture (x86_64, aarch64, etc.)
3. **Target installation**: Ensures the musl Rust target is installed (auto-installs if missing)
4. **Rust build**: Compiles `render-sys` to a static library for the musl target
5. **Stub compilation**: Compiles the GCC 14+ musl compatibility stub

### Usage

**Via go generate (recommended):**
```bash
go generate ./...
```

**Direct invocation:**
```bash
./scripts/build-rust.sh
```

### Environment variables

| Variable       | Default     | Description                             |
|----------------|-------------|-----------------------------------------|
| `CC`           | `musl-gcc`  | musl C compiler to use                  |
| `CARGO_FLAGS`  | (empty)     | Additional flags passed to cargo build  |

### Exit codes

- `0` – Success
- `1` – Missing dependency or build failure

### Requirements

- **musl-gcc**: musl C compiler
  - Ubuntu/Debian: `sudo apt-get install musl-tools`
  - Fedora/RHEL: `sudo dnf install musl-gcc`
  - Arch Linux: `sudo pacman -S musl`
  - Alpine Linux: `apk add musl-dev`
  - macOS: `brew install FiloSottile/musl-cross/musl-cross`

- **cargo**: Rust build tool
  - Install via: `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`

- **rustup**: Rust toolchain manager (for auto-installing musl target)

### Output artifacts

- `render-sys/target/<arch>-unknown-linux-musl/release/librender_sys.a` – Rust static library
- `internal/render/dl_find_object_stub.o` – musl compatibility stub

These artifacts are automatically linked by Go's CGO when building the final binary.

### Integration with Go build

After running `go generate`, the Go build system will automatically link against the generated artifacts via CGO_LDFLAGS. The Makefile sets these flags automatically:

```makefile
CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread"
```

### Cross-architecture builds

The script auto-detects the host architecture and builds for the corresponding musl target. For cross-compilation, use the Makefile with an appropriate cross-toolchain:

```bash
make build CC=aarch64-linux-musl-gcc
```

The build script will detect the target architecture from `rustc -vV` and build for the correct musl target.
