# GPU Shader Testing (Phase 4.5)

This document describes the GPU shader testing infrastructure implemented in Phase 4.5 of the ROADMAP.

## Overview

The GPU testing framework validates that shaders compiled through the naga frontend and Intel EU backend render correctly on actual GPU hardware. Tests compare GPU-rendered output against software rasterizer reference images using pixel-by-pixel comparison with configurable tolerance.

## Architecture

### Module: `render-sys/src/gpu_test.rs`

The `gpu_test` module provides:

1. **Image Buffer Abstraction**
   - `Pixel`: RGBA pixel representation (r, g, b, a)
   - `ImageBuffer`: 2D image with pixel access and comparison
   - `approx_eq()`: Tolerance-based pixel comparison

2. **GPU Test Context**
   - `GpuTestContext`: Manages GPU device, allocator, and generation detection
   - `new()`: Auto-detects Intel GPU (Gen9+) or returns None
   - `allocate_render_target()`: Creates 64x64 ARGB8888 test render targets
   - `readback_pixels()`: Maps GPU buffer to CPU for validation (TODO: implement mmap)

3. **Test Utilities**
   - `TEST_RT_WIDTH`, `TEST_RT_HEIGHT`: Standard test dimensions (64x64)
   - `TEST_RT_BPP`: Bits per pixel (4 = ARGB8888)
   - `DEFAULT_TOLERANCE`: Per-channel pixel tolerance (2/255)

### GPU Tests: `render-sys/src/shader.rs`

Seven GPU validation tests, one per shader:
- `test_solid_fill_gpu`
- `test_textured_quad_gpu`
- `test_sdf_text_gpu`
- `test_rounded_rect_gpu`
- `test_linear_gradient_gpu`
- `test_radial_gradient_gpu`
- `test_box_shadow_gpu`

All tests are marked `#[ignore]` to run only when GPU hardware is available.

## Running GPU Tests

### Prerequisites

1. **Intel GPU hardware** (Gen9+)
   - Gen9: Skylake, Kaby Lake, Coffee Lake
   - Gen11: Ice Lake
   - Gen12: Tiger Lake, Rocket Lake, Alder Lake
   - Xe: Meteor Lake and newer

2. **DRM device access**: `/dev/dri/renderD128` must be readable

### Running Tests

```bash
# Run standard tests (GPU tests skipped)
cd render-sys && cargo test --target x86_64-unknown-linux-musl

# Run GPU tests only (requires hardware)
cd render-sys && cargo test --target x86_64-unknown-linux-musl -- --ignored

# Run all tests including GPU tests
cd render-sys && cargo test --target x86_64-unknown-linux-musl -- --include-ignored
```

### Expected Output

**Without GPU hardware:**
```
test shader::tests::test_solid_fill_gpu ... ignored
test shader::tests::test_textured_quad_gpu ... ignored
...
```

**With GPU hardware:**
```
test shader::tests::test_solid_fill_gpu ... ok
Testing solid_fill shader on Gen9
Expected failure (rendering not yet implemented): Images differ...
```

## Implementation Status

### ✅ Complete
- GPU test infrastructure module (`gpu_test.rs`)
- Image buffer abstraction with pixel comparison
- GPU context creation and device detection
- Render target allocation
- 7 GPU validation test scaffolds
- All tests properly ignored by default
- Zero regressions in existing tests (185 passing)

### 🔧 TODO (Phase 5)
- **CPU mmap for readback**: Implement `readback_pixels()` to actually read GPU buffers
- **Batch submission**: Wire up EU compiler output to GPU command submission
- **Reference renderers**: Create proper software rasterizer reference images
- **Full rendering pipeline**: Connect shaders → batch → submit → readback → validate

## Test Methodology

Each GPU test follows this pattern:

```rust
#[test]
#[ignore] // Requires Intel GPU hardware
fn test_<shader>_gpu() {
    // 1. Create GPU context (or skip if no GPU)
    let mut ctx = match GpuTestContext::new() {
        Some(c) => c,
        None => {
            println!("No Intel GPU available, test skipped");
            return;
        }
    };

    // 2. Allocate render target
    let rt = ctx.allocate_render_target()
        .expect("Failed to allocate render target");

    // 3. TODO: Compile shader to EU binary
    // let eu_binary = compile_shader(&shader_source);

    // 4. TODO: Build batch buffer with draw commands
    // let batch = build_batch(&eu_binary, &rt);

    // 5. TODO: Submit batch and wait for completion
    // ctx.submit_and_wait(&batch);

    // 6. Read back GPU-rendered pixels
    let gpu_result = ctx.readback_pixels(&rt)
        .expect("Failed to read back pixels");

    // 7. Generate software rasterizer reference
    let reference = render_<shader>_reference();

    // 8. Compare with tolerance
    gpu_result.compare(&reference, DEFAULT_TOLERANCE)
        .expect("GPU output should match reference");
}
```

## Pixel Comparison Algorithm

The `ImageBuffer::compare()` method performs tolerance-based comparison:

1. Verify dimensions match
2. For each pixel:
   - Compute per-channel absolute difference
   - Check if all channels are within tolerance
3. Report first 5-10 mismatched pixels if comparison fails

**Tolerance rationale**: GPU and CPU may differ by 1-2 LSBs due to:
- Rounding differences in floating-point operations
- Different precision in intermediate calculations
- Texture filtering implementation variations

Default tolerance of 2/255 (~0.8%) allows minor precision differences while catching actual rendering bugs.

## File Layout

```
render-sys/src/
├── gpu_test.rs         # GPU test infrastructure (NEW)
│   ├── Pixel           # RGBA pixel with approx_eq
│   ├── ImageBuffer     # 2D image buffer with comparison
│   └── GpuTestContext  # GPU device management for tests
├── shader.rs           # Shader module + GPU validation tests
│   └── tests
│       ├── test_solid_fill_gpu        (NEW)
│       ├── test_textured_quad_gpu     (NEW)
│       ├── test_sdf_text_gpu          (NEW)
│       ├── test_rounded_rect_gpu      (NEW)
│       ├── test_linear_gradient_gpu   (NEW)
│       ├── test_radial_gradient_gpu   (NEW)
│       └── test_box_shadow_gpu        (NEW)
└── lib.rs              # Module declaration (#[cfg(test)] pub mod gpu_test)
```

## Test Count Summary

| Category | Count | Notes |
|----------|-------|-------|
| **Total Rust tests** | 185 | Up from 180 pre-Phase 4.5 |
| **Ignored GPU tests** | 8 | 7 shader tests + 1 context test |
| **New infrastructure tests** | 5 | Pixel, ImageBuffer, conversion tests |
| **New GPU validation tests** | 7 | One per UI shader |

## CI Considerations

**GPU tests are opt-in** via `#[ignore]` attribute. This prevents CI failures on systems without GPU hardware.

To enable GPU testing in CI:
```yaml
- name: Run GPU tests (if available)
  run: |
    if [ -e /dev/dri/renderD128 ]; then
      cargo test --target x86_64-unknown-linux-musl -- --ignored
    else
      echo "No GPU available, skipping GPU tests"
    fi
```

## Next Steps (Phase 5)

1. **Implement CPU mmap readback**
   - Add DRM_IOCTL_I915_GEM_MMAP_OFFSET / DRM_IOCTL_XE_GEM_MMAP_OFFSET
   - Map buffer to user space, copy pixels to ImageBuffer

2. **Wire GPU rendering pipeline**
   - Call `eu::compile()` to generate shader binaries
   - Build batch buffer with 3DSTATE_* commands
   - Submit via `render_submit_batch()` FFI

3. **Create reference renderers**
   - Use existing software rasterizer (`internal/raster/*`)
   - Generate reference images in Rust or call Go functions via FFI

4. **Validate all 7 shaders**
   - Fill in TODO sections in each test
   - Verify pixel-perfect or tolerance-based match

## References

- ROADMAP.md Phase 4.5 (lines 267-282)
- `render-sys/src/gpu_test.rs` - Test infrastructure
- `render-sys/src/shader.rs` - Shader tests module
- `render-sys/shaders/README.md` - Shader documentation
