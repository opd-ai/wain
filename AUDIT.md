# AUDIT — 2026-03-13

## Project Context

**Wain** is a statically-compiled Go UI toolkit that bridges a Rust rendering library
(via CGO + musl) for GPU-accelerated graphics on Linux.  It targets Go library consumers
who need a single, fully-static binary with no runtime dependencies.  The project
implements Wayland and X11 display protocols from scratch, provides a software 2D
rasterizer, and exposes a public `App`/`Window`/widget API.

Primary documentation audited: `README.md`, `API.md`, `ACCESSIBILITY.md`,
`ROADMAP.md`, `READINESS_SUMMARY.md`.  Analysis tool: `go-stats-generator` (installed
binary).

---

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| CRITICAL | 1     | CGO link failure prevents `go test ./...` from working without env setup |
| HIGH     | 5     | Partial/placeholder implementations in advertised features |
| MEDIUM   | 4     | Documented features absent or incomplete |
| LOW      | 5     | Naming, style, and cosmetic issues |

**Overall health:** Core protocol stacks (Wayland, X11), software rasterizer, and
widget layer are functional and well-tested (31 passing packages, 0 race conditions
detected).  The GPU rendering pipeline is structurally present but not integrated
end-to-end.  Several public API surface areas on X11 are silent no-ops.

**Diff vs. prior baseline (`baseline.json`, 2026-03-09):** The codebase is effectively
unchanged in size (13,101 vs 13,099 LOC, same function/method/type counts).  The prior
baseline reported 0% documentation coverage and 0 naming violations, which was a
tool-version artifact; current tool correctly measures 90.7% coverage and 32 low-severity
naming violations.  The single previously-flagged CC >10 function remains
(`writeGlyphMetadata`, cc=11 — now confirmed in `cmd/`).

---

## Findings

### CRITICAL

- [x] **`go test ./...` fails without CGO_LDFLAGS** — `internal/render/binding.go` (CGO header) — Running `go test ./...` in a clean environment produces linker errors (`undefined reference to render_add`, `buffer_allocate`, etc.) for `internal/render`, `internal/render/atlas`, `internal/render/backend`, and `internal/integration`.  The Rust static library (`librender_sys.a`) must be on `CGO_LDFLAGS` at link time.  The README documents this requirement ("Without direnv, you must use `make test-go`…") but presents it as a minor caveat.  In practice, any CI job or contributor running `go test ./...` without the `.envrc` environment will see 14 package build failures rather than a clear error message.  The root cause is that `internal/render/binding.go` embeds CGO `#cgo` flags that reference the library path implicitly via the build system, not via a self-contained `//go:generate` step in the affected package.  **Fix:** `scripts/build-rust.sh` now generates `internal/render/cgo_flags_generated.go` with `#cgo LDFLAGS: ${SRCDIR}/...` after building the library and stub; `go generate ./...` makes `go test ./...` self-contained.

---

### HIGH

- [x] **GPU→CPU readback is unimplemented (no-op)** — `internal/raster/consumer/gpu.go:104–112` — The `GPUConsumer.copyToBuffer` method, which is required to produce pixels from the GPU render target back to a CPU buffer, is documented as "a placeholder for future implementation" and returns `nil` without doing anything (`_ = buf`).  README claims "Display List Rendering — GPU backend with display list consumer"; the consumer exists but cannot deliver rendered pixels to the display without this step.  This silently produces blank frames on GPU paths.

- [x] **Text rendering is a no-op in the GPU backend** — `internal/render/backend/vertex.go:188–193` — `textToVertices` always returns `nil` with an in-code comment "text won't render" and a Phase 5.2 deferral note.  README claims "SDF text rendering" as a feature; this is functional only through the *software* rasterizer (`internal/raster/text/`), not through the GPU display-list backend.  Any widget that draws text via the GPU backend produces silent blank output.

- [x] **Intel EU and AMD RDNA shader backends not integrated into the rendering path** — `render-sys/src/eu/mod.rs`, `render-sys/src/rdna/`, `ROADMAP.md` — README claims "Intel EU Backend — register allocator, instruction lowering, and 128-bit binary encoding for Gen9+ execution units" and "AMD RDNA Backend — RDNA instruction set, register allocation, encoding, and PM4 command stream".  ROADMAP explicitly marks both as `⚠️ Partial`: the compilation pipeline produces EU/RDNA binaries but they are never executed on hardware.  No demo renders a frame using compiled shaders; `gpu-triangle-demo` uses fixed-function state.  The `eu/` and `rdna/` crates contain 89 `.unwrap()` calls in production paths (`eu/lower.rs:2039–2040`, `eu/mod.rs:316,322,369,375` etc.) that would panic on unexpected shader IR.

- [x] **`Panel.SetAlign` is a documented no-op** — `layout.go:205–213` — The method signature and doc comment fully describe cross-axis alignment ("For Row containers, controls vertical alignment…"), but the body is `_ = align` — the align value is silently discarded.  README claims "Flexbox-like Row/Column layout engine".  Any call to `SetAlign` with `AlignCenter`, `AlignEnd`, or `AlignStretch` has no effect; all layout is equivalent to `AlignStart`.

- [x] **X11 window-management operations are silent no-ops** — `app.go:477–650` — Three `Window` methods used by the documented public API have placeholder implementations for the X11 backend:
  - `SetTitle` (app.go:494): returns `nil` without setting the X11 window title ("simplified placeholder").
  - `SetMinSize` (app.go:559): returns `nil` without writing WM size hints.
  - `SetMaxSize` (app.go:575): returns `nil` without writing WM size hints.
  - `SetFullscreen` (app.go:615): returns `nil` without sending `_NET_WM_STATE_FULLSCREEN`.

  All four contain comments explaining the limitation but return success (`nil`), giving callers no way to detect failure.  The `x11-demo` and `window-demo` demos exercise these paths.

---

### MEDIUM

- [x] **`cmd/example-app` explicitly incomplete** — `cmd/example-app/main.go:20–21` — The top-of-file NOTE states "Full integration with the rendering pipeline is in progress."  The file is listed in README's project structure as a usage example and ships in the repository.  A developer following the README's "Create a window using the public API" section may build this binary and observe no rendering output.

- [x] **Clipboard not exposed in public API** — `app.go` (no clipboard functions), `internal/x11/selection/manager.go:99,127` — README features list "data-device clipboard" (Wayland) and the demo `cmd/clipboard-demo` exists, but neither `App` nor `Window` expose `SetClipboard`/`GetClipboard` methods.  X11 clipboard is implemented in `internal/x11/selection` but is inaccessible to public-API consumers.  Wayland data-device is implemented in `internal/wayland/datadevice` but similarly not wired into the public surface.

- [x] **AT-SPI2 accessibility absent** — `ACCESSIBILITY.md:5` — README feature description does not mention accessibility, but `ACCESSIBILITY.md` ships with the repository and describes a planned-but-unimplemented AT-SPI2 layer.  `accessibility_test.go` exists at the repo root.  The combination of a test file and a roadmap document creates an expectation gap for contributors.

- [x] **`cmd/gen-atlas` (SDF font atlas tool) uses panic for all error handling** — `cmd/gen-atlas/main.go:81–134` — The `writeAtlasFile` function uses 15 consecutive `panic(err)` calls for file I/O and atlas-writing operations instead of returning errors.  README documents this as a user-facing tool (`gen-atlas/`) under "Font Atlas Generation".  Any failure (missing font, disk full, bad path) terminates the process with a panic stack trace rather than a user-friendly error message.  `cyclomatic complexity = 11` (highest in the codebase; above the >10 threshold).

---

### LOW

- [x] **32 identifier naming violations** — Various files — Detected by `go-stats-generator`: 28 identifier issues including:
  - `buttonColors`, `textInputDisplay` (stuttering — `internal/ui/widgets/base.go:232,551`)
  - `ArgTypeUint32`, `DecodeUint32`, `EncodeUint32` (should be `UInt32` per Go acronym convention — `internal/wayland/wire/protocol.go:75,157,166`)
  All are low-severity; none affect behavior.
  **Fixed (2026-03-13)**: Renamed private stuttering types: `buttonColors`→`btnColors`, `textInputDisplay`→`inputDisplay` (`internal/ui/widgets/base.go`), `panelStyle`→`themeAdapter` (`layout.go`). Renamed package-stuttering exports `ScaleInt`→`Int`, `ScaleFloat`→`Float` (`internal/ui/scale/manager.go`). Remaining tool findings (`Uint32`→`UInt32`, `Idle`→`IDle`) are false positives per Go stdlib convention: `encoding/binary` uses `Uint32`, and `Idle` is an English word, not the `ID` acronym.

- [x] **3 generic file names** — `internal/x11/events/types.go`, `internal/demo/constants.go`, `internal/ui/widgets/base.go` — go-stats-generator reports names as too generic; naming convention violation only.
  **Fixed (2026-03-13)**: Renamed `types.go`→`event_types.go`, `constants.go`→`drm_paths.go`, `base.go`→`widget_impl.go`.

- [ ] **1 package name violation** — (go-stats-generator `package_name_violations: 1`) — Low-severity, cosmetic.

- [ ] **Deprecated comment on Wayland DMA-BUF protocol** — `internal/wayland/dmabuf/protocol.go:109` — Comment marks `zwp_linux_dmabuf_v1` format events as deprecated; no action tracking exists in code.

- [ ] **Cross-axis alignment issue URL is a placeholder** — `layout.go:211` — The tracking URL reads `https://github.com/opd-ai/wain/issues/TBD` — no actual issue has been filed to track the deferred `SetAlign` implementation.

---

## Metrics Snapshot

| Metric | Current (`audit-baseline.json`) | Prior (`baseline.json` 2026-03-09) | Delta |
|--------|---------------------------------|------------------------------------|-------|
| Total files | 171 | 171 | 0 |
| Total LOC | 13,101 | 13,099 | +2 |
| Total functions | 541 | 541 | 0 |
| Total methods | 950 | 950 | 0 |
| Total structs | 214 | 214 | 0 |
| Total interfaces | 31 | 31 | 0 |
| Total packages | 37 | 37 | 0 |
| Avg cyclomatic complexity | N/A (tool limitation) | N/A | — |
| Functions with CC >10 | 1 (`writeGlyphMetadata`, cc=11) | 1 | 0 |
| Functions with CC >15 | 0 | 0 | 0 |
| Functions >50 lines | 15 (mostly `cmd/` demos) | ~15 | ~0 |
| Doc coverage (overall) | 90.7% | 0%¹ | N/A¹ |
| Doc coverage (functions) | 98.6% | 0%¹ | N/A¹ |
| Doc coverage (methods) | 88.5% | 0%¹ | N/A¹ |
| Duplication ratio | 0% (no clones) | 3.35% | —² |
| Circular dependencies | 0 | 0 | 0 |
| Naming violations | 32 (all LOW) | 0¹ | N/A¹ |
| `go test ./...` passing packages | 31/45 | — | — |
| `go test ./...` build-fail packages | 14/45 (CGO link) | — | — |
| `go vet ./...` | PASS (0 issues) | — | — |
| Race detector | PASS on all 31 passing pkgs | — | — |

¹ Prior baseline.json shows 0% coverage and 0 violations due to a known `go-stats-generator` version
difference; values are not meaningfully comparable.

² Duplication metric algorithm differs between tool versions.

### High-Risk Functions (CC >10 OR length >50 lines, library code only)

| Function | File | Line | CC | Lines | Risk Reason |
|----------|------|------|----|-------|-------------|
| `writeGlyphMetadata` | `cmd/gen-atlas/main.go` | 107 | 11 | 29 | CC >10; panics in caller |
| `applyToTheme` | `theme.go` | 195 | 10 | 29 | CC at threshold |
| `decodeVisuals` | `internal/x11/wire/setup.go` | 231 | 10 | 31 | CC at threshold |
| `main` (auto-render-demo) | `cmd/auto-render-demo/main.go` | 24 | 9 | 100 | Length >50 |
| `main` (window-demo) | `cmd/window-demo/main.go` | 22 | 8 | 79 | Length >50 |
| `main` (callback-demo) | `cmd/callback-demo/main.go` | 19 | 2 | 77 | Length >50 |
| `main` (decorations-demo) | `cmd/decorations-demo/main.go` | 16 | 3 | 70 | Length >50 |
| `run` | `cmd/wain-build/main.go` | 65 | 9 | 54 | CC + length |
| `BlitScaled` | `internal/raster/composite/ops.go` | 107 | 7 | 51 | Length >50, inner loop |

All functions above the length threshold in library code are in `cmd/` (demo/tool
binaries) or `internal/raster/composite` (hot-path pixel blit, length is inherent).
No library functions exceed CC >10.
