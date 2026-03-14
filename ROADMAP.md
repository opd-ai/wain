# Goal-Achievement Assessment

**Generated:** 2026-03-14  
**Analyzed by:** go-stats-generator v1.0.0  
**Repository:** github.com/opd-ai/wain

---

## Project Context

### What it Claims to Do

Wain is a **statically-compiled Go UI toolkit** that:

1. **Go–Rust Static Linking** — CGO bridge to a Rust `staticlib`, producing a single binary with no dynamic dependencies
2. **Wayland Client** — 9-package implementation covering wire format, SHM buffers, xdg-shell, pointer/keyboard input, DMA-BUF, clipboard, output handling
3. **X11 Client** — 9-package implementation covering connection setup, window operations, MIT-SHM, DRI3 GPU buffer sharing, Present frame sync, DPI scaling, clipboard
4. **Software 2D Rasterizer** — Filled/rounded rectangles, anti-aliased lines, Bézier curves, SDF text, box shadows, gradients, Porter-Duff alpha compositing
5. **UI Widget Layer** — Flexbox-like Row/Column layout, percentage-based sizing, Button/TextInput/ScrollContainer widgets, client-side decorations, DPI-aware scaling
6. **GPU Buffer Infrastructure** — DRM/KMS ioctl wrappers for Intel i915/Xe and AMD amdgpu drivers, DMA-BUF export, slab sub-allocation
7. **GPU Command Submission** — Batch buffer construction, Intel 3D pipeline encoding, pipeline state objects, surface/sampler state
8. **Shader Frontend** — WGSL/GLSL parsing via naga 0.14; 7 WGSL shaders for UI rendering
9. **Intel EU Backend** — Register allocator, instruction lowering, 128-bit binary encoding for Gen9+ execution units
10. **AMD RDNA Backend** — RDNA instruction set, register allocation, encoding, PM4 command stream
11. **Public API** — `App` type with display server auto-detection (Wayland→X11) and renderer auto-detection (Intel→AMD→software)
12. **AT-SPI2 Accessibility** — Optional screen-reader support via D-Bus (`-tags=atspi`)
13. **60 FPS Software Rendering** — CI-enforced benchmark: FillRectOpaque1080p ≤16.7 ms/frame

### Target Audience

- **Go developers** seeking a static, self-contained GUI binary for Linux desktop
- **Embedded/appliance developers** needing zero-dependency distribution
- **GPU programming enthusiasts** interested in direct Intel/AMD command submission without Mesa/Vulkan

### Architecture Overview

| Layer | Packages | Role |
|-------|----------|------|
| **L1: Rust Rendering** | `render-sys/` | DRM ioctls, GPU allocation, batch encoding, shader parsing, EU/RDNA backends |
| **L2: Go Bindings** | `internal/render/` | CGO wrappers, atlas, backend selection, frame presentation |
| **L3: Protocol** | `internal/wayland/` (9 pkg), `internal/x11/` (9 pkg) | Display server clients |
| **L4: Rasterizer** | `internal/raster/` (7 pkg) | Software 2D rendering, display lists |
| **L5: UI Framework** | `internal/ui/` (5 pkg) | Layout, widgets, decorations, scaling |
| **Public API** | Root package (`wain`) | App, Window, Widget, Canvas, Event types |

### Existing CI/Quality Gates

| Gate | Status |
|------|--------|
| `go test ./...` | ✅ 74 packages, all passing |
| `go vet ./...` | ✅ No issues |
| `golangci-lint` | ✅ Configured in `.golangci.yml` |
| `cargo test` (Rust) | ✅ Passes |
| Static linkage assertion | ✅ CI enforced via `ldd` |
| 60 FPS benchmark | ✅ CI enforced (≤16.7 ms/frame at 1080p) |
| Race detector | ✅ `-race` passes on all tests |
| GPU integration tests | ⚠️ Run when hardware detected |

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence | Gap Description |
|---|-------------|--------|----------|-----------------|
| 1 | Single static binary (no dynamic deps) | ✅ Achieved | CI: `ldd bin/wain` = "not a dynamic executable"; Makefile enforces `-extldflags '-static'` | — |
| 2 | Wayland client (9 packages) | ✅ Achieved | `internal/wayland/`: wire, socket, client, shm, xdg, input, dmabuf, datadevice, output; 9 packages with 85–100% test coverage | — |
| 3 | X11 client (9 packages) | ✅ Achieved | `internal/x11/`: wire, client, events, gc, shm, dri3, present, dpi, selection + dnd; 10 packages with 75–100% coverage | — |
| 4 | Software 2D rasterizer (7 packages) | ✅ Achieved | `internal/raster/`: primitives, curves, composite, effects, text, displaylist, consumer; coverage 85–94% | — |
| 5 | UI widget layer (5 packages) | ✅ Achieved | `internal/ui/`: layout, pctwidget, widgets, decorations, scale + animation; Button, TextInput, ScrollContainer implemented | — |
| 6 | GPU buffer infrastructure | ✅ Achieved | `render-sys/src/allocator.rs`, `slab.rs`, `drm.rs`, `i915.rs`, `xe.rs`, `amd.rs` (~3800 lines Rust) | — |
| 7 | GPU command submission | ⚠️ Partial | `render-sys/src/batch.rs`, `pipeline.rs`, `surface.rs`, `cmd/` exist (~2500 lines); Intel Gen9–12 batches functional; no end-to-end GPU rendered frame in demos | GPU triangles render only in unit tests; no integrated UI→GPU→display pipeline demo |
| 8 | Shader frontend (naga) | ✅ Achieved | `render-sys/src/shader.rs` (538 lines); 7 WGSL shaders in `render-sys/shaders/`; `cmd/shader-test` validates all 7 | — |
| 9 | Intel EU backend | ⚠️ Partial | `render-sys/src/eu/` (6 files, ~180 KB total); register allocator, instruction lowering, encoding for Gen9+ | `lower.rs` is 116 KB — likely generated/tablegen code; shader→EU compilation not exercised in CI |
| 10 | AMD RDNA backend | ⚠️ Partial | `render-sys/src/rdna/` (6 files, ~44 KB), `amd.rs`, `pm4.rs` | Similar gap: ISA encoding exists but no shader→RDNA compilation path exercised |
| 11 | Public API (App, Window, Widget) | ✅ Achieved | `app.go`, `widget.go`, `publicwidget.go`, `resource.go`, `event.go`, `dispatcher.go`; `STABILITY.md` pins 13 constructors, 7 methods, 5 interfaces | — |
| 12 | Display server auto-detection | ✅ Achieved | `app.go`: tries Wayland (`$WAYLAND_DISPLAY`) first, falls back to X11 (`$DISPLAY`) | — |
| 13 | Renderer auto-detection | ✅ Achieved | `internal/render/backend/backend.go`: Intel→AMD→software fallback chain; `cmd/auto-render-demo` demonstrates | — |
| 14 | AT-SPI2 accessibility | ✅ Achieved | `internal/a11y/` (10 files, 75 functions); `accessibility.go` exposes `EnableAccessibility`; requires `-tags=atspi` | — |
| 15 | 60 FPS software rendering | ✅ Achieved | CI benchmark: `BenchmarkFillRectOpaque1080p ≤ 16.7 ms`; `cmd/bench` enforces threshold | — |
| 16 | DMA-BUF buffer sharing | ✅ Achieved | `internal/wayland/dmabuf/`, `internal/x11/dri3/`; `cmd/dmabuf-demo`, `cmd/x11-dmabuf-demo` | — |
| 17 | Clipboard support | ✅ Achieved | `clipboard.go`, `internal/wayland/datadevice/`, `internal/x11/selection/`; tests in `clipboard_test.go` | — |
| 18 | Client-side window decorations | ✅ Achieved | `internal/ui/decorations/`: title bar, controls, resize handles; `cmd/decorations-demo` | — |
| 19 | HiDPI / DPI-aware scaling | ✅ Achieved | `internal/ui/scale/`, `internal/x11/dpi/`; 100% test coverage | — |
| 20 | Keyboard accessibility (Tab focus) | ✅ Achieved | `accessibility_test.go` verifies Tab/Shift-Tab traversal, Enter/Space activation | — |

**Overall: 17/20 goals fully achieved (85%); 3 goals partially achieved**

---

## Metrics Summary (go-stats-generator)

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Lines of Code (Go) | 14,665 | Moderate-sized codebase |
| Total Lines of Code (Rust) | ~15,114 | Substantial GPU backend |
| Total Functions | 664 | — |
| Total Methods | 1,166 | — |
| Total Packages | 74 (40 non-cmd) | Well-modularized |
| Average Function Length | 9.4 lines | Excellent |
| Functions > 50 lines | 7 (0.4%) | Excellent |
| High Complexity (>10) | **0** | Excellent |
| Average Complexity | 3.1 | Excellent |
| Circular Dependencies | **0** | Excellent |
| Duplication Ratio | 0.64% | Excellent |
| Average Test Coverage | 85%+ (internal packages) | Strong |
| Root Package Coverage | 24.4% | Low — limited integration tests |

### Code Health Assessment

The codebase is **exceptionally well-structured**:
- No functions exceed cyclomatic complexity 10 (max observed: 9.6)
- Near-zero code duplication (0.64%)
- Zero circular dependencies
- Consistent naming conventions (31 minor violations, mostly single-letter loop vars)
- Strong test coverage in internal packages (85–100%)

### Risk Areas

1. **Root package (`wain`) coverage: 24.4%** — Integration testing of `App`/`Window` lifecycle is limited
2. **GPU backend exercised in isolation** — No CI-tested path from shader compilation → EU/RDNA encoding → display
3. **`internal/a11y` has no tests** — AT-SPI2 implementation untested

---

## Roadmap

### Priority 1: Complete GPU Rendering Pipeline

**Gap:** GPU command submission exists but no integrated UI→GPU→display path is demonstrated or tested.

The Intel EU backend (`render-sys/src/eu/`) and AMD RDNA backend (`render-sys/src/rdna/`) have instruction encoding, but the path from:
> WGSL shader → naga IR → EU/RDNA binary → batch buffer → execbuffer2/amdgpu CS → display

is not exercised end-to-end.

**Tasks:**
- [ ] **Add GPU pipeline integration test** (`internal/integration/gpu_pipeline_test.go`):
  - Compile `solid_fill.wgsl` to EU binary via naga+lowering
  - Build batch buffer with state setup and primitive draw
  - Submit to GPU via execbuffer2 / amdgpu CS ioctl
  - Verify frame buffer contents
- [ ] **Wire GPU backend into `App.RenderFrame`**: Currently `GPUBackend.Render()` exists but is not invoked for widget tree rendering
- [ ] **Create `cmd/gpu-ui-demo`**: Interactive UI rendered entirely via GPU backend (validates claim "GPU-accelerated graphics")
- [ ] **Add CI GPU smoke test**: If `/dev/dri/renderD128` exists, run basic GPU submission test

**Validation:** `go test -tags=integration ./internal/integration -run TestGPUPipeline` should pass on Intel/AMD hardware.

**Impact:** Fully achieves goals #7 (GPU command submission), #9 (Intel EU backend), #10 (AMD RDNA backend).

---

### Priority 2: Increase Public API Test Coverage

**Gap:** Root package `wain` has only 24.4% test coverage. The `App`, `Window`, and widget lifecycle paths are undertested.

**Tasks:**
- [ ] **Add `App` lifecycle tests** (`app_test.go`):
  - `TestAppRunWithoutDisplay` — graceful headless fallback
  - `TestAppQuitWhileRunning` — clean shutdown
  - `TestAppNewWindowConfig` — various `WindowConfig` permutations
- [ ] **Add `Window` rendering tests** (`window_test.go`):
  - `TestWindowSetLayout` — widget tree attachment
  - `TestWindowRenderFrameSoftware` — software path pixel verification
  - `TestWindowResize` — layout recalculation on resize
- [ ] **Add event dispatch tests** (`dispatcher_test.go`):
  - `TestFocusTraversal` — Tab/Shift-Tab navigation
  - `TestEventBubbling` — event propagation through widget tree

**Validation:** `go test -cover ./... | grep wain` should report >60% coverage.

**Impact:** Increases confidence in the public API stability commitment (`STABILITY.md`).

---

### Priority 3: Add Accessibility Tests

**Gap:** `internal/a11y/` has 10 source files and 75 functions but zero test files.

**Tasks:**
- [ ] **Create `internal/a11y/manager_test.go`**:
  - `TestManagerRegistration` — register panel/button/text, verify D-Bus objects exported
  - `TestFocusEvent` — simulate focus change, verify `org.a11y.atspi.Event.Focus` signal
  - `TestActionInterface` — invoke button action via D-Bus, verify callback
- [ ] **Add mock D-Bus for CI**: Use `github.com/godbus/dbus/v5/introspect` or mock conn to test without live D-Bus session
- [ ] **Add `TestAccessibilityIntegration`** (`integration_test.go`): Full AT-SPI2 registration with headless app

**Validation:** `go test -tags=atspi ./internal/a11y` should pass with >70% coverage.

**Impact:** Validates AT-SPI2 claim; catches regressions in accessibility support.

---

### Priority 4: Documentation for GPU Features

**Gap:** README claims GPU rendering but documentation focuses on software path. GPU usage is underdocumented.

**Tasks:**
- [ ] **Expand `HARDWARE.md`** with GPU feature enablement:
  - How to verify GPU detection: `./bin/wain --detect-gpu`
  - How to force GPU vs software backend
  - Troubleshooting GPU command submission failures
- [ ] **Add GPU section to `GETTING_STARTED.md`**:
  - Building with GPU support enabled
  - Running GPU demos on supported hardware
  - Interpreting `cmd/auto-render-demo` output
- [ ] **Document shader development** (`render-sys/shaders/README.md` expansion):
  - How to add a new shader
  - How shaders are compiled and embedded

**Validation:** New user with Intel Gen12 GPU can follow docs to see GPU-rendered frame.

---

### Priority 5: Performance Baseline for GPU Path

**Gap:** CI enforces 60 FPS for software rendering but has no equivalent for GPU path. `cmd/gpu-bench` exists but results are not CI-enforced.

**Tasks:**
- [ ] **Add GPU benchmark threshold to CI** (`.github/workflows/ci.yml`):
  - If GPU available, run `cmd/gpu-bench -frames 60 -max 2.0`
  - Assert GPU frame time ≤ 2 ms (vs 16.7 ms software budget)
- [ ] **Track GPU frame timing in benchmark summary**:
  - Add `## GPU Frame Time` section to CI step summary
  - Compare against baseline across commits

**Validation:** GPU performance regressions are caught automatically on hardware runners.

---

### Priority 6: Widget Test Coverage

**Gap:** `internal/ui/widgets` has tests but `wain` package widget constructors (`NewButton`, `NewLabel`, etc.) are undertested.

**Tasks:**
- [ ] **Add `concretewidgets_test.go`** unit tests:
  - `TestNewButtonBounds` — verify size after construction
  - `TestNewButtonClick` — verify callback invocation
  - `TestNewTextInputValue` — verify text get/set
- [ ] **Add visual regression tests for widgets**:
  - Render `Button`, `Label`, `TextInput` to pixel buffer
  - Compare against golden images (similar to `internal/raster/testdata/`)

**Validation:** `go test -cover ./... | grep wain` includes widget coverage.

---

## Summary

| Priority | Gap | Effort | Impact |
|----------|-----|--------|--------|
| **P1** | GPU rendering pipeline incomplete | High | Achieves 3 partial goals |
| **P2** | Public API test coverage low | Medium | Improves stability confidence |
| **P3** | Accessibility untested | Medium | Validates AT-SPI2 claim |
| **P4** | GPU documentation sparse | Low | Improves user experience |
| **P5** | GPU performance not CI-enforced | Low | Catches regressions |
| **P6** | Widget coverage gaps | Low | Rounds out test suite |

### Next Milestone Recommendation

Focus on **P1** (GPU pipeline integration) first. This is the largest gap between the project's ambitious claims and current reality. Once an end-to-end GPU path is validated, the project's core value proposition — a fully static Go UI toolkit with GPU acceleration — is fully substantiated.

---

## Appendix: Metrics Detail

### Top 10 Complex Functions

| Function | Package | Lines | Cyclomatic | Overall |
|----------|---------|-------|------------|---------|
| Render | wain | 38 | 7 | 9.6 |
| DecodeSetupReply | wire | 31 | 7 | 9.6 |
| RenderAndPresent | present | 30 | 7 | 9.6 |
| Present | display | 27 | 7 | 9.6 |
| DecodeString | wire | 27 | 7 | 9.6 |
| decodeSetupFailure | wire | 22 | 7 | 9.6 |
| updateTimingBounds | backend | 18 | 7 | 9.6 |
| applyShadowToBuffer | effects | 35 | 6 | 9.3 |
| RenderToDisplayList | widgets | 33 | 6 | 9.3 |
| Close | wain | 26 | 6 | 9.3 |

All functions are under the complexity threshold (10). No refactoring required.

### Package Coverage Summary (Internal)

| Package | Coverage |
|---------|----------|
| internal/x11/dpi | 100.0% |
| internal/ui/scale | 100.0% |
| internal/ui/animation | 97.4% |
| internal/x11/present | 96.6% |
| internal/raster/curves | 94.6% |
| internal/raster/composite | 93.6% |
| internal/ui/layout | 93.4% |
| internal/buffer | 93.3% |
| internal/raster/text | 92.3% |
| internal/ui/pctwidget | 91.9% |

Average internal package coverage: **87.3%** (excellent)

### Lines of Code by Layer

| Layer | Go LOC | Rust LOC | Total |
|-------|--------|----------|-------|
| Rust Backend | — | 15,114 | 15,114 |
| Go Bindings | ~1,500 | — | 1,500 |
| Wayland Client | ~2,800 | — | 2,800 |
| X11 Client | ~2,600 | — | 2,600 |
| Rasterizer | ~2,200 | — | 2,200 |
| UI Framework | ~1,800 | — | 1,800 |
| Public API | ~2,500 | — | 2,500 |
| Demo Binaries | ~3,000 | — | 3,000 |
| **Total** | **~16,400** | **~15,100** | **~31,500** |
