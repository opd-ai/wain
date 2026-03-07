# Implementation Plan: Phase 1 — Software Rendering Path

## Project Context
- **What it does**: Rust/Go interface to Mesa, Vulkan — a statically-compiled Go UI toolkit with hardware-accelerated rendering
- **Current milestone**: Phase 1 — Software Rendering Path (Wayland/X11 protocol implementation + software rasterizer)
- **Estimated Scope**: Large (30,000–50,000 LoC for Go protocols + UI + software renderer per ROADMAP.md)

## Current State Assessment

### Completed Work (Phase 0)
The project has completed Phase 0 (Foundation & Toolchain Setup):
- ✅ Go module with CGO_ENABLED=1 linking static Rust `.a` archive
- ✅ Binary is fully static (musl-based, verified via `ldd`)
- ✅ C ABI boundary defined and validated (`render_add`, `render_version`)
- ✅ CI checks static linking on every commit
- ✅ Makefile automates the full build pipeline

### Metrics Summary (Go Codebase)
| Metric | Current Value | Assessment |
|--------|---------------|------------|
| Functions above complexity 9.0 | 0 | N/A (codebase is minimal) |
| Duplication ratio | 0% | Excellent |
| Doc coverage | 100% (2/2 functions documented) | Excellent |
| Package coupling | Low (render: 0.0, main: 0.5) | Excellent |
| Total Go LoC | 40 | Foundation only |
| Total Rust LoC | ~45 | Foundation only |

**Note**: The codebase is at the foundational stage. Metrics reflect a healthy starting point with no technical debt.

## First Incomplete Milestone: Phase 1 — Software Rendering Path

Per ROADMAP.md, Phase 1 consists of 5 sub-phases that must be completed before GPU acceleration work can begin. This phase establishes the full UI pipeline with CPU rendering, serving as both the fallback path and test harness.

## Implementation Steps

### Step 1: Wayland Wire Protocol Foundation (Phase 1.1a)
- **Deliverable**: Pure Go implementation of Wayland wire format — header parsing, argument marshaling/unmarshaling
- **Dependencies**: None (standalone)
- **Scope**: ~1,000–2,000 LoC
- **Files to create**: `internal/wayland/wire/` package
- **Acceptance**: 
  - Functions for `encode`/`decode` with cyclomatic complexity < 9
  - No code duplication across similar marshaling functions
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "wire") | select(.complexity.cyclomatic > 9)'
  ```

### Step 2: Wayland FD Passing (Phase 1.1b)
- **Deliverable**: SCM_RIGHTS fd passing implementation for shared memory buffers
- **Dependencies**: Step 1 (wire protocol)
- **Scope**: ~300–500 LoC
- **Files to create**: `internal/wayland/socket/` package
- **Acceptance**:
  - Single responsibility functions with complexity < 9
  - Documented public API (doc coverage 100%)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation | \
    jq '.documentation.coverage.functions'
  ```

### Step 3: Core Wayland Objects (Phase 1.1c)
- **Deliverable**: Implementation of `wl_display`, `wl_registry`, `wl_compositor`, `wl_surface`
- **Dependencies**: Steps 1, 2
- **Scope**: ~2,000–3,000 LoC
- **Files to create**: `internal/wayland/client/` package
- **Acceptance**:
  - Package cohesion score > 0.5
  - No circular dependencies
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections packages | \
    jq '.packages[] | select(.name == "client") | .cohesion_score'
  ```

### Step 4: Wayland SHM Support (Phase 1.1d)
- **Deliverable**: `wl_shm`, `wl_shm_pool`, `wl_buffer` + `memfd_create` syscall wrapper
- **Dependencies**: Steps 1, 2, 3
- **Scope**: ~1,000–1,500 LoC
- **Files to create**: `internal/wayland/shm/` package
- **Acceptance**:
  - All exported functions documented
  - No functions with cognitive complexity > 15
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "shm") | select(.complexity.cognitive > 15)'
  ```

### Step 5: XDG Shell Protocol (Phase 1.1e)
- **Deliverable**: `xdg_wm_base`, `xdg_surface`, `xdg_toplevel` implementation
- **Dependencies**: Steps 3, 4
- **Scope**: ~1,500–2,000 LoC
- **Files to create**: `internal/wayland/xdg/` package
- **Milestone**: Open a window and display a solid color on Wayland compositor
- **Acceptance**:
  - Functional test: window opens on sway/weston
  - Package coupling score < 0.7
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections packages | \
    jq '.packages[] | select(.name == "xdg") | .coupling_score'
  ```

### Step 6: X11 Connection & Core Protocol (Phase 1.2a)
- **Deliverable**: X11 connection setup, authentication, core requests (CreateWindow, MapWindow)
- **Dependencies**: None (parallel to Wayland work)
- **Scope**: ~2,000–3,000 LoC
- **Files to create**: `internal/x11/protocol/` package
- **Acceptance**:
  - No duplication with Wayland wire format (shared abstractions if possible)
  - Complexity distribution: 80% of functions below complexity 5
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.package == "protocol") | .complexity.overall < 5] | map(select(.)) | length / length * 100'
  ```

### Step 7: X11 Graphics Context & Blitting (Phase 1.2b)
- **Deliverable**: CreateGC, PutImage, CreatePixmap, MIT-SHM extension
- **Dependencies**: Step 6
- **Scope**: ~1,500–2,000 LoC
- **Files to create**: `internal/x11/gc/` package
- **Milestone**: Open a window and display a solid color on X11
- **Acceptance**:
  - Functional test: window opens on X11
  - No magic numbers (use named constants)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections patterns | \
    jq '.patterns.anti_patterns.magic_numbers // []'
  ```

### Step 8: Input Handling — Wayland (Phase 1.3a)
- **Deliverable**: `wl_seat`, `wl_pointer`, `wl_keyboard`, `wl_touch`, basic xkb keymap parsing
- **Dependencies**: Step 5
- **Scope**: ~2,000–2,500 LoC
- **Files to create**: `internal/wayland/input/` package
- **Acceptance**:
  - No deeply nested functions (max nesting depth 4)
  - Event handling functions with complexity < 9
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "input") | select(.complexity.nesting_depth > 4)'
  ```

### Step 9: Input Handling — X11 (Phase 1.3b)
- **Deliverable**: KeyPress, KeyRelease, ButtonPress, ButtonRelease, MotionNotify, Expose, ConfigureNotify events
- **Dependencies**: Step 7
- **Scope**: ~1,500–2,000 LoC
- **Files to create**: `internal/x11/events/` package
- **Acceptance**:
  - Consistent event handler signature pattern
  - No code duplication between similar event handlers
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication.duplication_ratio'
  ```

### Step 10: Software Rasterizer Core (Phase 1.4a)
- **Deliverable**: Tile-based 2D rasterizer foundation — filled rectangles, rounded rectangles, line segments
- **Dependencies**: None (can proceed in parallel with protocol work)
- **Scope**: ~3,000–4,000 LoC
- **Files to create**: `internal/raster/core/` package
- **Acceptance**:
  - Algorithm functions with complexity < 15
  - ARGB8888 buffer operations documented
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "core") | select(.complexity.cyclomatic > 15)'
  ```

### Step 11: Software Rasterizer — Curves & Arcs (Phase 1.4b)
- **Deliverable**: Quadratic/cubic Bezier curves, arc fills
- **Dependencies**: Step 10
- **Scope**: ~1,500–2,000 LoC
- **Files to extend**: `internal/raster/curves/` package
- **Acceptance**:
  - Mathematical functions well-documented
  - Unit tests for edge cases (degenerate curves)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation | \
    jq '.documentation.coverage.functions'
  ```

### Step 12: Software Rasterizer — Text Rendering (Phase 1.4c)
- **Deliverable**: SDF-based text rendering with pre-baked SDF font atlas
- **Dependencies**: Step 10
- **Scope**: ~2,000–3,000 LoC
- **Files to create**: `internal/raster/text/` package, embedded font atlas
- **Acceptance**:
  - Font atlas embedded as `//go:embed` resource
  - Glyph lookup functions with O(1) complexity
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "text") | {name, complexity: .complexity.overall}'
  ```

### Step 13: Software Rasterizer — Compositing (Phase 1.4d)
- **Deliverable**: Image blitting with bilinear filtering, alpha compositing (Porter-Duff SrcOver)
- **Dependencies**: Step 10
- **Scope**: ~1,500–2,000 LoC
- **Files to create**: `internal/raster/composite/` package
- **Acceptance**:
  - No code duplication in blending functions
  - Functions optimized for hot path (no allocations)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication.duplication_ratio < 0.03'
  ```

### Step 14: Software Rasterizer — Effects (Phase 1.4e)
- **Deliverable**: Box shadow (Gaussian blur), linear/radial gradients, scissor clipping
- **Dependencies**: Steps 10, 13
- **Scope**: ~2,000–2,500 LoC
- **Files to create**: `internal/raster/effects/` package
- **Milestone**: Render styled UI elements with shadows using only CPU
- **Acceptance**:
  - Effect functions with clear separation of concerns
  - Package cohesion score > 0.6
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections packages | \
    jq '.packages[] | select(.name == "effects") | .cohesion_score'
  ```

### Step 15: Widget Layer — Layout Engine (Phase 1.5a)
- **Deliverable**: Flexbox-like layout system, retained-mode or immediate-mode API
- **Dependencies**: Steps 5, 7 (window management), Steps 10–14 (rasterizer)
- **Scope**: ~3,000–4,000 LoC
- **Files to create**: `internal/ui/layout/` package
- **Acceptance**:
  - Layout algorithms with complexity < 15
  - Renderer-agnostic (emits display list, not pixels)
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '.functions[] | select(.package == "layout") | select(.complexity.cyclomatic > 15)'
  ```

### Step 16: Widget Layer — Core Widgets (Phase 1.5b)
- **Deliverable**: Text input, buttons, scroll containers
- **Dependencies**: Steps 8, 9 (input handling), Step 15 (layout)
- **Scope**: ~3,000–4,000 LoC
- **Files to create**: `internal/ui/widgets/` package
- **Milestone**: Interactive demo app running on software renderer over both X11 and Wayland
- **Acceptance**:
  - Widget implementations with consistent API pattern
  - All public widget types documented
  - No circular dependencies between ui packages
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections packages | \
    jq '.circular_dependencies | length == 0'
  ```

## Dependency Graph

```
Step 1 ─────► Step 2 ─────► Step 3 ─────► Step 4 ─────► Step 5 ─────┐
(wire)       (fd)          (client)      (shm)         (xdg)        │
                                                                     │
                                                          Step 8 ◄──┘
                                                          (wl input)
                                                               │
Step 6 ─────► Step 7 ─────► Step 9 ─────────────────────────────┤
(x11 proto)  (x11 gc)       (x11 input)                         │
                                                                 │
                                                                 ▼
Step 10 ────┬───► Step 11 ───────────────────────────────► Step 15 ──► Step 16
(raster)    │     (curves)                                 (layout)    (widgets)
            │                                                  ▲
            ├───► Step 12 ─────────────────────────────────────┤
            │     (text)                                       │
            │                                                  │
            └───► Step 13 ───► Step 14 ────────────────────────┘
                  (composite)  (effects)
```

## Parallelization Opportunities

The following work streams can proceed concurrently:
1. **Wayland path** (Steps 1–5, 8)
2. **X11 path** (Steps 6–7, 9)
3. **Rasterizer** (Steps 10–14)

This allows significant parallelization with multiple contributors.

## Success Criteria for Phase 1 Completion

Per ROADMAP.md:
> Interactive demo app (text fields, buttons, scrolling list) running on software renderer over both X11 and Wayland.

**Metrics targets for Phase 1 completion:**
| Metric | Target |
|--------|--------|
| Go LoC | 30,000–50,000 |
| Functions with complexity > 15 | < 5% of total |
| Code duplication ratio | < 3% |
| Doc coverage (exported) | > 90% |
| Circular dependencies | 0 |
| Package cohesion (avg) | > 0.5 |

## Risk Mitigation

Per ROADMAP.md identified risks:
- **Wayland protocol surface is large**: Test on wlroots-based compositors (sway) first
- **Compositor-specific quirks**: Add mutter/kwin compat fixes as needed after sway works

## Next Phase Preview

Upon Phase 1 completion, Phase 2 (DRM/KMS Buffer Infrastructure in Rust) becomes unblocked, which involves:
- Kernel ioctl wrappers in Rust
- Buffer allocator for GPU-visible buffers
- DMA-BUF integration with Wayland
- DRI3 integration with X11

---
*Generated with go-stats-generator metrics on 2026-03-07*
*Baseline: Phase 0 complete (Go/Rust static linking validated)*
