/// C-ABI boundary functions exposed to Go via CGO.
///
/// This module validates the full Go → Rust build pipeline.
/// Start with trivial arithmetic to confirm the ABI works before
/// adding GPU/rendering functionality.

pub mod drm;
pub mod i915;
pub mod xe;
pub mod allocator;
pub mod slab;
pub mod detect;
pub mod cmd;
pub mod batch;

use std::os::unix::io::RawFd;
use drm::DrmDevice;
use allocator::{BufferAllocator, Buffer, TilingFormat, DriverType};
use detect::GpuGeneration;

/// Add two 32-bit integers and return the result.
///
/// # Safety
/// This function is called from Go via CGO and must be exported with the
/// C calling convention.
#[no_mangle]
pub extern "C" fn render_add(a: i32, b: i32) -> i32 {
    a.wrapping_add(b)
}

/// Return the version of the render library as a null-terminated C string.
///
/// The returned pointer is valid for the lifetime of the process (static storage).
#[no_mangle]
pub extern "C" fn render_version() -> *const std::ffi::c_char {
    static VERSION: &[u8] = b"0.1.0\0";
    VERSION.as_ptr() as *const std::ffi::c_char
}

/// Detect GPU generation from the DRM device at the given path.
///
/// Returns an integer representing the GPU generation:
/// - 0: Unknown
/// - 9: Gen9 (Skylake/Kaby Lake/Coffee Lake)
/// - 11: Gen11 (Ice Lake)
/// - 12: Gen12 (Tiger Lake/Rocket Lake/Alder Lake)
/// - 13: Xe (Meteor Lake+)
/// - -1: Error opening device or querying
///
/// # Safety
/// - path must be a valid null-terminated C string
#[no_mangle]
pub unsafe extern "C" fn render_detect_gpu(path: *const std::ffi::c_char) -> i32 {
    if path.is_null() {
        return -1;
    }

    let c_str = match std::ffi::CStr::from_ptr(path).to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    let device = match DrmDevice::open(c_str) {
        Ok(d) => d,
        Err(_) => return -1,
    };

    let generation = match device.detect_gpu_generation() {
        Ok(g) => g,
        Err(_) => return -1,
    };

    match generation {
        GpuGeneration::Gen9 => 9,
        GpuGeneration::Gen11 => 11,
        GpuGeneration::Gen12 => 12,
        GpuGeneration::Xe => 13,
        GpuGeneration::Unknown => 0,
    }
}


/// Create a buffer allocator for the DRM device at the given path.
///
/// Returns an opaque pointer to the allocator or null on error.
/// The caller must call buffer_allocator_destroy to free the allocator.
///
/// # Safety
/// - path must be a valid null-terminated C string
/// - The returned pointer must be freed with buffer_allocator_destroy
#[no_mangle]
pub unsafe extern "C" fn buffer_allocator_create(path: *const std::ffi::c_char) -> *mut BufferAllocator {
    if path.is_null() {
        return std::ptr::null_mut();
    }

    let c_str = match std::ffi::CStr::from_ptr(path).to_str() {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    let device = match DrmDevice::open(c_str) {
        Ok(d) => d,
        Err(_) => return std::ptr::null_mut(),
    };

    // Detect driver type (for now, assume i915 - could be enhanced with device query)
    let driver = DriverType::I915;

    let allocator = Box::new(BufferAllocator::new(device, driver));
    Box::into_raw(allocator)
}

/// Destroy a buffer allocator created with buffer_allocator_create.
///
/// # Safety
/// - allocator must be a valid pointer returned by buffer_allocator_create
/// - allocator must not be used after this call
#[no_mangle]
pub unsafe extern "C" fn buffer_allocator_destroy(allocator: *mut BufferAllocator) {
    if !allocator.is_null() {
        let _ = Box::from_raw(allocator);
    }
}

/// Allocate a GPU buffer with the given dimensions and format.
///
/// Returns an opaque pointer to the buffer or null on error.
/// The caller must call buffer_destroy to free the buffer.
///
/// # Safety
/// - allocator must be a valid pointer returned by buffer_allocator_create
/// - The returned pointer must be freed with buffer_destroy
#[no_mangle]
pub unsafe extern "C" fn buffer_allocate(
    allocator: *mut BufferAllocator,
    width: u32,
    height: u32,
    bpp: u32,
    tiling: u32,
) -> *mut Buffer {
    if allocator.is_null() {
        return std::ptr::null_mut();
    }

    let alloc = &*allocator;

    let tiling_format = match tiling {
        0 => TilingFormat::None,
        1 => TilingFormat::X,
        2 => TilingFormat::Y,
        _ => return std::ptr::null_mut(),
    };

    match alloc.allocate(width, height, bpp, tiling_format) {
        Ok(buffer) => Box::into_raw(Box::new(buffer)),
        Err(_) => std::ptr::null_mut(),
    }
}

/// Export a buffer as a DMA-BUF file descriptor.
///
/// Returns the file descriptor or -1 on error.
/// The caller owns the fd and must close it when done.
///
/// # Safety
/// - allocator must be a valid pointer returned by buffer_allocator_create
/// - buffer must be a valid pointer returned by buffer_allocate
#[no_mangle]
pub unsafe extern "C" fn buffer_export_dmabuf(
    allocator: *mut BufferAllocator,
    buffer: *mut Buffer,
) -> RawFd {
    if allocator.is_null() || buffer.is_null() {
        return -1;
    }

    let alloc = &*allocator;
    let buf = &*buffer;

    match alloc.export_dmabuf(buf) {
        Ok(fd) => fd,
        Err(_) => -1,
    }
}

/// Get buffer dimensions and stride.
///
/// # Safety
/// - buffer must be a valid pointer returned by buffer_allocate
/// - out_width, out_height, out_stride must be valid pointers
#[no_mangle]
pub unsafe extern "C" fn buffer_get_info(
    buffer: *mut Buffer,
    out_width: *mut u32,
    out_height: *mut u32,
    out_stride: *mut u32,
) -> i32 {
    if buffer.is_null() || out_width.is_null() || out_height.is_null() || out_stride.is_null() {
        return -1;
    }

    let buf = &*buffer;
    *out_width = buf.width;
    *out_height = buf.height;
    *out_stride = buf.stride;
    0
}

/// Destroy a buffer created with buffer_allocate.
///
/// # Safety
/// - allocator must be a valid pointer returned by buffer_allocator_create
/// - buffer must be a valid pointer returned by buffer_allocate
/// - buffer must not be used after this call
#[no_mangle]
pub unsafe extern "C" fn buffer_destroy(allocator: *mut BufferAllocator, buffer: *mut Buffer) -> i32 {
    if allocator.is_null() || buffer.is_null() {
        return -1;
    }

    let alloc = &*allocator;
    let buf = Box::from_raw(buffer);

    match alloc.deallocate(*buf) {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn add_positive() {
        assert_eq!(render_add(2, 3), 5);
    }

    #[test]
    fn add_negative() {
        assert_eq!(render_add(-1, 1), 0);
    }

    #[test]
    fn add_overflow_wraps() {
        assert_eq!(render_add(i32::MAX, 1), i32::MIN);
    }
}
