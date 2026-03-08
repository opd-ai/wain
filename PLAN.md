# Implementation Plan: Phase 4.2 – UI Shader Authoring

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust, targeting X11/Wayland with hardware-accelerated rendering (Intel first, then AMD).
- **Current milestone**: Phase 4.2 – Write UI shaders in GLSL/WGSL (per ROADMAP.md)
- **Estimated Scope**: **Medium** (6-10 shader pairs required, structural work on pipeline integration)

## Metrics Summary (Go Codebase)

| Metric | Value | Status |
|--------|-------|--------|
| **Total LOC** | 5,394 | — |
| **Total Functions** | 198 funcs + 309 methods | — |
| **Total Packages** | 23 | — |
| **Complexity Hotspots** | 8 functions above threshold (≥10) | ✅ Small |
| **Duplication Ratio** | 5.04% (633 duplicated lines) | ⚠️ Medium |
| **Doc Coverage** | 92.7% overall | ✅ Good |
| **Clone Pairs** | 32 (2 violations, 30 warnings) | ⚠️ Medium |

### High Complexity Functions (cyclomatic ≥10)
| Complexity | Location | Function |
|------------|----------|----------|
| 13 | `internal/x11/client/client.go` | `SendRequestAndReplyWithFDs` |
| 11 | `internal/ui/pctwidget/autolayout.go` | `AutoLayout` |
| 11 | `cmd/gpu-triangle-demo/main.go` | `setupX11AndGPU` |
| 11 | `internal/wayland/input/keymap.go` | `keycodeToAlphanumeric` |
| 11 | `internal/x11/wire/setup.go` | `DecodeSetupReply` |
| 10 | `internal/raster/core/line.go` | `lineCoverage` |
| 10 | `internal/raster/core/rect.go` | `FillRoundedRect` |
| 10 | `internal/raster/effects/effects.go` | `LinearGradient` |

### Package Coupling Analysis
| Package | Lines | Funcs | Cohesion | Coupling |
|---------|-------|-------|----------|----------|
| main (cmd/*) | 1,938 | 76 | 1.89 | 6.0 (high) |
| client (x11/wayland) | 1,108 | 47 | 2.36 | 1.5 |
| wire | 1,079 | 34 | 3.47 | 0.0 |
| widgets | 642 | 44 | 10.0 | 1.5 |
| render | 405 | 14 | 2.2 | 0.0 |

### Duplication Hotspots (Violations)
1. **X11 request/reply pattern** – 4 locations in `x11/{client,dri3,gc,shm}` (6-line blocks)
2. **Demo rendering loop** – 6 locations across `cmd/{demo,wayland-demo,widget-demo,x11-demo}` (6-line blocks)

---

## Milestone Scope: Phase 4.2

**Objective**: Author ~6-10 vertex/fragment shader pairs in GLSL or WGSL for common UI draw types.

**Per ROADMAP.md lines 199-206**:
> Author ~6-10 vertex/fragment shader pairs in GLSL or WGSL:
> solid fill, textured quad, SDF text, box shadow blur, rounded rect clip, linear gradient, radial gradient.
> These are simple shaders — most fragment shaders are <30 lines.

**Dependencies (complete)**:
- ✅ Phase 4.1: naga shader frontend (WGSL/GLSL parsing) – implemented in `render-sys/src/shader.rs`
- ✅ Phase 3: GPU command infrastructure – batch, pipeline, surface state implemented

---

## Implementation Steps

### Step 1: Create Shader Source Directory Structure
- **Deliverable**: Create `render-sys/shaders/` directory with organized structure for WGSL shader sources
- **Dependencies**: None
- **Files**: 
  - `render-sys/shaders/solid_fill.wgsl`
  - `render-sys/shaders/textured_quad.wgsl`
  - `render-sys/shaders/sdf_text.wgsl`
  - `render-sys/shaders/box_shadow.wgsl`
  - `render-sys/shaders/rounded_rect.wgsl`
  - `render-sys/shaders/linear_gradient.wgsl`
  - `render-sys/shaders/radial_gradient.wgsl`
- **Acceptance**: All `.wgsl` files exist and are parseable by naga
- **Validation**: `cd render-sys && cargo test shader -- --nocapture` should validate all shaders parse

### Step 2: Implement Solid Fill Shader
- **Deliverable**: Vertex shader passing position/UV, fragment shader returning constant color from push constant
- **Dependencies**: Step 1
- **Files**: `render-sys/shaders/solid_fill.wgsl`
- **Acceptance**: Shader compiles via naga, outputs valid naga IR with 1 vertex entry point + 1 fragment entry point
- **Validation**: 
```bash
cd render-sys && cargo test test_solid_fill_shader -- --nocapture
```

### Step 3: Implement Textured Quad Shader
- **Deliverable**: Vertex shader with UV coordinates, fragment shader sampling texture with bilinear filtering
- **Dependencies**: Step 1, Step 2 (pattern established)
- **Files**: `render-sys/shaders/textured_quad.wgsl`
- **Acceptance**: Shader includes texture binding, sampler binding, proper UV interpolation
- **Validation**: 
```bash
cd render-sys && cargo test test_textured_quad_shader -- --nocapture
```

### Step 4: Implement SDF Text Shader
- **Deliverable**: Fragment shader with signed distance field alpha calculation, configurable threshold/smoothing
- **Dependencies**: Step 3 (texture sampling pattern)
- **Files**: `render-sys/shaders/sdf_text.wgsl`
- **Acceptance**: Shader implements `smoothstep` for SDF edge softening, alpha output
- **Validation**: 
```bash
cd render-sys && cargo test test_sdf_text_shader -- --nocapture
```

### Step 5: Implement Box Shadow Shader (Two-Pass Blur)
- **Deliverable**: Separable Gaussian blur vertex/fragment shaders for horizontal and vertical passes
- **Dependencies**: Step 3 (texture sampling)
- **Files**: `render-sys/shaders/box_shadow.wgsl`
- **Acceptance**: Shader includes blur kernel weights, separable passes, rect mask support
- **Validation**: 
```bash
cd render-sys && cargo test test_box_shadow_shader -- --nocapture
```

### Step 6: Implement Rounded Rect Clip Shader
- **Deliverable**: Fragment shader with SDF-based discard for rounded corners
- **Dependencies**: Step 4 (SDF pattern)
- **Files**: `render-sys/shaders/rounded_rect.wgsl`
- **Acceptance**: Shader computes signed distance from rounded rect bounds, uses `discard` or alpha
- **Validation**: 
```bash
cd render-sys && cargo test test_rounded_rect_shader -- --nocapture
```

### Step 7: Implement Linear Gradient Shader
- **Deliverable**: Fragment shader interpolating colors along a direction vector
- **Dependencies**: Step 2 (color output pattern)
- **Files**: `render-sys/shaders/linear_gradient.wgsl`
- **Acceptance**: Shader accepts gradient direction, at least 2-4 color stops
- **Validation**: 
```bash
cd render-sys && cargo test test_linear_gradient_shader -- --nocapture
```

### Step 8: Implement Radial Gradient Shader
- **Deliverable**: Fragment shader with radial color interpolation from center point
- **Dependencies**: Step 7 (gradient pattern)
- **Files**: `render-sys/shaders/radial_gradient.wgsl`
- **Acceptance**: Shader accepts center, radius, color stops with radial falloff
- **Validation**: 
```bash
cd render-sys && cargo test test_radial_gradient_shader -- --nocapture
```

### Step 9: Add Shader Validation Test Suite
- **Deliverable**: Comprehensive Rust tests validating all shaders compile and produce valid naga IR
- **Dependencies**: Steps 1-8
- **Files**: `render-sys/src/shader.rs` (extend test module)
- **Acceptance**: All 7 shaders validate with naga, each has documented entry points and bindings
- **Validation**: 
```bash
cd render-sys && cargo test shader -- --nocapture 2>&1 | grep -E "(test_.*shader|PASSED|FAILED)"
```

### Step 10: Document Shader API and Uniform Layouts
- **Deliverable**: Documentation in `render-sys/shaders/README.md` describing each shader's purpose, bindings, and uniform buffer layout
- **Dependencies**: Steps 1-9
- **Files**: `render-sys/shaders/README.md`
- **Acceptance**: Each shader has documented: entry points, uniform struct layout, texture bindings, expected vertex format
- **Validation**: Manual review – file exists and covers all 7 shaders

---

## Scope Assessment Rationale

| Metric | Assessment | Impact on Scope |
|--------|------------|-----------------|
| Functions above complexity 9 | 8 (Small) | No immediate refactoring needed |
| Duplication ratio | 5.04% (Medium) | Deduplication optional, not blocking |
| Doc coverage gap | 7.3% (Small) | Adequate for development |
| Shader count | 7 shaders × 2 entry points = 14 shader stages | Primary scope driver |

**Estimated effort**: Medium (7 shader pairs, test infrastructure, documentation)

---

## Metrics Targets Post-Implementation

| Metric | Current | Target | Validation |
|--------|---------|--------|------------|
| Shader test count | 6 | 13+ | `cargo test shader -- --list \| wc -l` |
| Shader file count | 0 | 7 | `ls render-sys/shaders/*.wgsl \| wc -l` |
| Naga IR validation | N/A | 100% | All shaders pass `naga::valid::Validator` |

---

## Out of Scope (Deferred to Future Phases)

The following items were identified from metrics but are **not part of Phase 4.2**:

### Duplication Cleanup (5.04% ratio)
- **X11 request/reply pattern**: Could extract shared helper to `internal/x11/wire/`
- **Demo rendering loop**: Could create `internal/demo/renderloop.go` helper
- **Recommendation**: Address in Phase 5 or dedicated cleanup sprint

### Complexity Reduction
- `SendRequestAndReplyWithFDs` (complexity 13): Protocol-inherent, low refactoring ROI
- `AutoLayout` (complexity 11): Layout algorithm complexity is justified
- **Recommendation**: Monitor, don't refactor unless bugs emerge

### Documentation Gaps
- Method coverage at 89.9% (31 undocumented methods)
- **Recommendation**: Document as part of Phase 8.5 (Documentation milestone)

---

## Dependencies for Next Phase (4.3: Intel EU Backend)

Phase 4.2 completion enables:
1. **Shader IR available**: naga IR from each shader is the input to EU backend
2. **Binding layout documented**: EU backend needs to know uniform buffer layouts
3. **Entry points defined**: Vertex/fragment entry points map to 3DSTATE_VS/3DSTATE_PS

---

## Quick Reference Commands

```bash
# Build Rust library (includes shader validation)
make rust

# Run Rust tests (shader compilation)
cd render-sys && cargo test --target x86_64-unknown-linux-musl

# Run specific shader tests
cd render-sys && cargo test shader --target x86_64-unknown-linux-musl -- --nocapture

# Full project build
make build

# Full test suite
make test

# Validate metrics after changes
go-stats-generator analyze . --skip-tests --format json --output metrics.json --sections functions,duplication,documentation
```

---

*Generated from go-stats-generator metrics and ROADMAP.md Phase 4.2 specification.*
