/// Intel i915 DRM driver IOCTL wrappers.
///
/// This module provides safe Rust wrappers around Intel i915-specific ioctls
/// for GEM buffer management and GPU command submission.

use std::io;
use crate::drm::DrmDevice;

/// I915_GEM_CREATE: Allocate a GEM buffer object.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemCreate {
    pub size: u64,     // in: size in bytes
    pub handle: u32,   // out: GEM handle
    pub pad: u32,
}

impl GemCreate {
    /// Create a new GemCreate request.
    pub fn new(size: u64) -> Self {
        Self {
            size,
            handle: 0,
            pad: 0,
        }
    }
}

/// I915_GEM_MMAP_OFFSET: Get mmap offset for a GEM buffer.
///
/// Used to map a GEM buffer into CPU address space via mmap().
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemMmapOffset {
    pub handle: u32,
    pub flags: u32,
    pub offset: u64,   // out: offset for mmap()
}

/// Mmap type flags for I915_GEM_MMAP_OFFSET.
pub const I915_MMAP_OFFSET_WB: u32 = 0;   // Write-back cache mode
pub const I915_MMAP_OFFSET_WC: u32 = 1;   // Write-combining mode

impl GemMmapOffset {
    /// Create a new GemMmapOffset request.
    pub fn new(handle: u32, flags: u32) -> Self {
        Self {
            handle,
            flags,
            offset: 0,
        }
    }
}

/// I915_GEM_SET_TILING: Set tiling mode for a GEM buffer.
///
/// Tiling improves GPU cache performance for 2D operations.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemSetTiling {
    pub handle: u32,
    pub tiling_mode: u32,
    pub stride: u32,
    pub swizzle_mode: u32,  // out: swizzle mode
}

/// Tiling modes for I915_GEM_SET_TILING.
pub const I915_TILING_NONE: u32 = 0;
pub const I915_TILING_X: u32 = 1;
pub const I915_TILING_Y: u32 = 2;

impl GemSetTiling {
    /// Create a new GemSetTiling request.
    pub fn new(handle: u32, tiling_mode: u32, stride: u32) -> Self {
        Self {
            handle,
            tiling_mode,
            stride,
            swizzle_mode: 0,
        }
    }
}

/// I915_GEM_WAIT: Wait for a GEM buffer to become idle.
///
/// Used to synchronize CPU access after GPU operations.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemWait {
    pub bo_handle: u32,
    pub flags: u32,
    pub timeout_ns: i64,  // in: timeout in nanoseconds (-1 = infinite)
}

impl GemWait {
    /// Create a new GemWait request with infinite timeout.
    pub fn new(bo_handle: u32) -> Self {
        Self {
            bo_handle,
            flags: 0,
            timeout_ns: -1,
        }
    }

    /// Set a timeout in nanoseconds.
    pub fn with_timeout(mut self, timeout_ns: i64) -> Self {
        self.timeout_ns = timeout_ns;
        self
    }
}

/// I915_GEM_CONTEXT_CREATE: Create a GPU execution context.
///
/// Contexts isolate GPU state and allow multiple independent command streams.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ContextCreate {
    pub ctx_id: u32,   // out: context ID
    pub flags: u32,
}

impl ContextCreate {
    /// Create a new ContextCreate request.
    pub fn new() -> Self {
        Self {
            ctx_id: 0,
            flags: 0,
        }
    }
}

impl Default for ContextCreate {
    fn default() -> Self {
        Self::new()
    }
}

/// I915_GETPARAM: Query device parameters.
///
/// Used to detect GPU generation, available features, etc.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GetParam {
    pub param: i32,
    pub value: *mut i32,
}

/// Parameter codes for I915_GETPARAM.
pub const I915_PARAM_CHIPSET_ID: i32 = 4;
pub const I915_PARAM_HAS_EXECBUF2: i32 = 9;
pub const I915_PARAM_HAS_GEM: i32 = 5;

impl GetParam {
    /// Create a new GetParam request.
    pub fn new(param: i32, value: &mut i32) -> Self {
        Self {
            param,
            value: value as *mut i32,
        }
    }
}

/// I915_GEM_EXECBUFFER2: Submit GPU command buffer for execution.
///
/// This is the primary interface for submitting work to the GPU.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ExecBuffer2 {
    pub buffers_ptr: u64,       // pointer to exec_object2 array
    pub buffer_count: u32,
    pub batch_start_offset: u32,
    pub batch_len: u32,
    pub dr1: u32,
    pub dr4: u32,
    pub num_cliprects: u32,
    pub cliprects_ptr: u64,
    pub flags: u64,
    pub rsvd1: u64,
    pub rsvd2: u64,
}

/// Flags for I915_GEM_EXECBUFFER2.
pub const I915_EXEC_RENDER: u64 = 1 << 0;
pub const I915_EXEC_NO_RELOC: u64 = 1 << 11;

impl ExecBuffer2 {
    /// Create a new ExecBuffer2 request.
    pub fn new() -> Self {
        Self {
            buffers_ptr: 0,
            buffer_count: 0,
            batch_start_offset: 0,
            batch_len: 0,
            dr1: 0,
            dr4: 0,
            num_cliprects: 0,
            cliprects_ptr: 0,
            flags: I915_EXEC_RENDER,
            rsvd1: 0,
            rsvd2: 0,
        }
    }
}

impl Default for ExecBuffer2 {
    fn default() -> Self {
        Self::new()
    }
}

/// Execution object for I915_GEM_EXECBUFFER2.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ExecObject2 {
    pub handle: u32,
    pub relocation_count: u32,
    pub relocs_ptr: u64,
    pub alignment: u64,
    pub offset: u64,
    pub flags: u64,
    pub rsvd1: u64,
    pub rsvd2: u64,
}

const DRM_IOCTL_BASE: u8 = b'd';
const DRM_COMMAND_BASE: u64 = 0x40;

// i915 ioctl numbers
const I915_GEM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x1b, std::mem::size_of::<GemCreate>());
const I915_GEM_MMAP_OFFSET: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x24, std::mem::size_of::<GemMmapOffset>());
const I915_GEM_SET_TILING: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x21, std::mem::size_of::<GemSetTiling>());
const I915_GEM_WAIT: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x2c, std::mem::size_of::<GemWait>());
const I915_GEM_CONTEXT_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x2d, std::mem::size_of::<ContextCreate>());
const I915_GETPARAM: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x06, std::mem::size_of::<GetParam>());
const I915_GEM_EXECBUFFER2: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x29, std::mem::size_of::<ExecBuffer2>());

impl DrmDevice {
    /// Allocate a GEM buffer (i915-specific).
    pub fn i915_gem_create(&self, req: &mut GemCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_CREATE as _, req as *mut GemCreate)
        };
        Ok(())
    }

    /// Get mmap offset for a GEM buffer (i915-specific).
    pub fn i915_gem_mmap_offset(&self, req: &mut GemMmapOffset) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_MMAP_OFFSET as _, req as *mut GemMmapOffset)
        };
        Ok(())
    }

    /// Set tiling mode for a GEM buffer (i915-specific).
    pub fn i915_gem_set_tiling(&self, req: &mut GemSetTiling) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_SET_TILING as _, req as *mut GemSetTiling)
        };
        Ok(())
    }

    /// Wait for a GEM buffer to become idle (i915-specific).
    pub fn i915_gem_wait(&self, req: &mut GemWait) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_WAIT as _, req as *mut GemWait)
        };
        Ok(())
    }

    /// Create a GPU execution context (i915-specific).
    pub fn i915_context_create(&self, req: &mut ContextCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_CONTEXT_CREATE as _, req as *mut ContextCreate)
        };
        Ok(())
    }

    /// Query device parameters (i915-specific).
    pub fn i915_getparam(&self, req: &mut GetParam) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GETPARAM as _, req as *mut GetParam)
        };
        Ok(())
    }

    /// Submit command buffer for execution (i915-specific).
    pub fn i915_execbuffer2(&self, req: &mut ExecBuffer2) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), I915_GEM_EXECBUFFER2 as _, req as *mut ExecBuffer2)
        };
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gem_create_request() {
        let req = GemCreate::new(4096);
        assert_eq!(req.size, 4096);
        assert_eq!(req.handle, 0);
    }

    #[test]
    fn gem_mmap_offset_request() {
        let req = GemMmapOffset::new(42, I915_MMAP_OFFSET_WC);
        assert_eq!(req.handle, 42);
        assert_eq!(req.flags, I915_MMAP_OFFSET_WC);
    }

    #[test]
    fn gem_set_tiling_request() {
        let req = GemSetTiling::new(42, I915_TILING_X, 1024);
        assert_eq!(req.handle, 42);
        assert_eq!(req.tiling_mode, I915_TILING_X);
        assert_eq!(req.stride, 1024);
    }

    #[test]
    fn gem_wait_request() {
        let req = GemWait::new(42);
        assert_eq!(req.bo_handle, 42);
        assert_eq!(req.timeout_ns, -1);
    }

    #[test]
    fn context_create_request() {
        let req = ContextCreate::new();
        assert_eq!(req.ctx_id, 0);
        assert_eq!(req.flags, 0);
    }

    #[test]
    fn execbuffer2_request() {
        let req = ExecBuffer2::new();
        assert_eq!(req.flags & I915_EXEC_RENDER, I915_EXEC_RENDER);
    }
}
