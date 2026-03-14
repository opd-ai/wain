# Implementation Gaps — 2026-03-14

---

## Gap 1: DragDrop Data Is Never Delivered to the Handler

- **Stated Goal**: "Drag-and-drop (XDND)" — `SetDropTarget` registers a `DragDropHandler func(mimeType string, data []byte)` that "is called with the negotiated MIME type and transferred data when a drop is completed" (`app.go:1062-1065`).
- **Current State**: `dispatchDragEvent` in `app.go:2151-2152` calls `w.dropHandler("", nil)` unconditionally — the MIME type is always an empty string and the data payload is always nil. The Wayland `wl_data_offer` read path and the X11 `XdndDrop`/selection transfer path do not populate any fields on `DragEvent` that could be forwarded to the handler.
- **Impact**: Any application that calls `Window.SetDropTarget` to accept dropped files or text will silently receive nothing. The API compiles and runs without error, making the bug invisible until the developer inspects handler arguments. Drag-and-drop is listed as a feature in the X11 package (`internal/x11/dnd/`) and in the public API (`event.go:508-562`).
- **Closing the Gap**: Add `MimeType string` and `Data []byte` fields (or accessor methods) to `DragEvent`. Populate them in the Wayland data-offer read completion callback and in the X11 `XdndDrop` + ICCCM selection-transfer handler. In `dispatchDragEvent`, pass `evt.MimeType` and `evt.Data` to the handler instead of `"", nil`. Cover with an integration test that simulates a drop and asserts the handler receives the expected MIME type and payload.

---

## Gap 2: ImageWidget Does Not Render Images on the Software Path

- **Stated Goal**: "`ImageWidget` displays an image resource. `ImageWidget` renders an image loaded via `LoadImage`. The image is scaled to fit the widget bounds." (`concretewidgets.go:509-518`; also stated in `WIDGETS.md`).
- **Current State**: `bufferCanvas.DrawImage` (`concretewidgets.go:122-125`) is a silent no-op: all parameters are blank-identified (`_ *Image, _, _, _, _ int`) and the body contains only a comment. `ImageWidget.Draw` calls `c.DrawImage(iw.image, x, y, w, h)` which returns without rendering a single pixel. On the GPU display-list path, `CmdDrawImage` is also skipped with a comment in `internal/raster/consumer/software.go:97-99`.
- **Impact**: Any application that creates an `ImageWidget` (or any custom widget that calls `Canvas.DrawImage`) receives a transparent placeholder. The application does not error; users see blank space where images should appear. `Canvas` is a stability-pinned public interface (`STABILITY.md`).
- **Closing the Gap**: Implement `bufferCanvas.DrawImage` by decoding the `*Image` pixel buffer and calling `composite.Blit` or `composite.BlitScaled` from `internal/raster/composite` to alpha-composite the image into the `primitives.Buffer`. The raster composite package is already implemented and tested. Similarly, wire `CmdDrawImage` through `SoftwareConsumer.renderCommand` to the composite package.

---

## Gap 3: Canvas Gradient and Shadow Methods Are Silent No-Ops

- **Stated Goal**: The `Canvas` interface (stability-pinned, `STABILITY.md`) exposes `LinearGradient`, `RadialGradient`, and `BoxShadow` methods. The `README` lists "gradients, shadows" as software-rasterizer capabilities. The raster layer (`internal/raster/effects`) fully implements all three.
- **Current State**: The `bufferCanvas` adapter that bridges public widget `Draw` callbacks to the raster layer has stub implementations for all three methods (`concretewidgets.go:128-140`). Each contains only a comment "not supported in buffer canvas adapter yet" and does nothing.
- **Impact**: Custom widgets that use `Canvas.LinearGradient`, `Canvas.RadialGradient`, or `Canvas.BoxShadow` in their `Draw` method produce no visible output. Because `Canvas` is a published interface, third-party widget authors writing to the documented contract will encounter silent failures. The built-in widget set does not currently call these methods, so first-party widgets are unaffected.
- **Closing the Gap**: Wire each `bufferCanvas` method to the corresponding `internal/raster/effects` function: `effects.LinearGradient`, `effects.RadialGradient`, `effects.BoxShadow`. The `bufferCanvas` struct already holds a `*primitives.Buffer` field that matches the function signatures. Add unit tests that call each method and assert non-zero pixel output.

---

## Gap 4: CI Linting Is Broken

- **Stated Goal**: The README pre-commit checklist: "No vet warnings: `go vet ./...`". The CI pipeline description: "`golangci-lint`" as a quality gate (`README.md` §CI Workflow). `.golangci.yml` exists and configures 9 linters.
- **Current State**: `golangci-lint run ./...` exits with a hard error: `can't load config: the Go language version (go1.23) used to build golangci-lint is lower than the targeted Go version (1.24)`. This is caused by the installed golangci-lint binary (v1.64.8, built with Go 1.23) conflicting with the module's `go 1.24` directive. Separately, `.golangci.yml:17-19` enables `structcheck`, `varcheck`, and `deadcode`, which were removed from golangci-lint at v1.49.0 and will produce additional errors once the version is fixed.
- **Impact**: The lint CI job fails silently (or does not run at all) on any runner with a pre-Go-1.24 golangci-lint binary. Code quality regressions that would be caught by `staticcheck`, `errcheck`, or `unused` are not detected in CI. Contributors following the pre-commit checklist cannot verify their changes.
- **Closing the Gap**: (1) Update `.github/workflows/ci.yml` to install golangci-lint using the official installer action pinned to a release built with Go 1.24 or later. (2) Remove `structcheck`, `varcheck`, and `deadcode` from `.golangci.yml:17-19`. `unused` (already enabled) replaces all three.

---

## Gap 5: AT-SPI2 Build Tag Not Documented in README

- **Stated Goal**: README §Features: "**AT-SPI2 Accessibility** — D-Bus screen reader integration with Accessible, Component, Action, and Text interfaces (`internal/a11y/`)".
- **Current State**: `EnableAccessibility` compiles and links in all builds but returns `nil` without `-tags=atspi` (via `internal/a11y/manager_stub.go`). The `ACCESSIBILITY.md` document explains this clearly, but the README feature list and the `STABILITY.md` covered identifiers table make no mention of the build constraint.
- **Impact**: A developer reading only the README will call `EnableAccessibility("my-app")` in a default build, receive `nil`, and either crash on a nil pointer dereference (if they don't check the return) or assume accessibility is enabled. Only by reading `ACCESSIBILITY.md` would they learn the build tag is required. This is a documentation gap, not a code gap.
- **Closing the Gap**: Add a parenthetical to the README feature bullet: `requires \`-tags=atspi\` — see [ACCESSIBILITY.md](./ACCESSIBILITY.md)`. Add the same note to the `STABILITY.md` table row for `EnableAccessibility`.

---

## Gap 6: GPU Shader Compilation Not Exercised in Standard CI

- **Stated Goal**: "**Shader Compilation** — WGSL shaders compiled to Intel EU and AMD RDNA native ISA via naga" (README §Features). `HARDWARE.md` marks Intel Gen9–Xe and AMD RDNA 1–3 as "✅ Fully Supported".
- **Current State**: The WGSL→naga parse and validation path (`render-sys/src/shader.rs`) is tested and passing. The EU instruction lowering pipeline (`render-sys/src/eu/lower.rs`, ~4 000 lines) and RDNA encoding pipeline (`render-sys/src/rdna/`) compile and have unit tests for individual instructions, but the full shader→native-ISA compilation pipeline (parse → type-check → lower → register-alloc → encode → emit binary) is exercised only in Rust unit tests, not in an end-to-end CI job. The `cmd/gpu-triangle-demo` and `cmd/gpu-shader-demo` require physical DRM hardware and are noted as conditional.
- **Impact**: Regressions in the EU/RDNA lowering or encoding path may go undetected in standard CI. The HARDWARE.md claim of "✅ Fully Supported" for specific GPU generations implies hardware-validated execution, which CI cannot currently guarantee.
- **Closing the Gap**: Add a non-hardware CI step that compiles each of the 7 WGSL shaders through the full EU and RDNA lowering pipeline to native ISA bytes, then asserts a non-empty, structurally valid binary output (e.g., check the instruction count and magic header). This can run in CI without a GPU and would catch lowering regressions. Qualify the HARDWARE.md support status as "code-verified" vs. "hardware-validated" pending a hardware CI runner.

---

## Gap 7: `TECHNICAL_DEBT.md` Referenced but Absent

- **Stated Goal**: README §Contributing pre-commit checklist: "TODOs tracked in `TECHNICAL_DEBT.md`".
- **Current State**: `TECHNICAL_DEBT.md` does not exist in the repository root.
- **Impact**: Contributors are directed to a non-existent file. Known technical debt (canvas stubs, DragDrop data gap, GPU CI gap) is not centrally tracked, increasing the risk that it accumulates invisibly.
- **Closing the Gap**: Create `TECHNICAL_DEBT.md` listing at minimum the items already identified in this report (canvas stubs, DragDrop data delivery, GPU CI gap). Alternatively, remove the reference from `README.md` and `CONTRIBUTING.md` and use GitHub Issues instead.
