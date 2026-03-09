/// DRM/KMS kernel IOCTL wrappers.
///
/// This module provides safe Rust wrappers around DRM (Direct Rendering Manager)
/// ioctls for GPU buffer allocation and management.

use std::fs::File;
use std::os::unix::io::{AsRawFd, RawFd};
use std::io;

/// Execute an ioctl and return an error if it fails.
///
/// Unlike calling `nix::libc::ioctl` directly, this function checks the return
/// value and converts a negative result into an `io::Error`. Callers must not
/// ignore the returned `Result` — on failure the output fields of `arg` are
/// left in an undefined state.
pub(crate) fn checked_ioctl<T>(fd: RawFd, request: nix::libc::c_ulong, arg: *mut T) -> io::Result<()> {
    // SAFETY: ioctl syscall requires:
    // - Valid file descriptor (caller responsibility)
    // - arg points to initialized memory of type matching the ioctl request (caller responsibility)
    // - arg remains valid for call duration (synchronous syscall)
    // - On failure, arg contents are undefined (documented in function comment)
    let ret = unsafe { nix::libc::ioctl(fd, request as nix::libc::Ioctl, arg) };
    if ret < 0 {
        Err(io::Error::last_os_error())
    } else {
        Ok(())
    }
}

/// DRM device file handle.
pub struct DrmDevice {
    file: File,
}

impl DrmDevice {
    /// Open a DRM device from a path (e.g., "/dev/dri/renderD128").
    pub fn open(path: &str) -> io::Result<Self> {
        let file = File::options()
            .read(true)
            .write(true)
            .open(path)?;
        Ok(Self { file })
    }

    /// Get the raw file descriptor for ioctl calls.
    pub fn fd(&self) -> RawFd {
        self.file.as_raw_fd()
    }
}

/// DRM_IOCTL_MODE_CREATE_DUMB: Create a dumb buffer (CPU-accessible framebuffer).
///
/// Dumb buffers are simple, linear, CPU-accessible framebuffers suitable for
/// software rendering or as fallback when GPU buffers are not available.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct CreateDumb {
    pub height: u32,
    pub width: u32,
    pub bpp: u32,      // bits per pixel
    pub flags: u32,
    pub handle: u32,   // returned: GEM handle
    pub pitch: u32,    // returned: stride in bytes
    pub size: u64,     // returned: size in bytes
}

impl CreateDumb {
    /// Create a new CreateDumb request.
    pub fn new(width: u32, height: u32, bpp: u32) -> Self {
        Self {
            height,
            width,
            bpp,
            flags: 0,
            handle: 0,
            pitch: 0,
            size: 0,
        }
    }
}

const DRM_IOCTL_BASE: u8 = b'd';
const DRM_IOCTL_MODE_CREATE_DUMB: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, 0xB2, std::mem::size_of::<CreateDumb>());

impl DrmDevice {
    /// Allocate a dumb buffer.
    pub fn create_dumb(&self, req: &mut CreateDumb) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_IOCTL_MODE_CREATE_DUMB as u64, req as *mut CreateDumb)
    }
}

/// DRM_IOCTL_GEM_CLOSE: Release a GEM buffer object.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemClose {
    pub handle: u32,
    pub pad: u32,
}

const DRM_IOCTL_GEM_CLOSE: nix::libc::Ioctl = nix::request_code_write!(DRM_IOCTL_BASE, 0x09, std::mem::size_of::<GemClose>());

impl DrmDevice {
    /// Close (free) a GEM buffer object.
    pub fn gem_close(&self, handle: u32) -> io::Result<()> {
        let mut req = GemClose { handle, pad: 0 };
        checked_ioctl(self.fd(), DRM_IOCTL_GEM_CLOSE as u64, &mut req as *mut GemClose)
    }
}

/// DRM_IOCTL_PRIME_HANDLE_TO_FD: Export a GEM handle as a DMA-BUF fd.
///
/// This allows sharing GPU buffers with Wayland compositors and other processes.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct PrimeHandleToFd {
    pub handle: u32,
    pub flags: u32,
    pub fd: i32,       // returned: DMA-BUF file descriptor
}

const DRM_IOCTL_PRIME_HANDLE_TO_FD: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, 0x2D, std::mem::size_of::<PrimeHandleToFd>());
const DRM_CLOEXEC: u32 = 0x80000000;
const DRM_RDWR: u32 = 0x00000002;

impl DrmDevice {
    /// Export a GEM buffer as a DMA-BUF file descriptor.
    pub fn prime_handle_to_fd(&self, handle: u32) -> io::Result<RawFd> {
        let mut req = PrimeHandleToFd {
            handle,
            flags: DRM_CLOEXEC | DRM_RDWR,
            fd: -1,
        };
        checked_ioctl(self.fd(), DRM_IOCTL_PRIME_HANDLE_TO_FD as u64, &mut req as *mut PrimeHandleToFd)?;
        Ok(req.fd)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn create_dumb_request() {
        let req = CreateDumb::new(1920, 1080, 32);
        assert_eq!(req.width, 1920);
        assert_eq!(req.height, 1080);
        assert_eq!(req.bpp, 32);
        assert_eq!(req.flags, 0);
    }

    #[test]
    fn gem_close_structure() {
        let req = GemClose { handle: 42, pad: 0 };
        assert_eq!(req.handle, 42);
    }
}
