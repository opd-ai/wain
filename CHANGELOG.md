# Changelog

All notable changes to **wain** are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- `cmd/bench` — software-renderer benchmarking binary (500 rects, 100 text
  placeholder rects, 10 rounded box-shadows at 1920×1080; JSON output with
  pass/fail against a configurable ms threshold)
- CI `benchmarks` job extended with `cmd/bench -frames 60 -max 16` step and
  a GitHub Step Summary table for per-commit frame-time tracking
- SIMD-optimised `blendRow` in `internal/raster/primitives/rect.go` —
  precomputes per-colour blend factors once per call and processes pixels
  via vectorisation-friendly uint32 loop (auto-vectorises to AVX2 on x86-64)
- GoDoc `Example*` functions for all top-10 public API constructors:
  `NewApp`, `NewButton`, `NewLabel`, `NewTextInput`, `NewPanel`, `NewRow`,
  `NewColumn`, `NewGrid`, `NewScrollView`, `EnableAccessibility`
- `TUTORIAL.md` — step-by-step guide building a contact-form application
  (layout, events, theming, clipboard, window lifecycle)

---

## [1.0.0] — 2026-03-14

### Added
- **Public API** (`app.go`): `App`, `Window`, `WindowConfig`, event loop, display
  server auto-detection (Wayland preferred, X11 fallback), renderer auto-detection
  (Intel → AMD → software), `Window.SetLayout`, `Window.RenderFrame`
- **Software presentation path**: `SoftwareWaylandPresenter`, `SoftwareX11Presenter`
  in `internal/render/display/` — pixels delivered via `wl_shm` / X11 PutImage
- **GPU presentation path**: `WaylandPipeline` (DMA-BUF), `X11Pipeline` (DRI3+Present)
  wired into the `App` type through `initWaylandPresenter`/`initX11Presenter`
- **GPU command submission**: `GPUBackend.Render()` emits Intel/AMD batch commands
  (`submit.go`, `vertex.go`, `batch.go`); solid fills, rounded rects, text glyphs
- **AT-SPI2 accessibility** (`internal/a11y/`): `Manager`, `AccessibleObject`,
  `Accessible`, `Component`, `Action`, `Text` D-Bus interfaces; focus-change events;
  build tag `atspi` enables real D-Bus export, stub otherwise
- **`cmd/widget-demo`**: interactive Wayland and X11 window with widget tree
- **`example/hello`**: canonical hello-world showing `App.Notify` + widget layout
- `STABILITY.md`: deprecation policy and API-stability commitment

### Changed
- `go.mod`: `github.com/godbus/dbus/v5` promoted from indirect to direct dependency

---

## [0.2.0] — 2026-02-xx

### Added
- `Window.SetLayout` and `layoutAdapter` bridging public widgets to the internal
  panel tree
- `example/hello` headless fallback so `go test ./example/hello/...` always passes
- Client-side window decorations (`internal/ui/decorations`)
- HiDPI / DPI-aware scaling (`internal/ui/scale`, `internal/x11/dpi`)

### Fixed
- Display list damage tracking correctness for partial-frame re-renders

---

## [0.1.0] — 2026-01-xx

### Added
- Initial public release
- Wayland client (9 packages: wire, socket, client, shm, xdg, input, dmabuf,
  datadevice, output)
- X11 client (9 packages: wire, client, events, gc, shm, dri3, present, dpi,
  selection)
- Software 2D rasterizer (`internal/raster/`: primitives, curves, composite,
  effects, text, displaylist, consumer)
- UI widget layer (`internal/ui/`: layout, widgets, pctwidget, scale, decorations)
- GPU buffer infrastructure (`render-sys/src/allocator.rs`, `slab.rs`)
- Intel EU backend (Gen9+ execution unit compiler, `render-sys/src/eu/`)
- AMD RDNA backend (`render-sys/src/rdna/`, `amd.rs`, `pm4.rs`)
- Shader frontend (WGSL/GLSL via naga 0.14; 7 UI shaders)
- Go–Rust static linking via CGO/musl; `ldd bin/wain` = "not a dynamic executable"
- Clipboard support (Wayland `data_device`, X11 `CLIPBOARD` selection)
- Keyboard accessibility: Tab/Shift-Tab focus traversal, Enter/Space activation

[Unreleased]: https://github.com/opd-ai/wain/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/opd-ai/wain/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/opd-ai/wain/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/opd-ai/wain/releases/tag/v0.1.0
