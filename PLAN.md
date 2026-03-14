# Implementation Plan: GPU Pipeline Completion & Post-v1.0 Roadmap

> **Generated:** 2026-03-14  
> **Tool:** `go-stats-generator v1.0.0` + ROADMAP.md goal analysis  
> **Baseline commit:** v1.0.0

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit that links a Rust rendering
  library via CGO/musl for GPU-accelerated graphics on Linux, implementing Wayland and X11
  display protocols from scratch in a single zero-dependency binary.
- **Current goal**: Complete the GPU rendering pipeline (Phase 5.2) — the highest-value
  unachieved differentiator — then extend the public API and add post-v1.0 features.
- **Estimated Scope**: Medium (Phase 5.2 stubs); Small (API surface); Medium (DnD/animations)

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| Go–Rust static linking | ✅ Achieved | No |
| Wayland client (9 packages) | ✅ Achieved | No |
| X11 client (9 packages) | ✅ Achieved | No |
| Software 2D rasterizer | ✅ Achieved | No |
| UI widget layer with flexbox layout | ✅ Achieved | No |
| GPU buffer infrastructure (alloc, DMA-BUF) | ✅ Achieved | No |
| GPU command submission — Intel | ⚠️ Partial (Phase 5.2 stubs) | **Yes — Step 1** |
| GPU command submission — AMD | ⚠️ Partial (Phase 5.2 stubs) | **Yes — Step 1** |
| Shader frontend (WGSL/GLSL via naga) | ✅ Achieved | No |
| Intel EU backend (Gen9+) | ✅ Achieved | No |
| AMD RDNA backend | ✅ Achieved | No |
| Public API with auto-detection | ✅ Achieved | No |
| Display list → GPU rendering | ⚠️ Partial (pipeline stubs) | **Yes — Step 1** |
| <2ms GPU frame time | ⚠️ Unverified (no GPU CI) | **Yes — Step 2** |
| 60 FPS software @ 1080p | ✅ Achieved (CI benchmark) | No |
| Zero runtime dependencies | ✅ Achieved | No |
| Keyboard accessibility | ✅ Achieved | No |
| AT-SPI2 screen reader | ✅ Achieved (`-tags=atspi`) | No |
| Clipboard (Wayland + X11) | ✅ Achieved | No |
| HiDPI/DPI-aware scaling | ✅ Achieved | No |
| `LoadImageFromReader` in public API | ❌ Not exposed | **Yes — Step 3** |
| Multi-window support | ❌ Not exposed | **Yes — Step 4** |
| Drag-and-drop (Wayland + X11) | ❌ Not implemented | **Yes — Step 5** |
| Property animations | ❌ Not implemented | **Yes — Step 6** |
| NVIDIA nouveau backend | ❌ Not planned | No |

**Overall before this plan: 16/23 goals fully achieved, 3 partial, 4 not implemented.**

---

## Metrics Summary

| Metric | Value | Assessment |
|---|---|---|
| Lines of Code (Go) | 14,314 | Moderate; 38 packages |
| Lines of Code (Rust) | ~15,114 | Substantial GPU code |
| Total functions | 651 functions + 1,132 methods | Well-distributed |
| High complexity (CC > 9) | **0** | Excellent |
| Doc coverage | **90.79%** (methods: 88.67%) | Exceeds 80% target |
| Duplication ratio | **0.61%** (202 lines, 10 clone pairs) | Excellent |
| Naming violations | **0** | Clean |
| Circular dependencies | **0** | Clean architecture |
| Functions > 50 lines | 7 (all in `cmd/` demos) | Acceptable |
| Dead code | 0% | Clean |
| `go vet` warnings | 0 | Clean |
| Rust `.unwrap()` in production | 0 in `lib.rs`; all others in `#[cfg(test)]` | Acceptable |

### Complexity hotspots on goal-critical paths
- `internal/render/backend/submit.go`: 0 high-CC functions, but **4 stub/placeholder
  command slots** that block real GPU rendering (lines 190, 207, 268–283). These are
  the only blockers on the GPU pipeline path.
- `internal/render/backend/gpu.go` `Render()`: correct structure, awaits Phase 5.2 fill.

### Package coupling (notable)
| Package | Coupling | Dependents | Note |
|---|---|---|---|
| `wain` (root) | 10.0 | Many consumers | Expected for public API hub |
| `internal/render/display` | 5.0 | 10 deps | Gateway between protocol and GPU layers |
| `internal/render/backend` | 2.5 | 5 deps | GPU consumer — Phase 5.2 target |

---

## Implementation Steps

---

### Step 1: Complete GPU Pipeline State (Phase 5.2)

**Goal Impact:** Finishes the highest-priority stated goal — GPU rendering for real UI
workloads. Without this, `GPUBackend.Render()` emits structurally correct command buffers
but with zero-filled pointers for viewport state, scissor state, and Surface State Base
Address, meaning the hardware will fault or silently produce no output.

**Deliverable:**
- `internal/render/backend/submit.go`: replace the 4 stub locations with computed values:
  1. **Line 190** — Surface State Base Address: emit a valid GPU-virtual address relocation
     for the render-target surface state (reference `render-sys/src/surface.rs`
     `SurfaceState` encoding; use `render.BufferHandle.GPUVirtualAddress()` from
     `internal/render/binding.go`).
  2. **Line 207** — scissor test enable: replace the "placeholder" scissor command with a
     proper `3DSTATE_SCISSOR_STATE_POINTERS` command pointing to a scissor descriptor
     written into the batch buffer (reference `render-sys/src/cmd/` `Scissor` struct).
  3. **Lines 268–277** — viewport and scissor state pointers: populate
     `3DSTATE_VIEWPORT_STATE_POINTERS_SF_CLIP` and
     `3DSTATE_VIEWPORT_STATE_POINTERS_CC` with addresses of `SF_CLIP_VIEWPORT` and
     `CC_VIEWPORT` descriptors encoded inline in the batch (reference Intel PRMs
     encapsulated in `render-sys/src/cmd/`).
  4. **Line 283** — `3DSTATE_PS` stub: expand to full pixel-shader state block that
     references the compiled solid-fill kernel binary stored in `b.solidFillShader`
     (already loaded in `GPUBackend.New()`, used only for WGSL validation today).
- `internal/render/backend/gpu.go`: add a `RenderPhase52()` method that calls the
  updated `submit.go` path and returns a `*render.BufferHandle` for DMA-BUF export.
- `render-sys/src/submit.rs` (Rust): expose a `render_submit_batch_with_state()` FFI
  function that accepts a prebuilt viewport/scissor descriptor blob alongside the
  existing batch buffer, so the Go side does not need to re-encode Rust-owned structs.
- `internal/render/binding.go`: add `SubmitBatchWithState(batch, stateBlob []byte) error`
  Go wrapper for the new FFI function.
- New test: `internal/render/backend/gpu_phase52_test.go` — table-driven test that calls
  `GPUBackend.Render()` with a minimal display list (one solid-fill rect) and verifies
  the batch buffer contains no zero-filled state pointers.

**Dependencies:** Step 0 (baseline — already met: `render-sys/src/surface.rs`,
`render-sys/src/cmd/`, `internal/render/binding.go` all exist).

**Acceptance:**
- `cmd/gpu-display-demo` renders a complete widget hierarchy via GPU without falling back
  to software (verified by adding a `--assert-gpu` flag that exits non-zero if the
  software path is taken).
- `internal/render/backend/gpu_phase52_test.go` passes: no zero words at state pointer
  offsets in the emitted command buffer.

**Validation:**
```bash
go test ./internal/render/backend/... -run TestPhase52 -v
go-stats-generator analyze ./internal/render/backend --skip-tests --format json \
  --sections functions | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Must be 0
```

---

### Step 2: GPU Frame-Time Regression Gate

**Goal Impact:** The stated goal of <2ms GPU frame time is currently unverified in CI
because no GPU hardware exists in the runner. This step establishes a reproducible
measurement path for hardware environments and a simulation baseline for CI.

**Deliverable:**
- `cmd/gpu-bench/main.go` (new binary): renders the same standardised scene as `cmd/bench`
  (500 rects, 100 text runs, 10 shadows) but through the GPU backend. Outputs JSON with
  `backend`, `mean_ms`, `p99_ms`, `pass` fields. Accepts `-max float` threshold flag.
  Falls back gracefully when `/dev/dri/renderD128` is absent (prints `"backend":"none"`
  and exits 0 — no CI failure on non-GPU runners).
- `.github/workflows/ci.yml`: add a `gpu-bench` step inside the existing `benchmarks` job
  that runs `go run ./cmd/gpu-bench -max 2.0` and appends a "GPU Frame Time" row to
  `$GITHUB_STEP_SUMMARY`. The step uses `continue-on-error: true` so CI never fails on
  runners without `/dev/dri`.
- `HARDWARE.md`: update the "Performance Claims" section to link to the new benchmark
  output and document how to reproduce locally.

**Dependencies:** Step 1 (GPU backend must emit real commands before timing them).

**Acceptance:**
- On a machine with an Intel GPU: `go run ./cmd/gpu-bench -max 2.0` exits 0 and reports
  mean frame time ≤ 2.0 ms for the standard scene.
- On a machine without GPU: `go run ./cmd/gpu-bench` exits 0 with `"backend":"none"`.
- CI `benchmarks` job Step Summary contains a "GPU Frame Time" row.

**Validation:**
```bash
go build ./cmd/gpu-bench && echo "build OK"
go run ./cmd/gpu-bench | jq '{backend, mean_ms, pass}'
```

---

### Step 3: Expose `LoadImageFromReader` in Public API

**Goal Impact:** `resource-demo/main.go` line 65 documents that
`ResourceManager.LoadImageFromReader` is "not exposed in the public API yet." This makes
image loading from arbitrary `io.Reader` sources (HTTP responses, zip archives,
in-memory assets) impossible via the public interface. The method already exists on the
internal `ResourceManager`; this step promotes it.

**Deliverable:**
- `resource.go` (root package): add a public method `(*App).LoadImageFromReader(r
  io.Reader, filenameHint string) (*Image, error)` that delegates to
  `a.resources.LoadImageFromReader(r, filenameHint)`. The implementation is a one-liner.
- `resource.go`: add a `ExampleApp_LoadImageFromReader` GoDoc example showing HTTP image
  loading into a widget.
- `compat_test.go`: add a compile-time signature pin:
  `var _ func(*App, io.Reader, string) (*Image, error) = (*App).LoadImageFromReader`.
- `cmd/resource-demo/main.go`: replace the `NOTE` comment with a call to the new public
  API to confirm it works end-to-end.

**Dependencies:** None (method exists; this is purely a promotion).

**Acceptance:**
- `go doc github.com/opd-ai/wain App.LoadImageFromReader` prints the method signature.
- `go test ./... -run TestCompat` passes (compat_test.go pin compiles).
- `cmd/resource-demo` builds and runs without the `// Note:` workaround.

**Validation:**
```bash
go vet ./...
go test ./... -run TestCompat
go doc github.com/opd-ai/wain App.LoadImageFromReader
go-stats-generator analyze . --skip-tests --format json --sections documentation \
  | jq '.documentation.coverage.methods'
# Must remain ≥ 88.0
```

---

### Step 4: Multi-Window Public API

**Goal Impact:** The infrastructure for multiple windows already exists — `app.go` uses a
`surfaceToWindow` map and the dispatch loop routes input to the correct window. The
comment at `app.go:2016` explicitly notes routing is "fully supported." However, the
public API exposes no way to create a second window after `Run()` has started, making
the feature invisible to users.

**Deliverable:**
- `app.go`: promote `(*App).NewWindow(WindowConfig) (*Window, error)` to be callable
  after `Run()` has been invoked (currently it panics if called post-Run). Use a
  `sync.Mutex`-protected window registry and send a new-window request on an internal
  channel consumed by the event loop, so creation is safe from any goroutine.
- `window.go` (or `app.go`): add `(*App).Windows() []*Window` to enumerate open windows.
- `event.go`: document that `EventWindowClose` on the last window triggers `App.Quit()`
  automatically (this is the expected UX for multi-window apps).
- `example/multi-window/main.go` (new): canonical example showing two independent windows
  with separate layouts, shared theme, and per-window close handling.
- `window_test.go`: add `TestMultiWindowCreation` using the headless/software path to
  verify two windows can be created, rendered, and closed independently.
- `compat_test.go`: pin `(*App).Windows() []*Window`.

**Dependencies:** Step 3 (confirms API promotion pattern), but can be developed in
parallel.

**Acceptance:**
- `example/multi-window` builds and runs: two independent windows appear.
- `TestMultiWindowCreation` passes in `go test ./...`.
- No data race reported by `go test -race ./...`.

**Validation:**
```bash
go test -race ./... -run TestMultiWindow -v
go-stats-generator analyze . --skip-tests --format json --sections functions \
  | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Must remain 0
go vet ./...
```

---

### Step 5: Drag-and-Drop (Wayland + X11)

**Goal Impact:** Explicitly listed as a 1-week post-v1.0 feature. DnD is essential for
productivity applications (file managers, text editors) and is the last major protocol
feature missing from both display-server backends.

**Deliverable:**

**Wayland DnD (`internal/wayland/datadevice/dnd.go`, new file):**
- Implement `wl_data_source` offer + `wl_data_device.start_drag` on the source side.
- Implement `wl_data_device.enter`/`motion`/`drop`/`leave` events on the target side.
- Expose `DragSource` and `DropTarget` types with MIME-type negotiation.

**X11 DnD (`internal/x11/dnd/dnd.go`, new package):**
- Implement XDND protocol v5 (XdndEnter, XdndPosition, XdndStatus, XdndDrop,
  XdndFinished, XdndLeave client messages).
- Integrate with the existing `internal/x11/events` dispatch loop.

**Public API (`app.go`, `event.go`):**
- Add `EventDragEnter`, `EventDragMove`, `EventDragDrop`, `EventDragLeave` event types.
- Add `(*Window).SetDropTarget(mimeTypes []string, handler DragDropHandler)`.
- Add `(*Window).StartDrag(source DragDataProvider, icon *Image)`.

**Tests:**
- `internal/wayland/datadevice/dnd_test.go`: unit tests for source/offer encoding.
- `internal/x11/dnd/dnd_test.go`: unit tests for XDND message construction.
- `integration_test.go`: add `TestDragDropHeadless` using a mock event source.

**Dependencies:** Step 4 (multi-window event routing pattern).

**Acceptance:**
- `cmd/clipboard-demo` updated to also demonstrate drag-and-drop of text between two
  windows (serves as a live integration test).
- `go test ./internal/wayland/datadevice/... ./internal/x11/dnd/...` passes.
- No new high-complexity functions (CC ≤ 9): each handler dispatches to focused helpers
  following the pattern established in `internal/wayland/input/` (see
  `handleEnterEvent`, `handleKeyEvent`).

**Validation:**
```bash
go test ./internal/wayland/datadevice/... ./internal/x11/dnd/... -v
go-stats-generator analyze ./internal/wayland/datadevice ./internal/x11/dnd \
  --skip-tests --format json --sections functions \
  | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Must be 0
go test -race ./... -run TestDragDrop
```

---

### Step 6: Property Animations

**Goal Impact:** Explicitly listed as a 2-week post-v1.0 feature. Animations are the last
major UI-layer gap before wain is competitive with GTK/Qt for application development.

**Deliverable:**
- `internal/ui/animation/animation.go` (new package):
  - `Animation` struct: target widget, property name (e.g., `"Opacity"`, `"Width"`),
    from/to values (`float64`), duration, easing function, completion callback.
  - Built-in easing functions: `Linear`, `EaseIn`, `EaseOut`, `EaseInOut` (cubic
    Bézier), `Spring`.
  - `Animator` manages a list of running animations, driven by the frame loop — no
    goroutines, no timers; driven by `Tick(dt time.Duration)` called from the render loop.
- `app.go`: wire `Animator.Tick(dt)` into the existing `renderFrame()` path so animations
  advance on every frame without API changes to callers.
- Public API additions in `widget.go`:
  - `(*App).Animate(w Widget, property string, to float64, dur time.Duration,
    easing EasingFunc) *Animation`
  - `(*Animation).OnComplete(func())`
  - `(*Animation).Cancel()`
- `internal/ui/animation/animation_test.go`: table-driven tests for all easing functions
  and edge cases (zero duration, cancel mid-flight, chained animations).
- `cmd/animations-demo/main.go` (new): demonstrates slide-in panel, fade button, and
  spring-based scroll snap.

**Dependencies:** Step 4 (multi-window event loop must be stable before threading
animation ticks through it).

**Acceptance:**
- `TestAnimationLinear`, `TestAnimationSpring`, `TestAnimationCancel` all pass.
- `cmd/animations-demo` builds and runs.
- No new high-complexity functions (CC ≤ 9): Tick loop dispatches to per-animation
  `advance()` methods; easing is a pure `func(t float64) float64`.
- Doc coverage for `internal/ui/animation` ≥ 90%.

**Validation:**
```bash
go test ./internal/ui/animation/... -v
go-stats-generator analyze ./internal/ui/animation --skip-tests --format json \
  --sections functions,documentation \
  | jq '{cc_violations: ([.functions[] | select(.complexity.cyclomatic > 9)] | length),
         doc_coverage: .documentation.coverage.overall}'
# cc_violations must be 0, doc_coverage ≥ 90
go build ./cmd/animations-demo && echo "build OK"
```

---

## Dependency Graph

```
Step 1 (GPU Phase 5.2)
  └── Step 2 (GPU bench gate)

Step 3 (LoadImageFromReader)   ← independent

Step 4 (Multi-window)
  └── Step 5 (Drag-and-drop)
        └── Step 6 (Animations — needs stable event loop)
```

Steps 1–3 can be worked in parallel. Steps 4, 5, 6 form a chain.

---

## Scope Assessment

| Step | Metric Basis | Items Above Threshold | Scope |
|---|---|---|---|
| 1 — GPU Phase 5.2 | Stub command slots in `submit.go` | 4 placeholder locations; 1 new FFI function | Small |
| 2 — GPU bench gate | New binary + 1 CI step | 1 binary, 1 CI stanza | Small |
| 3 — LoadImageFromReader | New public method | 1 method, 1 example, 1 compat pin | Small |
| 4 — Multi-window API | Event-loop change + API | 3 new public methods, 1 channel | Small |
| 5 — Drag-and-drop | New packages (Wayland + X11) | 2 new packages, ~8 event types | Medium |
| 6 — Animations | New package + frame-loop wiring | 1 new package, ~5 public types | Medium |

---

## Thresholds Reference (project-calibrated)

| Metric | Green | Yellow | Red |
|---|---|---|---|
| Functions above CC 9 | 0 | 1–4 | ≥ 5 |
| Duplication ratio | < 1% | 1–3% | > 3% |
| Doc coverage | ≥ 90% | 80–90% | < 80% |
| Long functions > 50 lines in library code | 0 | 1–3 | > 3 |

All thresholds tightened from the generic defaults to match the project's own
demonstrated quality (0 CC violations, 0.61% duplication, 90.79% doc coverage at v1.0.0).

---

## Risk Register

| Risk | Severity | Mitigation |
|---|---|---|
| GPU Phase 5.2 causes driver faults on some Intel gens | HIGH | Test on Gen9, Gen11, Gen12 (Xe) before merging; software fallback always available |
| Multi-window Wayland concurrency (registry bind race) | MEDIUM | Protected by `sync.Mutex` window registry + channel dispatch; `go test -race` gate |
| DnD MIME negotiation complexity increases CC | MEDIUM | Each MIME handler is a lookup function; enforce CC ≤ 9 via `go-stats-generator` gate |
| Animation goroutine-free design missed edge cases | LOW | Property `Tick(dt)` model is fully deterministic; table-driven tests cover edge cases |
| `LoadImageFromReader` public promotion breaks compat | LOW | `compat_test.go` pin is additive-only; no signature changes to existing functions |

---

## Appendix: Baseline Metrics Command

```bash
go-stats-generator analyze . --skip-tests --format json \
  --sections functions,duplication,documentation,packages,patterns \
  | jq '{
      overview: {
        cc_violations: ([.functions[] | select(.complexity.cyclomatic > 9)] | length),
        long_functions: ([.functions[] | select(.lines.total > 50)] | length),
        duplication_pct: (.duplication.duplication_ratio * 100 | round / 100),
        doc_coverage: .documentation.coverage.overall
      }
    }'
# Expected at v1.0.0:
# { "overview": { "cc_violations": 0, "long_functions": 7,
#                 "duplication_pct": 0.61, "doc_coverage": 90.79 } }
```
