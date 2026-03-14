# Implementation Plan: Close Correctness, Coverage, and Complexity Gaps

**Generated:** 2026-03-14  
**Tool:** go-stats-generator v1.0.0 + AUDIT.md + GAPS.md cross-reference  
**Repository:** `github.com/opd-ai/wain`

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit for Linux that renders via a Rust GPU backend with automatic software fallback, implementing Wayland and X11 display protocols directly to produce fully static, zero-dependency binaries.
- **Current goal**: Achieve full correctness across all documented features — specifically closing the `ImageWidget` software-mode regression, the sole complexity outlier, and coverage gaps on critical frame-delivery paths.
- **Estimated Scope**: Medium (1 correctness bug + 1 complexity outlier + 5 coverage gaps + 2 feature completions + minor hygiene)

---

## Goal-Achievement Status

| # | Stated Goal | Current Status | This Plan Addresses |
|---|-------------|----------------|---------------------|
| 1 | Single static binary (zero deps) | ✅ Achieved | No |
| 2 | Wayland client (9 packages) | ✅ Achieved | No |
| 3 | X11 client (9 packages) | ✅ Achieved | No |
| 4 | Software 2D rasterizer (7 packages) | ✅ Achieved | No |
| **5** | **Widget system (all widgets functional)** | **⚠️ Partial — `ImageWidget` silently invisible in software mode** | **Yes (Step 1)** |
| 6 | Layout containers | ✅ Achieved | No |
| **7** | **GPU command submission end-to-end** | **⚠️ Partial — no CI-tested path; hardware-gated only** | **Yes (Step 8)** |
| 8 | Shader compilation (naga) | ✅ Achieved | No |
| 9 | Intel EU backend | ⚠️ Partial — compilation CI gate passes; no mock-hardware integration test | Yes (Step 8) |
| 10 | AMD RDNA backend | ⚠️ Partial — same gap as #9 | Yes (Step 8) |
| 11 | Public API (App, Window, Widget) | ✅ Achieved | No |
| 12 | Display server auto-detection | ✅ Achieved | No |
| 13 | Renderer auto-detection | ✅ Achieved | No |
| 14 | AT-SPI2 accessibility | ✅ Achieved (TD-6 resolved; 74.2% coverage with `-tags=atspi`) | No |
| 15 | 60 FPS software rendering | ✅ Achieved | No |
| 16 | DMA-BUF buffer sharing | ✅ Achieved | No |
| 17 | Clipboard support | ✅ Achieved | No |
| 18 | Client-side decorations | ✅ Achieved | No |
| 19 | HiDPI / DPI-aware scaling | ✅ Achieved | No |
| 20 | Keyboard accessibility (Tab focus) | ✅ Achieved | No |

---

## Metrics Summary

_Fresh go-stats-generator analysis: 2026-03-14, 199 files, --skip-tests._

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Go LOC | 14,786 | Moderate |
| Total functions | 667 | — |
| Total methods | 1,169 | — |
| Total packages | 40 (non-test) | Well-modularized |
| **Complexity outliers (CC > 10)** | **1 — `processWaylandDragEvents` CC=20** | ❌ Only outlier |
| All other functions max CC | 7 (overall 9.6) | ✅ Excellent |
| Duplication ratio | 0.63% | ✅ Excellent |
| Doc coverage | 91.2% (functions: 98.3%) | ✅ Excellent |
| Circular dependencies | 0 | ✅ Excellent |

### Coverage Hotspots (go test -cover ./...)

| Package | Coverage | Risk |
|---------|----------|------|
| `internal/render/present` | **0%** | ❌ Frame pipeline orchestration untested |
| `internal/a11y` | 0% (74.2% with `-tags=atspi`) | ✅ TD-6 resolved |
| `internal/render/display` | **1.9%** | ❌ Frame delivery path nearly untested |
| `internal/wayland/input` | **25.0%** | ⚠️ Core input handling undertested |
| `internal/x11/wire` | 27.1% | ⚠️ Protocol codec undertested |
| `internal/render/atlas` | **27.0%** | ⚠️ LRU eviction logic untested |
| `internal/x11/dri3` | 26.0% | ⚠️ DRI3 GPU buffer sharing undertested |
| `internal/render/backend` | 53.7% | ⚠️ GPU path undertested |
| `internal/render` | 44.0% | ⚠️ Go–Rust bindings undertested |
| `github.com/opd-ai/wain` (root) | 60.0% | ⚠️ Public API partially tested |

---

## Implementation Steps

Steps are ordered: **correctness first → complexity → critical coverage gaps → performance → feature completion → hygiene**.

---

### Step 1: Fix `ImageWidget` Software Rendering (Correctness)

- **Deliverable**: `internal/raster/consumer/software.go` — replace the `case displaylist.CmdDrawImage: // Skip for now` stub with a working `renderDrawImage` helper that blits the image's `image.RGBA` pixels using `composite.BlitScaled`. Add a test `TestImageWidgetSoftwarePath` in `internal/raster/consumer/`.
- **Files**: `internal/raster/consumer/software.go`, `internal/raster/consumer/software_test.go` (or existing test file)
- **Dependencies**: None (self-contained; `composite.BlitScaled` is already imported in related files)
- **Goal Impact**: Closes the only correctness gap in stated goal #5 (Widget system). README advertises `ImageWidget` as first-class; this makes that claim true on software-fallback systems (the majority of user machines).
- **Acceptance**: `go test -run TestImageWidgetSoftwarePath ./internal/raster/consumer/` passes; the test constructs an `ImageWidget`, renders it through `SoftwareConsumer`, and asserts the output buffer is non-zero within the widget's bounding box.
- **Validation**:
  ```bash
  go test -run TestImageWidgetSoftwarePath -count=1 ./internal/raster/consumer/
  go test -cover ./internal/raster/consumer/ | grep coverage
  # Expected: coverage climbs from 85.3% → ≥88%
  ```

---

### Step 2: Reduce `processWaylandDragEvents` Complexity (Code Quality)

- **Deliverable**: `app.go` — extract the four drag-event branches of `processWaylandDragEvents` (enter, motion, leave, drop) into four private helpers: `handleWaylandDragEnter`, `handleWaylandDragMotion`, `handleWaylandDragLeave`, `handleWaylandDrop`. The orchestrating function becomes a four-line dispatcher.
- **Files**: `app.go`
- **Dependencies**: None (pure refactor; no behavior change)
- **Goal Impact**: Eliminates the sole complexity outlier (CC=20, overall=27.5) in an otherwise excellent codebase (all other functions ≤ CC=7). Reduces regression risk in drag-and-drop, which was previously broken (TD-1) and was fixed in v1.1.
- **Acceptance**: `processWaylandDragEvents` reports CC ≤ 5; all four helper functions report CC ≤ 6; all existing tests continue to pass.
- **Validation**:
  ```bash
  go test -count=1 ./...
  go-stats-generator analyze . --skip-tests --format json --sections functions \
    | jq '[.functions[] | select(.name == "processWaylandDragEvents")] | .[0].complexity.cyclomatic'
  # Expected: ≤5
  ```

---

### Step 3: Test `internal/render/present` (Coverage)

- **Deliverable**: `internal/render/present/present_test.go` (new file) — unit tests for `RenderAndPresent` (CC=7, overall=9.6) using mock `FramebufferPool` and `PlatformPresenter` implementations that record `RenderToFramebuffer` and `PresentBuffer` call sequences. Tests cover: normal frame submission, back-pressure (pool full), and context cancellation.
- **Files**: `internal/render/present/present_test.go` (new)
- **Dependencies**: None (mock-only; no display server required)
- **Goal Impact**: Brings the frame pipeline orchestrator from 0% coverage to ≥70%. A regression in `RenderAndPresent` sequencing (wrong buffer hand-off order, double-present) is currently invisible to CI.
- **Acceptance**: `go test -cover ./internal/render/present/` reports ≥70%.
- **Validation**:
  ```bash
  go test -cover ./internal/render/present/
  # Expected: coverage: ≥70% of statements
  ```

---

### Step 4: Test `internal/render/display` Frame Delivery (Coverage)

- **Deliverable**: Expand `internal/render/display/software_test.go` (or add `display_test.go`) to cover `renderToFramebuffer`, `ensureWaylandBuffer`, and `PresentBuffer` on both `SoftwareWaylandPresenter` and `SoftwareX11Presenter`. Use a fake `wl_shm` pool (in-memory byte buffer) and a `net.Pipe()` mock socket to avoid needing a real compositor.
- **Files**: `internal/render/display/software_test.go` (extend existing), `internal/render/display/display_test.go` (new, if needed)
- **Dependencies**: Step 3 is not a hard prerequisite but provides the vocabulary for mock interfaces.
- **Goal Impact**: Raises `internal/render/display` from 1.9% → ≥50%. Frame delivery is the penultimate step in the render→display→compositor pipeline; a 1.9% coverage here means every frame-presentation regression escapes CI.
- **Acceptance**: `go test -cover ./internal/render/display/` reports ≥50%.
- **Validation**:
  ```bash
  go test -cover ./internal/render/display/
  # Expected: coverage: ≥50% of statements
  ```

---

### Step 5: Test `internal/render/atlas` Eviction Logic (Coverage)

- **Deliverable**: Extend `internal/render/atlas/` tests to cover: (1) eviction triggered when atlas capacity is exceeded, (2) re-insertion of an evicted glyph, (3) atlas reset / invalidation. These are the code paths at 27% that directly affect text and image rendering in GPU mode.
- **Files**: `internal/render/atlas/atlas_test.go` (extend or new)
- **Dependencies**: None
- **Goal Impact**: Raises `internal/render/atlas` from 27.0% → ≥70%. Bugs in LRU eviction produce invisible text or wrong images in GPU mode; this gap is purely untested logic, not hardware-gated.
- **Acceptance**: `go test -cover ./internal/render/atlas/` reports ≥70%.
- **Validation**:
  ```bash
  go test -cover ./internal/render/atlas/
  # Expected: coverage: ≥70% of statements
  ```

---

### Step 6: Test `internal/wayland/input` Event Handling (Coverage)

- **Deliverable**: Add or extend tests in `internal/wayland/input/` to cover fake Wayland wire messages for: pointer button press/release, axis scroll, keyboard key-press/release (including modifier keys), and touch-down/up sequences. Use in-process wire encoding to simulate compositor-to-client events without a live Wayland socket.
- **Files**: `internal/wayland/input/input_test.go` (new or extend existing)
- **Dependencies**: None (can reuse wire encoding already tested in `internal/wayland/wire/`)
- **Goal Impact**: Raises `internal/wayland/input` from 25.0% → ≥60%. Input handling is central to interactive applications; the `handleKeyEvent` and `handleEnterEvent` helpers are currently untested.
- **Acceptance**: `go test -cover ./internal/wayland/input/` reports ≥60%.
- **Validation**:
  ```bash
  go test -cover ./internal/wayland/input/
  # Expected: coverage: ≥60% of statements
  ```

---

### Step 7: Replace `AcquireForWriting` Polling with Condition Signal (Performance)

- **Deliverable**: `internal/buffer/ring.go` — replace the `time.After(5ms)` polling loop in `AcquireForWriting` (lines 160–175) with a `sync.Cond` wait. Add a `cond *sync.Cond` field to `Ring`. In `markSlotTransition` (called by `MarkReleased`), call `r.cond.Signal()` after the state change. `AcquireForWriting` calls `r.cond.Wait()` instead of sleeping.
- **Files**: `internal/buffer/ring.go`
- **Dependencies**: None (self-contained)
- **Goal Impact**: Advances the "Double/Triple Buffering — frame synchronization with compositor" claim by reducing worst-case slot acquisition latency from 5 ms → ≈100 µs (OS scheduling granularity). Under the 16.7 ms frame budget, 5 ms polling consumes 30% of the worst-case budget. Aligns with the `GAPS.md` finding on `AcquireForWriting`.
- **Acceptance**: Existing `internal/buffer/` tests pass without modification; a new benchmark `BenchmarkAcquireForWriting` shows p99 acquisition latency < 500 µs.
- **Validation**:
  ```bash
  go test -count=1 ./internal/buffer/
  go test -bench=BenchmarkAcquire -benchtime=5s ./internal/buffer/
  # Expected: all tests pass; benchmark shows reduced latency vs polling baseline
  ```

---

### Step 8: GPU Pipeline Mock-Hardware Integration Test (Feature Completion)

- **Deliverable**: `internal/integration/gpu_pipeline_test.go` (extend existing or add `//go:build integration` test) — create a `GPUBackend` with a mock DRM allocator (wraps an in-memory buffer instead of calling `ioctl`), render a minimal display list (one filled rectangle), and assert: (1) the produced batch byte slice is non-empty, (2) the batch header is structurally valid (correct magic, non-zero length), (3) no panic occurs. Runs unconditionally under `-tags=integration`; real-hardware submission path continues to be gated on `/dev/dri/renderD128`.
- **Files**: `internal/integration/gpu_pipeline_test.go` (extend), `.github/workflows/ci.yml` (add `-tags=integration` step)
- **Dependencies**: Steps 3–4 not required but Step 5 (atlas) ensures text rendering in GPU mode won't regress silently.
- **Goal Impact**: Moves GPU command submission (goals #7, #9, #10) from "hardware-gated only" to "CI-tested at structural level". Satisfies the `GAPS.md` and `AUDIT.md` finding that "GPU rendering regressions go undetected in CI".
- **Acceptance**: `go test -tags=integration -run TestGPUPipelineMock ./internal/integration/` passes without GPU hardware.
- **Validation**:
  ```bash
  go test -tags=integration -run TestGPUPipelineMock -count=1 ./internal/integration/
  go test -cover ./internal/integration/
  # Expected: test passes; integration coverage improves from 8.9%
  ```

---

### Step 9: GPU Documentation (Documentation)

- **Deliverable**: Two documentation additions:
  1. `GETTING_STARTED.md` — add a "GPU Usage" section covering: how `AppConfig{Verbose: true}` logs the detected backend, how `AppConfig{ForceSoftware: true}` forces software fallback, and how to interpret `cmd/auto-render-demo` output.
  2. `render-sys/shaders/README.md` — document how to add a new WGSL shader: write the shader, add it to the `shaders.rs` constant table, add a CI validation entry in `shader_compile.rs`.
- **Files**: `GETTING_STARTED.md`, `render-sys/shaders/README.md` (new)
- **Dependencies**: None
- **Goal Impact**: GPU-capable users currently default to software rendering because there is no guided path from "I have this GPU" to "I can see a GPU-rendered frame." This closes the `GAPS.md` "GPU Documentation Gap" finding.
- **Acceptance**: A new user with an Intel Gen12 laptop can follow the documentation to verify GPU detection (`Verbose: true` output) and run `cmd/auto-render-demo` successfully.
- **Validation**: Manual review — no automated metric.

---

### Step 10: Hygiene — go.sum Tidy and nil FontAtlas Warning (Low Risk)

- **Deliverable**: Two small fixes:
  1. Run `go mod tidy` to remove stale `golang.org/x/sys v0.20.0` and `v0.42.0` entries from `go.sum`.
  2. `internal/render/backend/gpu.go` — in `RenderWithDamage` (or equivalent), when a `CmdDrawText` command is encountered with a nil `FontAtlas`, emit a one-time `log.Printf("wain/backend: font atlas not set — text rendering disabled in GPU mode")` via `sync.Once`.
- **Files**: `go.sum`, `internal/render/backend/gpu.go`
- **Dependencies**: None
- **Goal Impact**: Eliminates silent failures (nil atlas produces invisible text with no diagnostic), and removes stale dependency entries that confuse `go mod verify` audits.
- **Acceptance**: `go mod verify` succeeds; a test that calls `Render` with a nil atlas and captures log output asserts the warning is logged exactly once.
- **Validation**:
  ```bash
  go mod verify
  go test -run TestGPUNilAtlasWarning -count=1 ./internal/render/backend/
  ```

---

## Dependency Order

```
Step 1  (ImageWidget)           → independent
Step 2  (processWaylandDragEvents) → independent
Step 3  (render/present tests)  → independent
Step 4  (render/display tests)  → Step 3 recommended first (shares mock patterns)
Step 5  (render/atlas tests)    → independent
Step 6  (wayland/input tests)   → independent
Step 7  (AcquireForWriting)     → independent
Step 8  (GPU mock test)         → Step 5 recommended first (atlas needed for GPU text)
Step 9  (GPU docs)              → independent
Step 10 (hygiene)               → independent
```

Steps 1–2 (correctness and complexity) should be completed before 3–10 to keep the codebase in a clean state for test authorship.

---

## Scope Assessment

| Metric | Threshold | Current | Assessment |
|--------|-----------|---------|------------|
| Functions above CC=10 | Small: <5 | 1 (`processWaylandDragEvents` CC=20) | Small |
| Duplication ratio | Small: <3% | 0.63% | Small |
| Doc coverage gap | Small: <10% | 8.8% gap (91.2% actual) | Small |
| Coverage gaps (packages < 30%) | Medium: 5–15 | 5 packages (render/present, render/display, wayland/input, render/atlas, x11/dri3) | Medium |

**Overall scope: Medium** — dominated by coverage work across 5 packages, plus one correctness bug and one complexity refactor.

---

## Metrics Targets After All Steps Complete

| Metric | Baseline | Target |
|--------|----------|--------|
| Functions above CC=10 | 1 | **0** |
| `internal/render/present` coverage | 0% | ≥70% |
| `internal/render/display` coverage | 1.9% | ≥50% |
| `internal/render/atlas` coverage | 27% | ≥70% |
| `internal/wayland/input` coverage | 25% | ≥60% |
| `internal/integration` coverage | 8.9% | ≥20% |
| ImageWidget software rendering | ❌ Silent skip | ✅ Functional |
| `AcquireForWriting` p99 latency | ≤5 ms | ≤500 µs |
| `go mod verify` | ⚠️ Stale entries | ✅ Clean |
