# Implementation Gaps — 2026-03-14

## Gap 1: Public API Window Does Not Display Pixels

- **Stated Goal**: The README shows a working `app.Run()` loop that opens a window and renders widgets to the screen. The `App` type claims to "auto-detect the display server and renderer" and the `Window` type supports `SetLayout()` with a full widget tree.
- **Current State**: `App.Run()` calls `win.RenderFrame()` → `RenderBridge.Render()` → `renderer.Render(dl)`, which fills an internal GPU or software buffer. However, `RenderBridge.Present()` is **never called** from the event loop, and neither `WaylandPipeline` nor `X11Pipeline` (from `internal/render/display`) are created or used by the `App` type. The Wayland surface receives one empty `wl_surface.commit` at initialization (`app.go:380`) with no buffer attached; the X11 window receives no pixel data at all. The rendered content lives in heap memory and is never sent to the compositor.
- **Impact**: Any application using the public API gets a blank window regardless of widgets added. This is the most fundamental user-facing gap in the project. The `cmd/wayland-demo` and `cmd/gpu-display-demo` work only because they bypass the public API and call the internal pipeline constructors directly.
- **Closing the Gap**:
  1. On Wayland: in `Window.initWaylandWindow()`, create a `display.WaylandPipeline` (for the GPU path) or a `SoftwareSHMPresenter` (for the software path). Store it on the `Window` struct.
  2. On X11: in `Window.initX11Window()`, create a `display.X11Pipeline` (GPU) or use `internal/x11/shm` + `gc.CopyArea` (software).
  3. In `Window.RenderFrame()`, replace `renderBridge.Render(rootWidget)` with `pipeline.RenderAndPresent(ctx, displayList)`.
  4. For the software fallback: add `SoftwareBackend.Pixels() []byte` to expose the CPU buffer, then implement a `SoftwareSHMPresenter` in `internal/render/display/` that writes pixels into a `wl_shm` pool and calls `wl_surface.commit`.
  5. Validate: `./bin/wain-demo` and `example/hello/hello` must open a visible window with rendered content.

---

## Gap 2: Software Backend Cannot Present Frames

- **Stated Goal**: The README and HARDWARE.md state that wain "automatically falls back to a software rasterizer that produces pixel-identical output to the GPU backends" when no compatible GPU is detected.
- **Current State**: `SoftwareBackend.Present()` always returns `ErrSoftwareNoDmabuf` (file descriptor = −1). The `display.WaylandPipeline` calls `renderer.Present()` expecting a DMA-BUF file descriptor; passing a `SoftwareBackend` causes every frame submission to fail with an error. There is no SHM-based presentation path in the display pipeline.
- **Impact**: The software fallback cannot display frames on any display server. Users without compatible Intel/AMD GPUs (the majority of CI environments and many embedded targets) would get a non-functional window.
- **Closing the Gap**:
  1. Add `Pixels() []byte` to `SoftwareBackend` returning the raw ARGB8888 framebuffer (already held in `primitives.Buffer`).
  2. Create `internal/render/display/software_wayland.go` implementing a `SoftwareWaylandPresenter` using `internal/wayland/shm` to upload pixels each frame.
  3. Create `internal/render/display/software_x11.go` using `internal/x11/shm` + `internal/x11/gc.CopyArea`.
  4. Update `auto.go` in `internal/render/backend` to route to the correct presenter when `GPUBackend` is unavailable.
  5. Validate: `go test ./internal/render/display/...` including a new `TestSoftwarePresent` that reads pixels back via `SoftwareBackend.Pixels()`.

---

## Gap 3: AT-SPI2 Screen-Reader Support Missing

- **Stated Goal**: ACCESSIBILITY.md documents the full AT-SPI2 D-Bus interface set (`Accessible`, `Component`, `Action`, `Text`, `Value`) and describes the implementation path. The project claims Linux-first accessibility support.
- **Current State**: `internal/a11y/` implements only keyboard-focus traversal and event routing. No D-Bus objects are exported, no `org.a11y.atspi.*` interfaces exist. `ACCESSIBILITY.md:14` explicitly acknowledges: "Wain does not currently implement AT-SPI2 accessibility support."
- **Impact**: Applications built with wain cannot be used with Orca, Accerciser, or any AT-SPI2-based assistive technology. This blocks adoption in any accessibility-required context (government, enterprise, or accessibility-compliant software).
- **Closing the Gap**:
  1. Promote `github.com/godbus/dbus/v5` from indirect to direct dependency in `go.mod`.
  2. Create `internal/a11y/atspi/` implementing `Accessible`, `Component`, and `Action` interfaces for `Button`, `TextInput`, `Label`, `Panel`.
  3. Integrate with `EventDispatcher`: emit `focus-changed` and `object:state-changed:focused` D-Bus signals when `FocusManager` moves focus.
  4. Register with `org.a11y.atspi.Registry` on `App.Run()`.
  5. Validate: `go test ./internal/a11y/atspi/...`; Orca announces button labels in `cmd/widget-demo`.

---

## Gap 4: GPU Rendering Path Not Exercised for Real UI Workloads

- **Stated Goal**: The README's "GPU Command Submission" feature entry describes "Batch buffer construction, Intel 3D pipeline command encoding … pipeline state objects, and surface/sampler state encoding." HARDWARE.md states `<2 ms` GPU frame time for typical UI (200–500 rectangles, 50–100 text runs).
- **Current State**: The GPU infrastructure (batch buffers, shaders, pipeline state) is implemented in `render-sys/src/` and tested in Rust unit tests. The Go-side `GPUBackend.Render()` submits a batch buffer via `render.SubmitBatch()` (real ioctl path), but the only end-to-end GPU demo is `cmd/gpu-triangle-demo` (triangle geometry only). No UI-level workload (rectangles from a display list, text, shadows) is ever submitted to the GPU in any demo or test. The `display/wayland.go` and `display/x11.go` pipelines, while structurally complete, are not wired into the `App` type (Gap 1).
- **Impact**: The `<2 ms` GPU claim is based on ad-hoc benchmarking; no automated path verifies that UI display lists produce correct GPU output. Regressions in shader or batch encoding would be invisible.
- **Closing the Gap**:
  1. After closing Gap 1, run `cmd/gpu-display-demo` with a display list that mirrors a typical UI (use the `buildDemoScene` helper from `cmd/auto-render-demo`).
  2. Add a CI job using `/dev/dri/renderD128` (conditioned on GPU presence, matching the existing `gpu-check` step in `ci.yml`) that runs `go test -tags=integration -run TestGPURenderUIScene ./internal/integration/...`.
  3. Add `cmd/perf-demo` as a CI step with `--quick` flag to smoke-test the profiler path.
  4. Validate: `TestGPURenderUIScene` passes on hardware with an Intel Gen9+ GPU.

---

## Gap 5: Performance Targets Not Verified in CI

- **Stated Goal**: HARDWARE.md states `<2 ms` GPU frame time and `≤16 ms` software frame time (60 FPS at 1080p). These are positioned as product guarantees.
- **Current State**: No benchmark runs in CI. `cmd/perf-demo` exists but is not invoked in `.github/workflows/ci.yml`. Software performance measurements in HARDWARE.md are from a single ad-hoc session on one machine.
- **Impact**: Performance regressions are undetectable in PR review. SIMD is acknowledged as not implemented (expected 2–4× improvement); without a baseline measurement, there is no way to confirm the claimed targets are met even on reference hardware.
- **Closing the Gap**:
  1. Add `go test -bench=BenchmarkFillRect -benchtime=3s -benchmem ./internal/raster/primitives/` to a new CI job. Set a soft baseline (e.g., ≥150 MP/s at 1080p) and emit a warning (non-blocking) if below.
  2. Implement AVX2 SIMD hot paths for `FillRect` and `FillRoundedRect` in `internal/raster/primitives/rect.go` (runtime-detected via `golang.org/x/sys/cpu`).
  3. After SIMD: gate the CI benchmark as a hard failure at the pre-SIMD baseline, letting SIMD be a performance bonus.
  4. Validate: `go test -bench=. ./internal/raster/primitives/` reports ≥150 MP/s on reference hardware.

---

## Gap 6: `cmd/widget-demo` Interactive Display Not Implemented

- **Stated Goal**: The README lists `widget-demo` under "Demonstration Binaries" as an "interactive widget demo (X11/Wayland)."
- **Current State**: Both `runWayland()` and `runX11()` in `cmd/widget-demo/main.go` are stubs that print a warning message and return immediately. The demo simulates click/scroll events in-process for X11, but no window is ever shown.
- **Impact**: A developer following the README's "run the demos" instructions gets no visual output from the most prominent demo. This damages trust in the project's completeness.
- **Closing the Gap**:
  1. Implement `runWayland()` using the `WaylandPipeline` pattern from `cmd/gpu-display-demo`. Attach the widget tree via `demo.AttachAndDisplayBuffer` or the pipeline's `RenderAndPresent`.
  2. Implement `runX11()` using the `X11Pipeline` or SHM path.
  3. Remove the stub warning messages once both paths are live.
  4. Validate: `make widget-demo && ./bin/widget-demo` opens a window with a visible `Button`, `TextInput`, and `ScrollView`.

---

## Gap 7: API Stability Not Declared

- **Stated Goal**: The README notes "(v0.2.0): The public API is functional but not yet API-stable. Signatures may change in future minor releases until v1.0.0 is tagged."
- **Current State**: No `v1.0.0` tag, no `STABILITY.md`, no compatibility-test suite. There are 32 identifier naming suggestions from `go-stats-generator` (e.g., `Uint32s`, `OnPixmapIdle`) that indicate names may still evolve.
- **Impact**: Downstream libraries or applications cannot safely depend on `github.com/opd-ai/wain` without risk of breakage in patch releases.
- **Closing the Gap**:
  1. Resolve open naming issues (32 identifier suggestions from static analysis; prioritize exported names).
  2. Write `STABILITY.md` documenting the deprecation policy and migration path.
  3. Add a `TestAPICompatibility` file that compiles against pinned function signatures to catch accidental breakage.
  4. Tag `v1.0.0` after Gaps 1–6 are resolved (rendering must work for stable API to be meaningful).
  5. Validate: `go get github.com/opd-ai/wain@v1.0.0` in a fresh module compiles without modification.
