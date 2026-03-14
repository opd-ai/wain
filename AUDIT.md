# AUDIT — 2026-03-14

## Project Goals

**Wain** is a statically-compiled Go UI toolkit for Linux that:

- Renders via a Rust GPU backend (Intel Gen9–Xe, AMD RDNA 1–3) with automatic software fallback
- Implements the Wayland and X11 display protocols directly (no libwayland, no Xlib)
- Produces fully static, zero-dependency binaries (musl libc + Rust staticlib)
- Provides a widget system (Button, Label, TextInput, ScrollView, ImageWidget, Spacer) with percentage-based sizing
- Provides layout containers (Row, Column, Stack, Grid, Panel) with flexbox-style alignment
- Software-rasterizes rectangles, rounded rects, anti-aliased lines, Bézier curves, gradients, shadows, SDF text
- Submits GPU commands natively to Intel i915/Xe and AMD RDNA drivers via DRM/KMS
- Compiles WGSL shaders to Intel EU and AMD RDNA native ISA via naga 0.14
- Integrates AT-SPI2 screen-reader accessibility via D-Bus
- Offers theming (DefaultDark, DefaultLight, HighContrast), clipboard, keyframe animations, and client-side decorations
- Achieves ≤16.7 ms/frame (60 FPS) on the software rasterizer at 1920×1080
- Targets Go developers, embedded/appliance developers, and GPU programming enthusiasts

---

## Goal-Achievement Summary

| # | Goal | Status | Evidence |
|---|------|--------|----------|
| 1 | Static binary (zero runtime deps) | ✅ Achieved | Makefile `-extldflags '-static'`; CI `ldd` check; `render-sys/Cargo.toml` `crate-type = ["staticlib"]` |
| 2 | Wayland protocol client (9 packages) | ✅ Achieved | `internal/wayland/`: wire, socket, client, shm, xdg, input, dmabuf, datadevice, output — 9 packages, all tests passing |
| 3 | X11 protocol client (9+ packages) | ✅ Achieved | `internal/x11/`: wire, client, events, gc, shm, dri3, present, dpi, selection, dnd — 10 packages, all tests passing |
| 4 | Software 2D rasterizer (7 packages) | ✅ Achieved | `internal/raster/`: primitives, curves, composite, effects, text, displaylist, consumer; visual regression tests 100% match |
| 5 | Widget system (Button, Label, TextInput, ScrollView, ImageWidget, Spacer) | ⚠️ Partial | Widgets implemented in `concretewidgets.go`; **ImageWidget renders nothing on software path** (`bufferCanvas.DrawImage` is a no-op, `concretewidgets.go:123-125`) |
| 6 | Layout containers (Row, Column, Stack, Grid, Panel) | ✅ Achieved | `layout.go`, `internal/ui/layout/`; layout tests pass |
| 7 | GPU buffer infrastructure (DRM, GEM, DMA-BUF) | ✅ Achieved | `render-sys/src/allocator.rs`, `slab.rs`, `drm.rs`, `i915.rs`, `xe.rs`, `amd.rs` (~3 800 Rust LOC); `internal/render/` CGO bindings |
| 8 | GPU command submission (Intel + AMD) | ⚠️ Partial | `render-sys/src/batch.rs`, `pipeline.rs`, `submit.rs`, `cmd/` exist (~2 500 Rust LOC); batch unit tests pass; no CI-verified end-to-end GPU rendered frame without hardware |
| 9 | Intel EU shader compiler (Gen9–Xe) | ⚠️ Partial | `render-sys/src/eu/` (6 files, ~4 400 Rust LOC); register allocator, instruction lowering, encoding present; shader→EU compilation path not exercised in standard CI |
| 10 | AMD RDNA shader compiler | ⚠️ Partial | `render-sys/src/rdna/` (6 files, ~1 400 Rust LOC), `amd.rs`, `pm4.rs`; same gap as Intel EU |
| 11 | Shader frontend (WGSL/GLSL via naga) | ✅ Achieved | `render-sys/src/shader.rs`; 7 WGSL shaders in `render-sys/shaders/`; `cmd/shader-test` validates all 7 |
| 12 | Display server auto-detection (Wayland→X11) | ✅ Achieved | `app.go`: checks `$WAYLAND_DISPLAY` then `$DISPLAY`; `ForcX11` config override |
| 13 | Renderer auto-detection (Intel→AMD→software) | ✅ Achieved | `internal/render/backend/auto.go`; graceful fallback chain with verbose logging |
| 14 | AT-SPI2 accessibility | ✅ Achieved | `internal/a11y/` (10 files, 75 functions); `accessibility.go`; requires `-tags=atspi` (documented in `ACCESSIBILITY.md` but **not in README**) |
| 15 | Theming (3 built-in themes) | ✅ Achieved | `theme.go`: `DefaultDark()`, `DefaultLight()`, `HighContrast()`; `cmd/theme-demo` |
| 16 | Clipboard (Wayland + X11) | ✅ Achieved | `clipboard.go`, `internal/wayland/datadevice/`, `internal/x11/selection/`; tests pass |
| 17 | Keyframe animations | ✅ Achieved | `animate.go`, `internal/ui/animation/`; 6 easing functions; animation tests pass |
| 18 | Client-side decorations | ✅ Achieved | `internal/ui/decorations/`; `cmd/decorations-demo` |
| 19 | HiDPI / DPI-aware scaling | ✅ Achieved | `internal/ui/scale/`, `internal/x11/dpi/`; 100% test coverage on scale package |
| 20 | ≤16.7 ms/frame software rendering at 1080p | ✅ Achieved | `cmd/bench`; CI-enforced; visual regression tests 100% match |
| 21 | Drag-and-drop (Wayland + X11) | ❌ Non-functional | `app.go:2152`: `dropHandler("", nil)` always called with empty MIME type and nil data; transferred content is never delivered to handler |
| 22 | Canvas gradient/shadow rendering (software path) | ⚠️ Partial | `bufferCanvas.LinearGradient`, `RadialGradient`, `BoxShadow` are silent no-ops (`concretewidgets.go:128-140`); raster layer supports them but the Canvas adapter bridge is not wired |
| 23 | CI linting (golangci-lint) | ❌ Broken | `golangci-lint run ./...` exits with error: `go1.23 build < go1.24 target`; also `.golangci.yml` references removed linters |

**Overall: 16/23 goals fully achieved (70%); 5 partial; 2 broken**

---

## Findings

### CRITICAL

- [ ] **DragDrop handler always receives empty MIME type and nil data** — `app.go:2152` — `dispatchDragEvent` calls `w.dropHandler("", nil)` unconditionally when a drop event fires. The `DragDropHandler` contract (`event.go:562`) promises "the negotiated MIME type and transferred data when a drop is completed", but both are always zero-valued. Any application that calls `Window.SetDropTarget` and relies on the handler arguments to read dropped content silently receives nothing. This makes the entire drop-target API non-functional for data transfer. **Remediation:** Wire the actual transferred MIME type and payload through `DragEvent` (add `mimeType string` and `data []byte` fields to `DragEvent`, populate them in the Wayland `wl_data_offer` read path and in the X11 `XdndDrop`/selection-read path, then pass `evt.MimeType()` and `evt.Data()` to the handler). Validate with: `go test -race -run TestDropTarget ./...` after adding an integration test that verifies the handler receives the expected MIME type and payload.

### HIGH

- [ ] **ImageWidget renders nothing on the software rasterizer path** — `concretewidgets.go:123-125` — `bufferCanvas.DrawImage` is a documented stub that discards all arguments. `ImageWidget.Draw` calls `c.DrawImage(iw.image, x, y, w, h)` but the canvas adapter never reaches the `internal/raster/composite` package. On the GPU path the `CmdDrawImage` display-list command is also silently skipped (`internal/raster/consumer/software.go:97-99`). The README and `WIDGETS.md` describe `ImageWidget` as a rendering widget without noting this limitation. **Remediation:** In `bufferCanvas.DrawImage`, decode the `*Image.pixels` (RGBA bytes) and call `composite.Blit` or `composite.BlitScaled` from `internal/raster/composite` to composite the image into the buffer. Add a test that creates a solid-colour `Image` and asserts at least one pixel in the rendered buffer matches. Validate with: `go test -race -run TestImageWidget ./...`.

- [ ] **`golangci-lint run ./...` fails — CI linting is broken** — `.golangci.yml:1` — golangci-lint v1.64.8 (available in the environment) was built with Go 1.23 but the module requires Go 1.24, causing a hard exit error: `can't load config: the Go language version (go1.23) used to build golangci-lint is lower than the targeted Go version (1.24)`. The linting CI job will fail on any runner that hasn't updated golangci-lint to a Go 1.24-built binary. Additionally, `.golangci.yml:17-19` lists `structcheck`, `varcheck`, and `deadcode` which were removed from golangci-lint in v1.49.0 and will produce unknown-linter errors. **Remediation:** (1) Update the golangci-lint installation in `.github/workflows/ci.yml` to use a release built with Go 1.24 (e.g. `v2.x`). (2) Remove `structcheck`, `varcheck`, `deadcode` from `.golangci.yml:17-19`; their functionality is subsumed by `unused` (already enabled). Validate with: `golangci-lint run ./...` returning exit code 0.

### MEDIUM

- [ ] **Canvas gradient and shadow methods are silent no-ops in the widget draw path** — `concretewidgets.go:128-140` — `bufferCanvas.LinearGradient`, `bufferCanvas.RadialGradient`, and `bufferCanvas.BoxShadow` all contain `// ... not supported in buffer canvas adapter yet.` and do nothing. The `Canvas` interface is part of the public API (`STABILITY.md`) and custom widgets that call these methods in their `Draw` callbacks will silently produce no output. The effects exist and are tested in `internal/raster/effects` but are not wired through the Canvas bridge. **Remediation:** Implement each method: (a) `LinearGradient` → call `effects.LinearGradient`; (b) `RadialGradient` → call `effects.RadialGradient`; (c) `BoxShadow` → call `effects.BoxShadow`. The required import (`internal/raster/effects`) is already used transitively. Validate with: `go test -race -run TestCanvas ./...`.

- [ ] **`TECHNICAL_DEBT.md` referenced but does not exist** — `README.md:481`, `CONTRIBUTING.md` — The contributor pre-commit checklist states "TODOs tracked in `TECHNICAL_DEBT.md`" but the file is absent. Contributors following these instructions have no canonical debt registry. **Remediation:** Either create `TECHNICAL_DEBT.md` (listing known debt such as the canvas stubs above) or remove the reference from `README.md` and `CONTRIBUTING.md`. Validate with: `ls TECHNICAL_DEBT.md`.

- [ ] **README omits `-tags=atspi` requirement for AT-SPI2** — `README.md` (Features section) — The README bullet "AT-SPI2 Accessibility" makes no mention that `EnableAccessibility` is a no-op returning `nil` without `-tags=atspi`. A developer who calls `EnableAccessibility("my-app")` in a default build will silently get back `nil` and conclude that AT-SPI2 is working (they must detect the nil and act accordingly, but nothing in the default README path signals this). `ACCESSIBILITY.md` does cover this, but README is the first-contact document. **Remediation:** Add a parenthetical to the README feature bullet: "requires `-tags=atspi` (see [ACCESSIBILITY.md](./ACCESSIBILITY.md))". Validate by inspecting README after the change.

### LOW

- [ ] **`go.sum` contains stale transitive entries for `golang.org/x/sys`** — `go.sum:5-10` — The file contains hash entries for `golang.org/x/sys v0.20.0` and `v0.42.0` even though `go.mod` only requires `v0.27.0`. These entries are harmless to correctness but indicate that `go mod tidy` has not been run after dependency updates, and they can confuse `go mod verify`. **Remediation:** Run `go mod tidy` to prune unused entries. Validate with: `go mod verify && go mod tidy && git diff --exit-code go.sum`.

- [ ] **Deprecated linters in `.golangci.yml`** — `.golangci.yml:17-19` — `structcheck`, `varcheck`, and `deadcode` were removed from golangci-lint at v1.49.0. They are listed as enabled linters and will produce "unknown linter" errors on any modern golangci-lint version (the high-severity finding above captures the primary failure; this finding addresses the root config correctness). **Remediation:** Remove lines 17–19 (`structcheck`, `varcheck`, `deadcode`) from `.golangci.yml`. Their functionality is fully covered by the already-enabled `unused` linter. Validate with: `golangci-lint run ./... 2>&1 | grep "unknown linter"` should produce no output.

---

## Metrics Snapshot

| Metric | Value | Assessment |
|--------|-------|------------|
| Total Go LOC | 14,665 | Moderate-sized codebase |
| Total Rust LOC | ~13,669 | Substantial GPU backend |
| Total Go functions | 664 | — |
| Total Go methods | 1,166 | — |
| Total Go packages | 40 (non-cmd) | Well-modularised |
| Avg function length | 9.4 lines | ✅ Excellent |
| Functions > 50 lines | 7 (0.4%) | ✅ Excellent |
| Functions > 100 lines | 0 (0.0%) | ✅ Excellent |
| Max cyclomatic complexity | 7 (Render, DecodeSetupReply, …) | ✅ No function > 10 |
| Documentation coverage | 91.0% overall (98.3% functions, 89.0% methods) | ✅ Above threshold |
| Duplication ratio | 0.64% (11 clone pairs, 218 lines) | ✅ Excellent |
| Naming violations | 34 total (2 file, 31 identifier, 1 package) | Low impact |
| `go test ./...` | ✅ All packages pass | — |
| `go vet ./...` | ✅ No issues | — |
| `go test -race ./...` | ✅ No races detected | — |
| `golangci-lint run ./...` | ❌ Fails (Go version mismatch) | Requires fix |
| Visual regression (8 primitives) | ✅ 100% pixel match | — |
| CI 60 FPS benchmark | ✅ ≤16.7 ms/frame at 1080p | — |

*Metrics produced by go-stats-generator v1.0.0 on 2026-03-14.*
