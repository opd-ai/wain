# AUDIT — 2026-03-14

## Project Goals

Wain is a **statically-compiled Go UI toolkit** targeting Linux desktop and
embedded developers who need a self-contained binary with zero runtime
dependencies. The README and supporting documentation make the following
concrete commitments:

| # | Stated Goal / Claim | Source |
|---|---------------------|--------|
| G1 | Go–Rust static linking via CGO/musl; single binary, no dynamic deps | README Features |
| G2 | Wayland client — 9 packages, wire format through DMA-BUF and clipboard | README Features |
| G3 | X11 client — 9 packages, connection setup through DRI3/Present/clipboard | README Features |
| G4 | Software 2D rasterizer — fill, curves, SDF text, shadows, gradients, compositing | README Features |
| G5 | UI widget layer — flexbox-like Row/Column, Button/TextInput/Scroll, decorations, HiDPI | README Features |
| G6 | GPU buffer infrastructure — DRM/KMS ioctls, tiling, DMA-BUF export, slab allocator | README Features |
| G7 | GPU command submission — batch buffers, 3D pipeline state, Intel + AMD paths | README Features |
| G8 | Shader frontend — WGSL/GLSL via naga, 7 bundled UI shaders | README Features |
| G9 | Intel EU backend — Gen9/11/12/Xe register allocation and binary encoding | README Features |
| G10 | AMD RDNA backend — RDNA1/2/3 instruction set, PM4 command stream | README Features |
| G11 | Display list rendering — GPU backend consumes display lists, texture atlas, frame presentation | README Features |
| G12 | Public API — `App`, `Window`, `Widget` with display-server and renderer auto-detection | README Usage |
| G13 | Accessibility — keyboard navigation (Tab/Shift-Tab, Enter/Space, arrows) | ACCESSIBILITY.md |
| G14 | AT-SPI2 screen-reader support | README / ACCESSIBILITY.md |
| G15 | Clipboard — Wayland data-device and X11 selection | README Features |
| G16 | HiDPI / DPI-aware scaling | README Features |
| G17 | `<2 ms` GPU frame time for typical UI workload | HARDWARE.md |
| G18 | 60 FPS software rendering at 1080p (≤16 ms frame time) | HARDWARE.md |
| G19 | `go generate ./...` → `go build` workflow with no extra env vars | README Usage |
| G20 | Fully static binary verified by CI (`ldd` check) | README / CI |

**Target audience:** developers building self-contained Linux GUI applications
and applications targeting embedded Linux, containers, or minimal distributions.

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| G1 — Go–Rust static linking | ✅ Achieved | `render-sys/Cargo.toml` `crate-type=["staticlib"]`; CI `ldd` check passes; `scripts/verify-build.sh` |
| G2 — Wayland client (9 pkg) | ✅ Achieved | `internal/wayland/{wire,socket,client,shm,xdg,input,dmabuf,datadevice,output}` fully tested |
| G3 — X11 client (9 pkg) | ✅ Achieved | `internal/x11/{wire,client,events,gc,shm,dri3,present,dpi,selection}` fully tested |
| G4 — Software 2D rasterizer | ✅ Achieved | `internal/raster/{primitives,curves,composite,effects,text,displaylist,consumer}` all tested |
| G5 — UI widget layer | ✅ Achieved | `internal/ui/{layout,widgets,decorations,scale,pctwidget}` all tested |
| G6 — GPU buffer infrastructure | ✅ Achieved | `render-sys/src/{allocator,slab,drm,i915,xe}.rs`; Go bindings in `internal/render/binding.go` |
| G7 — GPU command submission | ⚠️ Partial | `render-sys/src/{batch,cmd/,pipeline,surface}.rs` encode commands; UI path exercises software fallback only (see F1) |
| G8 — Shader frontend | ✅ Achieved | `render-sys/src/shader.rs` + 7 WGSL shaders; validated by `cmd/shader-test` |
| G9 — Intel EU backend | ✅ Achieved | `render-sys/src/eu/` — ~4,807 lines; regalloc, lower, encode, types |
| G10 — AMD RDNA backend | ✅ Achieved | `render-sys/src/rdna/` + `amd.rs` + `pm4.rs`; regalloc and encoding |
| G11 — Display list rendering | ⚠️ Partial | `internal/render/display/` (`WaylandPipeline`, `X11Pipeline`) exists and is tested; **not wired into `App`** (see F1) |
| G12 — Public API auto-detection | ⚠️ Partial | Display-server and renderer detection work; window renders internally but **pixels never presented to compositor** (see F1) |
| G13 — Keyboard accessibility | ✅ Achieved | `accessibility_test.go` passes; `internal/a11y` package |
| G14 — AT-SPI2 screen reader | ❌ Not Implemented | `ACCESSIBILITY.md` explicitly states "not yet implemented"; no D-Bus objects exported |
| G15 — Clipboard | ✅ Achieved | `clipboard.go`, `internal/wayland/datadevice`, `internal/x11/selection`; clipboard_test.go passes |
| G16 — HiDPI scaling | ✅ Achieved | `internal/ui/scale`, `internal/x11/dpi`, `Theme.Scale` field |
| G17 — <2 ms GPU frame time | ⚠️ Unverified | HARDWARE.md states measured 0.3–1.5 ms but no CI benchmark; GPU path not exercised by public API |
| G18 — 60 FPS software @ 1080p | ⚠️ Unverified | HARDWARE.md states ≤12 ms; no CI benchmark; SIMD explicitly not implemented |
| G19 — `go generate` workflow | ✅ Achieved | `internal/render/generate.go` drives `scripts/build-rust.sh` and writes `cgo_flags_generated.go` |
| G20 — Fully static binary in CI | ✅ Achieved | `.github/workflows/ci.yml` runs `ldd bin/wain` and exits 1 if dynamically linked |

**Summary: 13/20 fully achieved · 4 partial · 1 not implemented · 2 unverified**

---

## Findings

### CRITICAL

- [x] **Public API window renders to internal buffers but never presents pixels to compositor** — `app.go` — The `App` event loop calls `win.RenderFrame()` → `RenderBridge.Render()` → `renderer.Render(dl)`, which fills the GPU or software buffer, but **`Present()` is never called** and neither `WaylandPipeline` nor `X11Pipeline` (from `internal/render/display`) are wired into the `App` type. The Wayland surface receives one empty `wl_surface.commit` during initialization (`app.go:380`) but no buffer is ever attached; for X11, no `CopyArea` / DRI3 pixmap path exists in the event loop. A real app using the public API would open a window with a blank surface regardless of widgets added. The `cmd/gpu-display-demo` and `cmd/wayland-demo` work because they bypass the public API and call `WaylandPipeline.RenderAndPresent()` or `demo.AttachAndDisplayBuffer()` directly.
  - **Evidence:** `app.go:1912-1913` (`renderFrames` → `RenderFrame`); `app.go:925-939` (`RenderFrame` does not call `Present`); `render.go:107-114` (`RenderBridge.Present()` exists but is never called from the event loop); `internal/render/display/wayland.go:44` and `x11.go:98` (pipeline constructors unreferenced in `app.go`).
  - **Remediation:** In `initWaylandWindow` / `initX11Window`, create a `WaylandPipeline` or `X11Pipeline` and store it on `Window`. Replace the `renderBridge.Render()` call in `RenderFrame()` with `pipeline.RenderAndPresent(ctx, displayList)`. For the software path use `internal/wayland/shm` + `demo.AttachAndDisplayBuffer` or a dedicated `SoftwareWaylandPresenter`. Validate with: `./bin/wain-demo` showing a visible button; pixel colour must change when the button is hovered.

### HIGH

- [x] **`cmd/widget-demo` event loops are stubs** — `cmd/widget-demo/main.go:333,366` — Both `runWayland()` and `runX11()` print `⚠ Wayland/X11 event loop not yet implemented` and return immediately without displaying anything. The demo is listed in the README under "Demonstration Binaries". A user following the README would run it and see only console output.
  - **Evidence:** `cmd/widget-demo/main.go:329-340` (`runWayland` stub); `cmd/widget-demo/main.go:355-378` (`runX11` stub with simulated interactions only).
  - **Remediation:** Implement `runWayland` and `runX11` using the same `WaylandPipeline`/`X11Pipeline` pattern used by `cmd/gpu-display-demo`. Validate with: `make widget-demo && ./bin/widget-demo` opens a window with visible widgets.

- [x] **Software backend has no compositor presentation path** — `internal/render/backend/software.go:108-112` — `SoftwareBackend.Present()` always returns `ErrSoftwareNoDmabuf` (fd=-1). The display pipeline (`WaylandPipeline.renderToFramebuffer`) calls `renderer.Present()` expecting a DMA-BUF fd; calling it with a software renderer will always fail. There is no SHM-based software presenter that the display pipeline can fall back to, so the documented "automatic software fallback" cannot present frames when DMA-BUF is unavailable.
  - **Evidence:** `internal/render/backend/software.go:108-112`; `internal/render/display/wayland.go:143` (`fd, err := p.renderer.Present()`); no SHM presenter in `internal/render/display/`.
  - **Remediation:** Add a `SoftwareSHMPresenter` in `internal/render/display/` that obtains pixels via `SoftwareBackend.Pixels()`, writes them into a `wl_shm` pool buffer, and commits it to the surface. The `NewWaylandPipeline` should accept either a `GPUBackend` or a `SoftwareBackend`, selecting the presentation strategy automatically. Validate with: `go test ./internal/render/display/...` including a new `TestSoftwarePresent` test.

- [ ] **SIMD optimizations not implemented in software rasterizer** — `internal/raster/primitives/` — HARDWARE.md explicitly states "SIMD optimizations (AVX2/NEON) are not yet implemented. Performance could improve 2–4×." The software path is the always-available fallback and the primary path for users without compatible Intel/AMD GPUs. Without SIMD, the claimed 60 FPS at 1080p relies on benchmarks from a specific Intel i7; on lower-end or embedded hardware the target will not be met.
  - **Evidence:** `HARDWARE.md:290-292`; no `#[target_feature(enable="avx2")]` or `std::arch::x86_64` usage in `internal/raster/`.
  - **Remediation:** Add AVX2 hot paths for `FillRect` and `FillRoundedRect` in `internal/raster/primitives/rect.go` using `unsafe` + `golang.org/x/sys/cpu` for runtime detection. Validate with: new `BenchmarkFillRect` showing ≥2× throughput improvement on AVX2-capable hardware.

- [ ] **Performance targets unverified in CI** — `.github/workflows/ci.yml` — The CI pipeline has no benchmark job. The claimed `<2 ms` GPU and `≤16 ms` software frame-time targets (HARDWARE.md) can silently regress.
  - **Evidence:** `.github/workflows/ci.yml` (no benchmark step); `cmd/perf-demo/main.go` exists but is not run in CI; HARDWARE.md claims are based on a single ad-hoc measurement.
  - **Remediation:** Add a CI job that runs `go test -bench=. -benchtime=5s ./internal/raster/...` and fails if software raster throughput falls below a baseline (no GPU required in CI). Add `cmd/perf-demo` as a smoke test (`-quick` flag) to catch panics in the benchmark infrastructure. Validate with: CI step green on `main` after establishing a baseline.

### MEDIUM

- [ ] **AT-SPI2 screen-reader support is absent but publicly documented** — `ACCESSIBILITY.md` — The README lists keyboard accessibility as a feature; ACCESSIBILITY.md acknowledges AT-SPI2 is "not yet implemented" but documents it as a future path. Applications cannot be used with Orca or other AT-SPI2 tools. `github.com/godbus/dbus/v5` is already an indirect dependency (`go.mod:3`), so there is no additional build dependency to introduce.
  - **Evidence:** `ACCESSIBILITY.md:14-19`; `internal/a11y/` contains only keyboard-focus helpers, no D-Bus objects; `go.mod:3` (`godbus/dbus/v5 v5.1.0 // indirect`).
  - **Remediation:** Implement `internal/a11y/atspi` with `Accessible`, `Component`, and `Action` AT-SPI2 interfaces backed by `godbus/dbus/v5`. Expose focus changes from `EventDispatcher.Dispatch` as D-Bus signals. Validate with: `go test ./internal/a11y/atspi/...` and manual Orca announcement of button labels.

- [ ] **`TODO` in `internal/wayland/dmabuf/protocol.go:128`** — `internal/wayland/dmabuf/protocol.go` — A tracked `TODO` notes that version-2 fallback code should be removed once the minimum compositor version is raised to v3. This is a known stale annotation but creates a maintenance burden.
  - **Evidence:** `internal/wayland/dmabuf/protocol.go:128` (`// TODO: Once the minimum required compositor version is raised to v3, remove this`).
  - **Remediation:** Determine the minimum compositor version the project targets. If v3 is already ubiquitous (wlroots ≥0.16, GNOME ≥42, KDE ≥5.25), remove the v2 fallback branch and the TODO. Validate with: `go test ./internal/wayland/dmabuf/...`.

- [ ] **`TODO` in Wayland event dispatch** — `app.go:1905` — A `TODO(future)` marks incomplete routing of input events (pointer, keyboard) to window-specific handlers. Events currently rely on object-level callbacks, which means multiple windows sharing the same seat object cannot independently receive input events.
  - **Evidence:** `app.go:1905` (`// TODO(future): Also dispatch to window-specific handlers for input events`).
  - **Remediation:** After fixing F1 (render presentation), add event routing in `dispatchWaylandEvent` using `surfaceToWindow` to route pointer-enter/leave and keyboard-enter/leave events to the correct window. Validate with: `go test -run TestMultiWindowInput ./...`.

- [ ] **`min` and `max` helpers duplicate Go 1.21+ builtins** — `render.go:167-181` — The project targets Go 1.24 (`go.mod:3`). The hand-rolled `min` and `max` functions in `render.go` shadow the builtins introduced in Go 1.21, creating potential confusion and dead code.
  - **Evidence:** `render.go:167-181`; `go.mod:3` (`go 1.24`).
  - **Remediation:** Delete the two helper functions and rely on the built-in `min` and `max`. Validate with: `go build ./... && go vet ./...`.

### LOW

- [ ] **Demo code duplication in `concretewidgets.go`** — `concretewidgets.go:135-163` — Six-line button-creation blocks appear three times. While the overall duplication ratio (0.80%) is well within threshold, this clone is in a public-facing file.
  - **Evidence:** `go-stats-generator` clone hash `ad8347f2af285039`; `concretewidgets.go:135-140`, `147-152`, `158-163`.
  - **Remediation:** Extract a `newWidgetWithSize(name string, w, h int) *BaseWidget` helper. Validate with: `go-stats-generator analyze . --skip-tests --format json | python3 -c "import json,sys; d=json.loads(sys.stdin.read()); print(d['duplication']['duplication_ratio'])"` ≤ current baseline.

- [ ] **`internal/render/display/x11.go:270` — `OnPixmapIdle` name** — The method is named `OnPixmapIdle` but ROADMAP.md §5.1 notes it should be `OnPixmapReady` to match the actual semantics (the pixmap is available for re-use). Cosmetic but misleads readers of the internal API.
  - **Evidence:** `internal/render/display/x11.go:270`; ROADMAP.md §5.1.
  - **Remediation:** Rename `OnPixmapIdle` → `OnPixmapReady` and update any callers. Validate with: `go build ./... && go vet ./...`.

- [ ] **No GoDoc examples for top public API functions** — Root package — The README usage example is in prose only. `go doc github.com/opd-ai/wain NewApp` shows no example. Users cannot copy-paste a verified snippet from `go doc`.
  - **Evidence:** `go doc -all github.com/opd-ai/wain` shows 0 `Example*` functions; `example/hello/main.go` exists but is not linked from GoDoc.
  - **Remediation:** Add `Example_newApp()`, `Example_newButton()`, and `Example_newColumn()` functions in a new `example_test.go` at the root. Validate with: `go test -run Example ./...`.

---

## Metrics Snapshot

| Metric | Value | Status |
|--------|-------|--------|
| Total Lines of Code (Go) | 13,845 | — |
| Total Functions | 630 | — |
| Total Methods | 1,090 | — |
| Total Structs | 232 | — |
| Total Interfaces | 32 | — |
| Total Packages | 38 | — |
| Total Files | 188 | — |
| Average function length | 9.5 lines | ✅ Excellent |
| Functions > 50 lines | 7 (0.4%) — all in `cmd/` or `internal/demo` | ✅ Good |
| Functions > 100 lines | 0 | ✅ Excellent |
| Max cyclomatic complexity | 7 (`EndFrame`, `RenderAndPresent`, `DecodeString`) | ✅ Well below threshold |
| Functions CC > 10 | 0 | ✅ Excellent |
| Documentation coverage | 91.4% overall (100% packages, 98.7% functions) | ✅ Exceeds 80% target |
| Duplication ratio | 0.80% (16 clone pairs, 258 lines) | ✅ Well below 5% |
| Circular dependencies | 0 | ✅ Clean |
| `go vet` warnings | 0 | ✅ Clean |
| `go test -race ./...` | All pass (47 packages + 11 cmd packages) | ✅ Clean |
| Rust `.unwrap()` in production lib.rs | 2 `.expect()` calls, both with proven-safe justification | ✅ Acceptable |
| TODO/FIXME comments | 2 TODO (1 in dmabuf protocol, 1 in app.go event dispatch) | ⚠️ Track |
| Refactoring suggestions (tool) | 217 | ℹ️ Largely cosmetic |

---

## Dependency Health

| Dependency | Version | Notes |
|-----------|---------|-------|
| `github.com/godbus/dbus/v5` | v5.1.0 | Indirect dep; current stable; required for future AT-SPI2 work |
| `naga` (Rust) | 0.14 (via `render-sys/Cargo.toml`) | Shader validation; naga 0.19+ available but no breaking API issues for current usage |
| Go | 1.24 | Module minimum; current stable |
| Rust | stable (musl target) | No pinned version; stable channel is appropriate |
