# Implementation Plan: Wain v1.1 — Functional Completeness

**Generated:** 2026-03-14  
**Tool:** go-stats-generator v1.0.0 + repository analysis  
**Repository:** github.com/opd-ai/wain

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit for Linux that renders via a Rust GPU backend with automatic software fallback, producing fully static zero-dependency binaries via Wayland and X11.
- **Current goal**: Close the gap between the API surface that v1.0.0 *exposes* and the subset of it that *actually works* — eliminating silent no-ops and the broken CI lint gate before adding new features.
- **Estimated Scope**: Medium (7 independent steps; mix of bug fixes, test additions, and infrastructure repair)

---

## Goal-Achievement Status

| # | Stated Goal | Current Status | This Plan Addresses |
|---|-------------|---------------|---------------------|
| 1 | Single static binary (no dynamic deps) | ✅ Achieved | No |
| 2 | Wayland client (9 packages) | ✅ Achieved | No |
| 3 | X11 client (9 packages) | ✅ Achieved | No |
| 4 | Software 2D rasterizer (7 packages) | ✅ Achieved | No |
| 5 | UI widget layer | ✅ Achieved | Partial (Step 3) |
| 6 | GPU buffer infrastructure | ✅ Achieved | No |
| 7 | GPU command submission (end-to-end) | ⚠️ Partial | Yes (Step 5) |
| 8 | Shader frontend (naga) | ✅ Achieved | No |
| 9 | Intel EU backend (shader→ISA) | ⚠️ Partial | Yes (Step 5) |
| 10 | AMD RDNA backend (shader→ISA) | ⚠️ Partial | Yes (Step 5) |
| 11 | Public API (`App`, `Window`, `Widget`) | ✅ Achieved | Yes (Steps 1–4) |
| 12 | Display server auto-detection | ✅ Achieved | No |
| 13 | Renderer auto-detection | ✅ Achieved | No |
| 14 | AT-SPI2 accessibility | ✅ Achieved | Yes (Step 6) |
| 15 | 60 FPS software rendering (CI-enforced) | ✅ Achieved | No |
| 16 | DMA-BUF buffer sharing | ✅ Achieved | No |
| 17 | Clipboard support | ✅ Achieved | No |
| 18 | Client-side window decorations | ✅ Achieved | No |
| 19 | HiDPI / DPI-aware scaling | ✅ Achieved | No |
| 20 | Keyboard accessibility (Tab focus) | ✅ Achieved | No |
| 21 | `TECHNICAL_DEBT.md` (referenced by CONTRIBUTING.md) | ❌ Missing | Yes (Step 7) |

---

## Metrics Summary (go-stats-generator v1.0.0, 2026-03-14)

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Go LOC | 14,665 | Moderate |
| Total packages (non-test) | 40 | Well-modularized |
| Functions > 50 lines | 7 (all in `cmd/`/demo code) | ✅ Excellent |
| Functions with cyclomatic complexity > 9 | 0 | ✅ Excellent |
| Max cyclomatic complexity | 7 (9.6 overall score) | ✅ Below threshold |
| Circular dependencies | 0 | ✅ None |
| Duplication ratio | **0.64%** (11 clone pairs, 218 lines) | ✅ Excellent |
| Doc coverage — packages | 100% | ✅ |
| Doc coverage — functions | 98.3% | ✅ |
| Doc coverage — methods | 89.0% | ⚠️ Minor gap |
| Doc coverage — overall | 91.0% | ✅ |
| Naming violations | 0 | ✅ |
| Anti-patterns detected | 5 types (`god_objects`, `long_methods`, `deep_nesting`, `magic_numbers`, `performance_antipatterns`) | ⚠️ Localized to `wain` root + `cmd/` |

### Coupling Hotspots

| Package | Coupling Score | Note |
|---------|---------------|------|
| `wain` (root) | 10 | Expected — public API orchestrator |
| `cmd/` binaries | 10 | Expected — demo entry points |
| `internal/render/display` | 5 | Bridges GPU + Wayland + X11 |
| `internal/render/backend` | 2.5 | Acceptable multi-layer adapter |

No unexpected coupling. The high scores in `wain` and `cmd/` are architectural necessities, not design debt.

### Key Finding: Code Quality Is Excellent; Functional Gaps Are the Priority

The metrics confirm there are **zero complexity, duplication, or circular-dependency problems** to fix. The codebase is clean. The entire plan therefore targets **four functional gaps** identified in `GAPS.md` and one **infrastructure break** (CI linting), ordered by impact on user-facing correctness.

---

## Implementation Steps

### Step 1: Fix DragDrop Data Delivery (Bug — `app.go`)

**Severity:** High — silent data loss in a stability-pinned API  
**Gap source:** `GAPS.md` §Gap 1

- **Deliverable:**
  - `app.go`: populate `DragEvent.MimeType` and `DragEvent.Data` fields from the Wayland `wl_data_offer` read-completion callback and the X11 `XdndDrop` + ICCCM selection-transfer handler.
  - `app.go:2152`: replace `w.dropHandler("", nil)` with `w.dropHandler(evt.MimeType, evt.Data)`.
  - `event.go`: verify `DragEvent` has exported `MimeType string` and `Data []byte` fields (add them if absent).
  - New test in `integration_test.go` or a new `drag_test.go`: simulate a drop event with a known MIME type and payload; assert the registered handler receives them.

- **Dependencies:** None. Isolated to `app.go` and `event.go`.

- **Goal Impact:** Closes Goal #11 (public API correctness); removes a user-visible silent failure that undermines the v1.0.0 stability guarantee in `STABILITY.md`.

- **Acceptance Criterion:** `go test -run TestDragDropHandlerReceivesData ./...` passes. The handler receives a non-empty `mimeType` and `data` slice for any simulated drop event.

- **Validation:**
  ```bash
  go test -v -run TestDragDrop ./...
  ```

---

### Step 2: Fix CI Linting Gate (Infrastructure — `.golangci.yml` + CI)

**Severity:** High — CI quality gate is silently broken  
**Gap source:** `GAPS.md` §Gap 4

- **Deliverable:**
  - `.golangci.yml`: remove `structcheck`, `varcheck`, and `deadcode` from the `linters.enable` list (all three were removed from golangci-lint at v1.49.0; `unused` already enabled covers their functionality).
  - `.github/workflows/ci.yml`: verify the `golangci/golangci-lint-action@v6` step pins a version compatible with Go 1.24. Add `version: latest` or a specific `v1.57+` pin to the action `with:` block.
  - Confirm `golangci-lint run ./...` exits 0 locally after the fix.

- **Dependencies:** None. Isolated to config files.

- **Goal Impact:** Restores the CI quality gate. `staticcheck`, `errcheck`, and `unused` will now run on every PR, catching regressions the project currently misses.

- **Acceptance Criterion:** `golangci-lint run ./...` exits 0 with no "unknown linter" warnings. The CI `build-and-test` job's lint step shows green.

- **Validation:**
  ```bash
  golangci-lint run ./... 2>&1 | grep -c "unknown linter"
  # expected: 0
  ```

---

### Step 3: Implement `bufferCanvas` Stubs — Image, Gradient, Shadow (Feature Completion — `concretewidgets.go`)

**Severity:** High — three stability-pinned `Canvas` methods are silent no-ops  
**Gap source:** `GAPS.md` §Gap 2, §Gap 3

- **Deliverable:**
  - `concretewidgets.go`: Add imports for `internal/raster/composite`, `internal/raster/effects`.
  - `bufferCanvas.DrawImage`: decode the `*Image` pixel buffer into a `*primitives.Buffer`; call `composite.BlitScaled` (already implemented and tested in `internal/raster/composite`) to alpha-composite the image into `c.buf` at the requested bounds.
  - `bufferCanvas.LinearGradient(x, y, w, h int, start, end Color, angle float64)`: compute `startX/Y`, `endX/Y` from `angle`; call `effects.LinearGradient(c.buf, x+c.xOff, y+c.yOff, w, h, startX, startY, start.toInternal(), endX, endY, end.toInternal())`.
  - `bufferCanvas.RadialGradient(x, y, w, h int, center, edge Color)`: call `effects.RadialGradient(c.buf, x+c.xOff, y+c.yOff, w, h, x+c.xOff+w/2, y+c.yOff+h/2, min(w,h)/2, center.toInternal(), edge.toInternal())`.
  - `bufferCanvas.BoxShadow(x, y, w, h, offX, offY, blur int, color Color)`: call `effects.BoxShadow(c.buf, x+c.xOff+offX, y+c.yOff+offY, w, h, blur, color.toInternal())`.
  - Wire `internal/raster/consumer/software.go`: implement the `CmdDrawImage` case in `SoftwareConsumer.renderCommand` using `composite.BlitScaled`.
  - Unit tests in `concretewidgets_test.go`: for each method, create a canvas backed by a non-nil buffer, call the method, assert that at least one pixel in the affected region is non-zero.

- **Dependencies:** Step 1 (none in code; ordering by importance only). Raster packages (`composite`, `effects`, `primitives`) are already fully implemented and tested — this step is pure wiring.

- **Goal Impact:** Closes Goal #5 (full UI widget layer). Any third-party widget author using `Canvas.LinearGradient`, `Canvas.RadialGradient`, `Canvas.BoxShadow`, or `Canvas.DrawImage` will now see correct output instead of blank pixels.

- **Acceptance Criterion:** `go test -run TestBufferCanvas ./...` passes; each method produces ≥1 non-transparent pixel in the target region.

- **Validation:**
  ```bash
  go test -v -run TestBufferCanvas ./...
  go-stats-generator analyze . --skip-tests --format json --sections documentation \
    2>/dev/null | jq '.documentation.coverage.methods'
  # target: ≥ 90.0
  ```

---

### Step 4: Increase Public API Test Coverage (`wain` root package)

**Severity:** Medium — stability-pinned API has only 24.4% test coverage  
**Gap source:** `ROADMAP.md` §Priority 2

- **Deliverable:**
  - `app_test.go` (new or extend existing): add `TestAppRunWithoutDisplay` (graceful headless fallback via `AppConfig.ForceSoftware`), `TestAppQuitWhileRunning` (clean shutdown), `TestAppNewWindowConfig` (permutations of `WindowConfig`).
  - Extend `window_test.go`: add `TestWindowSetLayout` (widget tree attachment and bounds computation), `TestWindowRenderFrameSoftware` (call `RenderFrame()` on a headless software window; assert no error and non-nil pixel buffer).
  - Extend `dispatcher_test.go` or add `event_test.go`: `TestFocusTraversal` (Tab/Shift-Tab moves focus through the widget tree), `TestEventBubbling` (pointer event propagates through container→child).

- **Dependencies:** Step 3 (canvas stubs must work for `TestWindowRenderFrameSoftware` to be meaningful).

- **Goal Impact:** Closes Goal #11 (public API stability confidence). Raises root package coverage from ~24% toward ≥60%, backing the v1.0.0 stability commitment in `STABILITY.md`.

- **Acceptance Criterion:** `go test -cover . | grep "^ok"` shows coverage ≥ 60% for the `wain` root package.

- **Validation:**
  ```bash
  go test -cover . 2>/dev/null | grep coverage
  # target: coverage: 60.0% of statements or higher
  ```

---

### Step 5: GPU Shader-to-ISA Compilation CI Gate (Infrastructure — `render-sys/` + CI)

**Severity:** Medium — three "Partially Achieved" GPU goals have no automated regression protection  
**Gap source:** `GAPS.md` §Gap 6; `ROADMAP.md` §Priority 1

- **Deliverable:**
  - Add a Rust test in `render-sys/src/` (e.g., `tests/shader_compile.rs` or inline `#[cfg(test)]` in `shader.rs`) that: loads each of the 7 WGSL shaders from `render-sys/shaders/`, parses them through naga, lowers to Intel EU IR and AMD RDNA IR, and asserts the emitted binary blob is non-empty and begins with the expected ISA magic header or instruction count > 0. **No GPU hardware is needed** — this tests the compilation pipeline only.
  - `.github/workflows/ci.yml`: add a step after `Run Rust tests` that runs `cargo test --test shader_compile` with the musl target, ensuring the compilation gate runs on every push.
  - Update `HARDWARE.md` to distinguish "code-verified (shader compiles to ISA bytes)" from "hardware-validated (frame submitted and displayed on physical GPU)".

- **Dependencies:** None for the Rust test. The CI step depends on Step 2 (linting fixed) only for overall CI health.

- **Goal Impact:** Provides regression protection for Goals #7 (GPU command submission), #9 (Intel EU backend), #10 (AMD RDNA backend). Shader→ISA compilation regressions are caught in standard CI without requiring a GPU runner.

- **Acceptance Criterion:** `cargo test --manifest-path render-sys/Cargo.toml --target x86_64-unknown-linux-musl --test shader_compile` passes; all 7 WGSL shaders compile to non-empty EU and RDNA byte sequences.

- **Validation:**
  ```bash
  cargo test --manifest-path render-sys/Cargo.toml \
    --target x86_64-unknown-linux-musl \
    --test shader_compile -- --nocapture 2>&1 | grep -E "ok|FAILED"
  ```

---

### Step 6: Add `internal/a11y` Tests

**Severity:** Medium — AT-SPI2 implementation has 10 source files and 75 functions but zero test files  
**Gap source:** `ROADMAP.md` §Priority 3

- **Deliverable:**
  - `internal/a11y/manager_test.go` (new): `TestManagerRegistration` — instantiate the `Manager` with a stub `dbus.Conn`, register a panel + button + text `AccessibleObject`, assert the objects are exported at the expected D-Bus paths. `TestFocusEvent` — call the focus-change callback; assert the `org.a11y.atspi.Event.Focus` signal data is correct. `TestActionInterface` — invoke a button action via the `Action` interface; assert the registered Go callback fires.
  - Use `github.com/godbus/dbus/v5`'s test infrastructure (already a direct dependency) or a simple mock `dbus.Conn` to keep tests free of a live D-Bus session. The build tag `atspi` gates the real D-Bus export; tests should compile under the stub path without a running bus.
  - Target: ≥ 70% statement coverage for `internal/a11y`.

- **Dependencies:** Step 2 (linting must pass; `unused` linter may flag untested helper methods in `a11y` once it runs).

- **Goal Impact:** Validates Goal #14 (AT-SPI2 accessibility). Catches regressions in the screen-reader integration that are currently invisible to CI.

- **Acceptance Criterion:** `go test -tags=atspi ./internal/a11y/...` exits 0 with ≥ 70% coverage.

- **Validation:**
  ```bash
  go test -tags=atspi -cover ./internal/a11y/... 2>/dev/null | grep coverage
  # target: coverage: 70.0% of statements or higher
  ```

---

### Step 7: Create `TECHNICAL_DEBT.md`

**Severity:** Low — referenced by `CONTRIBUTING.md` and `README.md` but does not exist  
**Gap source:** `GAPS.md` §Gap 7

- **Deliverable:**
  - `TECHNICAL_DEBT.md` in the repository root, listing all known tracked items from `GAPS.md` with TD-N identifiers, file locations, priorities, and effort estimates:
    - TD-1: DragDrop data delivery (now fixed by Step 1 — mark Resolved)
    - TD-2: `bufferCanvas` image/gradient/shadow stubs (now fixed by Step 3 — mark Resolved)
    - TD-3: GPU shader-to-ISA compilation CI gate (addressed by Step 5 — mark In Progress)
    - TD-4: AT-SPI2 build tag undocumented in README/STABILITY.md (`GAPS.md` §Gap 5)
    - TD-5: `golangci-lint` version mismatch (now fixed by Step 2 — mark Resolved)
  - Any future `// TODO(TD-N):` comments in source code must reference entries in this file.

- **Dependencies:** Steps 1–6 must be complete or in progress so TD-N items reflect accurate status.

- **Goal Impact:** Satisfies the `CONTRIBUTING.md` pre-commit checklist requirement ("TODOs tracked in `TECHNICAL_DEBT.md`"). Prevents invisible accumulation of future debt.

- **Acceptance Criterion:** `ls TECHNICAL_DEBT.md` exits 0; `grep -c "^## TD-" TECHNICAL_DEBT.md` returns ≥ 5; `README.md` and `CONTRIBUTING.md` references resolve to the file.

- **Validation:**
  ```bash
  ls TECHNICAL_DEBT.md && grep -c "^## TD-" TECHNICAL_DEBT.md
  # expected: TECHNICAL_DEBT.md exists; count ≥ 5
  ```

---

## Step Dependency Graph

```
Step 1 (DragDrop bug)         ──────────────────────────────────┐
Step 2 (Fix CI linting)       ────────────────┐                 │
Step 3 (Canvas stubs)         ──────────────┐ │                 │
                                            ↓ ↓                 ↓
Step 4 (Root pkg test cov)    ← Step 3     Step 6 (a11y tests)  Step 7 (TECHNICAL_DEBT.md)
Step 5 (GPU shader CI gate)   ← Step 2 (CI health only)
```

Steps 1, 2, 3, and 5 have no inter-dependencies and can be executed in parallel.  
Step 4 depends on Step 3.  
Step 6 is aided by Step 2 (linter exposes unused symbols).  
Step 7 should come last to accurately reflect resolution status.

---

## Prioritization Rationale

| Step | Priority | Reason |
|------|----------|--------|
| **Step 1** — DragDrop bug | **P0 / Bug** | Silent data loss in a stability-pinned public API. Already at v1.0.0; any user calling `SetDropTarget` receives incorrect behavior. |
| **Step 2** — Fix CI linting | **P0 / Infra** | Quality gate is broken. Code regressions that `staticcheck`/`errcheck`/`unused` would catch are not caught. Blocks all other quality work. |
| **Step 3** — Canvas stubs | **P1 / Bug** | Three stability-pinned `Canvas` methods silently do nothing. Any widget author following the documented contract sees blank output. |
| **Step 4** — Root pkg coverage | **P2 / Test** | 24.4% coverage on the stability-committed root package undermines v1.0.0 guarantees. |
| **Step 5** — GPU shader CI | **P2 / Infra** | Three "Partial" goals gain regression protection without requiring GPU hardware. |
| **Step 6** — a11y tests | **P3 / Test** | 75 functions, 0 tests — any refactor silently breaks screen-reader support. |
| **Step 7** — TECHNICAL_DEBT.md | **P3 / Docs** | Low effort; fixes a broken CONTRIBUTING.md reference; establishes debt tracking for the future. |

---

## Scope Assessment (Calibrated to Codebase)

| Metric | Threshold | Current | Steps Affected |
|--------|-----------|---------|---------------|
| Functions > cyclomatic 9 | Small: <5 | **0** | None — no complexity work needed |
| Duplication ratio | Small: <3% | **0.64%** | None — no deduplication work needed |
| Doc coverage gap | Small: <10% | **9.0% gap** (91% coverage) | None — doc coverage is fine; method gap is 11% |
| Stability-pinned methods that are no-ops | N/A | **4** (`DrawImage`, `LinearGradient`, `RadialGradient`, `BoxShadow`) | Step 3 |
| Test files in packages with >10 functions | N/A | `internal/a11y` = 0 test files | Step 6 |

**Overall plan scope: Medium** — 7 steps, all independently actionable, estimated 3–5 developer-days total.

---

## Out of Scope (Deferred to Future Plans)

| Item | Reason for Deferral |
|------|---------------------|
| End-to-end GPU UI→display pipeline demo (`cmd/gpu-ui-demo`) | Requires GPU hardware for validation; hardware runners not in current CI |
| `App.RenderFrame` wired to `GPUBackend.Render()` for widget tree | Blocked on GPU pipeline integration test (Step 5 is a prerequisite) |
| GPU performance CI enforcement (≤2 ms/frame) | Requires GPU runner; `cmd/gpu-bench` already exists |
| AT-SPI2 build tag documented in README/STABILITY.md | Low complexity; tracked as TD-4 in `TECHNICAL_DEBT.md` (Step 7) |
| Method doc coverage from 89% to 95% target | Not a functional gap; 89% is acceptable at v1.0 |
