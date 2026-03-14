# Implementation Gaps — 2026-03-14

## Canvas.LinearGradient Ignores Angle Parameter

- **Stated Goal**: The `Canvas` interface (`publicwidget.go:111–112`) documents `LinearGradient` as:
  _"The angle is in degrees (0 = left-to-right, 90 = top-to-bottom)."_ This implies users can
  draw gradients at arbitrary angles (horizontal, vertical, diagonal) to achieve varied visual
  effects in custom widgets.
- **Current State**: Both concrete `Canvas` implementations silently discard the `angle`
  argument. `displayListCanvas.LinearGradient` (`publicwidget.go:270–278`) contains the comment
  _"For simplicity, assume 0 degrees = horizontal left-to-right"_ and always computes a
  horizontal gradient. `bufferCanvas.LinearGradient` (`concretewidgets.go:142–150`) uses
  `_ float64` for the angle, explicitly discarding it. Any call with `angle != 0` (e.g., vertical
  gradient at 90°, or diagonal at 45°) renders the same left-to-right gradient without any
  warning or error.
- **Impact**: Widgets that use `Canvas.LinearGradient` with a non-zero angle will silently
  render incorrectly. This affects custom widget implementations that try to draw vertical
  (top-to-bottom) or diagonal gradients — a common UI pattern for button highlight effects and
  backgrounds. There is no compile-time or runtime indication that the angle was ignored.
- **Closing the Gap**: In both `displayListCanvas.LinearGradient` and
  `bufferCanvas.LinearGradient`, replace the hard-coded horizontal calculation with proper
  angle-to-vector conversion:
  ```go
  import "math"
  rad := angle * math.Pi / 180
  cx, cy := float64(x+width/2), float64(y+height/2)
  hw, hh := float64(width)/2, float64(height)/2
  x0 := int(cx - hw*math.Cos(rad))
  y0 := int(cy - hh*math.Sin(rad))
  x1 := int(cx + hw*math.Cos(rad))
  y1 := int(cy + hh*math.Sin(rad))
  ```
  Apply this in both `publicwidget.go` and `concretewidgets.go`. Add a test
  `TestLinearGradientAngle90` that verifies `angle=90` produces a top-to-bottom gradient (top
  row ≈ `startColor`, bottom row ≈ `endColor`).

---

## SetOpacity Documented But Not Implemented

- **Stated Goal**: The public `animate.go` package-level godoc (`animate.go:46`) and the
  `internal/ui/animation/animation.go` usage example (`animation.go:20`) both use
  `widget.SetOpacity(v)` as the canonical demonstration of how to animate a widget property.
  This strongly implies to users that `SetOpacity` is a standard method on `wain` widgets.
- **Current State**: `SetOpacity` does not exist on any type in the codebase. A search for
  `func.*SetOpacity` across all `.go` files returns no results. Any application developer who
  reads the godoc for `App.Animate` or `Animator.Add` and copies the example will get a
  compile error.
- **Impact**: New users following the canonical example from the public API documentation will
  encounter an unexplained compile failure immediately. This undermines the "first application"
  experience described in `GETTING_STARTED.md` and `TUTORIAL.md`. The error is particularly
  confusing because the code compiles successfully for all other advertised APIs.
- **Closing the Gap**: Choose one of two fixes:
  1. **Implement `SetOpacity`** (preferred): Add `SetOpacity(alpha float64)` to the `Widget`
     interface and `PublicWidget` interface. Implement on `BasePublicWidget` (store `opacity`
     field, default 1.0). Pass the opacity as a multiplier on the `alpha` channel in each
     `drawListCanvas` draw call. This makes the example compile and provides a genuinely useful
     widget capability.
  2. **Update the examples**: Replace `widget.SetOpacity(v)` in both `animate.go:46` and
     `animation.go:20` with a method that actually exists, such as
     `label.SetText(fmt.Sprintf("%.0f%%", v*100))`. This is minimal but does not add the
     missing feature.

---

## Stale Package Documentation Contradicts Implementation

- **Stated Goal**: Package documentation in `internal/raster/consumer/doc.go` describes the
  `consumer` package's capabilities and limitations so contributors know what is and is not
  handled by the software rasterizer.
- **Current State**: Lines 13–17 of `doc.go` state:
  > _"The SoftwareConsumer does not implement the CmdDrawImage display list command.
  > Image compositing is available through the composite package's Blit and BlitScaled
  > functions, but DrawImage command execution requires a GPU backend."_
  This was accurate before technical debt item TD-2 was resolved in v1.1. However,
  `software.go:96–112` now contains a working `renderDrawImage` implementation that converts
  any `image.Image` to a `primitives.Buffer` and blits it via `composite.BlitScaled`. The
  `CmdDrawImage` case in `renderCommand` (line 85–87) calls this function. The stale comment
  is directly contradicted by the code.
- **Impact**: Contributors reading `doc.go` will incorrectly believe `ImageWidget` is a
  GPU-only feature and may duplicate work or avoid adding image-related functionality to the
  software path. It also obscures the fact that the old gap reported in the previous `GAPS.md`
  has been fixed, creating confusion about the project's actual state.
- **Closing the Gap**: Replace the "Software Rasterizer Limitations" subsection in
  `internal/raster/consumer/doc.go` with a description of what `renderDrawImage` does:
  > _"The SoftwareConsumer handles all `DisplayList` command types, including `CmdDrawImage`.
  > Image blitting uses bilinear scaling via `internal/raster/composite.BlitScaled`. If the
  > `DrawImageData.Src` field is nil (GPU-only path), the call is silently skipped."_
  Validate with `go doc github.com/opd-ai/wain/internal/raster/consumer`.

---

## GPU Rendering Pipeline Has No Hardware-Independent End-to-End Test

- **Stated Goal**: The README claims "GPU Command Submission — Intel i915/Xe and AMD RDNA batch
  command generation with DMA-BUF export" as a production feature. The CI pipeline runs the
  shader-to-ISA compilation gate (`render-sys/tests/shader_compile.rs`) unconditionally. Users
  and contributors expect the GPU rendering path to be regression-tested in CI.
- **Current State**: The shader→ISA compilation gate verifies that WGSL shaders parse and
  produce non-empty native ISA byte blobs. However, the path from a widget tree through the
  display list, GPU batch encoding, execbuffer submission, and DMA-BUF export to the compositor
  is only exercised when `/dev/dri/renderD128` is physically present (CI step
  `id: gpu-check`). Standard GitHub Actions runners have no DRM device, so every CI run skips
  the GPU integration job. `HARDWARE.md` correctly marks GPU rendering as "hardware-validated:
  manual only", but this gap is not surfaced in the README's feature table.
- **Impact**: GPU rendering regressions in batch encoding, pipeline state objects, DMA-BUF
  buffer layout, or the Go→Rust FFI boundary go undetected between hardware-validated releases.
  The shader-to-ISA gate covers compilation correctness but not the rendering pipeline
  orchestration in Go (`internal/render/backend/gpu.go`, `internal/render/display/`).
- **Closing the Gap**:
  1. Add `internal/integration/gpu_pipeline_test.go` with `//go:build integration`. The test
     should construct a `GPUBackend` with a mock `BufferAllocator` that returns a fixed-size
     heap allocation (using `primitives.NewBuffer`), render a minimal display list containing
     one `CmdFillRect`, and assert: (a) the returned batch byte slice is non-empty; (b) the
     first 4 bytes match the expected Intel MI command header or AMD PM4 header depending on
     detected GPU generation; (c) no error is returned.
  2. Wire `go test -tags integration ./internal/integration/...` into the CI `build-and-test`
     job unconditionally (it must not require hardware; the mock allocator must make it runnable
     in any environment).
  3. Update `README.md` to note that the hardware integration tests are manual-only, keeping
     the distinction visible to contributors.
