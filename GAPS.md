# Implementation Gaps — 2026-03-14

## ImageWidget Silently Invisible in Software Mode

- **Stated Goal**: The README lists `ImageWidget` as a first-class widget alongside Button, Label, and TextInput with no caveats. Users expect `NewImageWidget(img, size)` to display an image in all rendering modes.
- **Current State**: `ImageWidget.Draw` emits a `CmdDrawImage` display-list command (`concretewidgets.go:606`). The software rasterizer's consumer (`internal/raster/consumer/software.go:90–92`) hits the case branch `case displaylist.CmdDrawImage: // Skip for now — GPU backend handles this` and discards the command silently. On any system running in software mode (`ForceSoftware: true`, or no supported GPU), every `ImageWidget` renders as empty space with no error, no log message, and no visual placeholder.
- **Impact**: Any application running on a non-GPU machine (the majority of CI and user environments) that uses `ImageWidget` will display broken UIs with invisible images. The feature is advertised without the software-mode caveat.
- **Closing the Gap**: Implement `SoftwareConsumer.renderDrawImage` in `internal/raster/consumer/software.go`. Extract the image pixels from the display-list command's `Image` field, convert to `image.RGBA` if necessary, and call `composite.BlitScaled(dst, src, x, y, w, h)` (already available in `internal/raster/composite`). Add a unit test `TestImageWidgetSoftwarePath` that renders an `ImageWidget` through a `SoftwareConsumer` and asserts the output buffer is non-zero in the widget's bounding box.

---

## GPU Rendering Pipeline Not Exercised End-to-End in CI

- **Stated Goal**: The README claims "GPU Command Submission — Intel i915/Xe and AMD RDNA batch command generation with DMA-BUF export" and "Shader Compilation — WGSL shaders compiled to Intel EU and AMD RDNA native ISA via naga" as first-class features. The feature table presents GPU rendering as production-ready.
- **Current State**: `GPUBackend.Render()` constructs command batches and submits them via `submitBatchesWithScissor` (`internal/render/backend/gpu.go:221`). The shader-to-ISA compilation CI gate (`cargo test --test shader_compile`) verifies non-empty binary blobs without hardware. However, the complete path — widget tree → display list → GPU batch → execbuffer/amdgpu CS → DMA-BUF → compositor — is only tested when `/dev/dri/renderD128` is present (`ci.yml`, `gpu-check` step), which is never true on standard GitHub Actions runners. There is no mock-hardware integration test.
- **Impact**: GPU rendering regressions (wrong batch encoding, broken DMA-BUF handoff, pipeline state corruption) go undetected in CI. Developers cannot verify GPU correctness without physical Intel/AMD hardware.
- **Closing the Gap**: (1) Add a software-emulated GPU integration test in `internal/integration/gpu_pipeline_test.go` that creates a `GPUBackend` with a mock DRM allocator, renders a simple display list, and asserts the batch byte sequence is non-empty and structurally valid (correct header, non-zero length). (2) Wire the test under a build tag `//go:build integration` so it runs unconditionally in CI without requiring hardware. (3) Add a `cmd/gpu-ui-demo` that renders a complete widget tree via GPU path, exercising the full pipeline on hardware runners when available.

---

## AT-SPI2 Accessibility Has 0.0% Effective Test Coverage

- **Stated Goal**: The README states "AT-SPI2 Accessibility — D-Bus screen reader integration with Accessible, Component, Action, and Text interfaces." `ACCESSIBILITY.md` documents four D-Bus interfaces. `STABILITY.md` lists `EnableAccessibility` and `AccessibilityManager` as stability-pinned public API.
- **Current State**: `internal/a11y/` contains 10 source files and 75 functions implementing the four AT-SPI2 interfaces. `manager_test.go` and `a11y_test.go` exist but exercise only struct-field setters (e.g., `SetBounds`, `SetText`) without using the build tag `atspi`. Running `go test ./internal/a11y/` reports `coverage: 0.0% of statements` because all implementation code is guarded by `//go:build atspi`. The D-Bus export paths, method handlers (`GetRole`, `GetDescription`, `DoAction`, `GetText`), and the Manager's object registration are completely untested.
- **Impact**: Any regression in the D-Bus interface implementations (wrong return types, broken property serialization, panics in signal emission) is invisible to CI. Screen-reader users (the primary audience for this feature) would experience silent failures.
- **Closing the Gap**: Add `//go:build atspi` to the test files and update the CI matrix to run `go test -tags=atspi ./internal/a11y/` with a mock D-Bus session using `github.com/godbus/dbus/v5`'s `SessionBusPrivate` + `Auth(dbus.AuthAnonymous())` pattern (no display server required). Tests should cover: object registration, `GetRole`/`GetName`/`GetDescription` return values, `DoAction` callback invocation, and `SetFocused` signal emission.

---

## Frame-Delivery Path (`internal/render/display`) Undertested

- **Stated Goal**: The README and architecture documentation describe a complete frame delivery pipeline: GPU renders to a DMA-BUF or shared-memory buffer; the display layer hands it to the Wayland compositor or X11 server each frame.
- **Current State**: `internal/render/display/` implements `WaylandPipeline`, `X11Pipeline`, `SoftwareWaylandPresenter`, and `SoftwareX11Presenter`. `go test -cover ./internal/render/display/` reports **1.9% coverage**. The `renderToFramebuffer`, `ensureWaylandBuffer`, and `PresentBuffer` functions — the frame-delivery hot path — are not covered. `internal/render/present/` (which contains `RenderAndPresent`, the top-level orchestrator) has **0 test files**.
- **Impact**: A regression in any frame-presentation path produces a broken display or a crash with no CI signal. The 60 FPS benchmark only validates the rasterizer in isolation, not the pipeline from rasterizer to screen.
- **Closing the Gap**: (1) Add `present_test.go` with mock `FramebufferPool` and `PlatformPresenter` implementations that record `RenderToFramebuffer` and `PresentBuffer` calls; assert correct sequencing. (2) Add display tests using `net.Pipe()` to simulate a Wayland socket and verify that `WaylandPipeline.RenderAndPresent` sends a `wl_surface.attach`+`commit` sequence. Both test files require no physical display server.

---

## `AcquireForWriting` Uses Polling Instead of Condition Signaling

- **Stated Goal**: The README promises "Double/Triple Buffering — frame synchronization with compositor." The architecture description implies low-latency frame delivery.
- **Current State**: `internal/buffer/ring.go:160–175` — `AcquireForWriting` polls for an available slot using a 5 ms `time.After` tick. Under compositor back-pressure all slots may be in `StateDisplaying`, forcing the render goroutine to spin-wait for up to 5 ms before retrying. This adds up to 5 ms of extra latency per frame (≈30% of the 16.7 ms budget) and wastes CPU cycles.
- **Impact**: Frame delivery latency is degraded under load or on slow compositors. On a 60 FPS budget, a 5 ms poll adds 30% overhead to worst-case frame times.
- **Closing the Gap**: Replace the `time.After` poll with a condition variable (`sync.Cond`) or a release notification channel. In `markSlotTransition` (called by `MarkReleased`), call `cond.Signal()` after the state change. `AcquireForWriting` waits on the condition variable instead of sleeping. This reduces worst-case acquisition latency from 5 ms to ≈100 µs (OS scheduling granularity). Validate with: `go test -bench=BenchmarkAcquireForWriting ./internal/buffer/` and verify p99 latency drops.

---

## GPU Documentation Gap

- **Stated Goal**: The README prominently lists GPU rendering as a key feature. `HARDWARE.md` documents a supported GPU hardware matrix (Intel Gen9–Xe, AMD RDNA 1–3).
- **Current State**: `HARDWARE.md` covers hardware compatibility but does not explain: how to verify GPU detection at runtime (`./bin/wain --detect-gpu` or equivalent), how to force GPU vs software mode via `AppConfig.ForceSoftware`, how to interpret `auto-render-demo` output, or how to add new WGSL shaders. A new user with an Intel Gen12 laptop has no guided path from "I have this GPU" to "I can see a GPU-rendered frame."
- **Impact**: GPU-capable users default to software rendering because they cannot discover or enable GPU acceleration. The project's headline GPU feature goes unused.
- **Closing the Gap**: (1) Add a "GPU Usage" section to `GETTING_STARTED.md` covering runtime GPU detection (`AppConfig{Verbose: true}` outputs backend selection), forced software fallback, and `cmd/auto-render-demo` interpretation. (2) Add `render-sys/shaders/README.md` documenting how to add a shader: write WGSL, add to `shaders.rs` constant table, add a CI validation entry in `shader_compile.rs`.
