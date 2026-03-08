/// GPU buffer allocator with tiling support.
///
/// This module provides a buffer allocator that can create GPU-visible buffers
/// with appropriate tiling formats for optimal rendering performance.

use std::io;
use std::os::unix::io::RawFd;
use crate::drm::DrmDevice;
use crate::i915;
use crate::xe;

/// Tiling format for GPU buffers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TilingFormat {
    /// Linear (no tiling) - simple row-major layout.
    None,
    /// X-tiled - optimized for textures and render targets on Intel GPUs.
    X,
    /// Y-tiled - optimized for depth/stencil buffers on Intel GPUs.
    Y,
}

/// GPU driver backend type.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DriverType {
    /// Intel i915 driver (Gen9-Gen12).
    I915,
    /// Intel Xe driver (Gen12+).
    Xe,
    /// AMD AMDGPU driver (RDNA1+).
    Amdgpu,
}

/// GPU buffer handle.
#[derive(Debug)]
pub struct Buffer {
    pub handle: u32,
    pub size: u64,
    pub width: u32,
    pub height: u32,
    pub stride: u32,
    pub tiling: TilingFormat,
    /// Driver type that allocated this buffer (reserved for Phase 3+ GPU command submission).
    _driver: DriverType,
}

/// Buffer allocator for GPU-visible memory.
pub struct BufferAllocator {
    device: DrmDevice,
    driver: DriverType,
}

impl BufferAllocator {
    /// Create a new buffer allocator for the given DRM device.
    pub fn new(device: DrmDevice, driver: DriverType) -> Self {
        Self { device, driver }
    }

    /// Allocate a GPU buffer with the specified dimensions and tiling format.
    ///
    /// Returns a Buffer handle that can be used for rendering or exported as DMA-BUF.
    pub fn allocate(
        &self,
        width: u32,
        height: u32,
        bpp: u32,
        tiling: TilingFormat,
    ) -> io::Result<Buffer> {
        match self.driver {
            DriverType::I915 => self.allocate_i915(width, height, bpp, tiling),
            DriverType::Xe => self.allocate_xe(width, height, bpp, tiling),
            DriverType::Amdgpu => self.allocate_amdgpu(width, height, bpp, tiling),
        }
    }

    /// Allocate buffer using i915 driver.
    fn allocate_i915(
        &self,
        width: u32,
        height: u32,
        bpp: u32,
        tiling: TilingFormat,
    ) -> io::Result<Buffer> {
        // Calculate stride based on tiling format
        let stride = match tiling {
            TilingFormat::None => width * (bpp / 8),
            TilingFormat::X => {
                // X-tiled: 512-byte tiles, align stride to 512
                let bytes_per_row = width * (bpp / 8);
                ((bytes_per_row + 511) / 512) * 512
            }
            TilingFormat::Y => {
                // Y-tiled: 128-byte tiles, align stride to 128
                let bytes_per_row = width * (bpp / 8);
                ((bytes_per_row + 127) / 128) * 128
            }
        };

        let size = (stride * height) as u64;

        // Allocate GEM buffer
        let mut gem_create = i915::GemCreate::new(size);
        self.device.i915_gem_create(&mut gem_create)?;

        let handle = gem_create.handle;

        // Set tiling mode if not linear
        if tiling != TilingFormat::None {
            let tiling_mode = match tiling {
                TilingFormat::X => i915::I915_TILING_X,
                TilingFormat::Y => i915::I915_TILING_Y,
                _ => unreachable!(),
            };
            self.device.i915_gem_set_tiling(handle, tiling_mode, stride)?;
        }

        Ok(Buffer {
            handle,
            size,
            width,
            height,
            stride,
            tiling,
            _driver: DriverType::I915,
        })
    }

    /// Allocate buffer using Xe driver.
    fn allocate_xe(
        &self,
        width: u32,
        height: u32,
        bpp: u32,
        tiling: TilingFormat,
    ) -> io::Result<Buffer> {
        // Xe doesn't use explicit tiling modes like i915 - tiling is handled
        // by the GPU hardware based on buffer flags and usage patterns.
        let stride = width * (bpp / 8);
        let size = (stride * height) as u64;

        // Use system memory placement (can be promoted to VRAM by GPU)
        let placement = xe::XE_GEM_CREATE_PLACEMENT_SYSTEM;
        let mut gem_create = xe::GemCreate::new(size, placement);

        // Use write-combining for better performance on GPU access
        gem_create.cpu_caching = xe::XE_GEM_CPU_CACHING_WC;

        self.device.xe_gem_create(&mut gem_create)?;

        Ok(Buffer {
            handle: gem_create.handle,
            size,
            width,
            height,
            stride,
            tiling,
            _driver: DriverType::Xe,
        })
    }

    /// Allocate buffer using AMD AMDGPU driver.
    fn allocate_amdgpu(
        &self,
        width: u32,
        height: u32,
        bpp: u32,
        tiling: TilingFormat,
    ) -> io::Result<Buffer> {
        use crate::amd::{GemCreate, AMDGPU_GEM_DOMAIN_VRAM, AMDGPU_GEM_DOMAIN_GTT,
                         AMDGPU_GEM_CREATE_CPU_ACCESS_REQUIRED};

        // AMD handles tiling automatically based on buffer usage
        // Tiling format is informational only for this driver
        let stride = width * (bpp / 8);
        let size = (stride * height) as u64;

        // Prefer VRAM but allow fallback to GTT if VRAM is full
        let domains = AMDGPU_GEM_DOMAIN_VRAM | AMDGPU_GEM_DOMAIN_GTT;
        let flags = AMDGPU_GEM_CREATE_CPU_ACCESS_REQUIRED;

        let mut gem_create = GemCreate::new(size, domains, flags);
        self.device.amdgpu_gem_create(&mut gem_create)?;

        Ok(Buffer {
            handle: gem_create.handle,
            size,
            width,
            height,
            stride,
            tiling,
            _driver: DriverType::Amdgpu,
        })
    }

    /// Deallocate a GPU buffer.
    pub fn deallocate(&self, buffer: Buffer) -> io::Result<()> {
        self.device.gem_close(buffer.handle)
    }

    /// Export a buffer as a DMA-BUF file descriptor.
    ///
    /// The returned file descriptor can be shared with Wayland compositors
    /// or X11 servers for zero-copy buffer sharing.
    pub fn export_dmabuf(&self, buffer: &Buffer) -> io::Result<RawFd> {
        self.device.prime_handle_to_fd(buffer.handle)
    }

    /// Get a reference to the underlying DRM device.
    ///
    /// Useful for direct ioctl access (e.g., batch submission).
    pub fn device(&self) -> &DrmDevice {
        &self.device
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tiling_format_equality() {
        assert_eq!(TilingFormat::None, TilingFormat::None);
        assert_ne!(TilingFormat::None, TilingFormat::X);
        assert_ne!(TilingFormat::X, TilingFormat::Y);
    }

    #[test]
    fn buffer_stride_calculation() {
        // Linear: exact width * bpp
        let width: u32 = 1920;
        let bpp: u32 = 32;
        let stride = width * (bpp / 8);
        assert_eq!(stride, 7680);

        // X-tiled: align to 512 bytes
        let bytes_per_row: u32 = width * (bpp / 8);
        assert_eq!(bytes_per_row, 7680);
        let x_stride: u32 = ((bytes_per_row + 511) / 512) * 512;
        // Manual calculation: 7680 is already aligned to 512 (7680 / 512 = 15 exactly)
        // So (7680 + 511) / 512 = 8191 / 512 = 15 (integer division), 15 * 512 = 7680
        assert_eq!(x_stride, 7680, "X-tiled stride for already-aligned value stays same");

        // Y-tiled: align to 128 bytes
        let y_stride: u32 = ((bytes_per_row + 127) / 128) * 128;
        // Manual calculation: 7680 is already aligned to 128 (7680 / 128 = 60 exactly)
        assert_eq!(y_stride, 7680, "Y-tiled stride for already-aligned value stays same");

        // Test unaligned width that requires alignment
        let unaligned_width: u32 = 1000; // 1000 * 4 = 4000 bytes (not aligned to 512)
        let unaligned_bytes: u32 = unaligned_width * (bpp / 8);
        assert_eq!(unaligned_bytes, 4000);
        
        // X-tiled alignment: should round up to next 512-byte boundary
        let x_aligned: u32 = ((unaligned_bytes + 511) / 512) * 512;
        // (4000 + 511) / 512 = 4511 / 512 = 8 (integer), 8 * 512 = 4096
        assert_eq!(x_aligned, 4096, "X-tiled should align 4000 to 4096");
        
        // Y-tiled alignment: should round up to next 128-byte boundary
        let y_aligned: u32 = ((unaligned_bytes + 127) / 128) * 128;
        // (4000 + 127) / 128 = 4127 / 128 = 32 (integer), 32 * 128 = 4096
        assert_eq!(y_aligned, 4096, "Y-tiled should align 4000 to 4096");
    }

    #[test]
    fn driver_type_equality() {
        assert_eq!(DriverType::I915, DriverType::I915);
        assert_ne!(DriverType::I915, DriverType::Xe);
    }
}
