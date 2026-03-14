# Implementation Plan: Complete API Correctness & GPU Pipeline Validation

> **Generated:** 2026-03-14  
> **Tool:** `go-stats-generator v1.0.0` + project documentation cross-reference  
> **Analyzed files:** 199 Go source files (non-test), 40 packages  
> **Baseline run:** `go-stats-generator analyze . --skip-tests --format json --sections functions,duplication,documentation,packages,patterns`

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit for Linux that renders via a Rust GPU backend with automatic software fallback, implementing Wayland and X11 protocols directly to produce fully static, zero-dependency binaries.
- **Current goal**: Close the three remaining correctness gaps (`SetOpacity` undeclared, `LinearGradient` angle ignored, stale `consumer` doc) and finish the GPU pipeline validation milestone (`gpu_pipeline_test.go` + `cmd/gpu-ui-demo`).
- **Estimated Scope**: **Medium** — 5 independently actionable steps across 6 files; no circular-dependency risk; all changes are leaf-level.

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| Single static binary (no dynamic deps) | ✅ Achieved | No |
| Wayland client (9 packages) | ✅ Achieved | No |
| X11 client (10 packages) | ✅ Achieved | No |
| Software 2D rasterizer (7 packages) | ✅ Achieved | No |
| UI widget layer (6 packages) | ✅ Achieved | No |
| GPU buffer infrastructure | ✅ Achieved | No |
| GPU command submission | ⚠️ Partial | **Yes** — Step 4 & 5 |
| Shader frontend (naga) | ✅ Achieved | No |
| Intel EU backend | ⚠️ Partial | **Yes** — Step 4 |
| AMD RDNA backend | ⚠️ Partial | **Yes** — Step 4 |
| Public API (App, Window, Widget) | ✅ Achieved | **Yes** — Step 1 |
| Display server auto-detection | ✅ Achieved | No |
| Renderer auto-detection | ✅ Achieved | No |
| AT-SPI2 accessibility | ✅ Achieved | **Yes** — Step 6 (minor) |
| 60 FPS software rendering | ✅ Achieved | No |
| DMA-BUF / clipboard / decorations / HiDPI | ✅ Achieved | No |
| Canvas API correctness (LinearGradient angle) | ❌ Broken | **Yes** — Step 2 |
| Animate example compiles | ❌ Broken | **Yes** — Step 1 |
| `consumer` package docs accurate | ❌ Stale | **Yes** — Step 3 |

---

## Metrics Summary

| Metric | Value | Threshold | Assessment |
|---|---|---|---|
| Functions with cyclomatic complexity > 9 | **0** | 9 | ✅ Excellent |
| Functions with overall score > 9 | **25** | — | ✅ All sub-threshold |
| Functions with overall score > 13 | **1** (`packVertices`, score=13.7) | — | ⚠️ Single hotspot |
| Functions > 50 lines | **7** | — | All in `cmd/` (main funcs), acceptable |
| Duplication ratio | **0.69%** | 3% | ✅ Excellent |
| Doc coverage (functions) | **98.3%** | 90% | ✅ Excellent |
| Doc coverage (methods) | **89.3%** | 90% | ⚠️ Marginally below target |
| Undocumented exported identifiers | **0** | — | ✅ Perfect |
| Circular dependencies | **0** | — | ✅ Perfect |
| Naming violations | **0** | — | ✅ Perfect |
| Performance anti-patterns (tool-reported) | **330** | — | ⚠️ Mostly goroutines-without-context and error-without-wrapping; see Note |

> **Note on performance anti-patterns:** The 330 flagged items break down into two categories:
> (1) Goroutines launched from event-loop entry points — these are intentional and the concurrency model is documented in `app.go` and `dispatcher.go`.  
> (2) Errors returned without `fmt.Errorf("…: %w", err)` wrapping — a style gap, not a correctness problem.  
> Neither category affects the goals tracked in this plan. They are candidates for a separate clean-up pass after the correctness and feature work is complete.

### Complexity Hotspot Detail

| Function | Package | File | Lines | Cyclomatic | Overall |
|---|---|---|---|---|---|
| `packVertices` | `backend` | `internal/render/backend/vertex.go:27` | 44 | **9** | **13.7** |

`packVertices` is the sole function that exceeds overall score 9. Its cyclomatic complexity (9) is below the hard threshold (10), but its nesting depth (4) inflates the score. It is on the GPU command-submission critical path, so reducing its nesting is tracked as Step 5b (bundled with the GPU demo work to avoid churn).

---

## Implementation Steps

Steps are ordered: correctness fixes first (highest user impact), then documentation fixes, then feature completion, then quality improvement.

---

### Step 1: Implement `Widget.SetOpacity` — Fix Broken Animate Example

- **Priority:** P0 — the canonical `Animate` example in `animate.go:46` calls `widget.SetOpacity(v)`, which does not exist on any type. Any user who copies the documentation gets an immediate compile failure.
- **Deliverable:**
  1. Add `SetOpacity(alpha float64)` to the `PublicWidget` interface (`publicwidget.go`).
  2. Add an `opacity float64` field (default `1.0`) to `BasePublicWidget` (`publicwidget.go`).
  3. Implement `SetOpacity` on `BasePublicWidget`; clamp `alpha` to `[0.0, 1.0]`.
  4. Add `Opacity() float64` accessor to `BasePublicWidget` for use by renderers.
  5. Pass the opacity value into the `alpha` channel in `displayListCanvas` draw calls:  
     - In `displayListCanvas.FillRect`, `FillRoundedRect`, `DrawText`, `DrawImage`:  
       multiply the effective `Color.A` by the widget's `Opacity()` before appending to the display list.  
       The canvas already receives the widget's color; apply `alpha = uint8(float64(color.A) * widget.Opacity())`.
  6. Add `TestSetOpacity` in `concretewidgets_test.go` (or `publicwidget_test.go`):  
     - `SetOpacity(0.5)` → `Opacity() == 0.5`.  
     - `SetOpacity(-0.1)` → clamped to `0.0`.  
     - `SetOpacity(1.5)` → clamped to `1.0`.
- **Dependencies:** None — leaf change to root package and `publicwidget.go`.
- **Goal Impact:** Fixes the public API developer experience; `animate.go` example now compiles and runs correctly. Advances Goal #11 (Public API completeness).
- **Acceptance:** `go build ./...` passes; `go test -run TestSetOpacity ./...` passes; the animate example in `animate.go:46` compiles without modification.
- **Validation:**
  ```bash
  go vet ./... && go test -run TestSetOpacity ./...
  # Confirm example compiles:
  go build -o /dev/null ./example/hello
  ```

---

### Step 2: Fix `Canvas.LinearGradient` Angle Parameter — Both Canvas Implementations

- **Priority:** P1 — `Canvas.LinearGradient` is documented as supporting arbitrary angles (`0 = left-to-right, 90 = top-to-bottom`). Both implementations silently discard the angle and always render a horizontal gradient. Any widget using a non-zero angle renders incorrectly with no error.
- **Deliverable:**
  1. In `publicwidget.go`, replace the hard-coded horizontal calculation in `displayListCanvas.LinearGradient` (line 270–275) with proper angle-to-vector conversion using `math.Cos`/`math.Sin`:
     ```go
     rad := angle * math.Pi / 180
     cx, cy := float64(x+width/2), float64(y+height/2)
     hw, hh := float64(width)/2, float64(height)/2
     x0 := int(cx - hw*math.Cos(rad))
     y0 := int(cy - hh*math.Sin(rad))
     x1 := int(cx + hw*math.Cos(rad))
     y1 := int(cy + hh*math.Sin(rad))
     ```
  2. In `concretewidgets.go`, replace `_ float64` with `angle float64` in `bufferCanvas.LinearGradient` (line 142) and apply the same conversion before passing start/end points to `effects.LinearGradient`.
  3. Add `TestLinearGradientAngle` in `concretewidgets_test.go`:
     - `angle=0` → start point is on the left, end point is on the right.
     - `angle=90` → start point is on the top, end point is on the bottom (top row ≈ `startColor`, bottom row ≈ `endColor`).
     - `angle=180` → start/end points mirrored vs. `angle=0`.
- **Dependencies:** Step 1 (none — can be done in parallel, but ordering here reflects descending impact).
- **Goal Impact:** Corrects a documented API contract; custom widget authors can now use `Canvas.LinearGradient` with any angle.
- **Acceptance:** `TestLinearGradientAngle` passes; `angle=90` produces a top-to-bottom gradient verified by sampling the first and last rows of the rendered buffer.
- **Validation:**
  ```bash
  go test -run TestLinearGradientAngle ./...
  go-stats-generator analyze . --skip-tests --format json --sections documentation 2>/dev/null \
    | jq '.documentation.coverage.functions'
  # Should remain ≥ 98.3
  ```

---

### Step 3: Fix Stale Documentation in `internal/raster/consumer/doc.go`

- **Priority:** P2 — `doc.go` states "The SoftwareConsumer does not implement the `CmdDrawImage` display list command" but `software.go` contains a working `renderDrawImage` implementation. This misleads contributors into thinking `ImageWidget` is GPU-only and may cause duplicate work or incorrect architectural decisions.
- **Deliverable:**  
  Replace the "Software Rasterizer Limitations" paragraph in `internal/raster/consumer/doc.go` with an accurate description of what `renderDrawImage` does:
  > The SoftwareConsumer handles all `DisplayList` command types, including `CmdDrawImage`.
  > Image blitting uses bilinear scaling via `internal/raster/composite.BlitScaled`. If
  > `DrawImageData.Src` is nil (GPU-only path), the call is silently skipped.
- **Dependencies:** None — documentation-only change.
- **Goal Impact:** Eliminates contributor confusion about the software/GPU split; no feature work is duplicated.
- **Acceptance:** `go doc github.com/opd-ai/wain/internal/raster/consumer` no longer states CmdDrawImage is unsupported.
- **Validation:**
  ```bash
  go doc github.com/opd-ai/wain/internal/raster/consumer | grep -c "does not implement"
  # Must output: 0
  go test ./internal/raster/consumer/...
  ```

---

### Step 4: Add Hardware-Independent GPU Pipeline Integration Test

- **Priority:** P1 — The GPU rendering pipeline (widget tree → display list → GPU batch encoding) has no CI-testable integration test. GPU encoding regressions in `internal/render/backend/` go undetected between hardware-validated runs. This is the primary gap preventing goals #7, #9, #10 from being "fully achieved".
- **Deliverable:**  
  Create `internal/integration/gpu_pipeline_test.go` with build tag `//go:build integration` containing `TestGPUPipeline`:
  1. Construct a `backend.GPUBackend` using a mock `BufferAllocator` backed by a `primitives.NewBuffer(1920, 1080)` heap allocation (no DRM device needed).
  2. Build a minimal `displaylist.DisplayList` containing one `CmdFillRect` (e.g., red, 100×100 at origin).
  3. Call `GPUBackend.Render(dl)` and assert:
     - The returned batch byte slice is non-empty.
     - No error is returned.
     - The first 4 bytes match the expected Intel MI_BATCH_BUFFER_START header (little-endian `0x31000000`) or AMD PM4 IT_NOP header when running on AMD hardware. On hardware-less runners, accept any non-empty output.
  4. Wire the test into CI: add `go test -tags integration ./internal/integration/...` to the `build-and-test` job in `.github/workflows/ci.yml` (the mock allocator removes any hardware requirement).
- **Dependencies:** Step 3 should land first (it removes a confusing doc that might lead reviewers to question this test's scope).
- **Goal Impact:** Fully achieves Goals #7 (GPU command submission), #9 (Intel EU backend), #10 (AMD RDNA backend) as code-verified.
- **Acceptance:** `go test -tags integration ./internal/integration/...` passes on any Linux runner without GPU hardware. CI `build-and-test` job goes green.
- **Validation:**
  ```bash
  go test -tags integration -v -run TestGPUPipeline ./internal/integration/...
  # Must print: PASS
  go-stats-generator analyze . --skip-tests --format json --sections functions 2>/dev/null \
    | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
  # Must remain: 0 (no new complexity introduced)
  ```

---

### Step 5: Create `cmd/gpu-ui-demo` + Refactor `packVertices`

- **Priority:** P2 — Two related deliverables bundled to avoid churn in `internal/render/backend/`:
  1. `cmd/gpu-ui-demo` is the last unchecked task in ROADMAP Priority 1 and validates the "GPU-accelerated graphics" claim end-to-end.
  2. `packVertices` (overall score 13.7, cc=9, nesting depth=4) is the sole complexity hotspot identified by `go-stats-generator`; it lives in the same package and refactoring it naturally accompanies the GPU demo work.
- **Deliverable:**
  1. **`cmd/gpu-ui-demo/main.go`**: A `wain.NewAppWithConfig(wain.AppConfig{Verbose: true})` application that opens a window, sets a `Column` layout with a `Label` ("GPU Demo"), a `Button` ("Render Frame"), and a `TextInput`, then calls `win.RenderFrame()` on each button click. The binary must run on software fallback when no GPU is present (so CI passes without hardware).
  2. **Refactor `packVertices` in `internal/render/backend/vertex.go`**:
     - Extract the atlas-warning loop into a private helper `warnAtlasAbsent(batches []Batch)`.
     - Extract the vertex-estimation + allocation block into `allocVertexBuffer(batches []Batch) []byte`.
     - The remaining `packVertices` body becomes a 15-line orchestrator: warn, allocate, iterate, return.
     - Target: cyclomatic ≤ 5, overall score < 9.
  3. Add `make gpu-ui-demo` target to `Makefile` (mirroring `widget-demo`).
- **Dependencies:** Step 4 (GPU pipeline test should pass before adding the demo that exercises the same path).
- **Goal Impact:** Fully achieves Goal #7 (GPU command submission) as user-demonstrable; reduces the one remaining complexity hotspot.
- **Acceptance:**
  - `make gpu-ui-demo` builds without error.
  - Binary exits 0 with `WAYLAND_DISPLAY=` `DISPLAY=` (headless, no display server).
  - `packVertices` overall score drops below 9.
- **Validation:**
  ```bash
  make gpu-ui-demo
  go-stats-generator analyze . --skip-tests --format json --sections functions 2>/dev/null \
    | jq '[.functions[] | select(.name == "packVertices")] | .[0].complexity'
  # overall should be < 9.0
  ```

---

### Step 6: Add `TestAccessibilityIntegration` to `integration_test.go`

- **Priority:** P3 — The last unchecked task from ROADMAP Priority 3. `internal/a11y/` has 74% test coverage (`manager_test.go` + `a11y_test.go`), but there is no cross-layer integration test verifying that `wain.EnableAccessibility` wires up correctly with the headless app lifecycle.
- **Deliverable:**  
  Add `TestAccessibilityIntegration` to `integration_test.go` (root package, `package wain_test`):
  1. Create a headless `wain.NewApp()`.
  2. Call `wain.EnableAccessibility("TestApp")` and assert the returned `*AccessibilityManager` is non-nil.
  3. Register a `Label` widget via the manager; assert `manager.Lookup(id)` returns the correct name.
  4. Send a synthetic Tab key event via `app.Dispatcher()` and assert focus changes to the next widget.
  5. Test must run without D-Bus (`-tags=atspi` not required; stub path asserted to return `nil` gracefully).
- **Dependencies:** Step 1 (SetOpacity, because it affects the Widget interface which the test constructs).
- **Goal Impact:** Closes the last remaining item in ROADMAP Priority 3; AT-SPI2 claim fully validated at integration level.
- **Acceptance:** `go test -run TestAccessibilityIntegration ./...` passes without `-tags=atspi`.
- **Validation:**
  ```bash
  go test -v -run TestAccessibilityIntegration ./...
  go test -v -tags=atspi -run TestAccessibilityIntegration ./...
  # Both must print: PASS
  ```

---

## Dependency Graph

```
Step 3 (doc fix)
    └── Step 4 (GPU integration test)
            └── Step 5 (gpu-ui-demo + packVertices refactor)

Step 1 (SetOpacity)
    └── Step 6 (accessibility integration test)

Step 2 (LinearGradient angle)   [independent]
```

Steps 1, 2, and 3 are fully independent and can be worked in parallel. Step 4 depends on Step 3 landing first (to avoid confusion during review). Step 5 depends on Step 4. Step 6 depends on Step 1.

---

## Scope Calibration

| Metric | Baseline | After This Plan | Change |
|---|---|---|---|
| Cyclomatic complexity > 9 | 0 | 0 | — |
| Overall score > 13 | 1 (`packVertices`) | 0 | **−1** |
| Duplication ratio | 0.69% | ≤ 0.69% | No regression |
| Doc coverage (functions) | 98.3% | ≥ 98.3% | No regression |
| Undocumented exported identifiers | 0 | 0 | — |
| Goals fully achieved | 17/20 | **20/20** | **+3** |
| Broken API examples | 1 (`SetOpacity`) | 0 | **−1** |
| Silent runtime API bugs | 1 (`LinearGradient angle`) | 0 | **−1** |
| Stale package docs | 1 (`consumer/doc.go`) | 0 | **−1** |

---

## Tiebreaker Note

All stated goals will be fully achieved after this plan. The project's own ROADMAP and GAPS documents have no remaining P0/P1 items after these six steps. The natural next milestone — beyond this plan — is the **error-wrapping cleanup pass** (330 `fmt.Errorf("…: %w", err)` improvements across the codebase), followed by the GPU benchmark commit-over-commit tracking task identified in ROADMAP Priority 5 (last unchecked item). These are P4 tasks and do not block any stated goal.
