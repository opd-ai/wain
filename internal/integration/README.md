# Integration Tests

This directory contains integration tests that verify cross-layer functionality between protocol handling, buffer management, and GPU rendering.

## Prerequisites

Integration tests require CGO linking against the Rust `render-sys` static library. Running tests directly with `go test` will fail with linker errors if the environment is not properly configured.

## Running Tests

### Option 1: Use direnv (Recommended)

```bash
# One-time setup:
direnv allow

# Now standard test commands work:
go test ./internal/integration
go test ./...
```

### Option 2: Use Make

```bash
make test-go
```

### Option 3: Manual CGO Setup

```bash
# Build dependencies first
make rust

# Set CGO flags
export CGO_LDFLAGS="$(pwd)/render-sys/target/x86_64-unknown-linux-musl/release/librender.a $(pwd)/internal/render/dl_find_object_stub.o -ldl -lm -lpthread"
export CGO_LDFLAGS_ALLOW=".*"

# Run tests
go test ./internal/integration
```

## Common Errors

### "undefined reference to render_*"

This error means `CGO_LDFLAGS` is not set. See "Running Tests" above for setup options.

Example error:
```
/usr/bin/ld: /tmp/go-link-*/000001.o: in function `_cgo_*_Cfunc_render_add':
/tmp/go-build/cgo-gcc-prolog:68:(.text+0x29): undefined reference to `render_add'
```

**Solution**: Use `direnv allow` or `make test-go` instead of `go test` directly.

## Test Coverage

- `gpu_test.go` - GPU detection and context management
- `dri3_test.go` - DRI3 buffer allocation and export  
- `wayland_test.go` - Wayland protocol integration
- `x11_sync_test.go` - X11 synchronization primitives
- `screenshot_test.go` - Screenshot capture utilities
