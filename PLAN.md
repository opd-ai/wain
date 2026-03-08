# Implementation Plan: Code Quality & Maintenance Enhancement

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust (Intel + AMD backends)
- **Current milestone**: All 8 roadmap phases complete — project ready for quality refinement and public API (Phases 9-10)
- **Estimated Scope**: Medium (15 complexity hotspots, 4% duplication, 90% doc coverage)

## Metrics Summary
| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Functions > complexity 9.0 | **41** | <15 | ⚠️ Medium |
| Functions > complexity 15.0 | **0** | <5 | ✅ Good |
| Duplication ratio | **4.03%** | <3% | ⚠️ Medium |
| Duplicate lines | 861 | — | — |
| Clone pairs | 45 | — | — |
| Doc coverage (overall) | **90.15%** | >90% | ✅ Good |
| Doc coverage (methods) | **87.24%** | >90% | ⚠️ Near threshold |
| High coupling packages | 7/35 | — | — |

### Complexity Hotspots (>10.0 overall)
| Function | Package | File | Complexity |
|----------|---------|------|------------|
| FillRoundedRect | core | internal/raster/core/rect.go:47 | 14.5 |
| lineCoverage | core | internal/raster/core/line.go:60 | 14.0 |
| handleGeometry | output | internal/wayland/output/output.go:149 | 13.5 |
| main | main | cmd/auto-render-demo/main.go:23 | 13.2 |
| drawGlyph | text | internal/raster/text/text.go:47 | 13.2 |
| createBufferRing | main | cmd/double-buffer-demo/main.go:144 | 12.7 |
| encodeArgument | wire | internal/wayland/wire/wire.go:381 | 12.7 |
| Coalesce | displaylist | internal/raster/displaylist/damage.go:53 | 12.4 |
| AttachAndDisplayBuffer | demo | internal/demo/shm.go:19 | 12.2 |
| FillRect | core | internal/raster/core/rect.go:7 | 11.9 |
| Blit | composite | internal/raster/composite/composite.go:35 | 11.9 |
| SetupWaylandGlobals | demo | internal/demo/wayland.go:41 | 10.9 |
| setupX11Context | main | cmd/x11-dmabuf-demo/main.go:159 | 10.9 |
| MakePair | socket | internal/wayland/socket/socket.go:236 | 10.9 |
| AcquireForWriting | buffer | internal/buffer/ring.go:160 | 10.6 |

### Top Duplication Violations
| Lines | Type | Locations |
|-------|------|-----------|
| 34 | renamed | cmd/gpu-triangle-demo/main.go (293-326, 421-454) |
| 29 | renamed | internal/render/display/wayland.go:82-110, x11.go:142-170 |
| 26 | renamed | cmd/dmabuf-demo/main.go (151-176, 160-185) |
| 25 | exact | cmd/gpu-triangle-demo/main.go (3 locations) |
| 25 | renamed | cmd/perf-demo/main.go (4 locations) |

### Package Coupling Analysis
| Package | Coupling Score | Dependencies |
|---------|----------------|--------------|
| main (cmd/*) | 10.0 | 21 |
| display | 3.5 | 7 |
| demo | 3.5 | 7 |
| backend | 2.5 | 5 |
| consumer | 2.0 | 4 |
| decorations | 2.0 | 4 |
| widgets | 2.0 | 4 |

---

## Implementation Steps

### Step 1: Extract Render Presentation Logic ✅ COMPLETE
- **Deliverable**: Create shared `internal/render/present/present.go` to deduplicate 29-line render-and-present pattern across Wayland and X11 display modules
- **Dependencies**: None
- **Files**: 
  - `internal/render/display/wayland.go` (lines 82-110)
  - `internal/render/display/x11.go` (lines 142-170)
- **Acceptance**: Duplication ratio reduced by ≥0.3%, no new complexity hotspots
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication | {ratio: .duplication_ratio, lines: .duplicated_lines}'
  ```

### Step 2: Refactor FillRoundedRect (Highest Complexity) ✅ COMPLETE
- **Deliverable**: Extract corner-drawing and edge-filling into helper functions to reduce cyclomatic complexity
- **Dependencies**: None
- **Files**: `internal/raster/core/rect.go` (line 47, complexity 14.5)
- **Acceptance**: `FillRoundedRect` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "FillRoundedRect")] | .[0].complexity.overall'
  ```

### Step 3: Refactor lineCoverage Function ✅ COMPLETE
- **Deliverable**: Simplify line coverage calculation by extracting slope-handling and pixel-iteration logic
- **Dependencies**: None
- **Files**: `internal/raster/core/line.go` (line 60, complexity 14.0 → 4.4)
- **Acceptance**: `lineCoverage` complexity ≤9.0 ✅ (achieved 4.4)
- **Implementation**: Extracted three helper functions:
  - `perpendicularCoverage` (complexity 3.1)
  - `startCapCoverage` (complexity 5.7)
  - `endCapCoverage` (complexity 5.7)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "lineCoverage")] | .[0].complexity.overall'
  ```
  Result: 4.4 (68.6% improvement from baseline)

### Step 4: Deduplicate Demo Setup Patterns ✅ COMPLETE
- **Deliverable**: Create shared demo setup helpers in `internal/demo/` to eliminate GPU triangle/dmabuf/perf demo duplication (25-34 line blocks)
- **Dependencies**: None
- **Files**:
  - `cmd/gpu-triangle-demo/main.go` (3+ duplicated blocks)
  - `cmd/dmabuf-demo/main.go`
  - `cmd/perf-demo/main.go`
- **Acceptance**: Clone pairs reduced by ≥5, duplication ratio <3.5%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication | {pairs: .clone_pairs, ratio: .duplication_ratio}'
  ```

### Step 5: Simplify Wayland Output Event Handler
- **Deliverable**: Refactor `handleGeometry` in output package to extract field assignments into a struct builder
- **Dependencies**: None
- **Files**: `internal/wayland/output/output.go` (line 149, complexity 13.5)
- **Acceptance**: `handleGeometry` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "handleGeometry")] | .[0].complexity.overall'
  ```

### Step 6: Refactor encodeArgument Switch Statement
- **Deliverable**: Replace type-switch with map-based encoder dispatch or extract type-specific encoders
- **Dependencies**: None
- **Files**: `internal/wayland/wire/wire.go` (line 381, complexity 12.7)
- **Acceptance**: `encodeArgument` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "encodeArgument")] | .[0].complexity.overall'
  ```

### Step 7: Improve Method Documentation Coverage
- **Deliverable**: Add godoc comments to undocumented exported methods (target: >90% method coverage)
- **Dependencies**: Steps 1-6 (avoid documenting code that will be refactored)
- **Files**: Focus on high-export packages: `backend`, `widgets`, `display`
- **Acceptance**: Method documentation coverage ≥90%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation | \
    jq '.documentation.coverage.methods'
  ```

### Step 8: Extract Damage Coalescing Logic
- **Deliverable**: Simplify `Coalesce` function by extracting overlap detection and region merging into helpers
- **Dependencies**: None (can run parallel with Steps 2-6)
- **Files**: `internal/raster/displaylist/damage.go` (line 53, complexity 12.4)
- **Acceptance**: `Coalesce` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "Coalesce")] | .[0].complexity.overall'
  ```

---

## Success Criteria

| Metric | Current | Target | Validation |
|--------|---------|--------|------------|
| Functions > complexity 9.0 | 41 | ≤20 | `jq '[.functions[] \| select(.complexity.overall > 9.0)] \| length'` |
| Duplication ratio | 4.03% | <3.0% | `jq '.duplication.duplication_ratio'` |
| Doc coverage (methods) | 87.24% | ≥90% | `jq '.documentation.coverage.methods'` |
| Clone pairs | 45 | ≤35 | `jq '.duplication.clone_pairs'` |

---

## Deferred Work (Out of Scope)

These items from the roadmap are **not addressed** by this plan:

1. **VT Switch Handling** (Phase 7.2 deferred) — Requires kernel/terminal signal integration
2. **DPMS Handling** (Phase 7.2 deferred) — Requires compositor event integration
3. **AT-SPI2 Accessibility** (Phase 8.4 documented) — Documented limitation, 6-week effort
4. **GPU Backend Screenshot Tests** (screenshot_test.go TODO) — Requires Phase 5 GPU rendering maturity

---

## Public API Implementation Plan (Phases 9-10)

This section defines the concrete implementation steps for the public API
described in ROADMAP.md Phases 9-10. Steps are ordered by dependency.

### Step 9: Create Public Package Structure
- **Deliverable**: Create root-level Go package files that will form the public API surface. Create `wain.go`, `app.go`, `window.go`, `widget.go`, `event.go`, `theme.go`, `color.go` at the module root alongside `go.mod`. These files expose public types that delegate to `internal/` packages.
- **Dependencies**: Steps 1-8 (code quality improvements reduce friction during promotion)
- **Files**:
  - `wain.go` (package doc, version)
  - `app.go` (App type, NewApp, Run, Quit)
  - `window.go` (Window type, WindowConfig)
  - `widget.go` (Widget/Container interfaces, Size struct)
  - `event.go` (public event types)
  - `theme.go` (Theme struct, built-in themes)
  - `color.go` (Color type, RGB/RGBA helpers)
- **Acceptance**: `go doc github.com/opd-ai/wain` shows public types; package compiles with no internal/ leaks in public API signatures
- **Validation**:
  ```bash
  go build github.com/opd-ai/wain
  go doc github.com/opd-ai/wain | head -50
  # Verify no internal/ types appear in the public API:
  go doc github.com/opd-ai/wain 2>&1 | grep -c 'internal/' | xargs test 0 -eq
  ```

### Step 10: Implement App Lifecycle & Window Abstraction
- **Deliverable**: Implement `App` struct with display-server auto-detection, event loop, and graceful shutdown. Implement `Window` struct wrapping internal Wayland/X11 windows with title, size, and close handling. Bridge to existing `internal/render/backend.NewRenderer()` for auto-detection.
- **Dependencies**: Step 9
- **Files**:
  - `app.go` (App.Run event loop, platform detection)
  - `window.go` (Window creation, resize, close)
  - `internal/render/display/` (existing, consumed by bridge)
  - `internal/wayland/client/` (existing, consumed by bridge)
  - `internal/x11/` (existing, consumed by bridge)
- **Acceptance**: `wain.NewApp()` successfully opens a window on both X11 and Wayland
- **Validation**:
  ```bash
  go build ./cmd/example-app/
  # Manual: run on Wayland and X11, verify window appears and closes cleanly
  ```

### Step 11: Implement Public Event System
- **Deliverable**: Define public event types (`PointerEvent`, `KeyEvent`, `WindowEvent`, `TouchEvent`) and event dispatch from the internal Wayland/X11 event loops to public callbacks. Implement hit-testing to route pointer events to the correct widget. Implement keyboard focus management with tab-order traversal.
- **Dependencies**: Step 10
- **Files**:
  - `event.go` (event type definitions)
  - `app.go` (event dispatch integration)
  - `widget.go` (hit-test routing, focus management)
- **Acceptance**: Pointer and keyboard events dispatch correctly to widgets; tab key moves focus between interactive widgets
- **Validation**:
  ```bash
  go test -run TestEventDispatch -v github.com/opd-ai/wain
  go test -run TestKeyboardFocus -v github.com/opd-ai/wain
  ```

### Step 12: Implement Render Integration Bridge
- **Deliverable**: Create internal bridge (`render_bridge.go`) that walks the public widget tree, emits DisplayList commands, submits to the Renderer (GPU or software), and presents to the compositor. Integrate damage tracking so only changed widgets are re-rendered. Manage frame lifecycle (acquire buffer → render → present → release).
- **Dependencies**: Steps 10, 11
- **Files**:
  - `render_bridge.go` (new, internal bridge logic)
  - `internal/render/backend/interface.go` (existing Renderer)
  - `internal/raster/displaylist/` (existing DisplayList)
  - `internal/buffer/ring.go` (existing buffer management)
- **Acceptance**: Widgets render to screen through the display list pipeline with damage tracking
- **Validation**:
  ```bash
  go test -run TestRenderBridge -v github.com/opd-ai/wain
  # Verify damage tracking reduces redundant rendering:
  go test -run TestDamageTracking -v github.com/opd-ai/wain
  ```

### Step 13: Implement Public Widget Types
- **Deliverable**: Create public widget types wrapping internal implementations:
  - `Panel` (wraps `pctwidget.Panel`) — styled container with percentage-based sizing
  - `Button` (wraps `widgets.Button`) — clickable with text and OnClick
  - `Label` — static text display (new, thin wrapper over text rendering)
  - `TextInput` (wraps `widgets.TextInput`) — editable text field
  - `ScrollView` (wraps `widgets.ScrollContainer`) — scrollable area
  - `Image` — displays loaded image resources (new)
  - `Spacer` — invisible layout spacer (new, trivial)
- All widgets accept `wain.Size{Width, Height float64}` for percentage-based sizing.
- **Dependencies**: Steps 9, 12
- **Files**:
  - `panel.go`, `button.go`, `label.go`, `text_input.go`, `scroll_view.go`, `image.go`, `spacer.go`
- **Acceptance**: Each widget type renders correctly with percentage sizing; all satisfy the `Widget` interface
- **Validation**:
  ```bash
  go test -run TestPanel -v github.com/opd-ai/wain
  go test -run TestButton -v github.com/opd-ai/wain
  go test -run TestWidgetInterface -v github.com/opd-ai/wain
  ```

### Step 14: Implement Container Types (Row, Column, Stack, Grid)
- **Deliverable**: Create public container types that arrange children using percentage-based auto-layout:
  - `Row` — horizontal flow (delegates to `AutoLayout` with `FlowRow`)
  - `Column` — vertical flow (delegates to `AutoLayout` with `FlowColumn`)
  - `Stack` — layered overlay (new, z-order by add order)
  - `Grid` — fixed-column grid (new, divides space into equal cells)
- Containers support `Padding`, `Gap`, and `Align` properties.
- **Dependencies**: Step 13
- **Files**:
  - `row.go`, `column.go`, `stack.go`, `grid.go`
  - `internal/ui/pctwidget/autolayout.go` (existing, consumed)
  - `internal/ui/layout/layout.go` (existing, consumed for flex semantics)
- **Acceptance**: Nested container layouts render correctly; percentage sizes resolve relative to parent container bounds
- **Validation**:
  ```bash
  go test -run TestRow -v github.com/opd-ai/wain
  go test -run TestColumn -v github.com/opd-ai/wain
  go test -run TestGrid -v github.com/opd-ai/wain
  go test -run TestNestedLayout -v github.com/opd-ai/wain
  ```

### Step 15: Implement Theming & Styling API
- **Deliverable**: Create public `Theme` struct and built-in themes (DefaultDark, DefaultLight, HighContrast). Implement theme inheritance — widgets inherit from parent container unless overridden. Expose `Style` interface for per-widget customization. Bridge to existing `pctwidget.Style` and `widgets.Theme`.
- **Dependencies**: Step 13
- **Files**:
  - `theme.go` (Theme struct, built-in themes, inheritance)
  - `color.go` (Color type, RGB/RGBA/Hex constructors)
  - `internal/ui/pctwidget/style.go` (existing, bridged)
  - `internal/ui/widgets/widgets.go` (existing Theme, bridged)
- **Acceptance**: Changing the app theme re-renders all widgets with new colors; per-widget style override works
- **Validation**:
  ```bash
  go test -run TestThemeInheritance -v github.com/opd-ai/wain
  go test -run TestStyleOverride -v github.com/opd-ai/wain
  go test -run TestBuiltInThemes -v github.com/opd-ai/wain
  ```

### Step 16: Implement Resource Management
- **Deliverable**: Implement font loading (LoadFont wrapping SDF atlas generation), image loading (LoadImage wrapping PNG/JPEG decode + GPU atlas upload), and embedded default font. All resources are reference-counted and freed on App destruction.
- **Dependencies**: Steps 10, 12
- **Files**:
  - `font.go` (LoadFont, default font embedding)
  - `image.go` (LoadImage, image resource type — note: distinct from Image widget)
  - `internal/render/atlas/` (existing atlas management)
  - `internal/raster/text/` (existing SDF text rendering)
- **Acceptance**: Text renders with default font without any developer setup; custom fonts load from file path
- **Validation**:
  ```bash
  go test -run TestDefaultFont -v github.com/opd-ai/wain
  go test -run TestLoadFont -v github.com/opd-ai/wain
  go test -run TestLoadImage -v github.com/opd-ai/wain
  ```

### Step 17: Implement State & Callback System
- **Deliverable**: Implement callback registration for all interactive widgets (OnClick, OnChange, OnScroll, etc.). Implement `wain.Notify()` for safe cross-goroutine UI updates via a channel read in the event loop. Ensure all callbacks execute on the UI goroutine.
- **Dependencies**: Steps 11, 13
- **Files**:
  - `app.go` (Notify channel integration in event loop)
  - `button.go`, `text_input.go`, `scroll_view.go` (callback registration)
- **Acceptance**: Callbacks fire on the UI goroutine; `Notify()` from a background goroutine updates UI without races
- **Validation**:
  ```bash
  go test -run TestCallbacks -v github.com/opd-ai/wain
  go test -run TestNotifyCrossGoroutine -v -race github.com/opd-ai/wain
  ```

### Step 18: Build & Distribution Verification
- **Deliverable**: Verify the full `go get` → `go build` pipeline works from scratch using pre-built Rust artifacts (no `go generate` required in consumer modules). Add CI jobs for clean-environment builds. Create and publish pre-built release assets for the Rust static library. Provide a separate generator tool (e.g., `cmd/wain-build`) that can be invoked from within a consuming project when regeneration is needed. Update README.md with getting-started instructions for the public API and clear guidance on when/how to run the generator.
- **Dependencies**: All previous steps
- **Files**:
  - `scripts/build-rust.sh` (existing, verify)
  - `internal/render/generate.go` (existing go:generate, used only in this repo / generator tool, not via `go generate github.com/opd-ai/wain/...`)
  - `cmd/wain-build/main.go` (new, optional Rust rebuild tool for consumers)
  - `.github/workflows/ci.yml` (add public API build test)
  - `README.md` (update with public API getting-started and generator usage)
- **Acceptance**: `go get github.com/opd-ai/wain && go build ./...` succeeds on a clean x86_64 Linux environment without requiring `go generate` on the `github.com/opd-ai/wain/...` import path; documentation shows how to optionally run the generator tool from a consumer module.
- **Validation**:
  ```bash
  # CI job verifies clean build from a fresh consumer module:
  mkdir /tmp/test-app && cd /tmp/test-app
  go mod init testapp
  go get github.com/opd-ai/wain
  cat > main.go << 'EOF'
  package main
  import "github.com/opd-ai/wain"
  func main() { wain.NewApp().Run() }
  EOF
  go build -o testapp .
  ldd testapp 2>&1 | grep -q "not a dynamic executable"
  # Optional: if rebuilding Rust backend from source is needed
  # (run from project root):
  # go install github.com/opd-ai/wain/cmd/wain-build@latest
  # wain-build
  ```

### Step 19: Reference Application & Documentation
- **Deliverable**: Create `cmd/example-app/` demonstrating the full public API: multi-panel layout with header/sidebar/content/footer, buttons, text input, scroll view, theme switching, image display. Write GETTING_STARTED.md and WIDGETS.md documentation. Ensure 100% godoc coverage for all public types.
- **Dependencies**: All previous steps
- **Files**:
  - `cmd/example-app/main.go` (reference application, ~100-150 LOC)
  - `GETTING_STARTED.md` (new)
  - `WIDGETS.md` (new)
  - All public API files (godoc coverage)
- **Acceptance**: Example app compiles, runs, and demonstrates all public widget types on both X11 and Wayland; godoc shows complete documentation
- **Validation**:
  ```bash
  go build ./cmd/example-app/
  go doc github.com/opd-ai/wain | wc -l  # Should be substantial
  # Verify godoc coverage:
  go doc github.com/opd-ai/wain.App
  go doc github.com/opd-ai/wain.Window
  go doc github.com/opd-ai/wain.Widget
  go doc github.com/opd-ai/wain.Panel
  go doc github.com/opd-ai/wain.Button
  go doc github.com/opd-ai/wain.Row
  go doc github.com/opd-ai/wain.Theme
  ```

### Step 20: Integration Testing & v0.1.0 Tag
- **Deliverable**: Screenshot comparison tests for the reference application across all backends. API contract tests verifying all widgets satisfy public interfaces. Build pipeline tests on CI matrix (x86_64, aarch64). Keyboard navigation tests. Tag v0.1.0 release.
- **Dependencies**: Step 19
- **Files**:
  - `wain_test.go` (API contract tests)
  - `internal/integration/public_api_test.go` (screenshot comparisons)
  - `.github/workflows/ci.yml` (matrix build tests)
- **Acceptance**: All tests pass; v0.1.0 tagged; `go get github.com/opd-ai/wain@v0.1.0` resolves
- **Validation**:
  ```bash
  go test -v github.com/opd-ai/wain/...
  make test-go
  git tag v0.1.0
  ```

---

## Public API Success Criteria

| Metric | Target | Validation |
|--------|--------|------------|
| Public types with godoc | 100% | `go doc` shows all types |
| Hello World LOC | ≤20 lines | Manual review of minimal example |
| `go get` + `go build` works | ✅ | CI clean-build job |
| Binary is fully static | ✅ | `ldd` reports "not a dynamic executable" |
| X11 + Wayland parity | ✅ | Screenshot comparison tests |
| GPU + Software parity | ✅ | Screenshot comparison tests |
| No internal/ in public API | 0 references | `go doc` grep check |
| Widget types available | ≥7 | Panel, Button, Label, TextInput, ScrollView, Image, Spacer |
| Container types available | ≥4 | Row, Column, Stack, Grid |
| Built-in themes | ≥3 | DefaultDark, DefaultLight, HighContrast |

---

## Metrics Baseline Reference

Generated: 2026-03-08  
Tool: go-stats-generator 1.0.0  
Files analyzed: 137 Go files  
Total LOC: 9,409  
Total functions: 362  
Total methods: 626  

```bash
# Regenerate baseline
go-stats-generator analyze . --skip-tests --format json --output metrics.json \
  --sections functions,duplication,documentation,packages,patterns
```
