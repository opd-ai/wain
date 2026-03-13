# Implementation Plan: Complete GPU Rendering & API Stabilisation

> Generated: 2026-03-13  
> Baseline tool: `go-stats-generator analyze . --skip-tests --format json --sections functions,duplication,documentation,packages,patterns`

---

## Project Context

- **What it does**: Wain is a statically-compiled Go UI toolkit that bridges a Rust rendering
  library (via CGO and musl) for GPU-accelerated graphics on Linux, with from-scratch Wayland
  and X11 client implementations and a software 2D rasterizer, producing a single fully-static
  binary with zero runtime dependencies.
- **Current milestone**: Priority 1 — Complete GPU-Rendered Frame Path  
  _(first task with ☐ checkboxes in `ROADMAP.md`; Intel EU and AMD RDNA backends compile WGSL
  shaders but never drive a displayed frame)_
- **Estimated Scope**: **Large** — new Rust source files, Go integration code, and a new demo
  binary; complexity and duplication work are Small/Medium alongside it.

---

## Metrics Summary (baseline, 2026-03-13)

| Metric | Value | Threshold | Rating |
|--------|-------|-----------|--------|
| Total Go source files | 171 | — | — |
| Total LOC (Go) | 13,101 | — | — |
| Functions / methods | 541 / 950 | — | — |
| **Library functions CC > 9** | **2** | < 5 | ✅ Small |
| **Library functions CC > 7** | **11** | < 5 | ⚠️ Small+ |
| **Duplication ratio** | **3.35%** | < 3% | ⚠️ Medium |
| **Internal library clone groups** | **31** | — | Medium |
| **Doc coverage (overall)** | **90.67%** | ≥ 80% | ✅ |
| Doc coverage (methods) | 88.48% | ≥ 80% | ✅ |
| Circular dependencies | 0 | 0 | ✅ |
| Library panics (Go) | 0 | 0 | ✅ |
| Rust `.unwrap()` in `render-sys/` | 62 | 0 | ⚠️ |

### Complexity Hotspots (library code only)

| Function | File | CC |
|----------|------|----|
| `applyToTheme` | `theme.go` | 10 |
| `decodeVisuals` | `internal/x11/wire/setup.go` | 10 |
| `parseGeometryArgs` | `internal/wayland/output/protocol.go` | 9 |
| `decodeSetupFailure` | `internal/x11/wire/setup.go` | 9 |
| `Coalesce` | `internal/raster/displaylist/damage.go` | 8 |
| `validateAndNormalizeConfig` | `app.go` | 8 |
| `SetSize` | `app.go` | 8 |
| `FillRect` | `internal/raster/primitives/rect.go` | 8 |
| `ReadMessage` | `internal/wayland/client/connection.go` | 8 |
| `handleDownEvent` | `internal/wayland/input/touch.go` | 8 |
| `DecodeSetupReply` | `internal/x11/wire/setup.go` | 8 |

_(Build-tool outlier: `writeGlyphMetadata` in `cmd/gen-atlas/main.go` CC=11)_

### Top Internal Duplication Hotspots

| Clone | Location | Lines | Severity |
|-------|----------|-------|----------|
| Input event dispatch pattern | `wayland/input/{keyboard,pointer,touch}.go` | 21 | violation |
| X11 setup decode pattern | `x11/wire/setup.go` (3×) | 16–20 | warning |
| Render command builders | `render/commands.go` (5 pairs) | 7–18 | warning |
| Bezier sub-segment helpers | `raster/curves/bezier.go` (3×) | 7–16 | warning |
| Flex layout sub-axis | `ui/layout/flex.go` | 11 | warning |
| Widget state helpers | `ui/widgets/base.go` | 19 | warning |
| Buffer ring patterns | `buffer/ring.go` | 13 | warning |

### Package Coupling (notable)

| Package | Coupling Score | Notes |
|---------|---------------|-------|
| `wain` (root) | 7.5 | Expected — public API orchestrates all layers |
| `internal/demo` | 5.5 | High; demo helpers still reach into protocol internals |
| `internal/render/display` | 4.0 | Bridges Wayland/X11 + present + backend — acceptable |
| `internal/render/backend` | 2.5 | Rasterizer + render layer coupling |

---

## Implementation Steps

Dependencies flow top-to-bottom within each step. Steps 2–4 are independent of each other
and can proceed in parallel once Step 1 is complete.

---

### Step 1: Complete GPU-Rendered Frame Path  _(Priority 1 — Critical)_

**Deliverable**: A new `cmd/gpu-shader-demo` binary that renders a solid-colour triangle using
a WGSL shader compiled at runtime to native Intel EU or AMD RDNA instructions and displayed
on screen via the existing DMA-BUF/Present path.

**Why first**: This is the project's stated primary differentiator and the only goal flagged
as a critical gap in `ROADMAP.md`. All other steps are either independent or lower risk.

**Sub-tasks** (ordered by dependency):

#### 1a — `render-sys/src/submit.rs`: Shader-to-batch binding
- **File**: `render-sys/src/submit.rs` (new)
- **What**: Expose a `submit_shader_batch` function that accepts a compiled EU/RDNA binary
  (from `shader.rs` → `eu/mod.rs` or `rdna/mod.rs`) and a `BatchBuffer` (from `batch.rs`),
  binds the shader as the pipeline's kernel, and returns a submittable batch descriptor.
- **Dependencies**: `render-sys/src/shader.rs`, `render-sys/src/batch.rs`,
  `render-sys/src/eu/mod.rs`, `render-sys/src/rdna/mod.rs`
- **Acceptance**: `cargo test -p render-sys submit` passes; unit test confirms a
  `SolidFill` WGSL shader compiles and binds without panicking.
- **Validation**:
  ```bash
  cd render-sys && cargo test submit 2>&1 | grep -E "test .* ok|FAILED"
  ```

#### 1b — `render-sys/src/lib.rs`: FFI exports for shader submission
- **File**: `render-sys/src/lib.rs`
- **What**: Add `render_submit_shader_batch(...)` as a `#[no_mangle] extern "C"` function.
  Inputs: shader source pointer + length, GPU type enum, batch pointer. Returns status code.
- **Dependencies**: Step 1a
- **Acceptance**: `cargo build --target x86_64-unknown-linux-musl` succeeds; new symbol is
  visible in `librender_sys.a` via `nm`.
- **Validation**:
  ```bash
  cd render-sys && cargo build --target x86_64-unknown-linux-musl 2>&1 | tail -3
  nm render-sys/target/x86_64-unknown-linux-musl/debug/librender_sys.a | grep render_submit_shader_batch
  ```

#### 1c — `internal/render/binding.go`: Go CGO wrapper
- **File**: `internal/render/binding.go`
- **What**: Add `SubmitShaderBatch(shaderSrc []byte, gpuType GPUType, batch *BatchBuffer) error`
  as a CGO wrapper around `render_submit_shader_batch`. Follow the existing pattern from
  `render_submit_batch` binding.
- **Dependencies**: Step 1b
- **Acceptance**: `go build ./internal/render/` succeeds with CGO enabled.
- **Validation**:
  ```bash
  CGO_ENABLED=1 go build ./internal/render/ 2>&1
  ```

#### 1d — `internal/render/backend/gpu.go`: Wire shader into frame pipeline
- **File**: `internal/render/backend/gpu.go`
- **What**: Call `render.SubmitShaderBatch` inside `renderFrame` (or equivalent) after
  building the display list, replacing the current fixed-function batch construction.
  Load `render-sys/shaders/solid_fill.wgsl` at startup via `go:embed`.
- **Dependencies**: Step 1c
- **Acceptance**: `go build ./internal/render/backend/` succeeds; existing GPU backend
  tests pass (`go test ./internal/render/backend/`).
- **Validation**:
  ```bash
  go test ./internal/render/backend/ -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze ./internal/render/backend/ --skip-tests --format json | \
    jq '.complexity.average_function_complexity'
  ```

#### 1e — `cmd/gpu-shader-demo/main.go`: End-to-end demonstration
- **File**: `cmd/gpu-shader-demo/main.go` (new)
- **What**: A demo that creates a window, compiles `solid_fill.wgsl` via the new path, and
  renders a coloured triangle. Modelled after `cmd/gpu-triangle-demo` but using the
  shader-compiled path rather than fixed-function state.
- **Dependencies**: Step 1d
- **Acceptance**: Binary builds statically (`ldd` shows "not a dynamic executable"); runs on
  Intel Gen9+ or AMD RDNA2 hardware without crash. On CI without GPU, exits cleanly with
  "no GPU available" message.
- **Validation**:
  ```bash
  make gpu-shader-demo
  ldd bin/gpu-shader-demo | grep -q "not a dynamic" && echo STATIC_OK
  ```

#### 1f — `API.md`: Document shader → GPU → screen data flow
- **File**: `API.md`
- **What**: Add a "GPU Rendering Pipeline" section describing the path from WGSL source to
  EU/RDNA binary to batch submission to displayed frame, with a code example using the
  new `SubmitShaderBatch` API.
- **Dependencies**: Step 1d
- **Acceptance**: Section exists in `API.md`; covers Intel and AMD paths.

---

### Step 2: Stabilise Public API Surface  _(Priority 2)_

**Deliverable**: External Go projects can import `github.com/opd-ai/wain` and use
`wain.Button`, `wain.TextInput`, and `wain.ScrollContainer` without vendoring internal
packages. A working minimal example lives in `wain/example/`.

**Why next**: The public API layer (`app.go`, `publicwidget.go`, etc.) is complete, but the
concrete widget types remain inaccessible behind `internal/ui/widgets`. This is the
largest usability blocker for adopters.

**Sub-tasks** (ordered by dependency):

#### 2a — Audit root package exports
- **File**: `publicwidget.go`, `concretewidgets.go`, `widget.go`
- **What**: Enumerate all exported types and functions in the root `wain` package.
  Identify which widget behaviours are only accessible via `internal/ui/widgets` and
  not exposed in the public surface.
- **Acceptance**: A written list of gaps (comment block at top of `publicwidget.go`, or
  separate tracking note).
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json | \
    jq '[.functions[] | select(.package == "wain" and .is_exported)] | length'
  ```

#### 2b — Re-export `internal/ui/widgets` types
- **Files**: `publicwidget.go` (extend), or new `widgets.go` in root package
- **What**: Add type aliases or thin wrapper constructors for `Button`, `TextInput`, and
  `ScrollContainer` so consumers use `wain.NewButton(...)` etc. without importing
  `internal/ui/widgets`. Follow the existing pattern in `concretewidgets.go`.
- **Dependencies**: Step 2a
- **Acceptance**: `go vet ./...` clean; `go doc github.com/opd-ai/wain` shows all three
  widget constructors.
- **Validation**:
  ```bash
  go vet ./... 2>&1 | grep -v "^#" | head -20
  go doc github.com/opd-ai/wain | grep -E "func New(Button|TextInput|Scroll)"
  ```

#### 2c — `example/` directory: minimal working application
- **Files**: `example/hello/main.go` (new)
- **What**: A minimal working application — creates a window, adds a `Button`, handles the
  click event, prints to stdout — importable as `github.com/opd-ai/wain/example/hello`.
  Must build with `go build ./example/...`.
- **Dependencies**: Step 2b
- **Acceptance**: `go build ./example/...` succeeds; the example uses only the public API
  (no `internal/` imports).
- **Validation**:
  ```bash
  go build ./example/...
  grep "internal/" example/hello/main.go && echo "FAIL: internal import" || echo "OK"
  ```

#### 2d — Version bump
- **File**: `go.mod`, `README.md`, `RELEASE.md`
- **What**: Bump to `v0.2.0` with an "unstable but usable" notice. Tag the commit.
- **Dependencies**: Steps 2b + 2c
- **Acceptance**: `go list -m github.com/opd-ai/wain` returns `v0.2.0`; git tag exists.

---

### Step 3: Reduce Internal Duplication  _(Priority 3)_

**Deliverable**: Duplication ratio drops from 3.35% to below 2.5%; the 31 internal library
clone groups are reduced to ≤ 10 by extracting the highest-severity repeated patterns.

**Why here**: Duplication in `wayland/input/` and `x11/wire/` sits directly in the
critical path of Step 1 (input handling and GPU buffer sharing). Cleaning it now lowers
risk and cognitive load during GPU integration work.

**Sub-tasks** (ordered by impact, high first):

#### 3a — `internal/wayland/input/`: Extract shared event dispatch helper
- **Files**: `keyboard.go`, `pointer.go`, `touch.go` → new `dispatch.go` in same package
- **What**: The 15–21-line input dispatch pattern (serialise axis/button/motion event, call
  registered callbacks) appears in 4 locations across the three input files. Extract into
  `dispatchInputEvent(kind InputEventKind, data InputEventData)` or equivalent.
- **Acceptance**: `go test ./internal/wayland/input/` passes; clone groups in these files
  drop from 11 instances to ≤ 2.
- **Validation**:
  ```bash
  go test ./internal/wayland/input/ -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze ./internal/wayland/input/ --skip-tests --format json | \
    jq '.duplication.clone_pairs'
  ```

#### 3b — `internal/x11/wire/setup.go`: Extract decode sub-steps
- **Files**: `internal/x11/wire/setup.go`
- **What**: The 16–20-line X11 setup decode pattern appears 3× in `setup.go` (lines 340,
  361, 385). Extract `decodeScreen(r *Reader) Screen` and `decodeDepth(r *Reader) Depth`
  helpers to eliminate the repetition. This also directly reduces `decodeVisuals` CC from
  10 to < 7.
- **Acceptance**: `go test ./internal/x11/wire/` passes; `decodeVisuals` CC ≤ 7.
- **Validation**:
  ```bash
  go test ./internal/x11/wire/ -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze ./internal/x11/wire/ --skip-tests --format json | \
    jq '[.functions[] | select(.name == "decodeVisuals")] | .[0].complexity.cyclomatic'
  ```

#### 3c — `internal/render/commands.go`: Extract command builder helpers
- **Files**: `internal/render/commands.go`
- **What**: Five clone pairs (7–18-line blocks) exist in `commands.go` for repeated render
  command encoding patterns (lines 69, 79, 93, 98, 109, 125, 138, 160, 232, 240).
  Extract `encodeVertexCommand(...)` and `encodeStateCommand(...)` helpers.
- **Acceptance**: `go test ./internal/render/` passes; clone count in `commands.go` drops
  to 0.
- **Validation**:
  ```bash
  go test ./internal/render/ -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze ./internal/render/ --skip-tests --format json | \
    jq '.duplication.clone_pairs'
  ```

#### 3d — `internal/raster/curves/bezier.go`: Extract sub-segment helper
- **Files**: `internal/raster/curves/bezier.go`
- **What**: A 7–16-line curve sub-segment pattern appears 3× (lines 226, 235, 249, 258,
  267). Extract `subdivideBezier(p0, p1, p2, t float32) (Point, Point, Point)` helper.
- **Acceptance**: `go test ./internal/raster/curves/` passes; clone pairs in bezier.go → 0.
- **Validation**:
  ```bash
  go test ./internal/raster/curves/ -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze ./internal/raster/curves/ --skip-tests --format json | \
    jq '.duplication.clone_pairs'
  ```

#### 3e — `internal/demo/`: Consolidate demo setup boilerplate
- **Files**: `internal/demo/common.go` (new or extend `internal/demo/`)
- **What**: 6–8-line X11/Wayland setup patterns appear across `cmd/*/main.go` (35+ cmd
  instances). Extract `AutoConnect() (Display, error)` and `CreateWindow(display Display,
  title string, w, h int) (Window, error)` into `internal/demo/`. Refactor
  `cmd/amd-triangle-demo`, `cmd/decorations-demo`, and `cmd/example-app` to use them.
- **Acceptance**: Duplication ratio drops below 2.5%.
- **Validation**:
  ```bash
  go build ./cmd/... 2>&1
  go-stats-generator analyze . --skip-tests --format json | \
    jq '.duplication.duplication_ratio'
  ```

---

### Step 4: Reduce Complexity Hotspots  _(Priority 4)_

**Deliverable**: All library functions have CC ≤ 9; the two CC=10 functions in library code
drop to CC ≤ 7; `writeGlyphMetadata` (build tool) drops to CC ≤ 8.

**Scope**: Small — only 2 library functions above CC 9, plus the build-tool outlier.

**Sub-tasks** (ordered by severity):

#### 4a — `theme.go`: Refactor `applyToTheme` (CC=10 → ≤ 7)
- **File**: `theme.go`
- **What**: `applyToTheme` contains a large switch on theme-token type. Extract each case
  group into a dedicated helper: `applyColourTokens`, `applyTypographyTokens`,
  `applySpacingTokens`. Pattern matches the `handleEnterEvent`/`handleKeyEvent` refactoring
  convention already used in `wayland/input/`.
- **Acceptance**: `go test .` passes; `applyToTheme` CC ≤ 7.
- **Validation**:
  ```bash
  go test . -run TestTheme -v 2>&1 | grep -E "PASS|FAIL"
  go-stats-generator analyze . --skip-tests --format json | \
    jq '[.functions[] | select(.name == "applyToTheme")] | .[0].complexity.cyclomatic'
  ```

#### 4b — `internal/x11/wire/setup.go`: `decodeVisuals` (CC=10 → ≤ 7)
- **File**: `internal/x11/wire/setup.go`
- **Note**: This is covered by Step 3b; if Step 3b is completed first, this step is already
  satisfied. Listed here for tracking visibility.
- **Acceptance**: `decodeVisuals` CC ≤ 7 after Step 3b.

#### 4c — `cmd/gen-atlas/main.go`: `writeGlyphMetadata` (CC=11 → ≤ 8)
- **File**: `cmd/gen-atlas/main.go`
- **What**: Extract `iterateGlyphs(font *Font, callback func(Glyph))` and
  `encodeGlyphMetadata(g Glyph, w io.Writer) error`. Replace loop body with calls.
  This is a build tool, so no runtime risk.
- **Acceptance**: `go build ./cmd/gen-atlas/` passes; `writeGlyphMetadata` CC ≤ 8.
- **Validation**:
  ```bash
  go build ./cmd/gen-atlas/
  go-stats-generator analyze ./cmd/gen-atlas/ --skip-tests --format json | \
    jq '[.functions[] | select(.name == "writeGlyphMetadata")] | .[0].complexity.cyclomatic'
  ```

#### 4d — Library functions CC 8–9: targeted extraction
- **Files**: `app.go` (`validateAndNormalizeConfig`, `SetSize`),
  `internal/raster/primitives/rect.go` (`FillRect`),
  `internal/wayland/client/connection.go` (`ReadMessage`),
  `internal/wayland/output/protocol.go` (`parseGeometryArgs`)
- **What**: Apply the extract-helper pattern to each. Each has 1–3 natural split points
  (validation vs. execution, or per-case handlers).
- **Acceptance**: All targeted functions reach CC ≤ 7; all existing tests pass.
- **Validation**:
  ```bash
  go test ./... 2>&1 | tail -20
  go-stats-generator analyze . --skip-tests --format json | \
    jq '[.functions[] | select(.complexity.cyclomatic > 7)] | length'
  ```

---

### Step 5: Rust FFI Safety Audit  _(from READINESS_SUMMARY.md — HIGH risk)_

**Deliverable**: All `unsafe` `.unwrap()` calls at the FFI boundary in `render-sys/` are
replaced with `Result`-returning code; panics across the CGO boundary are eliminated.

**Scope**: Medium — 62 `.unwrap()` calls; priority order by call count per file.

**Sub-tasks** (ordered by unwrap count, highest first):

#### 5a — `internal/render/eu/mod.rs` (23 unwraps)
- **What**: Audit each `.unwrap()`. Where the value can legitimately be absent (register
  allocation failures, instruction lowering failures), replace with
  `ok_or(EuError::...)?.` and propagate to a `Result<T, EuError>` return type.
  Expose the error to Go via a new status enum in the FFI layer.
- **Acceptance**: `cargo test -p render-sys eu` passes; 0 `.unwrap()` remain in `eu/mod.rs`.
- **Validation**:
  ```bash
  cd render-sys && cargo test eu 2>&1 | grep -E "test .* ok|FAILED"
  grep -c "\.unwrap()" render-sys/src/eu/mod.rs
  ```

#### 5b — `render-sys/src/shader.rs` (11 unwraps)
- **What**: Replace naga parse/validate `.unwrap()` calls with `Result` propagation.
  The FFI entry point should return a structured error code rather than panicking on
  invalid WGSL input.
- **Acceptance**: `cargo test -p render-sys shader` passes; fuzzing with malformed WGSL
  returns error code rather than SIGABRT.
- **Validation**:
  ```bash
  cd render-sys && cargo test shader 2>&1 | grep -E "test .* ok|FAILED"
  grep -c "\.unwrap()" render-sys/src/shader.rs
  ```

#### 5c — Remaining render-sys files (`eu/types.rs`, `batch.rs`, etc.)
- **What**: Sweep remaining 28 `.unwrap()` calls across `eu/types.rs` (11), `batch.rs` (6),
  `eu/regalloc.rs` (6), `shaders.rs` (1). Apply same Result-propagation pattern.
- **Acceptance**: Total `.unwrap()` count in production render-sys code (excluding
  `gpu_test.rs`) drops to 0.
- **Validation**:
  ```bash
  grep -r "\.unwrap()" render-sys/src/ --include="*.rs" | grep -v gpu_test | wc -l
  # Target: 0
  cd render-sys && cargo test 2>&1 | grep -E "test result"
  ```

---

## Full Validation Sweep

After all steps are complete, run this sequence to confirm targets are met:

```bash
# 1. Static build still works
make wain 2>&1 | tail -3
ldd bin/wain | grep -q "not a dynamic" && echo "STATIC: OK"

# 2. All Go tests pass
go test ./... 2>&1 | grep -E "FAIL|ok" | tail -20

# 3. Complexity: 0 library functions above CC 9
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select(.complexity.cyclomatic > 9 and (.file | contains("/cmd/") | not))] | length'
# Target: 0

# 4. Duplication below 2.5%
go-stats-generator analyze . --skip-tests --format json --sections duplication | \
  jq '.duplication.duplication_ratio'
# Target: < 0.025

# 5. Doc coverage still above 90%
go-stats-generator analyze . --skip-tests --format json --sections documentation | \
  jq '.documentation.coverage.overall'
# Target: >= 90

# 6. Rust unwraps in production code: 0
grep -r "\.unwrap()" render-sys/src/ --include="*.rs" | grep -v gpu_test | wc -l
# Target: 0

# 7. Rust tests pass
cd render-sys && cargo test --target x86_64-unknown-linux-musl 2>&1 | tail -5
```

---

## Out of Scope (this plan)

The following items appear in the backlog but are deferred:

| Item | Reason |
|------|--------|
| AT-SPI2 accessibility (ROADMAP Priority 5) | Requires D-Bus runtime; architectural trade-off; explicitly deferred in `ACCESSIBILITY.md` |
| Cross-axis alignment in layout (`layout.go:205` NOTE) | Flagged as future-phase work in existing code comment |
| `cmd/resource-demo` `LoadImageFromReader` exposure | Noted in demo comment as "not exposed yet"; depends on public API work in Step 2 |

---

## Gaps Document

### Metrics Gaps Not Tied to a Roadmap Priority

The following metric findings have no corresponding backlog item. They are tracked here
for future planning cycles:

1. **`internal/ui/widgets/base.go` 19-line clone** (lines 813, 874) — widget state helper
   duplication; low risk but worth extracting in a follow-up.
2. **`internal/ui/layout/flex.go` 11-line clone** (lines 326, 357) — sub-axis calculation
   repeated; refactor candidate if layout engine is extended.
3. **`internal/buffer/ring.go` 13-line clone** (lines 195, 214) — buffer acquire/release
   symmetry; consider a `withBuffer(fn)` helper.
4. **`internal/x11/client/connection.go` ↔ `internal/x11/dri3/extension.go` 8-line clone** —
   X11 byte-read pattern shared across packages; a shared `x11/wire` helper would eliminate it.
5. **Method doc coverage at 88.48%** — 11.5% of methods lack doc comments; consider a
   `go generate` lint pass to catch regressions.
