# Implementation Plan: Phase 3 — GPU Command Submission (Intel)

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust, targeting X11/Wayland with Intel/AMD GPU backends.
- **Current milestone**: Phase 3 — GPU Command Submission (Intel GPUs: Gen9-Gen12, i915/Xe drivers)
- **Estimated Scope**: **Large** — Phase 3 introduces GPU command emission, state encoding, and kernel submission infrastructure (~30,000 LOC estimated for Rust Intel driver + EU compiler).

## Metrics Summary (from go-stats-generator)

| Metric | Current Value | Assessment |
|--------|---------------|------------|
| Total LOC | 5,071 | Go layer mature; Rust layer pending Phase 3 expansion |
| Functions | 187 | Protocol + rasterizer layers complete |
| Methods | 293 | Well-structured OO design |
| Packages | 23 | Clean architectural boundaries |
| **Complexity hotspots** | **7** functions CC > 9 | Within healthy bounds |
| **Duplication ratio** | **4.1%** | Medium (threshold: 3-10%) |
| **Doc coverage** | **89.9%** overall | Good (97.9% functions, 84.9% methods) |

### Complexity Hotspots (CC > 9)
| Function | File | CC | Lines |
|----------|------|-----|-------|
| `SendRequestAndReplyWithFDs` | internal/x11/client/client.go | 13 | 59 |
| `AutoLayout` | internal/ui/pctwidget/autolayout.go | 11 | 64 |
| `keycodeToAlphanumeric` | internal/wayland/input/keymap.go | 11 | 42 |
| `DecodeSetupReply` | internal/x11/wire/setup.go | 11 | 127 |
| `lineCoverage` | internal/raster/core/line.go | 10 | 42 |
| `FillRoundedRect` | internal/raster/core/rect.go | 10 | 47 |
| `LinearGradient` | internal/raster/effects/effects.go | 10 | 52 |

### Duplication Clusters (violations, >10 lines)
| Clone Size | Files | Priority |
|------------|-------|----------|
| 70 lines | cmd/demo/main.go ↔ cmd/x11-demo/main.go | Demo consolidation (defer) |
| 36 lines | internal/x11/dri3/dri3.go ↔ internal/x11/present/present.go | Extension reply parsing |
| 29 lines | internal/raster/core/buffer.go ↔ internal/raster/curves/curves.go | Scanline iteration |
| 25 lines | cmd/dmabuf-demo/main.go ↔ cmd/x11-dmabuf-demo/main.go | Demo consolidation (defer) |

*Note: Demo duplication is acceptable for clarity; core library duplication should be addressed opportunistically.*

---

## Implementation Steps

### Step 1: Hardware Detection Module
- **Deliverable**: Create `render-sys/src/detect.rs` to query GPU generation from i915/Xe kernel parameters via `I915_GETPARAM` and `DRM_IOCTL_XE_DEVICE_QUERY`.
- **Dependencies**: Existing `render-sys/src/{i915.rs,xe.rs,drm.rs}` ioctls
- **Acceptance**: Function returns `GpuGeneration` enum (Gen9/Gen11/Gen12/Xe) with ≥95% test coverage
- **Validation**: 
  ```bash
  cd render-sys && cargo test detect -- --nocapture
  ```

### Step 2: GPU Command Encoding Tables (Gen9-Gen12)
- **Deliverable**: Create `render-sys/src/cmd/` module with Rust structs for Intel 3D pipeline commands:
  - `MI_BATCH_BUFFER_START`, `PIPELINE_SELECT`, `STATE_BASE_ADDRESS`
  - `3DSTATE_VIEWPORT`, `3DSTATE_CLIP`, `3DSTATE_SF`, `3DSTATE_WM`, `3DSTATE_PS`
  - `3DSTATE_VERTEX_BUFFERS`, `3DSTATE_VERTEX_ELEMENTS`, `3DPRIMITIVE`, `PIPE_CONTROL`
- **Dependencies**: Step 1 (generation detection for command variants)
- **Acceptance**: Each command struct serializes to correct binary per Intel PRM Vol. 2; unit tests verify dword layout
- **Validation**:
  ```bash
  cd render-sys && cargo test cmd:: -- --nocapture | grep -E "test.*ok"
  ```

### Step 3: Batch Buffer Builder
- **Deliverable**: Create `render-sys/src/batch.rs` implementing `BatchBuilder` that:
  - Allocates GEM buffer object for command stream
  - Provides typed emit methods for each 3D command
  - Handles relocation entries for buffer references
  - Supports GPU address patching
- **Dependencies**: Step 2 (command encoding), existing `allocator.rs`
- **Acceptance**: `BatchBuilder::emit_*` methods accept command structs; `finalize()` returns submittable batch
- **Validation**:
  ```bash
  cd render-sys && cargo test batch:: -- --nocapture
  ```

### Step 4: Pipeline State Configuration
- **Deliverable**: Create `render-sys/src/pipeline.rs` with pre-baked pipeline state configurations:
  - (a) Solid color fill
  - (b) Textured quad (bilinear sampling)
  - (c) SDF text rendering
  - (d) Box shadow (separable blur, two-pass)
  - (e) Rounded rect clip
  - (f) Linear/radial gradient
- **Dependencies**: Step 2 (3DSTATE commands), Step 3 (batch emitter)
- **Acceptance**: Each pipeline config is a unit-testable function returning encoded state; matches Go rasterizer output
- **Validation**:
  ```bash
  cd render-sys && cargo test pipeline:: -- --nocapture
  ```

### Step 5: Surface State & Sampler State
- **Deliverable**: Create `render-sys/src/surface.rs` to encode:
  - `RENDER_SURFACE_STATE` for render targets and texture sources
  - `SAMPLER_STATE` for bilinear/nearest filtering
  - Binding table management in surface state heap
- **Dependencies**: Step 1 (generation-specific layouts), Step 3 (batch builder)
- **Acceptance**: Surface state entries match Intel PRM Vol. 5 layouts; binding table indices are validated
- **Validation**:
  ```bash
  cd render-sys && cargo test surface:: -- --nocapture
  ```

### Step 6: Batch Submission (i915)
- **Deliverable**: Extend `render-sys/src/i915.rs` with:
  - `I915_GEM_EXECBUFFER2` wrapper
  - Context creation via `I915_GEM_CONTEXT_CREATE`
  - Synchronization via `I915_GEM_WAIT`
- **Dependencies**: Step 3 (batch builder output), existing drm ioctls
- **Acceptance**: Submitted batch completes without GPU hang; verified via sync wait return code
- **Validation**:
  ```bash
  cd render-sys && cargo test i915::submit -- --nocapture
  ```

### Step 7: Batch Submission (Xe)
- **Deliverable**: Extend `render-sys/src/xe.rs` with:
  - `DRM_IOCTL_XE_EXEC` wrapper
  - VM creation/binding via `DRM_IOCTL_XE_VM_CREATE`, `DRM_IOCTL_XE_VM_BIND`
  - Fence-based synchronization
- **Dependencies**: Step 3 (batch builder), Step 6 (parallel to i915)
- **Acceptance**: Same batch submits on Xe driver when available; graceful fallback when unavailable
- **Validation**:
  ```bash
  cd render-sys && cargo test xe::submit -- --nocapture || echo "Xe not available"
  ```

### Step 8: Go CGO Bindings for Submission
- **Deliverable**: Extend `internal/render/render.go` with C ABI bindings:
  - `render_detect_gpu() -> int` (returns generation enum)
  - `render_submit_batch(buf *C.uint8_t, len C.size_t) -> int` (submits and waits)
  - `render_create_context() -> uint64` (returns context handle)
- **Dependencies**: Steps 6-7 (Rust submission), existing CGO infrastructure
- **Acceptance**: Go code can call submission functions; static linking verified via `make check-static`
- **Validation**:
  ```bash
  make test-go && make check-static
  ```

### Step 9: First Triangle Demonstration
- **Deliverable**: Create `cmd/gpu-triangle-demo/` that:
  - Detects GPU, creates context
  - Builds batch: clear render target (blue), draw single triangle (white)
  - Submits batch, waits for completion
  - Copies GPU buffer to X11/Wayland surface via existing DRI3/dmabuf path
- **Dependencies**: All previous steps
- **Acceptance**: Visible white triangle on blue background in window
- **Validation**:
  ```bash
  make build && ./bin/gpu-triangle-demo
  # Visual verification: triangle renders correctly
  ```

### Step 10: Integration Test Suite
- **Deliverable**: Create `internal/integration/gpu_test.go` with:
  - GPU detection test (passes on Intel hardware, skips on others)
  - Batch construction test (verifies command serialization)
  - Submission test (clear + draw, read back via CPU mmap, verify pixel values)
- **Dependencies**: Step 9 complete
- **Acceptance**: `make test-go` passes on Intel GPU systems; tests skip gracefully on non-Intel
- **Validation**:
  ```bash
  make test-go 2>&1 | grep -E "(PASS|SKIP).*gpu"
  ```

---

## Deferred Work (Outside Phase 3 Scope)

### Code Quality (opportunistic, not blocking)
1. **Duplication in X11 extensions** (36-line clone): Extract shared reply parsing into `internal/x11/wire/extensions.go`
   - **Validation**: `go-stats-generator analyze . --sections duplication | jq '.duplication.duplication_ratio < 0.035'`

2. **Complexity in `SendRequestAndReplyWithFDs`** (CC=13): Split into `sendRequest` + `receiveReplyWithFDs`
   - **Validation**: `go-stats-generator analyze . | jq '[.functions[] | select(.file | contains("client.go")) | select(.complexity.cyclomatic > 10)] | length == 0'`

### Future Phases
- **Phase 4**: Shader compiler pipeline (naga IR → Intel EU binary)
- **Phase 5**: Rendering backend integration (display list → GPU batch)
- **Phase 6**: AMD GPU support (AMDGPU ioctls + RDNA ISA backend)

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| First triangle takes >4 weeks | Medium | High | Timebox; consult Mesa iris driver line-by-line |
| i915/Xe command encoding mismatch | Low | Medium | Target Gen12 first (most documented), backport to Gen9 |
| GPU hangs during development | High | Low | Run tests with `IGT_HANG_LIMIT=1`; fallback to software |
| Static linking breaks with new Rust code | Low | Medium | CI enforces `make check-static` on every commit |

---

## Validation Commands Summary

```bash
# Full Phase 3 validation suite
make test-rust                                    # Rust unit tests
make test-go                                      # Go unit tests (CGO-linked)
make check-static                                 # Verify static binary

# Metrics monitoring
go-stats-generator analyze . --skip-tests --format json | jq '{
  complexity_hotspots: [.functions[] | select(.complexity.cyclomatic > 9)] | length,
  duplication_ratio: .duplication.duplication_ratio,
  doc_coverage: .documentation.coverage.overall
}'
# Target: complexity_hotspots ≤ 10, duplication_ratio < 5%, doc_coverage > 85%
```

---

*Generated: 2026-03-07 | Next review: After Step 5 completion*
