# Goal-Achievement Assessment

**Generated:** 2026-03-14  
**Updated:** 2026-03-14  
**Analyzed by:** go-stats-generator v1.0.0  
**Repository:** github.com/opd-ai/wain

---

## Project Context

### What it Claims to Do

Wain is a **statically-compiled Go UI toolkit** that:

1. **Go–Rust Static Linking** — CGO bridge to a Rust `staticlib`, producing a single binary with no dynamic dependencies
2. **Wayland Client** — 9-package implementation covering wire format, SHM buffers, xdg-shell, pointer/keyboard input, DMA-BUF, clipboard, output handling
3. **X11 Client** — 10-package implementation covering connection setup, window operations, MIT-SHM, DRI3 GPU buffer sharing, Present frame sync, DPI scaling, clipboard, drag-and-drop
4. **Software 2D Rasterizer** — Filled/rounded rectangles, anti-aliased lines, Bézier curves, SDF text, box shadows, gradients, Porter-Duff alpha compositing
5. **UI Widget Layer** — Flexbox-like Row/Column layout, percentage-based sizing, Button/TextInput/ScrollContainer widgets, client-side decorations, DPI-aware scaling, animation
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
| **L3: Protocol** | `internal/wayland/` (9 pkg), `internal/x11/` (10 pkg) | Display server clients |
| **L4: Rasterizer** | `internal/raster/` (7 pkg) | Software 2D rendering, display lists |
| **L5: UI Framework** | `internal/ui/` (6 pkg) | Layout, widgets, decorations, scaling, animation |
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
| 3 | X11 client (10 packages) | ✅ Achieved | `internal/x11/`: wire, client, events, gc, shm, dri3, present, dpi, selection, dnd; 10 packages with 75–100% coverage | — |
| 4 | Software 2D rasterizer (7 packages) | ✅ Achieved | `internal/raster/`: primitives, curves, composite, effects, text, displaylist, consumer; coverage 85–94% | — |
| 5 | UI widget layer (6 packages) | ✅ Achieved | `internal/ui/`: layout, pctwidget, widgets, decorations, scale, animation; Button, TextInput, ScrollContainer implemented | — |
| 6 | GPU buffer infrastructure | ✅ Achieved | `render-sys/src/allocator.rs`, `slab.rs`, `drm.rs`, `i915.rs`, `xe.rs`, `amd.rs` (~3800 lines Rust) | — |
| 7 | GPU command submission | ⚠️ Partial | `render-sys/src/batch.rs`, `pipeline.rs`, `surface.rs`, `cmd/` exist (~2500 lines); Intel Gen9–12 batches functional; GPU backend wired into `App.RenderFrame` via `backend.Renderer` interface | No dedicated `cmd/gpu-ui-demo` for interactive UI rendered entirely via GPU; `gpu_pipeline_test.go` end-to-end test not yet created |
| 8 | Shader frontend (naga) | ✅ Achieved | `render-sys/src/shader.rs` (538 lines); 7 WGSL shaders in `render-sys/shaders/`; `cmd/shader-test` validates all 7 | — |
| 9 | Intel EU backend | ⚠️ Partial | `render-sys/src/eu/` (6 files, ~180 KB total); register allocator, instruction lowering, encoding for Gen9+ | `lower.rs` is 116 KB — likely generated/tablegen code; shader→EU compilation not exercised in CI |
| 10 | AMD RDNA backend | ⚠️ Partial | `render-sys/src/rdna/` (6 files, ~44 KB), `amd.rs`, `pm4.rs` | Similar gap: ISA encoding exists but no shader→RDNA compilation path exercised |
| 11 | Public API (App, Window, Widget) | ✅ Achieved | `app.go`, `widget.go`, `publicwidget.go`, `resource.go`, `event.go`, `dispatcher.go`; `STABILITY.md` pins 13 constructors, 7 methods, 5 interfaces | — |
| 12 | Display server auto-detection | ✅ Achieved | `app.go`: tries Wayland (`$WAYLAND_DISPLAY`) first, falls back to X11 (`$DISPLAY`) | — |
| 13 | Renderer auto-detection | ✅ Achieved | `internal/render/backend/backend.go`: Intel→AMD→software fallback chain; `cmd/auto-render-demo` demonstrates | — |
| 14 | AT-SPI2 accessibility | ✅ Achieved | `internal/a11y/` (10 files, 75 functions); `accessibility.go` exposes `EnableAccessibility`; requires `-tags=atspi`; `a11y_test.go` and `manager_test.go` provide test coverage | — |
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
| Total Lines of Code (Go, non-test) | 34,427 | Substantial codebase |
| Total Lines of Code (Go, tests) | 32,094 | Extensive test suite |
| Total Lines of Code (Rust) | ~15,114 | Substantial GPU backend |
| Total Functions | 668 | — |
| Total Methods | 1,174 | — |
| Total Packages | 74 (43 non-cmd) | Well-modularized |
| Total Test Files | 98 | — |
| Average Function Length | 9.4 lines | Excellent |
| Functions > 50 lines | 7 (0.4%) | Excellent |
| High Complexity (>10) | **0** | Excellent |
| Average Complexity | 3.1 | Excellent |
| Circular Dependencies | **0** | Excellent |
| Duplication Ratio | 0.64% | Excellent |
| Average Test Coverage | 85%+ (internal packages) | Strong |

### Code Health Assessment

The codebase is **exceptionally well-structured**:
- No functions exceed cyclomatic complexity 10 (max observed: 9.6)
- Near-zero code duplication (0.64%)
- Zero circular dependencies
- Consistent naming conventions (31 minor violations, mostly single-letter loop vars)
- Strong test coverage in internal packages (85–100%)
- Root package has 325+ test functions across 12 test files

### Risk Areas

1. **GPU backend exercised in isolation** — No CI-tested path from shader compilation → EU/RDNA encoding → display
2. **`internal/a11y` test coverage is growing** — `manager_test.go` (40 tests) and `a11y_test.go` (11 tests) exist; mock D-Bus coverage could be expanded

---

## Roadmap

### Priority 1: Complete GPU Rendering Pipeline

**Gap:** GPU command submission exists but no dedicated GPU-rendered UI demo is available, and the end-to-end shader→EU/RDNA→display path is not exercised.

The Intel EU backend (`render-sys/src/eu/`) and AMD RDNA backend (`render-sys/src/rdna/`) have instruction encoding, but the path from:
> WGSL shader → naga IR → EU/RDNA binary → batch buffer → execbuffer2/amdgpu CS → display

is not exercised end-to-end.

**Tasks:**
- [ ] **Add GPU pipeline integration test** (`internal/integration/gpu_pipeline_test.go`):
  - Compile `solid_fill.wgsl` to EU binary via naga+lowering
  - Build batch buffer with state setup and primitive draw
  - Submit to GPU via execbuffer2 / amdgpu CS ioctl
  - Verify frame buffer contents
- [x] **Wire GPU backend into `App.RenderFrame`**: `backend.Renderer` interface is invoked for widget tree rendering via `renderBridge.Render(rootWidget)` in `Window.RenderFrame()`
- [ ] **Create `cmd/gpu-ui-demo`**: Interactive UI rendered entirely via GPU backend (validates claim "GPU-accelerated graphics")
- [x] **Add CI GPU smoke test**: `.github/workflows/ci.yml` includes `gpu-integration-tests` job that runs `TestGPURenderingTriangle` when `/dev/dri/renderD128` is available

**Validation:** `go test -tags=integration ./internal/integration -run TestGPUPipeline` should pass on Intel/AMD hardware.

**Impact:** Fully achieves goals #7 (GPU command submission), #9 (Intel EU backend), #10 (AMD RDNA backend).

---

### Priority 2: Increase Public API Test Coverage

**Gap:** Root package `wain` test coverage has improved significantly with 325+ test functions across 12 test files. Some specific lifecycle tests remain unimplemented.

**Tasks:**
- [x] **Add `App` lifecycle tests** (`app_test.go`):
  - 71 test functions covering headless operation, window config, display server fallback, and shutdown paths
- [x] **Add `Window` rendering tests** (`window_test.go`):
  - 8 test functions covering window configuration and construction
  - Additional window lifecycle tests in `app_test.go`
- [x] **Add event dispatch tests** (`event_test.go`):
  - `TestFocusTraversal` — Tab/Shift-Tab navigation ✅
  - `TestEventBubbling` — event propagation through widget tree ✅
  - Plus 18 additional event dispatch tests covering pointer, key, touch, custom events, and focus management

**Validation:** `go test -cover ./... | grep wain` should report >60% coverage.

**Impact:** Increases confidence in the public API stability commitment (`STABILITY.md`).

---

### Priority 3: ~~Add Accessibility Tests~~ ✅ Complete

**Gap:** ~~`internal/a11y/` has 10 source files and 75 functions but zero test files.~~ Resolved.

**Tasks:**
- [x] **Create `internal/a11y/manager_test.go`**: 40 test functions covering manager registration, focus events, and action interfaces
- [x] **Create `internal/a11y/a11y_test.go`**: 11 test functions covering accessible interface, roles, and states
- [ ] **Add `TestAccessibilityIntegration`** (`integration_test.go`): Full AT-SPI2 registration with headless app

**Validation:** `go test -tags=atspi ./internal/a11y` should pass with >70% coverage.

**Impact:** AT-SPI2 claim is now validated with test coverage; integration test would further strengthen confidence.

---

### Priority 4: ~~Documentation for GPU Features~~ ✅ Complete

**Gap:** ~~README claims GPU rendering but documentation focuses on software path. GPU usage is underdocumented.~~ Resolved.

**Tasks:**
- [x] **Expand `HARDWARE.md`** with GPU feature enablement:
  - Supported Intel generations (Gen9–12, Xe) with chipset detection IDs
  - Supported AMD generations (RDNA1–3) with family mappings
  - Kernel feature requirements, validation checklist, performance characteristics
- [x] **Add GPU section to `GETTING_STARTED.md`**:
  - Verifying GPU detection, forcing software rendering, running auto-render demo
  - GPU requirements table and reference to `HARDWARE.md`
- [x] **Document shader development** (`render-sys/shaders/README.md`):
  - All 7 WGSL shaders documented with purpose, uniforms, vertex format, algorithm
  - Integration guide for Intel EU Backend, resource binding translation
  - Instructions for adding new shaders (562-line comprehensive guide)

**Validation:** New user with Intel Gen12 GPU can follow docs to see GPU-rendered frame.

---

### Priority 5: ~~Performance Baseline for GPU Path~~ ✅ Complete

**Gap:** ~~CI enforces 60 FPS for software rendering but has no equivalent for GPU path.~~ Resolved.

**Tasks:**
- [x] **Add GPU benchmark threshold to CI** (`.github/workflows/ci.yml`):
  - `benchmarks` job runs `cmd/gpu-bench -frames 60 -max 2.0` on all runners; the binary exits 0 with `backend=none` when no GPU is present
  - GPU frame time threshold: ≤ 2 ms (vs 16.7 ms software budget)
  - Results output to `/tmp/gpu-bench.json`
- [ ] **Track GPU frame timing in benchmark summary**:
  - Add `## GPU Frame Time` section to CI step summary
  - Compare against baseline across commits

**Validation:** GPU performance regressions are caught automatically on hardware runners.

---

### Priority 6: ~~Widget Test Coverage~~ ✅ Complete

**Gap:** ~~`internal/ui/widgets` has tests but `wain` package widget constructors (`NewButton`, `NewLabel`, etc.) are undertested.~~ Resolved.

**Tasks:**
- [x] **Add `concretewidgets_test.go`** unit tests (72 test functions):
  - `TestNewButton`, `TestButtonOnClick`, `TestButtonSetEnabled` — button construction, click behavior, state management
  - `TestNewLabel`, `TestLabelSetText`, `TestLabelSetTextColor`, `TestLabelSetFontSize` — label construction and properties
  - `TestNewTextInput`, `TestTextInputSetText`, `TestTextInputOnChange`, `TestTextInputHandleKeyPress` — text input behavior
  - `TestNewScrollView`, `TestScrollViewSetScrollOffset`, `TestScrollViewHandleScrollEvent` — scroll container
  - `TestNewImageWidget`, `TestNewSpacer` — additional widgets
- [x] **Add visual regression tests for widgets**:
  - `internal/integration/screenshot_test.go`: `TestScreenshotGoldenImages` with `.rgba` golden files
  - `internal/raster/visual_test.go`: `TestVisual` rendering primitives against reference images

**Validation:** `go test -cover ./... | grep wain` includes widget coverage.

---

## Summary

| Priority | Gap | Effort | Impact |
|----------|-----|--------|--------|
| **P1** | GPU rendering pipeline incomplete (2 of 4 tasks remain) | Medium | Achieves 3 partial goals |
| **P2** | ~~Public API test coverage low~~ ✅ Complete | — | Stability confidence achieved |
| **P3** | ~~Accessibility untested~~ ✅ Complete (integration test remaining) | Low | AT-SPI2 claim validated |
| **P4** | ~~GPU documentation sparse~~ ✅ Complete | — | User experience improved |
| **P5** | ~~GPU performance not CI-enforced~~ ✅ Complete (tracking remaining) | Low | Regressions caught |
| **P6** | ~~Widget coverage gaps~~ ✅ Complete | — | Test suite rounded out |

### Next Milestone Recommendation

Focus on the remaining **P1** tasks: creating `cmd/gpu-ui-demo` and `internal/integration/gpu_pipeline_test.go`. These are the last major gaps between the project's claims and current reality. Once an end-to-end GPU UI demo is validated, the project's core value proposition — a fully static Go UI toolkit with GPU acceleration — is fully substantiated. The P5 benchmark tracking task is a low-effort follow-up.

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
| Go Bindings | ~4,800 | — | 4,800 |
| Wayland Client | ~5,300 | — | 5,300 |
| X11 Client | ~4,200 | — | 4,200 |
| Rasterizer | ~2,900 | — | 2,900 |
| UI Framework | ~2,900 | — | 2,900 |
| Accessibility | ~900 | — | 900 |
| Public API | ~6,100 | — | 6,100 |
| Demo Binaries | ~5,100 | — | 5,100 |
| Other (buffer, demo, integration) | ~2,100 | — | 2,100 |
| **Total** | **~34,400** | **~15,100** | **~49,500** |
