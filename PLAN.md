# Implementation Plan: Pixel Presentation → GPU Integration → v1.0

> Generated 2026-03-14 from `go-stats-generator` metrics + project documentation analysis.

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit that links a Rust rendering library
  via CGO/musl for GPU-accelerated graphics on Linux, implementing Wayland and X11 from scratch and
  producing a single fully-static binary with zero runtime dependencies.
- **Current goal**: Make the public API produce visible pixels — the `App`/`Window` render pipeline
  is structurally complete but `Present()` is never called, so all rendered content stays in heap
  memory and no window ever shows output.
- **Estimated Scope**: **Large** — 7 implementation steps, spanning display pipeline wiring, GPU
  integration, CI benchmarks, error-handling hygiene (174 library violations), and AT-SPI2.

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| Go–Rust static linking via CGO/musl | ✅ Achieved | No |
| Wayland client (9 packages) | ✅ Achieved | No |
| X11 client (9 packages) | ✅ Achieved | No |
| Software 2D rasterizer | ✅ Achieved | No |
| UI widget layer with flexbox layout | ✅ Achieved | No |
| GPU buffer infrastructure | ✅ Achieved | No |
| GPU command submission for UI (Intel + AMD) | ⚠️ Partial | Yes — Step 4 |
| Shader frontend (naga WGSL/GLSL) | ✅ Achieved | No |
| Intel EU backend (Gen9+) | ✅ Achieved | No |
| AMD RDNA backend | ✅ Achieved | No |
| Public API with auto-detection | ✅ Achieved | No |
| Display list rendering | ✅ Achieved | No |
| **Public API actually renders pixels** | ❌ Gap 1: `Present()` never called | **Yes — Steps 1–2** |
| **Software fallback presentation** | ❌ Gap 2: always returns `ErrSoftwareNoDmabuf` | **Yes — Steps 1–2** |
| **`widget-demo` shows a real window** | ❌ Gap 6: both `runWayland/runX11` are stubs | **Yes — Step 3** |
| <2ms GPU frame time (typical UI) | ⚠️ Unverified in CI | Yes — Step 5 |
| 60 FPS software rendering @ 1080p | ⚠️ Unverified in CI | Yes — Step 5 |
| Zero runtime dependencies | ✅ Achieved | No |
| Accessibility — keyboard navigation | ✅ Achieved | No |
| Accessibility — AT-SPI2 screen reader | ❌ Not implemented | Yes — Step 6 |
| Clipboard support | ✅ Achieved | No |
| HiDPI / DPI-aware scaling | ✅ Achieved | No |
| API stability declared (v1.0) | ❌ Not yet | Yes — Step 7 |

---

## Metrics Summary (go-stats-generator, 2026-03-14, `--skip-tests`)

| Metric | Value | Threshold | Assessment |
|---|---|---|---|
| Total Go LOC | 13,845 | — | Moderate, well-distributed |
| Total packages | 38 | — | Clean layered architecture |
| Total functions + methods | 1,720 | — | — |
| Functions with CC > 9 | **0** | <5 = Small | ✅ Zero blockers |
| Functions > 50 lines | **7** (all in `cmd/`) | — | ✅ No library issue |
| Duplication ratio | **0.80%** | <3% = Small | ✅ Excellent |
| Clone pairs | 16 | — | Minor; all in demo/cmd code |
| Documentation coverage (overall) | **91.4%** | ≥80% = target met | ✅ |
| — packages | 100% | | |
| — functions | 98.7% | | |
| — methods | 89.6% | | Gap in concrete widget receivers |
| — types | 90.9% | | |
| Circular dependencies | **0** | 0 = target | ✅ |
| `bare_error_return` (library only) | **174** | >15 = Large | ❌ Errors lack context |
| `unused_receiver` (library only) | **76** | >15 = Large | ❌ Interface stubs unimplemented |
| `memory_allocation` warnings | 18 | — | Medium priority |
| `resource_leak` warnings | 15 | — | Medium priority |
| `goroutine_leak` warnings | 4 | — | Medium priority |

### Package Coupling Hotspots

| Package | Coupling Score | Deps | Note |
|---|---|---|---|
| `wain` (root public API) | 9.0 | 18 | Expected for public API |
| `internal/render/display` | 4.0 | 8 | Highest non-cmd coupling; key integration point |
| `internal/render/backend` | 2.5 | 5 | Core rendering glue |
| `internal/ui/decorations` | 2.0 | 4 | Normal |
| `internal/demo` | 6.0 | 12 | Expected for demo helpers |

### Anti-Pattern Hot Spots (library code, by package)

| Package | `bare_error_return` | `unused_receiver` |
|---|---|---|
| `internal/x11/wire` | 33 | 0 |
| `app.go` | 30 | 3 |
| `internal/wayland/input` | 20 | 14 |
| `internal/wayland/wire` | 10 | 0 |
| `internal/x11/client` | 10 | 0 |
| `concretewidgets.go` | 0 | 14 |
| `internal/ui/widgets` | 0 | 11 |
| `internal/wayland/client` | 7 | 0 |
| `internal/wayland/socket` | 7 | 0 |
| `internal/render/backend` | 6 | 3 |

---

## Implementation Steps

Steps are ordered by **dependency** (prerequisite first), then by **descending impact** on stated
project goals.

---

### Step 1: Wire Software SHM Presentation (Wayland + X11)

> **Root cause**: `Window.RenderFrame()` populates an in-memory pixel buffer via the software
> rasterizer but never calls `Present()`, so the compositor never receives a frame.
> `SoftwareBackend.Present()` always returns `ErrSoftwareNoDmabuf` because it has no SHM path.

- **Deliverable**:
  1. Add `Pixels() []byte` method to `internal/render/backend/software.go` that exposes the
     rasterizer's raw ARGB8888 framebuffer.
  2. Create `internal/render/display/software_wayland.go` — `SoftwareWaylandPresenter` that writes
     the pixel buffer into a `wl_shm` pool (using `internal/wayland/shm`) and calls
     `wl_surface.commit` each frame.
  3. Create `internal/render/display/software_x11.go` — `SoftwareX11Presenter` that writes pixels
     via `internal/x11/shm` + `internal/x11/gc.CopyArea`.
  4. In `app.go`: `Window.initWaylandWindow()` creates and stores a `SoftwareWaylandPresenter`;
     `Window.initX11Window()` creates and stores a `SoftwareX11Presenter`.
  5. In `app.go`: `Window.RenderFrame()` calls `presenter.Present()` after `renderBridge.Render()`.

- **Dependencies**: None. This step is self-contained.

- **Goal Impact**:
  - Gap 1: public API windows display rendered content for the first time.
  - Gap 2: software fallback becomes functional on all hardware, including CI.
  - Unlocks Steps 2, 3, 4.

- **Acceptance**:
  - `./bin/example-app` and `./example/hello/hello` open windows showing rendered widgets.
  - `go test -run TestSoftwarePresent ./internal/render/display/...` passes.

- **Validation**:
  ```bash
  go test -run TestSoftwarePresent ./internal/render/display/...
  # Manual: make wain-demo && ./bin/wain-demo  →  window shows rendered widgets
  ```

---

### Step 2: Restore `cmd/widget-demo` as a Functional Interactive Demo

> **Root cause**: Both `runWayland()` and `runX11()` in `cmd/widget-demo/main.go` are stubs that
> print a warning and return. This is the demo README users are told to run first.

- **Deliverable**:
  1. Implement `runWayland()` using the `SoftwareWaylandPresenter` from Step 1 (or the
     `WaylandPipeline` pattern from `cmd/gpu-display-demo` for GPU path).
  2. Implement `runX11()` using the `SoftwareX11Presenter` (or `X11Pipeline` for GPU).
  3. Remove stub warning log lines once both paths are live.
  4. The demo must render at minimum: one `Button`, one `TextInput`, and one `ScrollView` widget.

- **Dependencies**: Step 1.

- **Goal Impact**: Gap 6 — the README's primary demo now works.

- **Acceptance**:
  - `make widget-demo && ./bin/widget-demo` opens a window on Wayland or X11 with visible widgets.
  - No stub warning messages appear in stdout.

- **Validation**:
  ```bash
  make widget-demo
  ./bin/widget-demo  # manual visual check: window must show button + text input + scroll
  ```

---

### ~~Step 3: Wire GPU DMA-BUF Presentation Path~~ ✅ DONE

> **Root cause**: The GPU backend (`internal/render/backend/gpu.go`) builds batch buffers and
> submits them via `render.SubmitBatch()`, but `GPUBackend.Present()` does not connect to
> `internal/wayland/dmabuf` (Wayland) or `internal/x11/dri3` (X11). The `display.WaylandPipeline`
> and `display.X11Pipeline` exist but are not instantiated by the `App` type.

- **Deliverable**:
  1. In `Window.initWaylandWindow()`: when GPU backend is selected, create and store a
     `display.WaylandPipeline` (already implemented in `internal/render/display/wayland.go`).
  2. In `Window.initX11Window()`: when GPU backend is selected, create and store a
     `display.X11Pipeline` (already implemented in `internal/render/display/x11.go`).
  3. In `Window.RenderFrame()`: route to the GPU presenter when `GPUBackend` is active.
  4. Add `TestGPUPresent` integration test in `internal/integration/` gated by
     `//go:build integration`.

- **Dependencies**: Step 1 (ensures software fallback works; GPU path is additive).

- **Goal Impact**: Gaps 4 & 7 — GPU path is live for real UI workloads (not just triangle demos).
  This verifies the Intel EU and AMD RDNA backends produce visible output end-to-end.

- **Acceptance**:
  - `./bin/gpu-display-demo` renders a complete widget hierarchy using GPU batch submission.
  - `go test -tags=integration -run TestGPUPresent ./internal/integration/...` passes on hardware
    with `/dev/dri/renderD128` (gated; skipped in CI).

- **Validation**:
  ```bash
  make gpu-display-demo && ./bin/gpu-display-demo
  go test -tags=integration -run TestGPUPresent ./internal/integration/...
  ```

---

### ~~Step 4: Add Software Rasterizer Benchmarks to CI~~ ✅ DONE

> **Root cause**: HARDWARE.md claims ≤16ms software frame time at 1080p (60 FPS) but no benchmark
> runs in CI. `cmd/perf-demo` exists but is not invoked in `.github/workflows/ci.yml`. SIMD
> optimizations are documented as not yet implemented.

- **Deliverable**:
  1. Add `BenchmarkFillRect1080p` and `BenchmarkRoundedRect1080p` in
     `internal/raster/primitives/rect_bench_test.go`.
  2. Add a non-blocking CI job in `.github/workflows/ci.yml`:
     ```yaml
     - name: Software raster benchmark
       run: go test -bench=BenchmarkFillRect1080p -benchtime=3s ./internal/raster/primitives/ | tee bench.txt
     - name: Check baseline (warn only)
       run: grep BenchmarkFillRect bench.txt | awk '{if ($3 > 16000000) print "WARN: frame time above 16ms baseline"}'
     ```
  3. Implement AVX2 fast-path in `internal/raster/primitives/rect.go` using
     `golang.org/x/sys/cpu` for runtime detection (add as direct dependency).
     - NEON path for arm64 in a separate `rect_arm64.go` file.
  4. After SIMD: tighten the CI check to a hard failure at 16ms.

- **Dependencies**: Step 1 (software presenter must work to validate the end-to-end stack).

- **Goal Impact**: Gap 5 — performance claims gain CI protection; SIMD delivers the expected
  2–4× speedup documented in HARDWARE.md.

- **Acceptance**:
  ```bash
  go test -bench=BenchmarkFillRect1080p -benchtime=3s ./internal/raster/primitives/
  # target: reported ns/op implies ≥150 MPixels/s throughput
  go-stats-generator analyze ./internal/raster/... --format json | jq '.functions[] | select(.complexity.cyclomatic > 9)'
  # must return empty — no new complexity hotspots introduced by SIMD dispatch
  ```

---

### Step 5: Wrap Errors with Context in Library Code (174 violations)

> **Root cause**: `go-stats-generator` reports 174 `bare_error_return` violations in library
> packages — errors propagated with no `fmt.Errorf("context: %w", err)` wrapping. This makes
> debugging production issues nearly impossible (callers see "EOF" or "no such file or directory"
> with no call-site context).

- **Deliverable**: Add `fmt.Errorf("…: %w", err)` wrapping at every `bare_error_return` site in
  the following packages (in priority order, highest count first):
  1. `internal/x11/wire` (33 sites)
  2. `app.go` (30 sites)
  3. `internal/wayland/input` (20 sites)
  4. `internal/wayland/wire` (10 sites)
  5. `internal/x11/client` (10 sites)
  6. Remaining packages (71 sites across `wayland/client`, `wayland/socket`, `render/backend`,
     `render/present`, `render/atlas`, `ui/pctwidget`, `render.go`, and others)

  Each wrapped error message must identify the operation:
  ```go
  // before
  return err
  // after
  return fmt.Errorf("x11/wire: read setup: %w", err)
  ```

- **Dependencies**: None. Can proceed in parallel with Step 2.

- **Goal Impact**: Directly improves debuggability of all display-server and rendering failures.
  Reduces risk classification for the "Rust FFI panics crash entire process" risk item (better
  error context distinguishes Go errors from FFI panics).

- **Acceptance**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections patterns \
    | jq '[.patterns.anti_patterns.performance_antipatterns[]
           | select(.type == "bare_error_return")
           | select(.file | test("/cmd/") | not)] | length'
  # target: ≤ 10 (residual false positives in generated or interface-mandatory returns)
  go test ./...  # all existing tests must continue to pass
  ```

---

### Step 6: Implement AT-SPI2 Screen-Reader Support

> **Root cause**: `ACCESSIBILITY.md` explicitly documents AT-SPI2 as "not yet implemented."
> The `internal/a11y/` package has interface stubs (76 `unused_receiver` violations) but no D-Bus
> object export. `github.com/godbus/dbus/v5` is already in `go.mod` as an indirect dependency.

- **Deliverable**:
  1. Promote `github.com/godbus/dbus/v5` to a direct dependency in `go.mod`.
  2. Create `internal/a11y/atspi/` package:
     - `bus.go` — session bus connection + `org.a11y.atspi.Registry` registration.
     - `accessible.go` — `Accessible` interface for `wain.Button`, `wain.TextInput`, `wain.Label`,
       `wain.Panel`.
     - `component.go` — `Component` interface with screen-coordinate bounding boxes.
     - `action.go` — `Action` interface (DoDefaultAction for Button → click event).
     - `text.go` — `Text` interface for `wain.TextInput` (caret position, content).
  3. Wire into `EventDispatcher`: emit `focus-changed` and
     `object:state-changed:focused` D-Bus signals when `FocusManager` changes focus.
  4. Gate behind a `+atspi` build tag to keep the zero-dependency guarantee for users who do not
     need screen-reader support; document the build tag in `ACCESSIBILITY.md`.
  5. Fix the 76 `unused_receiver` violations in existing `internal/a11y/` stubs by filling in
     each method body (or removing unused stubs where the interface was abandoned).

- **Dependencies**: Steps 1–2 (window must display for AT-SPI2 coordinates to be meaningful).

- **Goal Impact**: Gap 3 — AT-SPI2 support lands; ACCESSIBILITY.md caveat is resolved.

- **Acceptance**:
  ```bash
  go test -tags=atspi ./internal/a11y/atspi/...
  go-stats-generator analyze ./internal/a11y/... --format json --sections patterns \
    | jq '[.patterns.anti_patterns.performance_antipatterns[]
           | select(.type == "unused_receiver")] | length'
  # target: 0 unused_receiver in internal/a11y/
  # manual: Orca reads button label in ./bin/widget-demo
  ```

---

### Step 7: API Stabilization and v1.0 Tag

> **Root cause**: README states "not yet API-stable. Signatures may change … until v1.0.0 is
> tagged." There is no `STABILITY.md`, no compatibility test suite, and no `v1.0.0` git tag. This
> prevents downstream libraries from safely depending on `github.com/opd-ai/wain`.

- **Deliverable**:
  1. **Audit exported identifiers**: run `go doc github.com/opd-ai/wain` and review every exported
     name. Resolve any names that do not follow Go naming conventions or that conflict across
     packages. Prioritize names referenced in `example/` and README code snippets.
  2. **Compatibility test file**: add `compat_test.go` at the module root that compiles against
     pinned public API signatures (function signatures as Go compile-time assertions), failing the
     build if any signature changes.
  3. **Write `STABILITY.md`**: document the deprecation policy (deprecate before removing,
     minimum one minor release notice), migration path template, and semver commitment.
  4. **Tag `v1.0.0`** after Steps 1–6 are complete and `go test ./...` passes cleanly.

- **Dependencies**: Steps 1–6 must be complete (rendering must work; AT-SPI2 stubs resolved;
  errors wrapped; benchmarks green).

- **Goal Impact**: Gap 7 — downstream consumers can safely `go get github.com/opd-ai/wain@v1.0.0`.

- **Acceptance**:
  ```bash
  go test -run TestAPICompat ./...         # compile-time signature assertions pass
  git tag v1.0.0 && git push --tags
  # fresh module test:
  cd /tmp && mkdir testmod && cd testmod
  go mod init example.com/test
  go get github.com/opd-ai/wain@v1.0.0
  go build .  # must compile without modification
  ```

---

## Step Dependency Graph

```
Step 1 (SHM presentation)
    └── Step 2 (widget-demo)
    └── Step 3 (GPU DMA-BUF path)
    └── Step 4 (CI benchmarks)
    └── Step 6 (AT-SPI2)  ──────────────────┐
                                              │
Step 5 (error wrapping)  [parallel to 1–4]   │
                                              │
Step 6 (AT-SPI2)         [needs 1+2]         │
                                              │
Step 7 (v1.0 tag)        [needs 1–6] ────────┘
```

Steps 5 can proceed in parallel with Steps 1–4 since it touches a disjoint set of call sites.

---

## Default Thresholds Reference

| Metric | Small | Medium | Large | Current |
|---|---|---|---|---|
| Functions above CC 9.0 | <5 | 5–15 | >15 | **0** ✅ |
| Duplication ratio | <3% | 3–10% | >10% | **0.80%** ✅ |
| Doc coverage gap | <10% | 10–25% | >25% | **8.6%** ✅ |
| `bare_error_return` (library) | <5 | 5–15 | >15 | **174** ❌ Large |
| `unused_receiver` (library) | <5 | 5–15 | >15 | **76** ❌ Large |

---

## Risk Register

| Risk | Severity | Step | Mitigation |
|---|---|---|---|
| SHM buffer size mismatch on resize crashes compositor | HIGH | 1 | Reallocate pool on `wl_surface.configure`; add resize test |
| GPU presenter references invalid DMA-BUF fd after GPU backend error | HIGH | 3 | Check fd ≥ 0 before `zwp_linux_buffer_params_v1.add`; return error, not panic |
| Rust FFI `.unwrap()` calls (89 found) crash entire process | HIGH | 3 | Audit FFI boundary in `render-sys/src/lib.rs`; wrap hot paths in `catch_unwind` |
| AT-SPI2 D-Bus adds optional runtime dep (session bus) | MEDIUM | 6 | Build tag `+atspi`; gracefully disable if bus unavailable |
| Error context wrapping breaks `errors.Is` callers | LOW | 5 | Use `%w` verb (not `%v`) in all wrapping; verify with `errors.Is` unit tests |
| v1.0 tag with undiscovered API warts | MEDIUM | 7 | Compatibility test file catches regressions; announce RC period |

---

## Post-v1.0 Backlog (Out of Scope for This Plan)

| Feature | Estimated Effort |
|---|---|
| NVIDIA nouveau GPU backend | 2–3 weeks |
| Multi-window support | 1 week |
| Drag-and-drop (Wayland `wl_data_device` + X11 XDND) | 1 week |
| Property animations with easing | 2 weeks |
| SVG vector icon rendering | 1–2 weeks |
| SIMD NEON path for arm64 rasterizer | 3 days |

---

*Metrics collected with `go-stats-generator analyze . --skip-tests --format json --sections functions,duplication,documentation,packages,patterns` at commit HEAD (2026-03-14).*
