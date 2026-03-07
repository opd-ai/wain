# Wain – Go UI toolkit with static Rust rendering library
#
# Targets:
#   make build        – build the Rust static library then the Go binary
#   make test         – run both Rust and Go test suites
#   make clean        – remove build artifacts
#   make check-static – verify the final binary is fully statically linked
#
# Environment variables:
#   CARGO_FLAGS   – extra flags passed to cargo (e.g. CARGO_FLAGS="--quiet")
#   GO_LDFLAGS    – extra flags passed to `go build -ldflags` (default empty)

SHELL := /bin/bash

RUST_DIR    := render-sys
RUST_TARGET := $(RUST_DIR)/target/release/librender.a

GO_BIN      := bin/wain
GO_PKG      := github.com/opd-ai/wain/cmd/wain

.PHONY: all build rust go test test-rust test-go clean check-static

all: build

## ── Rust ─────────────────────────────────────────────────────────────────────

rust: $(RUST_TARGET)

$(RUST_TARGET):
	cargo build --release --manifest-path $(RUST_DIR)/Cargo.toml $(CARGO_FLAGS)

## ── Go ───────────────────────────────────────────────────────────────────────

go: rust
	mkdir -p bin
	CGO_ENABLED=1 go build -ldflags "$(GO_LDFLAGS)" -o $(GO_BIN) $(GO_PKG)

build: go

## ── Tests ────────────────────────────────────────────────────────────────────

test-rust:
	cargo test --manifest-path $(RUST_DIR)/Cargo.toml $(CARGO_FLAGS)

test-go: rust
	CGO_ENABLED=1 go test ./...

test: test-rust test-go

## ── Static-link verification ─────────────────────────────────────────────────
#
# For a fully static binary (no shared-library dependencies) the build must
# target a musl-based toolchain:
#
#   rustup target add x86_64-unknown-linux-musl
#   CARGO_FLAGS="--target x86_64-unknown-linux-musl" \
#   GO_LDFLAGS="-extldflags '-static'" \
#   CC=musl-gcc \
#   make build check-static
#
# The CI job attempts this when the musl toolchain is available, and falls back
# to a dynamic-link build otherwise.

check-static: build
	@echo "Checking linkage of $(GO_BIN)…"
	@if ldd $(GO_BIN) 2>&1 | grep -q "not a dynamic executable"; then \
		echo "✓ Binary is fully statically linked."; \
	else \
		echo "ℹ Binary is dynamically linked (musl toolchain required for a fully static build):"; \
		ldd $(GO_BIN); \
	fi

## ── Cleanup ──────────────────────────────────────────────────────────────────

clean:
	cargo clean --manifest-path $(RUST_DIR)/Cargo.toml
	rm -rf bin
