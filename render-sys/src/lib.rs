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
