# AUDIT — 2026-03-14

## Project Goals

**Wain** (`github.com/opd-ai/wain`) is a statically-compiled Go UI toolkit for Linux that renders via a Rust GPU backend with automatic software fallback.  It implements the Wayland and X11 display protocols directly, producing fully static, zero-dependency binaries that run on any Linux distribution.

### Primary Audiences
- Go developers seeking a self-contained GUI binary for Linux desktop
- Embedded/appliance developers needing zero-dependency distribution
- GPU programming enthusiasts interested in direct Intel/AMD command submission without Mesa/Vulkan

### Stated Feature Claims (audit checklist)
1. Display-server auto-detection — Wayland preferred, X11 fallback
2. GPU renderer auto-detection — Intel Gen9–Xe → AMD RDNA 1–3 → software
3. Fully static binaries (zero runtime dependencies, musl + Rust staticlib)
4. Widget system — Button, Label, TextInput, ScrollView, ImageWidget, Spacer
5. Layout containers — Row, Column, Stack, Grid, Panel with flexbox-style alignment
6. Software rasterizer — rect, rounded-rect, anti-aliased lines, Bézier, gradients, shadows, SDF text
7. GPU command submission — Intel i915/Xe and AMD RDNA batch command generation with DMA-BUF export
8. Shader compilation — WGSL shaders compiled to Intel EU and AMD RDNA native ISA via naga
9. Wayland protocol — compositor, wl_shm, xdg_shell, input, clipboard, DMA-BUF, output
10. X11 protocol — connection, DRI3, Present, MIT-SHM, clipboard, drag-and-drop, HiDPI
11. AT-SPI2 Accessibility — D-Bus screen reader integration (requires `-tags=atspi`)
12. Theming — DefaultDark, DefaultLight, HighContrast built-in themes
13. Clipboard — read/write on both Wayland and X11
14. Animations — keyframe system with easing functions
15. Client-side decorations — title bar and resize handles
16. HiDPI support — automatic scale-factor detection on Wayland and X11
17. Double/triple buffering — frame synchronization with compositor
18. 60 FPS software rendering at 1920×1080 (≤16.7 ms/frame CI gate)

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Display-server auto-detection | ✅ Achieved | `app.go:209` — `WAYLAND_DISPLAY` → `DISPLAY` fallback; `cmd/auto-render-demo` |
| 2 | GPU renderer auto-detection | ✅ Achieved | `internal/render/backend/auto.go` — `render.DetectGPU` → Intel/AMD/software chain |
| 3 | Fully static binaries | ✅ Achieved | `Makefile` + CI `ldd` assertion; musl-gcc + `-extldflags '-static'` |
| 4 | Widget system | ⚠️ Partial | `concretewidgets.go` — all widgets exist; `ImageWidget.Draw` calls `Canvas.DrawImage`, which is silently skipped (`// Skip for now`) in the software rasterizer (`internal/raster/consumer/software.go:90–92`) |
| 5 | Layout containers | ✅ Achieved | `layout.go`, `internal/ui/layout/`; Row/Column/Stack/Grid/Panel with padding and gap |
| 6 | Software rasterizer | ✅ Achieved | `internal/raster/` (7 packages); benchmarks enforce ≤16.7 ms/frame |
| 7 | GPU command submission | ⚠️ Partial | `render-sys/src/batch.rs`, `pipeline.rs`, `submit.rs` exist; `GPUBackend.Render()` builds and submits batches. End-to-end UI→GPU→display path is hardware-gated in CI (`/dev/dri/renderD128` required) |
| 8 | Shader compilation | ✅ Achieved | `render-sys/src/shader.rs`; 7 WGSL shaders in `render-sys/shaders/`; CI gate `cargo test --test shader_compile` verifies non-empty ISA output without hardware |
| 9 | Wayland protocol | ✅ Achieved | `internal/wayland/` (9 packages); `go test ./internal/wayland/...` passes |
| 10 | X11 protocol | ✅ Achieved | `internal/x11/` (10 packages); `go test ./internal/x11/...` passes |
| 11 | AT-SPI2 accessibility | ⚠️ Partial | `internal/a11y/` (10 files, 75 functions) compiles and has unit tests; however `go test ./internal/a11y/` reports **0.0% coverage** — the manager tests exercise struct setters but no D-Bus paths run without a live session bus |
| 12 | Theming | ✅ Achieved | `theme.go`; DefaultDark, DefaultLight, HighContrast; `theme_test.go` |
| 13 | Clipboard | ✅ Achieved | `clipboard.go`, `internal/wayland/datadevice/`, `internal/x11/selection/`; tests pass |
| 14 | Animations | ✅ Achieved | `animate.go` + `internal/ui/animation/`; 97.4% test coverage |
| 15 | Client-side decorations | ✅ Achieved | `internal/ui/decorations/`; title bar, controls, resize handles; 85.7% coverage |
| 16 | HiDPI support | ✅ Achieved | `internal/ui/scale/` (100%), `internal/x11/dpi/` (100%) |
| 17 | Double/triple buffering | ✅ Achieved | `internal/buffer/ring.go`; no panics; 93.3% test coverage |
| 18 | 60 FPS software rendering | ✅ Achieved | CI `benchmarks` job asserts `BenchmarkFillRectOpaque1080p ≤ 16700000 ns/op`; `cmd/bench -frames 60 -max 16` |

**Summary: 15/18 goals fully achieved (83%); 3 partially achieved**

---

## Findings

### CRITICAL

_No CRITICAL findings._ All tests pass under `go test -race ./...`, `go vet` produces no warnings, and no data-corruption paths were identified.

### HIGH

- [x] **ImageWidget renders nothing in software mode** — `concretewidgets.go:606`, `internal/raster/consumer/software.go:90–92` — `ImageWidget.Draw` calls `Canvas.DrawImage`, which emits a `CmdDrawImage` display-list command. The `SoftwareConsumer.renderCommand` switch contains `case displaylist.CmdDrawImage: // Skip for now — GPU backend handles this`. Any application running in software-fallback mode (no GPU, or `ForceSoftware: true`) will silently display a blank area wherever an `ImageWidget` is placed. The README claims "ImageWidget" as a first-class widget with no caveats for the software path. **Remediation:** Implement software-path image drawing in `internal/raster/consumer/software.go`. Use `internal/raster/composite.BlitScaled` (already imported in related files) to blit the image's `image.RGBA` pixels into the `primitives.Buffer`. Add a `SoftwareConsumer.renderDrawImage` helper mirroring the existing `renderDrawText` pattern. Validate with: `go test -run TestImageWidgetSoftware ./... -count=1`.

- [x] **`internal/a11y` has 0.0% effective test coverage** — `internal/a11y/` — `go test ./internal/a11y/` reports `coverage: 0.0% of statements`. Test files (`manager_test.go`, `a11y_test.go`) exist but they only exercise struct-field setters. The AT-SPI2 D-Bus interface code (`accessible_iface.go`, `component_iface.go`, `action_iface.go`, `text_iface.go`, `manager.go`) is completely untested. A regression in any D-Bus method handler goes undetected. **Remediation:** Add mock-D-Bus tests using `github.com/godbus/dbus/v5` session bus in test mode. At minimum, test that `EnableAccessibility` returns a non-nil manager under `-tags=atspi`, and that `RegisterButton`/`RegisterLabel` export objects with the expected D-Bus paths. Validate with: `go test -tags=atspi -cover ./internal/a11y/ | grep coverage` — target ≥ 50%.

- [x] **`processWaylandDragEvents` exceeds complexity threshold** — `app.go:2090` — cyclomatic complexity 20, overall score 27.5 (only function in the codebase above CC=15). The function chains four independent `select`/`default` blocks covering enter/move/leave/drop events, with nested nil-checks and MIME negotiation. Although functionally correct, the high complexity raises regression risk during future maintenance. **Remediation:** Extract each drag-event branch into a dedicated helper: `handleWaylandDragEnter`, `handleWaylandDragMotion`, `handleWaylandDragLeave`, `handleWaylandDrop`. `processWaylandDragEvents` becomes a four-line dispatcher. Validate with: `go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.name=="processWaylandDragEvents")] | .[].complexity.cyclomatic'` — target <10.

### MEDIUM

- [x] **`internal/render/display` has 1.9% test coverage** — `internal/render/display/` — `WaylandPipeline`, `X11Pipeline`, and the software presenter paths have almost no test coverage. These are on the critical frame-delivery path (UI→render→display→compositor). A regression in `renderToFramebuffer`, `ensureWaylandBuffer`, or `presentBuffer` would be invisible to CI unless `/dev/dri/renderD128` is present. **Remediation:** Add unit tests for `renderToFramebuffer` using a mock `Renderer` that returns a fixed DMA-BUF fd, and for `ensureWaylandBuffer`/`PresentBuffer` using a fake `wl_shm` pool. Validate with: `go test -cover ./internal/render/display/ | grep coverage` — target ≥ 50%.

- [x] **`internal/render/present` has 0.0% coverage and no test files** — `internal/render/present/` — `present.go` contains `RenderAndPresent`, a core orchestration function for the frame pipeline, with cyclomatic complexity 7. Zero tests exist. **Remediation:** Add `present_test.go` exercising `RenderAndPresent` with mock `FramebufferPool` and `PlatformPresenter` implementations that record calls. Validate with: `go test -cover ./internal/render/present/` — target ≥ 70%.

- [ ] **`internal/render/atlas` has only 27.0% test coverage** — `internal/render/atlas/` — The GPU texture atlas (font SDF + image LRU) is a cache with eviction logic; at 27% coverage, the LRU eviction path and atlas compaction are untested. Bugs here produce invisible text or wrong images in GPU mode. **Remediation:** Add tests covering (1) eviction triggered when atlas capacity is exceeded, (2) re-insertion of an evicted glyph, (3) atlas reset. Validate with: `go test -cover ./internal/render/atlas/` — target ≥ 70%.

- [ ] **`internal/wayland/input` has only 25.0% test coverage** — `internal/wayland/input/` — Input handling (pointer, keyboard, touch) is central to interactive applications but only a quarter of the code is tested. Edge cases in `handleKeyEvent` or `handleEnterEvent` are untested. **Remediation:** Add tests using fake Wayland wire messages for pointer button, axis, keyboard key-press/release, and touch-down/up sequences. Validate with: `go test -cover ./internal/wayland/input/` — target ≥ 60%.

- [ ] **`AcquireForWriting` uses a polling loop with `time.After`** — `internal/buffer/ring.go:160–175` — The buffer-slot acquisition loop polls every 5 ms using `time.After` instead of a condition variable or channel notification. Under compositor back-pressure this wastes CPU and adds up to 5 ms of latency per frame. **Remediation:** Replace the `time.After` poll with a `sync.Cond` or an additional channel in `Slot` that the `MarkReleased` path signals. Validate with: existing buffer tests + `go test -bench=BenchmarkAcquire ./internal/buffer/` to confirm throughput improves.

### LOW

- [ ] **`go.sum` contains stale transitive entries for `golang.org/x/sys`** — `go.sum` — The sum database contains entries for `golang.org/x/sys v0.20.0` and `v0.42.0` which are not referenced by `go.mod` (which pins `v0.27.0`). This is a hygiene issue; stale entries do not affect builds but can confuse auditors and `go mod verify`. **Remediation:** Run `go mod tidy` to remove unused entries. Validate with: `go mod verify`.

- [ ] **`internal/render/backend/gpu.go`: `FontAtlas` field is documented "optional"** — `internal/render/backend/gpu.go:76` — When `FontAtlas` is `nil`, text commands in GPU mode produce no visible output, but no error is returned and no log warning is emitted. Silent failures are hard to diagnose. **Remediation:** In `GPUBackend.RenderWithDamage`, when a `CmdDrawText` command is encountered with a nil atlas, log a one-time `log.Printf("wain/backend: font atlas not set — text rendering disabled")` warning. Validate with: add a test that passes a nil atlas, calls `Render` with a `CmdDrawText` command, and asserts the warning is logged.

- [ ] **`cmd/` README claims 22 additional demo binaries; actual count is 20** — `README.md:337` — The README states "22 additional demo and tool binaries" under `cmd/`. `go list ./cmd/...` returns 20 packages (22 total including `cmd/wain` and `cmd/wain-build`). The off-by-two discrepancy is cosmetic but may confuse users looking for missing demos. **Remediation:** Update README.md line 337 to read "18 additional demo and tool binaries" (20 total minus `wain` and `wain-build`).

---

## Metrics Snapshot

| Metric | Value | Source |
|--------|-------|--------|
| Total Go LOC | 14,786 | go-stats-generator |
| Total Rust LOC (render-sys) | ~15,100 | wc estimate |
| Total Go functions | 667 | go-stats-generator |
| Total Go methods | 1,169 | go-stats-generator |
| Total packages | 40 (non-cmd) | go-stats-generator |
| Average function length | 9.5 lines | go-stats-generator |
| Functions > 50 lines | 8 (0.4%) | go-stats-generator |
| Functions > 100 lines | 0 (0.0%) | go-stats-generator |
| Average cyclomatic complexity | 3.2 | go-stats-generator |
| Functions with CC > 15 | 1 (`processWaylandDragEvents`, CC=20) | go-stats-generator |
| `go vet` warnings | 0 | `go vet ./...` |
| `go test -race` | All pass | `go test -race ./...` |
| Root package coverage | 60.0% | `go test -cover .` |
| Lowest internal package coverage | `internal/a11y`: 0.0% | `go test -cover ./internal/...` |
| Average internal package coverage | ~74% | `go test -cover ./internal/...` |
| Duplication ratio | Low (no clone groups flagged) | go-stats-generator |
| Circular dependencies | 0 | go-stats-generator |
| Rust `unwrap()` in production code | 2 (`expect("valid layout…")` in lib.rs) | grep render-sys/src/lib.rs |

### Package Coverage Detail (internal packages only)

| Package | Coverage |
|---------|----------|
| `internal/x11/dpi` | 100.0% |
| `internal/ui/scale` | 100.0% |
| `internal/ui/animation` | 97.4% |
| `internal/x11/present` | 96.6% |
| `internal/raster/curves` | 94.6% |
| `internal/raster/composite` | 93.6% |
| `internal/ui/layout` | 93.4% |
| `internal/buffer` | 93.3% |
| `internal/raster/text` | 92.3% |
| `internal/ui/pctwidget` | 91.9% |
| `internal/raster/effects` | 88.2% |
| `internal/x11/dnd` | 88.3% |
| `internal/wayland/output` | 87.2% |
| `internal/raster/primitives` | 87.8% |
| `internal/ui/decorations` | 85.7% |
| `internal/raster/consumer` | 85.3% |
| `internal/x11/selection` | 85.5% |
| `internal/x11/events` | 80.6% |
| `internal/x11/shm` | 79.8% |
| `internal/wayland/shm` | 76.7% |
| `internal/wayland/xdg` | 75.2% |
| `internal/ui/widgets` | 55.2% |
| `internal/render/backend` | 53.7% |
| `internal/render` | 44.0% |
| `internal/wayland/wire` | 41.9% |
| `internal/wayland/datadevice` | 41.2% |
| `internal/x11/client` | 42.5% |
| `internal/render/atlas` | 27.0% |
| `internal/x11/wire` | 27.1% |
| `internal/x11/dri3` | 26.0% |
| `internal/wayland/input` | 25.0% |
| `internal/render/display` | 1.9% |
| `internal/a11y` | 0.0% |
| `internal/render/present` | 0.0% (no test files) |
