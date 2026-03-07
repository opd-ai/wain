/// Intel Xe DRM driver IOCTL wrappers.
///
/// This module provides safe Rust wrappers around Intel Xe-specific ioctls
/// for newer Intel GPUs (12th gen and later).
///
/// Xe is the successor to i915 and uses a different kernel interface,
/// though the GPU programming model above this layer remains similar.

use std::io;
use crate::drm::DrmDevice;

/// DRM_IOCTL_XE_DEVICE_QUERY: Query device capabilities.
///
/// Used to detect GPU properties, engine classes, memory regions, etc.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct DeviceQuery {
    pub extensions: u64,
    pub query: u32,
    pub size: u32,
    pub data: u64,  // pointer to query-specific data
}

/// Query types for DRM_IOCTL_XE_DEVICE_QUERY.
pub const DRM_XE_DEVICE_QUERY_ENGINES: u32 = 0;
pub const DRM_XE_DEVICE_QUERY_MEM_REGIONS: u32 = 1;
pub const DRM_XE_DEVICE_QUERY_CONFIG: u32 = 2;
pub const DRM_XE_DEVICE_QUERY_GT_LIST: u32 = 3;

impl DeviceQuery {
    /// Create a new DeviceQuery request.
    pub fn new(query: u32, data: u64, size: u32) -> Self {
        Self {
            extensions: 0,
            query,
            size,
            data,
        }
    }
}

/// DRM_IOCTL_XE_GEM_CREATE: Allocate a GEM buffer object.
///
/// Xe uses a different creation interface than i915, with explicit
/// placement flags for memory regions.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemCreate {
    pub extensions: u64,
    pub size: u64,
    pub placement: u32,
    pub flags: u32,
    pub vm_id: u32,
    pub handle: u32,       // out: GEM handle
    pub cpu_caching: u32,
    pub pad: u32,
}

/// Placement flags for DRM_IOCTL_XE_GEM_CREATE.
pub const XE_GEM_CREATE_FLAG_DEFER_BACKING: u32 = 1 << 0;
pub const XE_GEM_CREATE_FLAG_SCANOUT: u32 = 1 << 1;

/// Memory placement regions for Xe GEM buffers.
pub const XE_GEM_CREATE_PLACEMENT_SYSTEM: u32 = 1 << 0;
pub const XE_GEM_CREATE_PLACEMENT_VRAM0: u32 = 1 << 1;

/// CPU caching modes for Xe GEM buffers.
pub const XE_GEM_CPU_CACHING_WB: u32 = 0;   // Write-back
pub const XE_GEM_CPU_CACHING_WC: u32 = 1;   // Write-combining

impl GemCreate {
    /// Create a new GemCreate request.
    pub fn new(size: u64, placement: u32) -> Self {
        Self {
            extensions: 0,
            size,
            placement,
            flags: 0,
            vm_id: 0,
            handle: 0,
            cpu_caching: XE_GEM_CPU_CACHING_WB,
            pad: 0,
        }
    }

    /// Set CPU caching mode.
    pub fn with_cpu_caching(mut self, caching: u32) -> Self {
        self.cpu_caching = caching;
        self
    }

    /// Set VM ID for this buffer.
    pub fn with_vm(mut self, vm_id: u32) -> Self {
        self.vm_id = vm_id;
        self
    }
}

/// DRM_IOCTL_XE_VM_CREATE: Create a GPU virtual memory context.
///
/// Xe uses explicit VM management instead of i915's implicit address spaces.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VmCreate {
    pub extensions: u64,
    pub flags: u32,
    pub vm_id: u32,        // out: VM ID
}

/// Flags for DRM_IOCTL_XE_VM_CREATE.
pub const DRM_XE_VM_CREATE_SCRATCH_PAGE: u32 = 1 << 0;
pub const DRM_XE_VM_CREATE_COMPUTE_MODE: u32 = 1 << 1;

impl VmCreate {
    /// Create a new VmCreate request.
    pub fn new() -> Self {
        Self {
            extensions: 0,
            flags: DRM_XE_VM_CREATE_SCRATCH_PAGE,
            vm_id: 0,
        }
    }

    /// Enable compute mode for this VM.
    pub fn with_compute_mode(mut self) -> Self {
        self.flags |= DRM_XE_VM_CREATE_COMPUTE_MODE;
        self
    }
}

impl Default for VmCreate {
    fn default() -> Self {
        Self::new()
    }
}

/// DRM_IOCTL_XE_VM_BIND: Bind a GEM buffer to a VM address range.
///
/// Xe requires explicit VM binding instead of relocation-based addressing.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VmBind {
    pub extensions: u64,
    pub vm_id: u32,
    pub exec_queue_id: u32,
    pub num_binds: u32,
    pub pad: u32,
    pub binds: u64,        // pointer to bind_op array
    pub num_syncs: u32,
    pub pad2: u32,
    pub syncs: u64,        // pointer to sync array
}

impl VmBind {
    /// Create a new VmBind request.
    pub fn new(vm_id: u32, binds_ptr: u64, num_binds: u32) -> Self {
        Self {
            extensions: 0,
            vm_id,
            exec_queue_id: 0,
            num_binds,
            pad: 0,
            binds: binds_ptr,
            num_syncs: 0,
            pad2: 0,
            syncs: 0,
        }
    }
}

/// Bind operation for DRM_IOCTL_XE_VM_BIND.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VmBindOp {
    pub extensions: u64,
    pub obj: u32,          // GEM handle (0 = unbind)
    pub pad: u32,
    pub range: u64,        // size in bytes
    pub addr: u64,         // GPU virtual address
    pub op: u32,
    pub flags: u32,
    pub prefetch_mem_region_instance: u32,
    pub pad2: u32,
}

/// Operation types for VmBindOp.
pub const XE_VM_BIND_OP_MAP: u32 = 0;
pub const XE_VM_BIND_OP_UNMAP: u32 = 1;
pub const XE_VM_BIND_OP_MAP_USERPTR: u32 = 2;
pub const XE_VM_BIND_OP_PREFETCH: u32 = 3;

/// Flags for VmBindOp.
pub const XE_VM_BIND_FLAG_READONLY: u32 = 1 << 0;
pub const XE_VM_BIND_FLAG_IMMEDIATE: u32 = 1 << 1;

impl VmBindOp {
    /// Create a new bind operation.
    pub fn new_map(obj: u32, addr: u64, range: u64) -> Self {
        Self {
            extensions: 0,
            obj,
            pad: 0,
            range,
            addr,
            op: XE_VM_BIND_OP_MAP,
            flags: 0,
            prefetch_mem_region_instance: 0,
            pad2: 0,
        }
    }

    /// Create a new unmap operation.
    pub fn new_unmap(addr: u64, range: u64) -> Self {
        Self {
            extensions: 0,
            obj: 0,
            pad: 0,
            range,
            addr,
            op: XE_VM_BIND_OP_UNMAP,
            flags: 0,
            prefetch_mem_region_instance: 0,
            pad2: 0,
        }
    }
}

/// DRM_IOCTL_XE_EXEC: Submit GPU execution queue.
///
/// Xe's execution model uses queues instead of i915's execbuffer.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct Exec {
    pub extensions: u64,
    pub exec_queue_id: u32,
    pub num_batch_buffer: u32,
    pub address: u64,      // GPU address of batch buffer
    pub num_syncs: u32,
    pub pad: u32,
    pub syncs: u64,        // pointer to sync array
}

impl Exec {
    /// Create a new Exec request.
    pub fn new(exec_queue_id: u32, address: u64) -> Self {
        Self {
            extensions: 0,
            exec_queue_id,
            num_batch_buffer: 1,
            address,
            num_syncs: 0,
            pad: 0,
            syncs: 0,
        }
    }
}

/// DRM_IOCTL_XE_EXEC_QUEUE_CREATE: Create an execution queue.
///
/// Execution queues are the Xe equivalent of i915 contexts.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ExecQueueCreate {
    pub extensions: u64,
    pub vm_id: u32,
    pub width: u16,
    pub num_placements: u16,
    pub instances: u64,    // pointer to engine instance array
    pub exec_queue_id: u32,  // out: execution queue ID
    pub flags: u32,
}

/// Flags for DRM_IOCTL_XE_EXEC_QUEUE_CREATE.
pub const DRM_XE_EXEC_QUEUE_CREATE_PRIORITY_LOW: u32 = 0 << 0;
pub const DRM_XE_EXEC_QUEUE_CREATE_PRIORITY_NORMAL: u32 = 1 << 0;
pub const DRM_XE_EXEC_QUEUE_CREATE_PRIORITY_HIGH: u32 = 2 << 0;

impl ExecQueueCreate {
    /// Create a new ExecQueueCreate request.
    pub fn new(vm_id: u32, instances: u64, num_placements: u16) -> Self {
        Self {
            extensions: 0,
            vm_id,
            width: 1,
            num_placements,
            instances,
            exec_queue_id: 0,
            flags: DRM_XE_EXEC_QUEUE_CREATE_PRIORITY_NORMAL,
        }
    }
}

const DRM_IOCTL_BASE: u8 = b'd';
const DRM_COMMAND_BASE: u64 = 0x40;

// Xe ioctl numbers
const DRM_XE_DEVICE_QUERY: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x00, std::mem::size_of::<DeviceQuery>());
const DRM_XE_GEM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x01, std::mem::size_of::<GemCreate>());
const DRM_XE_VM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x03, std::mem::size_of::<VmCreate>());
const DRM_XE_VM_BIND: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x05, std::mem::size_of::<VmBind>());
const DRM_XE_EXEC: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x06, std::mem::size_of::<Exec>());
const DRM_XE_EXEC_QUEUE_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x09, std::mem::size_of::<ExecQueueCreate>());

impl DrmDevice {
    /// Query device capabilities (Xe-specific).
    pub fn xe_device_query(&self, req: &mut DeviceQuery) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_DEVICE_QUERY as _, req as *mut DeviceQuery)
        };
        Ok(())
    }

    /// Allocate a GEM buffer (Xe-specific).
    pub fn xe_gem_create(&self, req: &mut GemCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_GEM_CREATE as _, req as *mut GemCreate)
        };
        Ok(())
    }

    /// Create a GPU virtual memory context (Xe-specific).
    pub fn xe_vm_create(&self, req: &mut VmCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_VM_CREATE as _, req as *mut VmCreate)
        };
        Ok(())
    }

    /// Bind a GEM buffer to a VM address (Xe-specific).
    pub fn xe_vm_bind(&self, req: &mut VmBind) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_VM_BIND as _, req as *mut VmBind)
        };
        Ok(())
    }

    /// Submit execution queue (Xe-specific).
    pub fn xe_exec(&self, req: &mut Exec) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_EXEC as _, req as *mut Exec)
        };
        Ok(())
    }

    /// Create an execution queue (Xe-specific).
    pub fn xe_exec_queue_create(&self, req: &mut ExecQueueCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_XE_EXEC_QUEUE_CREATE as _, req as *mut ExecQueueCreate)
        };
        Ok(())
    }
}

/// Detect at runtime whether i915 or Xe driver is active.
///
/// This function attempts to detect the active Intel DRM driver by checking
/// the driver name via DRM_IOCTL_VERSION.
pub enum IntelDriver {
    I915,
    Xe,
    Unknown,
}

impl DrmDevice {
    /// Detect which Intel driver is active (i915 vs Xe).
    ///
    /// Uses DRM_IOCTL_VERSION to query the driver name.
    pub fn detect_intel_driver(&self) -> io::Result<IntelDriver> {
        // DRM_IOCTL_VERSION structure
        #[repr(C)]
        struct DrmVersion {
            version_major: i32,
            version_minor: i32,
            version_patchlevel: i32,
            name_len: usize,
            name: *mut u8,
            date_len: usize,
            date: *mut u8,
            desc_len: usize,
            desc: *mut u8,
        }

        let mut name_buf = [0u8; 16];
        let mut version = DrmVersion {
            version_major: 0,
            version_minor: 0,
            version_patchlevel: 0,
            name_len: name_buf.len(),
            name: name_buf.as_mut_ptr(),
            date_len: 0,
            date: std::ptr::null_mut(),
            desc_len: 0,
            desc: std::ptr::null_mut(),
        };

        const DRM_IOCTL_VERSION: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, 0x00, std::mem::size_of::<DrmVersion>());
        
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_VERSION as _, &mut version as *mut DrmVersion)
        };

        let name = std::str::from_utf8(&name_buf[..version.name_len])
            .unwrap_or("")
            .trim_end_matches('\0');

        Ok(match name {
            "i915" => IntelDriver::I915,
            "xe" => IntelDriver::Xe,
            _ => IntelDriver::Unknown,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn device_query_request() {
        let req = DeviceQuery::new(DRM_XE_DEVICE_QUERY_CONFIG, 0x1000, 64);
        assert_eq!(req.query, DRM_XE_DEVICE_QUERY_CONFIG);
        assert_eq!(req.data, 0x1000);
        assert_eq!(req.size, 64);
    }

    #[test]
    fn gem_create_request() {
        let req = GemCreate::new(4096, 1);
        assert_eq!(req.size, 4096);
        assert_eq!(req.placement, 1);
        assert_eq!(req.cpu_caching, XE_GEM_CPU_CACHING_WB);
    }

    #[test]
    fn vm_create_request() {
        let req = VmCreate::new();
        assert_eq!(req.flags & DRM_XE_VM_CREATE_SCRATCH_PAGE, DRM_XE_VM_CREATE_SCRATCH_PAGE);
    }

    #[test]
    fn vm_bind_op_map() {
        let op = VmBindOp::new_map(42, 0x10000, 4096);
        assert_eq!(op.obj, 42);
        assert_eq!(op.addr, 0x10000);
        assert_eq!(op.range, 4096);
        assert_eq!(op.op, XE_VM_BIND_OP_MAP);
    }

    #[test]
    fn vm_bind_op_unmap() {
        let op = VmBindOp::new_unmap(0x10000, 4096);
        assert_eq!(op.obj, 0);
        assert_eq!(op.addr, 0x10000);
        assert_eq!(op.range, 4096);
        assert_eq!(op.op, XE_VM_BIND_OP_UNMAP);
    }

    #[test]
    fn exec_request() {
        let req = Exec::new(1, 0x20000);
        assert_eq!(req.exec_queue_id, 1);
        assert_eq!(req.address, 0x20000);
        assert_eq!(req.num_batch_buffer, 1);
    }

    #[test]
    fn exec_queue_create_request() {
        let req = ExecQueueCreate::new(1, 0x1000, 2);
        assert_eq!(req.vm_id, 1);
        assert_eq!(req.instances, 0x1000);
        assert_eq!(req.num_placements, 2);
    }
}
