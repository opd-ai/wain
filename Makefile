# Wain – Go UI toolkit with static Rust rendering library
#
# musl libc is required for all builds. This ensures the final binary is
# fully statically linked and has no glibc version dependency.
#
# Targets:
#   make build        – check deps, build Rust (musl), build Go (static)
#   make demo         – build the Phase 1 demonstration binary
#   make gen-atlas    – build the SDF font atlas generator tool
#   make test         – run both Rust and Go test suites
#   make check-static – assert the final binary is fully statically linked
#   make clean        – remove build artifacts
#
# Prerequisites:
#   musl C compiler  Ubuntu/Debian:  sudo apt-get install musl-tools   (musl-gcc)
#                    Fedora/RHEL:    sudo dnf install musl-gcc
#                    Arch Linux:     sudo pacman -S musl
#                    Alpine Linux:   apk add musl-dev
#                    macOS (cross):  brew install FiloSottile/musl-cross/musl-cross
#                                    then: make build CC=x86_64-linux-musl-gcc
#
#   musl Rust target:  rustup target add $(RUST_MUSL_TARGET)
#
# Configurable variables:
#   CC          – musl C compiler (default: musl-gcc).
#                 Override for cross toolchains, e.g. CC=x86_64-linux-musl-gcc.
#   CARGO_FLAGS – extra flags passed to cargo commands.

SHELL := /bin/bash

## ── Configurable variables ───────────────────────────────────────────────────

# The musl C compiler used for CGO.
# Defaults to musl-gcc; override on the command line for cross toolchains:
#   make build CC=x86_64-linux-musl-gcc    (macOS cross via musl-cross)
#   make build CC=aarch64-linux-musl-gcc   (aarch64 cross)
CC := musl-gcc

## ── Architecture & target detection ─────────────────────────────────────────

# Detect host architecture from Rust's host triple (e.g. x86_64, aarch64).
RUST_HOST        := $(shell rustc -vV 2>/dev/null | awk '/^host:/{print $$2}')
# Strip the vendor+OS suffix to get the raw arch (x86_64, aarch64, …).
HOST_ARCH        := $(firstword $(subst -, ,$(RUST_HOST)))
RUST_MUSL_TARGET := $(HOST_ARCH)-unknown-linux-musl

RUST_DIR  := render-sys
RUST_LIB  := $(RUST_DIR)/target/$(RUST_MUSL_TARGET)/release/librender.a

GO_BIN       := bin/wain
GO_PKG       := github.com/opd-ai/wain/cmd/wain
GEN_ATLAS_BIN := bin/gen-atlas
GEN_ATLAS_PKG := github.com/opd-ai/wain/cmd/gen-atlas

.PHONY: all build rust go test test-rust test-go clean check-static check-deps gen-atlas

all: build

## ── Dependency checks ────────────────────────────────────────────────────────
#
# These checks run before any build step and fail immediately with actionable
# installation instructions if a required tool is absent.

check-deps: check-musl-gcc check-musl-rust-target

check-musl-gcc:
	@if ! command -v $(CC) >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: musl C compiler '$(CC)' not found."; \
		echo ""; \
		echo "musl libc is required for all wain builds."; \
		echo ""; \
		echo "Install the musl C compiler for your platform:"; \
		echo ""; \
		echo "  Ubuntu / Debian:  sudo apt-get install musl-tools"; \
		echo "  Fedora / RHEL:    sudo dnf install musl-gcc"; \
		echo "  Arch Linux:       sudo pacman -S musl"; \
		echo "  Alpine Linux:     apk add musl-dev"; \
		echo "  macOS (cross):    brew install FiloSottile/musl-cross/musl-cross"; \
		echo "                    then re-run: make build CC=x86_64-linux-musl-gcc"; \
		echo ""; \
		echo "After installing, re-run: make build"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ musl C compiler found: $$(command -v $(CC))"

check-musl-rust-target:
	@if ! rustup target list --installed 2>/dev/null | grep -q "^$(RUST_MUSL_TARGET)$$"; then \
		echo ""; \
		echo "ERROR: Rust target '$(RUST_MUSL_TARGET)' is not installed."; \
		echo ""; \
		echo "Install it with:"; \
		echo ""; \
		echo "  rustup target add $(RUST_MUSL_TARGET)"; \
		echo ""; \
		echo "After installing, re-run: make build"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Rust musl target installed: $(RUST_MUSL_TARGET)"

## ── Rust ─────────────────────────────────────────────────────────────────────

rust: check-deps $(RUST_LIB)

$(RUST_LIB):
	cargo build --release \
	  --manifest-path $(RUST_DIR)/Cargo.toml \
	  --target $(RUST_MUSL_TARGET) \
	  $(CARGO_FLAGS)

## ── Go ───────────────────────────────────────────────────────────────────────

go: rust
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o $(GO_BIN) $(GO_PKG)

build: go

demo: rust
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/demo github.com/opd-ai/wain/cmd/demo

gen-atlas:
	mkdir -p bin
	go build -o $(GEN_ATLAS_BIN) $(GEN_ATLAS_PKG)

## ── Tests ────────────────────────────────────────────────────────────────────

test-rust: check-deps
	cargo test --manifest-path $(RUST_DIR)/Cargo.toml \
	  --target $(RUST_MUSL_TARGET) \
	  $(CARGO_FLAGS)

test-go: rust
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go test ./...

test: test-rust test-go

## ── Static-link verification ─────────────────────────────────────────────────

check-static: build
	@echo "Verifying static linkage of $(GO_BIN)…"
	@if ldd $(GO_BIN) 2>&1 | grep -q "not a dynamic executable"; then \
		echo "✓ Binary is fully statically linked."; \
	else \
		echo ""; \
		echo "ERROR: $(GO_BIN) is dynamically linked."; \
		echo ""; \
		ldd $(GO_BIN); \
		echo ""; \
		echo "Ensure CC=$(CC) and -extldflags '-static' are set."; \
		echo "Run 'make build' which enforces these flags automatically."; \
		echo ""; \
		exit 1; \
	fi

## ── Cleanup ──────────────────────────────────────────────────────────────────

clean:
	cargo clean --manifest-path $(RUST_DIR)/Cargo.toml
	rm -rf bin
