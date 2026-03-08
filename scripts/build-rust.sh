#!/bin/bash
#
# build-rust.sh – Build script for the Rust rendering library and musl compatibility stub.
# Called by `go generate` in internal/render/generate.go.
#
# This script:
# 1. Checks for required tools (musl-gcc, cargo, rustup)
# 2. Detects the host architecture and musl Rust target
# 3. Ensures the musl Rust target is installed
# 4. Builds the Rust static library (librender.a)
# 5. Compiles the musl compatibility stub (dl_find_object_stub.o)
#
# Exit codes:
#   0 – Success
#   1 – Missing dependency or build failure

set -e

# Color output for better readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}ERROR:${NC} $1" >&2
    exit 1
}

info() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Determine script directory (project root)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

# Configuration
CC="${CC:-musl-gcc}"
RUST_DIR="render-sys"
DL_STUB_SRC="internal/render/dl_find_object_stub.c"
DL_STUB_OBJ="internal/render/dl_find_object_stub.o"

# ── Step 1: Check for required tools ────────────────────────────────────────────

echo "Checking dependencies..."

# Check for musl C compiler
if ! command -v "${CC}" >/dev/null 2>&1; then
    error "musl C compiler '${CC}' not found.

musl libc is required for wain builds.

Install the musl C compiler for your platform:

  Ubuntu / Debian:  sudo apt-get install musl-tools
  Fedora / RHEL:    sudo dnf install musl-gcc
  Arch Linux:       sudo pacman -S musl
  Alpine Linux:     apk add musl-dev
  macOS (cross):    brew install FiloSottile/musl-cross/musl-cross
                    then: export CC=x86_64-linux-musl-gcc

After installing, re-run: go generate ./..."
fi
info "musl C compiler found: $(command -v "${CC}")"

# Check for cargo
if ! command -v cargo >/dev/null 2>&1; then
    error "cargo (Rust build tool) not found.

Install Rust from https://rustup.rs/:

  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

After installing, re-run: go generate ./..."
fi
info "cargo found: $(cargo --version)"

# Check for rustup
if ! command -v rustup >/dev/null 2>&1; then
    warn "rustup not found; cannot auto-install musl target"
else
    info "rustup found: $(rustup --version | head -1)"
fi

# ── Step 2: Detect host architecture and musl Rust target ───────────────────────

echo ""
echo "Detecting host architecture..."

# Get Rust host triple (e.g., x86_64-unknown-linux-gnu)
RUST_HOST="$(rustc -vV 2>/dev/null | awk '/^host:/{print $2}')"
if [ -z "${RUST_HOST}" ]; then
    error "Failed to detect Rust host triple. Is rustc installed?"
fi

# Extract architecture (x86_64, aarch64, etc.)
HOST_ARCH="${RUST_HOST%%-*}"
RUST_MUSL_TARGET="${HOST_ARCH}-unknown-linux-musl"

info "Detected architecture: ${HOST_ARCH}"
info "Rust musl target: ${RUST_MUSL_TARGET}"

# ── Step 3: Ensure musl Rust target is installed ────────────────────────────────

echo ""
echo "Checking Rust musl target..."

if ! rustup target list --installed 2>/dev/null | grep -q "^${RUST_MUSL_TARGET}\$"; then
    warn "Rust target '${RUST_MUSL_TARGET}' is not installed."
    if command -v rustup >/dev/null 2>&1; then
        echo "Installing ${RUST_MUSL_TARGET}..."
        rustup target add "${RUST_MUSL_TARGET}" || error "Failed to install Rust musl target"
        info "Installed ${RUST_MUSL_TARGET}"
    else
        error "Rust target '${RUST_MUSL_TARGET}' is not installed and rustup is not available.

Install it with:

  rustup target add ${RUST_MUSL_TARGET}

After installing, re-run: go generate ./..."
    fi
else
    info "Rust musl target installed: ${RUST_MUSL_TARGET}"
fi

# ── Step 4: Build Rust static library ───────────────────────────────────────────

echo ""
echo "Building Rust static library..."

RUST_LIB="${RUST_DIR}/target/${RUST_MUSL_TARGET}/release/librender.a"

cargo build --release \
    --manifest-path "${RUST_DIR}/Cargo.toml" \
    --target "${RUST_MUSL_TARGET}" \
    ${CARGO_FLAGS} || error "Rust build failed"

if [ ! -f "${RUST_LIB}" ]; then
    error "Rust library not found at expected path: ${RUST_LIB}"
fi

info "Rust library built: ${RUST_LIB}"

# ── Step 5: Compile musl compatibility stub ─────────────────────────────────────

echo ""
echo "Compiling musl compatibility stub..."

# GCC 14+ libgcc_eh.a references _dl_find_object (glibc 2.35+), which musl
# does not provide. This stub allows static linking with musl-gcc.

if [ ! -f "${DL_STUB_SRC}" ]; then
    error "Stub source not found: ${DL_STUB_SRC}"
fi

"${CC}" -c -o "${DL_STUB_OBJ}" "${DL_STUB_SRC}" || error "Failed to compile stub"

if [ ! -f "${DL_STUB_OBJ}" ]; then
    error "Stub object not found at expected path: ${DL_STUB_OBJ}"
fi

info "Stub compiled: ${DL_STUB_OBJ}"

# ── Success ──────────────────────────────────────────────────────────────────────

echo ""
echo -e "${GREEN}✓ All build artifacts ready${NC}"
echo ""
echo "Next steps:"
echo "  1. Run: go build ./..."
echo "  2. Or:  go build -o bin/wain ./cmd/wain"
echo ""
echo "The Go build will automatically link against:"
echo "  - ${RUST_LIB}"
echo "  - ${DL_STUB_OBJ}"
echo ""
