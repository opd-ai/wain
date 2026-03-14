# Implementation Plan: Functional Public API and Software Rendering Path

> Generated: 2026-03-14  
> Metrics baseline: `go-stats-generator analyze . --skip-tests --format json`  
> Scope assessment: **Large** (7 gaps, cross-layer changes in 6+ packages)

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit that links a Rust rendering library (via CGO/musl) for GPU-accelerated graphics on Linux, implementing Wayland and X11 display protocols from scratch and producing a single fully-static binary with zero runtime dependencies.
- **Current goal**: Make the public API produce a visible window — `App.Run()` currently renders into heap memory and never presents pixels to any display server.
- **Estimated Scope**: Large (changes touch `app.go`, `internal/render/backend/`, `internal/render/display/`, `internal/render/present/`, and related Rust FFI bindings)

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|-------------|---------------|---------------------|
| Public API window renders visibly | ❌ Blank window; `Present()` never called | **Yes** — Steps 1–3 |
| Software backend can present frames | ❌ `SoftwareBackend.Present()` returns `ErrSoftwareNoDmabuf` | **Yes** — Step 2 |
| `cmd/widget-demo` shows a window | ❌ Both `runWayland()` and `runX11()` are stubs | **Yes** — Step 4 |
| Performance targets verified in CI | ❌ No benchmarks run in CI; SIMD not implemented | **Yes** — Step 5 |
| GPU UI workloads validated end-to-end | ⚠️ Triangle demo only; no UI display-list GPU path | **Yes** — Step 6 |
| AT-SPI2 screen reader support | ❌ `internal/a11y/atspi` does not exist | **Yes** — Step 7 |
| API stability (v1.0.0) | ⚠️ No tag; STABILITY.md written but not enforced | **Yes** — Step 8 |
| Go–Rust static linking | ✅ Achieved | No |
| Wayland client (9 packages) | ✅ Achieved | No |
| X11 client (9 packages) | ✅ Achieved | No |
| Software 2D rasterizer | ✅ Achieved | No |
| UI widget layer with flexbox layout | ✅ Achieved | No |
| GPU buffer infrastructure | ✅ Achieved | No |
| Shader frontend (WGSL/naga) | ✅ Achieved | No |
| Intel EU backend (Gen9+) | ✅ Achieved | No |
| AMD RDNA backend | ✅ Achieved | No |
| Zero runtime dependencies | ✅ Achieved | No |
| Keyboard accessibility (focus traversal) | ✅ Achieved | No |
| Clipboard support (Wayland/X11) | ✅ Achieved | No |
| HiDPI/DPI-aware scaling | ✅ Achieved | No |

---

## Metrics Summary (2026-03-14 Baseline)

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Total lines of code (Go) | 14,160 | — | — |
| Total packages | 38 | — | — |
| Functions above CC 9 | **1** (`runWayland` in `cmd/widget-demo/main.go:335`, CC=10) | <5 | ✅ Small |
| Functions > 50 lines | **8** (all in `cmd/` demos, none in library) | — | ✅ |
| Duplication ratio | **0.83%** (15 clone pairs, 273 lines) | <3% | ✅ Small |
| Doc coverage overall | **90.85%** (packages 100%, functions 98.7%, methods 88.7%) | ≥80% | ✅ |
| Naming violations | **0** | 0 | ✅ |
| Bare error returns (library) | **96** across `internal/` | — | ⚠️ Medium |
| Append in loop without pre-alloc (library) | **17** in hot paths | — | ⚠️ Medium |
| Resources without defer close (library) | **15** | — | ⚠️ Medium |
| Goroutine without context (`clipboard.go:80`) | **1** | 0 | ⚠️ |
| `log.Fatal` in non-main package | **2** (`internal/demo/bootstrap.go:19`, `internal/demo/config.go:38`) | 0 | ⚠️ |
| Unused method receivers (stub impls) | **86** (mostly `internal/a11y/manager_stub.go`) | — | ℹ️ Expected for stubs |

### Complexity Hotspots on Goal-Critical Paths

| Function | File | CC | Relevance |
|----------|------|----|-----------|
| `runWayland` | `cmd/widget-demo/main.go:335` | **10** | Gap 6 (widget-demo stub is entire function) |
| `Window.initWaylandWindow` | `app.go` | ~6 | Gap 1 (presentation wiring here) |
| `Window.RenderFrame` | `app.go` | ~5 | Gap 1 (must call Present) |
| `SoftwareBackend.Present` | `internal/render/backend/` | ~2 | Gap 2 (returns error unconditionally) |

### Notable Package Coupling

| Package | Efferent deps | Issue |
|---------|---------------|-------|
| `main` (cmd bins) | 23 | Expected; demo binaries |
| `wain` (root public API) | 20 | Acceptable for orchestration layer |
| `demo` | 12 | `internal/demo` is a shared helper; high coupling expected |
| `display` | 10 | Core presentation layer; dependencies are structural |

---

## Implementation Steps

Steps are ordered by dependency chain: software presentation path (Steps 1–2) must
precede public API wiring (Step 3), which must precede demo completion (Step 4).
Performance (Step 5) and GPU integration (Step 6) require Steps 1–4. Accessibility
(Step 7) and API stabilization (Step 8) are independent but follow the functional work.

---

### Step 1: Add `Pixels() []byte` to `SoftwareBackend` and implement SHM presenters ✅ DONE

**Why first**: Every other step depends on software rendering working. CI has no GPU;
the software path is the only one exercisable in automated tests and by most users.

**Deliverable**:
- `internal/render/backend/software.go`: Add `Pixels() []byte` returning the raw
  ARGB8888 framebuffer held in `primitives.Buffer`. The buffer is already allocated
  per-frame; this is a zero-copy accessor.
- `internal/render/display/software_wayland.go`: New file. `SoftwareWaylandPresenter`
  struct implementing `Presenter` interface (`Present(context.Context) error` +
  `Close() error`). Uses `internal/wayland/shm` to create/reuse a `wl_shm_pool`,
  writes `SoftwareBackend.Pixels()` into the shared memory region, attaches the
  `wl_buffer`, and calls `wl_surface.commit`. Handles buffer release events via the
  `wl_buffer.release` callback to implement double-buffering.
- `internal/render/display/software_x11.go`: New file. `SoftwareX11Presenter` struct
  implementing `Presenter`. Uses `internal/x11/shm` to create an MIT-SHM segment,
  copies pixels via `XShmPutImage` equivalent (the existing `shm.PutImage` wrapper
  in `internal/x11/shm`), and calls `internal/x11/gc.CopyArea` to blit to the window.

**Dependencies**: None. `internal/wayland/shm` and `internal/x11/shm` are already
fully implemented.

**Goal Impact**: Closes Gap 2. Unblocks Step 3 software path.

**Acceptance**: `go test ./internal/render/display/...` passes including a new
`TestSoftwareWaylandPresenterPixels` test that creates a mock `wl_shm` connection,
calls `SoftwareWaylandPresenter.Present()`, and asserts pixels are written without
error. The test can use a pipe-based mock `wl_socket` (same pattern as
`internal/wayland/wire/wire_test.go`).

**Validation**:
```bash
go test ./internal/render/display/... -run TestSoftware -v
go-stats-generator analyze ./internal/render/display/ --skip-tests --format json \
  --sections functions | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Expected: 0
```

---

### Step 2: Wire presentation pipeline into `Window.RenderFrame()` (public API) ✅ DONE

**Why second**: Depends on Step 1 (software presenters). This is the core functional
gap: Gap 1. Without this step, every application using the public API shows a blank
window.

**Deliverable**:
- `app.go`: Modify `Window` struct to add a `presenter Presenter` field (the
  `internal/render/present` `Presenter` interface).
- `app.go` `initWaylandWindow()`: After creating the `wl_surface`, instantiate either
  a `display.WaylandPipeline` (when `GPUBackend` is selected) or a
  `SoftwareWaylandPresenter` (when `SoftwareBackend` is selected). Store in
  `win.presenter`.
- `app.go` `initX11Window()`: Same pattern — `display.X11Pipeline` or
  `SoftwareX11Presenter`. Store in `win.presenter`.
- `app.go` `RenderFrame()`: Replace the current `renderBridge.Render(rootWidget)`
  call with a two-phase sequence:
  1. `renderBridge.Render(rootWidget)` — builds the display list.
  2. `win.presenter.Present(ctx)` — uploads pixels to the compositor.
- `app.go` `Close()`: Call `win.presenter.Close()` to release SHM segments or
  DMA-BUF fds.

**Dependencies**: Step 1 (SoftwareWaylandPresenter, SoftwareX11Presenter).

**Goal Impact**: Closes Gap 1. Makes `App.Run()` produce visible windows. Enables
`example/hello/hello` and `cmd/wain-demo` to function.

**Acceptance**: `example/hello/hello` opens a visible window with "Hello, wain!"
rendered. In headless CI (no display server): `go test ./... -tags=noheadless`
continues to pass; the headless build gate in `compat_test.go` must still compile.

**Validation**:
```bash
go test ./... -count=1
# On a system with Wayland or X11:
make wain-demo && ./bin/wain-demo
# Must show a window with rendered content, not a blank frame.
go-stats-generator analyze . --skip-tests --format json --sections functions \
  | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Expected: ≤1 (only the existing widget-demo stub)
```

---

### Step 3: Implement `cmd/widget-demo` interactive display (replace stubs) ✅ DONE

**Why third**: Depends on Step 2 (working `App.Run()` + presentation pipeline).
`cmd/widget-demo` is the most prominent demo in the README; its stub state damages
developer trust.

**Deliverable**:
- `cmd/widget-demo/main.go` `runWayland()`: Replace the stub (current: prints
  warning, returns) with a real implementation using the now-working `App` type:
  create `App`, create `Window`, build the widget tree (Button, TextInput,
  ScrollView), call `app.Run()`. Remove the warning print.
  This eliminates the CC=10 function (the stub's complex switch).
- `cmd/widget-demo/main.go` `runX11()`: Same replacement pattern.
- Remove the stub warning messages from both functions.

**Dependencies**: Step 2 (working App/Window presentation).

**Goal Impact**: Closes Gap 6. Demonstrates the public API to new developers.

**Acceptance**: `make widget-demo && ./bin/widget-demo` opens a window with a visible
Button, TextInput, and ScrollView on both Wayland and X11.

**Validation**:
```bash
make widget-demo
go-stats-generator analyze cmd/widget-demo/ --skip-tests --format json \
  --sections functions | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Expected: 0 (stub removed, CC=10 function gone)
```

---

### Step 4: Add software-path CI benchmark and raster SIMD pre-allocation fixes ✅ DONE

**Why fourth**: Performance targets are stated product guarantees (HARDWARE.md:
`≤16ms` software frame at 1080p). This step creates the measurement infrastructure
and fixes the 17 `append`-in-loop anti-patterns in hot rendering paths.

**Deliverable**:
- **4a — Append pre-allocation** (medium effort, high frequency impact):
  Fix the 17 `append`-in-loop patterns flagged in library hot paths:
  - `internal/render/backend/vertex.go:54,209` — pre-allocate vertex slices
  - `internal/render/backend/batch.go:51,58` — pre-allocate batch entries
  - `internal/raster/displaylist/damage.go:88,240` — pre-allocate damage rects
  - `internal/render/atlas/texture.go:432,473` — pre-allocate glyph entries
  - `internal/ui/layout/flex.go:344` — pre-allocate child size entries
  - `internal/wayland/xdg/toplevel.go:252` — pre-allocate state change list
  - `internal/wayland/socket/connection.go:166` — pre-allocate message buffer
  - `internal/wayland/wire/protocol.go:380` — pre-allocate protocol args

- **4b — CI benchmark job** (`.github/workflows/ci.yml`):
  Add a new job `benchmark-software` that runs:
  ```bash
  go test -bench=BenchmarkFillRect -benchtime=3s -benchmem \
    ./internal/raster/primitives/ | tee /tmp/bench.txt
  ```
  Emit the result as a workflow step summary. The job is non-blocking (warning only)
  until SIMD is implemented (Step 5b).

- **4c — `SoftwareBackend` bench helper** (`internal/render/backend/software_bench_test.go`):
  `BenchmarkSoftwareRender1080p` — render a standard scene (500 filled rects, 50
  text glyphs, 10 box shadows) through the full software path and record ns/frame.
  This creates the regression-detectable baseline for Gap 5.

**Dependencies**: Step 1 (`SoftwareBackend.Pixels()` needed for bench).

**Goal Impact**: Partially closes Gap 5 (CI benchmark established; SIMD deferred to
Step 5).

**Acceptance**: `go test -bench=. ./internal/raster/primitives/` runs in CI without
error. `go-stats-generator` reports 0 `append`-in-loop patterns in the files listed
above.

**Validation**:
```bash
go test -bench=BenchmarkFillRect -benchtime=1s ./internal/raster/primitives/
go-stats-generator analyze ./internal/render/backend/ --skip-tests --format json \
  --sections patterns \
  | jq '[.patterns.anti_patterns.performance_antipatterns[]
         | select(.description | contains("append"))] | length'
# Expected: 0 in backend/
```

---

### Step 5: GPU UI rendering — wire display list to GPU batch submission ⚠️ BLOCKED (requires Intel Gen9+ hardware and Rust shader implementation)

**Why fifth**: Depends on Steps 1–2 (pipeline plumbing in place). The GPU
infrastructure exists (batch buffers, shaders, pipeline state) but no UI-level
workload (rectangles, text, shadows from a display list) is ever submitted to the GPU.
The Rust TODOs TD-7 through TD-13 in `render-sys/src/shader.rs` capture the shader
compilation stubs.

**Deliverable**:
- **5a — Rust shader dispatch** (`render-sys/src/shader.rs`):
  Implement the body of the functions annotated with TODO TD-7 through TD-13:
  `render_solid_fill`, `render_gradient`, `render_textured_quad`, `render_sdf_text`,
  `render_rounded_rect`, `render_radial_gradient`, `render_blur`.
  Each: compile the associated WGSL shader (already in `render-sys/shaders/`),
  set vertex attribute layout matching shader inputs, populate a `BatchBuffer` with
  the primitive's vertex data, and submit via `submit_batch`.

- **5b — Go GPU backend dispatch** (`internal/render/backend/gpu.go`):
  Implement `renderSolidRect()`, `renderRoundedRect()`, `renderText()` in
  `GPUBackend.Render()` to call the new Rust FFI functions from 5a instead of
  falling back to the software path.

- **5c — Integration test** (`internal/integration/`):
  Add `TestGPURenderUIScene` (build tag: `integration`): renders a display list with
  200 solid rects + 20 text runs through `GPUBackend`, calls `Present()`, reads back
  pixels via `SoftwareBackend.Pixels()` reference render, and asserts PSNR ≥ 40dB.
  Gate in CI on `gpu-check` step (existing `/dev/dri/renderD128` detection).

**Dependencies**: Steps 1–2. Rust: TD-6 (swizzle bits in `eu/lower.rs`) should be
addressed before or alongside TD-7.

**Goal Impact**: Closes Gap 4. Verifies GPU path against real UI workloads. Closes
Gap 5 for GPU path.

**Acceptance**: `cmd/gpu-display-demo` renders a complete widget hierarchy via GPU,
not software fallback. `TestGPURenderUIScene` passes on hardware with Intel Gen9+.

**Validation**:
```bash
cargo test --manifest-path render-sys/Cargo.toml \
  --target x86_64-unknown-linux-musl -- shader
go test -tags=integration -run TestGPURenderUIScene ./internal/integration/... -v
```

---

### Step 6: AT-SPI2 screen reader integration (`internal/a11y/atspi`) ✅ DONE

**Why sixth**: The only gap not on the critical rendering path. Depends on a working
`App.Run()` (Step 2) since AT-SPI2 needs to register with the running application.
Independent of GPU work (Steps 5–6 can be parallelized after Step 2).

**Deliverable**:
- `go.mod`: Promote `github.com/godbus/dbus/v5` from `indirect` to direct dependency.
- `internal/a11y/atspi/` (new package):
  - `accessible.go`: `AccessibleObject` exporting `org.a11y.atspi.Accessible` D-Bus
    interface. Fields: `Name`, `Role`, `Parent`, `Children`.
  - `component.go`: `org.a11y.atspi.Component` interface — `GetExtents`, `Contains`,
    `GetPosition`.
  - `action.go`: `org.a11y.atspi.Action` interface — `GetNActions`, `DoAction`,
    `GetDescription`.
  - `registry.go`: `Register()` connects to `org.a11y.atspi.Registry2` on the
    session D-Bus, registers the application, and exports the root accessible object.
  - `bridge.go`: `FocusBridge` — subscribes to `FocusManager` focus-change events
    and emits `object:state-changed:focused` and `focus:` event signals to the AT-SPI2
    event bus.
  - `widget_adapters.go`: AT-SPI2 adapters for `Button`, `TextInput`, `Label`,
    `Panel` — wraps each widget in an `AccessibleObject` with correct role
    (`ROLE_PUSH_BUTTON`, `ROLE_TEXT`, `ROLE_LABEL`, `ROLE_PANEL`).

- `app.go` `App.Run()`: When `ACCESSIBILITY_BUS_ADDRESS` env var is set or
  `WAIN_ATSPI=1`, call `atspi.Register(app)` before entering the event loop.
  Guard behind build tag `atspi` (consistent with the `atspi` tag documented in
  `STABILITY.md`).

**Dependencies**: Step 2 (working `App.Run()`). `github.com/godbus/dbus/v5` already
in `go.sum`.

**Goal Impact**: Closes Gap 3. Enables Orca/Accerciser to announce widget labels.
Required for government/enterprise accessibility compliance.

**Acceptance**: `go test ./internal/a11y/atspi/...` passes. In a session with Orca
running and `WAIN_ATSPI=1`, `cmd/widget-demo` announces "Button: Submit" when the
button receives focus.

**Validation**:
```bash
go build -tags atspi ./...
go test -tags atspi ./internal/a11y/atspi/... -v
go-stats-generator analyze ./internal/a11y/atspi/ --skip-tests --format json \
  --sections documentation | jq '.documentation.coverage.overall'
# Expected: ≥90
```

---

### Step 7: Error context wrapping and resource leak remediation ✅ DONE

**Why seventh**: 96 bare error returns in library code make debugging production
failures difficult; 15 resource-acquisition sites lack `defer close()`. These are
correctness and operational quality issues, not feature work. Address after the
functional gaps are closed.

**Deliverable**:
- **7a — Error wrapping** (96 sites in `internal/`):
  Wrap bare `return err` with `fmt.Errorf("context: %w", err)` in library code.
  Priority order (by caller visibility):
  1. `internal/render/display/wayland.go`, `x11.go` — frame-submission path
  2. `internal/buffer/ring.go:203,239` — buffer lifecycle
  3. `internal/render/present/interface.go:66` — presenter interface
  4. Remaining `internal/wayland/`, `internal/x11/`, `internal/raster/` sites

  Do **not** wrap errors in `internal/demo/` — demo code is not library-facing.

- **7b — Deferred close** (15 sites in `internal/`):
  Add `defer` to resource acquisitions that lack it:
  - `internal/render/display/wayland.go:100,216` — DMA-BUF fd cleanup
  - `internal/render/present/interface.go:66` — presenter resource cleanup
  - `internal/demo/display.go:20,62` — (demo; lower priority)
  - `internal/demo/wayland.go:29` — (demo)

- **7c — `log.Fatal` in non-main** (`internal/demo/bootstrap.go:19`,
  `internal/demo/config.go:38`):
  Replace with `return error` propagation. `log.Fatal` in a library package prevents
  callers from handling errors gracefully.

- **7d — Clipboard goroutine** (`clipboard.go:80`):
  Add `ctx context.Context` parameter or a `done <-chan struct{}` to the goroutine
  closure so it terminates cleanly when the `App` shuts down.

**Dependencies**: None (independent quality work). Should be batched into a single PR.

**Goal Impact**: Improves debuggability (Gap 1 fix surfaced several nil-presenter
errors that were swallowed). Required for Gap 7 (API stability) — error types must
be stable before v1.0.0.

**Acceptance**: Zero `bare_error_return` in `internal/` (excluding `internal/demo/`).
Zero `Resource acquisition without defer close` in `internal/render/`. Goroutine in
`clipboard.go` respects context cancellation.

**Validation**:
```bash
go test ./... -count=1 -race
go-stats-generator analyze . --skip-tests --format json --sections patterns \
  | jq '[.patterns.anti_patterns.performance_antipatterns[]
         | select(.description | contains("Error returned without context"))
         | select(.file | contains("/internal/") and (contains("/internal/demo/") | not))
        ] | length'
# Expected: 0
```

---

### Step 8: API stabilization — compatibility tests and v1.0.0 tag ✅ DONE

**Why eighth**: Requires Steps 1–7 to be done — a stable API commitment is only
meaningful when the core functionality works. The `STABILITY.md` and `compat_test.go`
are already written (verified by `STABILITY.md` in the repo); this step enforces them.

**Deliverable**:
- `compat_test.go`: Verify that all 13 constructors and 7 methods listed in
  `STABILITY.md` have the exact signatures documented. The file already exists
  (`compat_test.go` is listed in repo root); audit it to confirm all identifiers from
  `STABILITY.md`'s "Covered Identifiers (v1.0.0)" section are included. Add any
  missing assertions.

- `CHANGELOG.md` (new file): Document the changes from v0.2.0 to v1.0.0 including:
  - Gap 1 fix (windows now display content)
  - Gap 2 fix (software backend SHM presenter)
  - Gap 6 fix (widget-demo now interactive)
  - AT-SPI2 support (atspi build tag)
  - Error wrapping improvements

- Git tag `v1.0.0`: After all prior steps pass CI, tag with "API stable" message.
  Run `make` target to produce `wain-libs-x86_64-unknown-linux-musl.tar.gz` per
  `RELEASE.md` workflow and attach to the GitHub release.

**Dependencies**: Steps 1–7.

**Goal Impact**: Closes Gap 7. Enables downstream projects to safely depend on
`github.com/opd-ai/wain`.

**Acceptance**: `go get github.com/opd-ai/wain@v1.0.0` in a fresh module compiles
a "hello" app without modification. `go test ./...` in the tagged state passes.

**Validation**:
```bash
go test ./... -run TestAPICompat -v
git tag -a v1.0.0 -m "Release v1.0.0 — API stable"
go list -m github.com/opd-ai/wain@v1.0.0
```

---

## Scope Assessment Summary

| Step | Effort | Risk | Blocks |
|------|--------|------|--------|
| 1 — Software SHM presenters | Medium (2 new files, ~150 LOC) | Low | Steps 2,4 |
| 2 — Wire Present() into App | Medium (surgical edit to `app.go`) | Medium (display server init) | Steps 3,5,6 |
| 3 — widget-demo de-stub | Small (~60 LOC replacement) | Low | — |
| 4 — Bench + append fixes | Medium (17 sites + CI job) | Low | — |
| 5 — GPU UI render path | Large (Rust TD-7–13 + Go dispatch) | High (GPU hardware required) | — |
| 6 — AT-SPI2 integration | Large (new package, D-Bus protocol) | Medium | — |
| 7 — Error/resource cleanup | Medium (96 sites, mechanical) | Low | Step 8 |
| 8 — v1.0.0 tag + compat | Small (audit + tag) | Low | — |

**Total estimated effort**: 6–10 developer-weeks for Steps 1–7; Step 8 is one day.

---

## Dependency Graph

```
Step 1 (SHM presenters)
    └── Step 2 (wire Present into App)
            ├── Step 3 (widget-demo)
            ├── Step 5 (GPU UI path)
            └── Step 6 (AT-SPI2)
Step 4 (bench + append fixes)  ←── Step 1 (needs Pixels())
Step 7 (error/resource cleanup) ←── independent
Step 8 (v1.0.0) ←── Steps 1–7 complete
```

---

## Thresholds Used for Scope Assessment

| Metric | Small | Medium | Large | This Codebase |
|--------|-------|--------|-------|---------------|
| Functions above CC 9 | <5 | 5–15 | >15 | **1** (Small) |
| Duplication ratio | <3% | 3–10% | >10% | **0.83%** (Small) |
| Doc coverage gap | <10% | 10–25% | >25% | **9.15% gap** (Small) |
| Bare error returns (library) | <10 | 10–30 | >30 | **96** (Large) |
| Resources without defer | <5 | 5–15 | >15 | **15** (Medium) |

---

## Quick-Reference Validation Commands

```bash
# After Step 1
go test ./internal/render/display/... -run TestSoftware -v

# After Step 2
go test ./... -count=1 && make wain-demo && echo "OK"

# After Step 3
go-stats-generator analyze cmd/widget-demo/ --skip-tests --format json \
  --sections functions | jq '[.functions[] | select(.complexity.cyclomatic > 9)] | length'
# Expected: 0

# After Step 4
go test -bench=BenchmarkFillRect -benchtime=3s ./internal/raster/primitives/

# After Step 5 (GPU hardware required)
go test -tags=integration -run TestGPURenderUIScene ./internal/integration/... -v

# After Step 6
go test -tags=atspi ./internal/a11y/atspi/... -v

# After Step 7
go test ./... -race && \
go-stats-generator analyze . --skip-tests --format json --sections patterns \
  | jq '.patterns.anti_patterns.performance_antipatterns | length'
# Expected: ≤10 (cmd/ code remainder acceptable)

# After Step 8
go test ./... -run TestAPICompat -v
```
