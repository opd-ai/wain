# Wain – Go UI toolkit with static Rust rendering library
#
# musl libc is required for all builds. This ensures the final binary is
# fully statically linked and has no glibc version dependency.
#
# Targets:
#   make build        – check deps, build Rust (musl), build Go (static)
#   make demo         – build the Phase 1 demonstration binary
#   make wayland-demo – build the Wayland demonstration binary
#   make x11-demo     – build the X11 demonstration binary
#   make widget-demo  – build the interactive widget demonstration binary
#   make gen-atlas    – build the SDF font atlas generator tool
#   make test         – run both Rust and Go test suites
#   make test-visual  – run visual regression tests for rendering primitives
#   make coverage     – run Go tests with coverage reporting
#   make coverage-html – generate HTML coverage report (coverage/coverage.html)
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
RUST_LIB  := $(RUST_DIR)/target/$(RUST_MUSL_TARGET)/release/librender_sys.a

# Stub object for _dl_find_object (GCC 14+ with musl compatibility)
DL_STUB_SRC := internal/render/dl_find_object_stub.c
DL_STUB_OBJ := internal/render/dl_find_object_stub.o

GO_BIN       := bin/wain
GO_PKG       := github.com/opd-ai/wain/cmd/wain
GEN_ATLAS_BIN := bin/gen-atlas
GEN_ATLAS_PKG := github.com/opd-ai/wain/cmd/gen-atlas

.PHONY: all build rust go test test-rust test-go test-visual coverage coverage-html clean check-static check-deps gen-atlas wayland-demo x11-demo x11-dmabuf-demo widget-demo gpu-triangle-demo gpu-shader-demo double-buffer-demo dmabuf-demo stats wain-demo event-demo example-app bench

all: build

## ── Dependency checks ────────────────────────────────────────────────────────
#
# These checks run before any build step and fail immediately with actionable
# installation instructions if a required tool is absent.

check-deps: check-os check-go check-rust check-musl-gcc check-musl-rust-target

check-os:
	@if [ "$$(uname -s)" != "Linux" ]; then \
		echo ""; \
		echo "ERROR: wain requires Linux (found: $$(uname -s))"; \
		echo ""; \
		echo "wain implements Wayland and X11 display protocols and is designed"; \
		echo "for Linux desktop environments. Cross-compilation from macOS is"; \
		echo "possible but requires a Linux musl cross-toolchain."; \
		echo ""; \
		echo "If you are on macOS and want to cross-compile:"; \
		echo "  brew install FiloSottile/musl-cross/musl-cross"; \
		echo "  make build CC=x86_64-linux-musl-gcc"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Operating system: Linux"

check-go:
	@if ! command -v go >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: Go compiler not found"; \
		echo ""; \
		echo "Install Go 1.24 or later from https://go.dev/dl/"; \
		echo ""; \
		exit 1; \
	fi
	@GO_VERSION=$$(go version | sed -n 's/.*go\([0-9]*\)\.\([0-9]*\).*/\1.\2/p'); \
	GO_MAJOR=$$(echo $$GO_VERSION | cut -d. -f1); \
	GO_MINOR=$$(echo $$GO_VERSION | cut -d. -f2); \
	if [ "$$GO_MAJOR" -lt 1 ] || ([ "$$GO_MAJOR" -eq 1 ] && [ "$$GO_MINOR" -lt 24 ]); then \
		echo ""; \
		echo "ERROR: Go 1.24 or later is required (found: $$(go version))"; \
		echo ""; \
		echo "Download from https://go.dev/dl/"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Go version: $$(go version | awk '{print $$3}')"

check-rust:
	@if ! command -v rustc >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: Rust compiler not found"; \
		echo ""; \
		echo "Install Rust from https://rustup.rs/"; \
		echo "  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"; \
		echo ""; \
		exit 1; \
	fi
	@if ! command -v cargo >/dev/null 2>&1; then \
		echo ""; \
		echo "ERROR: cargo not found"; \
		echo ""; \
		echo "cargo should be installed with Rust. Re-install from https://rustup.rs/"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Rust version: $$(rustc --version | awk '{print $$2}')"

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

## ── C stub for musl compatibility ────────────────────────────────────────────
#
# GCC 14+ libgcc_eh.a references _dl_find_object (glibc 2.35+), which musl
# does not provide. This stub allows static linking with musl-gcc.

$(DL_STUB_OBJ): $(DL_STUB_SRC)
	$(CC) -c -o $(DL_STUB_OBJ) $(DL_STUB_SRC)

## ── Go ───────────────────────────────────────────────────────────────────────

go: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o $(GO_BIN) $(GO_PKG)
	@echo "Verifying static linkage…"
	@if ! ldd $(GO_BIN) 2>&1 | grep -q "not a dynamic executable"; then \
		echo ""; \
		echo "ERROR: Binary has dynamic dependencies:"; \
		echo ""; \
		ldd $(GO_BIN); \
		echo ""; \
		echo "Ensure CC=$(CC) and CGO_LDFLAGS are correctly set for static linking."; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Binary is fully statically linked."

build: go

wayland-demo:
	mkdir -p bin
	CGO_ENABLED=0 go build \
	  -ldflags "-s -w" \
	  -o bin/wayland-demo github.com/opd-ai/wain/cmd/wayland-demo

x11-demo:
	mkdir -p bin
	CGO_ENABLED=0 go build \
	  -ldflags "-s -w" \
	  -o bin/x11-demo github.com/opd-ai/wain/cmd/x11-demo

x11-dmabuf-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/x11-dmabuf-demo github.com/opd-ai/wain/cmd/x11-dmabuf-demo
	@if ! ldd bin/x11-dmabuf-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/x11-dmabuf-demo has dynamic dependencies:" && ldd bin/x11-dmabuf-demo && exit 1; \
	fi

widget-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/widget-demo github.com/opd-ai/wain/cmd/widget-demo
	@if ! ldd bin/widget-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/widget-demo has dynamic dependencies:" && ldd bin/widget-demo && exit 1; \
	fi

dmabuf-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/dmabuf-demo github.com/opd-ai/wain/cmd/dmabuf-demo
	@if ! ldd bin/dmabuf-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/dmabuf-demo has dynamic dependencies:" && ldd bin/dmabuf-demo && exit 1; \
	fi

gpu-triangle-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/gpu-triangle-demo github.com/opd-ai/wain/cmd/gpu-triangle-demo
	@if ! ldd bin/gpu-triangle-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/gpu-triangle-demo has dynamic dependencies:" && ldd bin/gpu-triangle-demo && exit 1; \
	fi

gpu-shader-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/gpu-shader-demo github.com/opd-ai/wain/cmd/gpu-shader-demo
	@if ! ldd bin/gpu-shader-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/gpu-shader-demo has dynamic dependencies:" && ldd bin/gpu-shader-demo && exit 1; \
	fi


gen-atlas: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o $(GEN_ATLAS_BIN) $(GEN_ATLAS_PKG)
	@if ! ldd $(GEN_ATLAS_BIN) 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: $(GEN_ATLAS_BIN) has dynamic dependencies:" && ldd $(GEN_ATLAS_BIN) && exit 1; \
	fi

double-buffer-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/double-buffer-demo github.com/opd-ai/wain/cmd/double-buffer-demo
	@if ! ldd bin/double-buffer-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/double-buffer-demo has dynamic dependencies:" && ldd bin/double-buffer-demo && exit 1; \
	fi

wain-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/wain-demo github.com/opd-ai/wain/cmd/wain-demo
	@if ! ldd bin/wain-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/wain-demo has dynamic dependencies:" && ldd bin/wain-demo && exit 1; \
	fi

event-demo: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/event-demo github.com/opd-ai/wain/cmd/event-demo
	@if ! ldd bin/event-demo 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/event-demo has dynamic dependencies:" && ldd bin/event-demo && exit 1; \
	fi

example-app: rust $(DL_STUB_OBJ)
	mkdir -p bin
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go build \
	    -ldflags "-extldflags '-static'" \
	    -o bin/example-app github.com/opd-ai/wain/cmd/example-app
	@if ! ldd bin/example-app 2>&1 | grep -q "not a dynamic executable"; then \
		echo "ERROR: bin/example-app has dynamic dependencies:" && ldd bin/example-app && exit 1; \
	fi

bench:
	mkdir -p bin
	go build -o bin/bench github.com/opd-ai/wain/cmd/bench

## ── Tests ────────────────────────────────────────────────────────────────────

test-rust: check-deps
	cargo test --manifest-path $(RUST_DIR)/Cargo.toml \
	  --target $(RUST_MUSL_TARGET) \
	  $(CARGO_FLAGS)

test-go: rust $(DL_STUB_OBJ)
	CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go test ./...

test: test-rust test-go

test-visual: rust $(DL_STUB_OBJ)
	@echo "Running visual regression tests…"
	@CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go test -v ./internal/raster -run TestVisual
	@echo ""
	@echo "Visual regression tests passed. Reference images: $(shell ls -1 internal/raster/testdata/*.png 2>/dev/null | wc -l) images"

coverage: rust $(DL_STUB_OBJ)
	@echo "Running Go tests with coverage…"
	@CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go test -cover ./... 2>&1 | tee coverage.txt
	@echo ""
	@echo "Coverage summary (library packages only, excluding cmd/):"
	@grep "internal.*coverage:" coverage.txt | awk '{ \
	    match($$0, /([0-9.]+)%/, arr); \
	    pkg = $$2; \
	    gsub(/^github.com\/opd-ai\/wain\//, "", pkg); \
	    if (arr[1] > 0) { \
	        total += arr[1]; \
	        count++; \
	        printf "  %-50s %6.1f%%\n", pkg, arr[1] \
	    } \
	} END { \
	    if (count > 0) { \
	        printf "\nAverage coverage: %.1f%% across %d packages with tests\n", total/count, count \
	    } \
	}'

coverage-html: rust $(DL_STUB_OBJ)
	@echo "Generating HTML coverage report…"
	@mkdir -p coverage
	@CC=$(CC) CGO_ENABLED=1 \
	  CGO_LDFLAGS="$(CURDIR)/$(RUST_LIB) $(CURDIR)/$(DL_STUB_OBJ) -ldl -lm -lpthread" \
	  CGO_LDFLAGS_ALLOW=".*" \
	  go test -coverprofile=coverage/coverage.out ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "HTML coverage report generated at coverage/coverage.html"

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

## ── Statistics ───────────────────────────────────────────────────────────────

stats:
	@echo "=== Lines of Code Summary ==="
	@echo ""
	@echo "Rust (render-sys/src):"
	@RUST_TOTAL=$$(find render-sys/src -name "*.rs" -exec wc -l {} + | tail -1 | awk '{print $$1}'); \
	RUST_CODE=$$(find render-sys/src -name "*.rs" -exec cat {} \; | grep -v '^\s*$$' | grep -v '^\s*//' | grep -v '^\s*/\*' | grep -v '^\s*\*' | wc -l); \
	echo "  ~$${RUST_CODE} LOC (code only, excludes comments/blanks)"; \
	echo "  ~$${RUST_TOTAL} LOC total (includes all lines)"
	@echo ""
	@echo "Go packages:"
	@for pkg in wayland x11 raster ui render buffer; do \
		if [ -d "internal/$$pkg" ]; then \
			TOTAL=$$(find internal/$$pkg -name "*.go" ! -name "*_test.go" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $$1}' || echo "0"); \
			echo "  internal/$$pkg: ~$${TOTAL} LOC"; \
		fi; \
	done

## ── Cleanup ──────────────────────────────────────────────────────────────────

clean:
	cargo clean --manifest-path $(RUST_DIR)/Cargo.toml
	rm -rf bin
	rm -f $(DL_STUB_OBJ)
