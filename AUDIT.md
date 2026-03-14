# AUDIT — 2026-03-14

## Project Goals

Wain is a **statically-compiled Go UI toolkit** for Linux that renders via a Rust GPU backend
with automatic software fallback. It implements the Wayland and X11 display protocols directly,
targeting Go developers who need zero-dependency Linux GUI binaries, embedded/appliance developers,
and GPU programming enthusiasts interested in direct Intel/AMD command submission.

### Stated Goals Checklist

The following were extracted from `README.md`, `ROADMAP.md`, `STABILITY.md`, `HARDWARE.md`, and
`CHANGELOG.md`:

1. Display server auto-detection (Wayland → X11)
2. GPU renderer auto-detection (Intel → AMD → software fallback)
3. Fully static binaries (zero runtime dependencies via musl + Rust staticlib)
4. Widget system: Button, Label, TextInput, ScrollView, ImageWidget, Spacer
5. Layout containers: Row, Column, Stack, Grid, Panel with flexbox-style alignment
6. Software rasterizer: rectangles, rounded rects, anti-aliased lines, Bézier curves,
   gradients, shadows, SDF text
7. GPU command submission (Intel i915/Xe and AMD RDNA batch command generation with DMA-BUF)
8. Shader compilation (WGSL → Intel EU ISA and AMD RDNA ISA via naga)
9. Wayland protocol (compositor, wl_shm, xdg_shell, input, clipboard, DMA-BUF, output)
10. X11 protocol (connection, windows, DRI3, Present, MIT-SHM, clipboard, DnD, HiDPI)
11. AT-SPI2 accessibility (D-Bus screen reader integration via `-tags=atspi`)
12. Theming: DefaultDark, DefaultLight, HighContrast
13. Clipboard read/write on Wayland and X11
14. Keyframe animation system with easing functions
15. Client-side decorations (title bar, resize handles)
16. HiDPI support (automatic scale factor detection)
17. Double/triple buffering with frame synchronisation
18. 60 FPS software rendering (≤16.7 ms/frame at 1920×1080, CI-enforced)
19. Public API stability guarantee (SemVer, `STABILITY.md`)
20. `Canvas.LinearGradient` with angle-based direction control

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Display server auto-detection | ✅ Achieved | `app.go:1427–1448`; tries `WAYLAND_DISPLAY`, falls back to `DISPLAY` |
| GPU renderer auto-detection | ✅ Achieved | `internal/render/backend/backend.go`; Intel→AMD→software chain |
| Fully static binaries | ✅ Achieved | Makefile enforces `-extldflags '-static'`; CI verifies with `ldd` |
| Widget system (6 widgets) | ✅ Achieved | `concretewidgets.go`; Button, Label, TextInput, ScrollView, ImageWidget, Spacer all present |
| Layout containers (5 types) | ✅ Achieved | `layout.go`; Row, Column, Stack, Grid, Panel with flexbox alignment |
| Software rasterizer (7 sub-packages) | ✅ Achieved | `internal/raster/`: primitives, curves, composite, effects, text, displaylist, consumer |
| GPU command submission | ✅ Achieved | `render-sys/src/batch.rs`, `pipeline.rs`, `cmd/`; Intel Gen9–12 and AMD RDNA batches |
| Shader compilation (WGSL→ISA) | ✅ Achieved | `render-sys/tests/shader_compile.rs` CI gate; 7 WGSL shaders → EU+RDNA binary blobs |
| Wayland protocol (9 packages) | ✅ Achieved | `internal/wayland/`: wire, socket, client, shm, xdg, input, dmabuf, datadevice, output |
| X11 protocol (10 packages) | ✅ Achieved | `internal/x11/`: wire, client, events, gc, shm, dri3, present, dpi, selection, dnd |
| AT-SPI2 accessibility | ✅ Achieved | `internal/a11y/`; real D-Bus with `-tags=atspi`, no-op stub otherwise; documented |
| Theming (3 built-in themes) | ✅ Achieved | `theme.go`; DefaultDark, DefaultLight, HighContrast exported |
| Clipboard (Wayland + X11) | ✅ Achieved | `clipboard.go`, `internal/wayland/datadevice/`, `internal/x11/selection/` |
| Animation system | ✅ Achieved | `animate.go`, `internal/ui/animation/`; 5 easing functions, `Animator.Tick` frame-driven |
| Client-side decorations | ✅ Achieved | `internal/ui/decorations/`; title bar, resize handles, pointer enter/leave |
| HiDPI support | ✅ Achieved | `internal/ui/scale/`, `internal/x11/dpi/`; 100% test coverage |
| Double/triple buffering | ✅ Achieved | `internal/buffer/`; ring buffer with compositor sync |
| 60 FPS software rendering | ✅ Achieved | `cmd/bench`; CI gate ≤16.7 ms/frame at 1920×1080 |
| Public API stability | ✅ Achieved | `STABILITY.md`; SemVer; `compat_test.go` compile-time assertions |
| Canvas.LinearGradient angle control | ❌ Missing | `publicwidget.go:270–278`; documented as "0=left-to-right, 90=top-to-bottom" but `angle` parameter is silently ignored in both Canvas implementations |

**Overall: 19/20 goals achieved (95%). 1 goal documented but not implemented.**

---

## Findings

### CRITICAL

_None identified._ All tests pass under `go test -race ./...` and `go vet ./...`.

### HIGH

- [x] **GPU pipeline has no hardware-independent end-to-end integration test** — `.github/workflows/ci.yml` (gpu-check step) — The shader→ISA CI gate (`render-sys/tests/shader_compile.rs`) validates individual compilation passes. However, the full GPU rendering pipeline (widget tree → display list → GPU batch → execbuffer/amdgpu CS → DMA-BUF → compositor) is only tested when `/dev/dri/renderD128` is physically present, which is never true on standard GitHub Actions runners. GPU rendering regressions (wrong batch encoding, broken DMA-BUF handoff, pipeline state corruption) therefore go undetected in standard CI runs. The `HARDWARE.md` correctly labels this "hardware-validated: manual only" but the README feature table presents GPU rendering as fully production-ready without this caveat. — **Remediation:** Add `internal/integration/gpu_pipeline_test.go` with a `//go:build integration` tag. Use the existing `internal/render/backend.NewSoftwareBackend` as a reference for how to wire a mock allocator; create a `GPUBackend` backed by an in-memory `BufferAllocator` stub, render a minimal display list (one `CmdFillRect`), and assert that the returned batch byte slice is non-empty and starts with the correct Intel MI_BATCH_BUFFER_START header (`0x31000000` or equivalent). Wire this test unconditionally into CI. Validate with: `go test -tags integration ./internal/integration/... -run TestGPUPipelineEndToEnd`.

### MEDIUM

- [x] **`Canvas.LinearGradient` silently ignores the `angle` parameter** — `publicwidget.go:270–278`, `concretewidgets.go:142–150` — The `Canvas` interface (line 111) documents `angle` as "degrees (0 = left-to-right, 90 = top-to-bottom)". Both concrete implementations (`displayListCanvas.LinearGradient` and `bufferCanvas.LinearGradient`) discard the angle with a comment "For simplicity, assume 0 degrees = horizontal left-to-right". Any call with `angle != 0` renders an incorrect horizontal gradient without warning. — **Remediation:** Implement angle→vector conversion using `math.Sin`/`math.Cos` in both `displayListCanvas.LinearGradient` and `bufferCanvas.LinearGradient`. For `displayListCanvas`, convert `angle` (degrees) to a unit direction vector `(cos θ, sin θ)`, then compute `x0 = cx - w/2*cos(θ)`, `y0 = cy - h/2*sin(θ)`, `x1 = cx + w/2*cos(θ)`, `y1 = cy + h/2*sin(θ)`. For `bufferCanvas`, apply the same calculation before calling `effects.LinearGradient`. Add a test `TestLinearGradientAngle90` in `publicwidget_test.go` that renders a 4×4 grid at angle=90 and asserts the top-row pixels are `startColor` and bottom-row pixels are `endColor`. Validate with: `go test ./... -run TestLinearGradientAngle`.

- [x] **`SetOpacity` used in public documentation but not implemented on any widget** — `animate.go:46`, `internal/ui/animation/animation.go:20` — Both files use `widget.SetOpacity(v)` as the canonical example for `App.Animate` and `Animator.Add`. The method does not exist on any type in the codebase (`grep -rn "func.*SetOpacity"` returns no matches). Users who copy the documented example will receive a compile error. — **Remediation:** Add `SetOpacity(alpha float64)` to the `PublicWidget` interface in `publicwidget.go`, implement it on `BasePublicWidget` (store an `opacity float64` field defaulting to 1.0), and pass the opacity to `displayListCanvas` draw calls via a multiplied alpha in `FillRect`/`FillRoundedRect`. Alternatively (less API surface), update both example code comments to use a real, existing method (e.g., `label.SetText(fmt.Sprintf("%.0f%%", v*100))`) as the OnTick example. Validate with: `go test ./... -run TestAnimate` and a manual build of `example/hello` that compiles without error.

- [ ] **`internal/raster/consumer/doc.go` contradicts the actual implementation** — `internal/raster/consumer/doc.go:13–15` — The package-level comment states: _"The SoftwareConsumer does not implement the CmdDrawImage display list command. [...] DrawImage command execution requires a GPU backend."_ This was accurate before TD-2 was resolved, but `software.go` now contains a working `renderDrawImage` function that bilinear-scales images into the destination buffer via `composite.BlitScaled`. The stale comment will mislead contributors and users into thinking `ImageWidget` is GPU-only. — **Remediation:** Delete lines 13–17 of `internal/raster/consumer/doc.go` (the "Software Rasterizer Limitations" subsection). Replace with: _"The SoftwareConsumer implements all `DisplayList` command types, including `CmdDrawImage`, using bilinear scaling from `internal/raster/composite`."_ Validate with: `go doc github.com/opd-ai/wain/internal/raster/consumer` and confirm the outdated paragraph is absent.

### LOW

- [ ] **12 clone pairs (0.69% duplication) concentrated in `cmd/` demo binaries** — `cmd/amd-triangle-demo/main.go:52`, `cmd/gpu-triangle-demo/main.go:229` — `go-stats-generator` reports 12 clone pairs totalling 238 duplicated lines, with the two highest-ROI duplications (25-line and 24-line blocks) in demo binaries. Duplication in demo binaries does not affect the library API but increases maintenance burden when shared logic changes. — **Remediation:** Extract the duplicated demo setup block (GPU context creation, error handling, display loop) into `internal/demo/gpu_setup.go` as a `NewGPUTriangleSetup(devicePath string) (*GpuSetup, error)` helper. Update both demo binaries to call the shared helper. Validate with: `go-stats-generator analyze . --sections duplication | grep "Duplication ratio"` confirming the ratio drops below 0.5%.

- [ ] **`internal/render/stats.go:GetMemoryStats` placement suggestion** — `internal/render/stats.go` — `go-stats-generator` flags `GetMemoryStats` as misplaced: its sole caller is `cmd/perf-demo/main.go`, suggesting it should either move to that binary or be more clearly documented as a monitoring API. — **Remediation:** If `GetMemoryStats` is intended as a stable monitoring API, add a godoc example in `internal/render/stats.go` and export it via the public `wain` package. If it is demo-only, move it inline to `cmd/perf-demo/main.go` and remove it from `internal/render`. Validate with: `go vet ./...` after the move.

---

## Metrics Snapshot

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Go files (non-test) | 199 | Substantial codebase |
| Total Go LoC (non-test) | 14,840 | Lean for the feature set |
| Total functions | 668 | — |
| Total methods | 1,174 | — |
| Total structs | 248 | — |
| Total interfaces | 34 | — |
| Total packages | 40 (internal); 74 (incl. cmd/) | Well-modularised |
| Average function length | 9.5 lines | Well below 30-line target |
| Functions > 50 lines | 7 (0.4%) | Only 7; none > 100 lines |
| Highest cyclomatic complexity | 13.7 (`packVertices`, `backend`) | Well below 15 threshold |
| Average cyclomatic complexity | 3.2 | Excellent |
| Documentation coverage (overall) | 91.16% | Above 90% target |
| Documentation coverage (functions) | 98.3% | Excellent |
| Documentation coverage (methods) | 89.3% | Above 70% threshold |
| Duplication ratio | 0.69% (12 clone pairs, 238 lines) | Very low |
| `go test -race ./...` | ✅ All pass | No race conditions |
| `go vet ./...` | ✅ No warnings | Clean |
| Rust `cargo test` | ✅ All pass (incl. shader_compile) | — |

### High-Complexity Functions (top 5)

| Rank | Function | Package | Lines | Overall Score |
|------|----------|---------|-------|---------------|
| 1 | `packVertices` | `backend` | 44 | 13.7 |
| 2 | `Render` | `wain` | 38 | 9.6 |
| 3 | `DecodeSetupReply` | `wire` | 31 | 9.6 |
| 4 | `RenderAndPresent` | `present` | 30 | 9.6 |
| 5 | `Present` | `display` | 27 | 9.6 |

No function exceeds the cyclomatic complexity threshold of 15.

### Oversized Files (top 3 by burden score)

| File | Lines | Burden |
|------|-------|--------|
| `app.go` | 1,603 | 3.89 |
| `event.go` | 348 | 2.65 |
| `internal/ui/widgets/widget_impl.go` | 634 | 2.46 |

`app.go` at 1,603 lines is large but its complexity is low (avg 3.2); it aggregates display-server
and renderer initialisation that is inherently broad. No refactoring is flagged as necessary unless
a specific function exceeds the per-function thresholds.

---

## Environment

| Item | Value |
|------|-------|
| Analysis date | 2026-03-14 |
| Go version | 1.24 (from `go.mod`) |
| go-stats-generator | v1.0.0 |
| Analysis time | 969 ms |
| Files processed | 199 |
