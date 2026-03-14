# Release Workflow for wain

This document outlines the process for creating wain releases with pre-built static libraries.

## Overview

Each wain release should include pre-built static libraries (`librender_sys.a` and `dl_find_object_stub.o`) for common platforms so that users can `go get` and `go build` without needing Rust or musl toolchains.

## Supported Platforms

Pre-built libraries are provided for:
- `x86_64-unknown-linux-musl` (x86_64 Linux)
- `aarch64-unknown-linux-musl` (ARM64 Linux)

## Building Release Artifacts

### On x86_64 Linux

```bash
# Build for x86_64
make clean
make build

# Copy artifacts
mkdir -p release/x86_64
cp render-sys/target/x86_64-unknown-linux-musl/release/librender_sys.a release/x86_64/
cp internal/render/dl_find_object_stub.o release/x86_64/

# Create tarball
cd release
tar czf wain-libs-x86_64-unknown-linux-musl.tar.gz x86_64/
```

### On ARM64 Linux (or cross-compilation)

```bash
# For native ARM64 build
make clean
make build

# Copy artifacts
mkdir -p release/aarch64
cp render-sys/target/aarch64-unknown-linux-musl/release/librender_sys.a release/aarch64/
cp internal/render/dl_find_object_stub.o release/aarch64/

# Create tarball
cd release
tar czf wain-libs-aarch64-unknown-linux-musl.tar.gz aarch64/
```

## Creating a GitHub Release

1. **Tag the release:**
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```

2. **Create GitHub release:**
   - Go to https://github.com/opd-ai/wain/releases/new
   - Select the tag you just created
   - Add release notes describing changes
   - Attach the pre-built library tarballs:
     - `wain-libs-x86_64-unknown-linux-musl.tar.gz`
     - `wain-libs-aarch64-unknown-linux-musl.tar.gz`

3. **Publish the release**

## Using Pre-built Libraries

Users can download and extract the pre-built libraries:

```bash
# Download for your platform
wget https://github.com/opd-ai/wain/releases/download/v0.2.0/wain-libs-x86_64-unknown-linux-musl.tar.gz

# Extract
tar xzf wain-libs-x86_64-unknown-linux-musl.tar.gz

# Set CGO_LDFLAGS to point to the libraries
export CGO_LDFLAGS="$(pwd)/x86_64/librender_sys.a $(pwd)/x86_64/dl_find_object_stub.o -ldl -lm -lpthread"
export CGO_LDFLAGS_ALLOW=".*"
export CC=musl-gcc
export CGO_ENABLED=1

# Build your wain application
go build -ldflags "-extldflags '-static'" .
```

## Future Automation

This process can be automated with GitHub Actions to build and publish release artifacts automatically when a tag is pushed.

## Verification

After creating a release, verify that users can use it:

```bash
# On a clean machine with only Go installed
go get github.com/opd-ai/wain@v0.2.0
cd /tmp/test-wain
go mod init test
go get github.com/opd-ai/wain@v0.2.0

# Should fail gracefully, pointing users to wain-build or release downloads
go build .
```
