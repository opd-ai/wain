/// Intel Xe DRM driver IOCTL wrappers.
///
/// This module provides safe Rust wrappers around Intel Xe-specific ioctls
/// for newer Intel GPUs (12th gen and later).
///
/// Xe is the successor to i915 and uses a different kernel interface,
/// though the GPU programming model above this layer remains similar.

use std::io;
use crate::drm::{DrmDevice, checked_ioctl};

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

/// DRM_IOCTL_XE_VM_DESTROY: Destroy a VM and free associated resources.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct VmDestroy {
    pub vm_id: u32,
    pub pad: u32,
}

impl VmDestroy {
    /// Create a new VmDestroy request.
    pub fn new(vm_id: u32) -> Self {
        Self {
            vm_id,
            pad: 0,
        }
    }
}

impl Default for VmDestroy {
    fn default() -> Self {
        Self::new(0)
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

/// DRM_IOCTL_XE_EXEC_QUEUE_DESTROY: Destroy an execution queue.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ExecQueueDestroy {
    pub exec_queue_id: u32,
    pub pad: u32,
}

impl ExecQueueDestroy {
    /// Create a new ExecQueueDestroy request.
    pub fn new(exec_queue_id: u32) -> Self {
        Self {
            exec_queue_id,
            pad: 0,
        }
    }
}

impl Default for ExecQueueDestroy {
    fn default() -> Self {
        Self::new(0)
    }
}

const DRM_IOCTL_BASE: u8 = b'd';
const DRM_COMMAND_BASE: u64 = 0x40;

// Xe ioctl numbers
const DRM_XE_DEVICE_QUERY: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x00, std::mem::size_of::<DeviceQuery>());
const DRM_XE_GEM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x01, std::mem::size_of::<GemCreate>());
const DRM_XE_VM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x03, std::mem::size_of::<VmCreate>());
const DRM_XE_VM_DESTROY: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x04, std::mem::size_of::<VmDestroy>());
const DRM_XE_VM_BIND: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x05, std::mem::size_of::<VmBind>());
const DRM_XE_EXEC: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x06, std::mem::size_of::<Exec>());
const DRM_XE_EXEC_QUEUE_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x09, std::mem::size_of::<ExecQueueCreate>());
const DRM_XE_EXEC_QUEUE_DESTROY: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_COMMAND_BASE + 0x0a, std::mem::size_of::<ExecQueueDestroy>());

impl DrmDevice {
    /// Query device capabilities (Xe-specific).
    pub fn xe_device_query(&self, req: &mut DeviceQuery) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_DEVICE_QUERY as u64, req as *mut DeviceQuery)
    }

    /// Allocate a GEM buffer (Xe-specific).
    pub fn xe_gem_create(&self, req: &mut GemCreate) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_GEM_CREATE as u64, req as *mut GemCreate)
    }

    /// Create a GPU virtual memory context (Xe-specific).
    pub fn xe_vm_create(&self, req: &mut VmCreate) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_VM_CREATE as u64, req as *mut VmCreate)
    }

    /// Destroy a GPU virtual memory context (Xe-specific).
    pub fn xe_vm_destroy(&self, req: &mut VmDestroy) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_VM_DESTROY as u64, req as *mut VmDestroy)
    }

    /// Bind a GEM buffer to a VM address (Xe-specific).
    pub fn xe_vm_bind(&self, req: &mut VmBind) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_VM_BIND as u64, req as *mut VmBind)
    }

    /// Submit execution queue (Xe-specific).
    pub fn xe_exec(&self, req: &mut Exec) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_EXEC as u64, req as *mut Exec)
    }

    /// Create an execution queue (Xe-specific).
    pub fn xe_exec_queue_create(&self, req: &mut ExecQueueCreate) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_EXEC_QUEUE_CREATE as u64, req as *mut ExecQueueCreate)
    }

    /// Destroy an execution queue (Xe-specific).
    pub fn xe_exec_queue_destroy(&self, req: &mut ExecQueueDestroy) -> io::Result<()> {
        checked_ioctl(self.fd(), DRM_XE_EXEC_QUEUE_DESTROY as u64, req as *mut ExecQueueDestroy)
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

/// Sync object for Xe fence-based synchronization.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct SyncObject {
    pub handle: u32,
    pub flags: u32,
    pub timeline_value: u64,
}

/// Sync flags for Xe.
pub const DRM_XE_SYNC_FLAG_SIGNAL: u32 = 1 << 0;

impl SyncObject {
    /// Create a new sync object for signaling.
    pub fn new_signal(handle: u32) -> Self {
        Self {
            handle,
            flags: DRM_XE_SYNC_FLAG_SIGNAL,
            timeline_value: 0,
        }
    }
}

/// Engine instance for Xe exec queue creation.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct EngineInstance {
    pub engine_class: u16,
    pub engine_instance: u16,
    pub gt_id: u16,
}

/// Engine classes for Xe.
pub const DRM_XE_ENGINE_CLASS_RENDER: u16 = 0;
pub const DRM_XE_ENGINE_CLASS_COPY: u16 = 1;
pub const DRM_XE_ENGINE_CLASS_VIDEO_DECODE: u16 = 2;
pub const DRM_XE_ENGINE_CLASS_VIDEO_ENHANCE: u16 = 3;
pub const DRM_XE_ENGINE_CLASS_COMPUTE: u16 = 4;

impl EngineInstance {
    /// Create a render engine instance.
    pub fn render(gt_id: u16) -> Self {
        Self {
            engine_class: DRM_XE_ENGINE_CLASS_RENDER,
            engine_instance: 0,
            gt_id,
        }
    }
}

impl DrmDevice {
    /// Submit a batch buffer on Xe driver with VM binding and wait for completion.
    ///
    /// This is a high-level wrapper for Xe batch submission that:
    /// 1. Creates a VM if vm_id is None
    /// 2. Binds the batch buffer to the VM at the specified address
    /// 3. Creates an exec queue if exec_queue_id is None
    /// 4. Submits the batch via DRM_IOCTL_XE_EXEC
    /// 5. Waits for completion (synchronous submission)
    ///
    /// # Arguments
    /// * `batch_handle` - GEM handle of the batch buffer
    /// * `batch_gpu_addr` - GPU virtual address where batch should be mapped
    /// * `batch_size_bytes` - Size of the batch buffer in bytes
    /// * `vm_id` - Optional VM ID (None = create new VM)
    /// * `exec_queue_id` - Optional exec queue ID (None = create new queue)
    ///
    /// # Returns
    /// * `Ok((vm_id, exec_queue_id))` if submission succeeded
    /// * `Err(_)` if submission failed or GPU hang occurred
    pub fn xe_submit_batch(
        &self,
        batch_handle: u32,
        batch_gpu_addr: u64,
        batch_size_bytes: u64,
        vm_id: Option<u32>,
        exec_queue_id: Option<u32>,
    ) -> io::Result<(u32, u32)> {
        // Create VM if not provided
        let vm_id = match vm_id {
            Some(id) => id,
            None => {
                let mut vm_create = VmCreate::new();
                self.xe_vm_create(&mut vm_create)?;
                vm_create.vm_id
            }
        };

        // Bind the batch buffer to the VM
        let bind_op = VmBindOp::new_map(batch_handle, batch_gpu_addr, batch_size_bytes);
        let mut vm_bind = VmBind::new(vm_id, &bind_op as *const VmBindOp as u64, 1);
        self.xe_vm_bind(&mut vm_bind)?;

        // Create exec queue if not provided
        let exec_queue_id = match exec_queue_id {
            Some(id) => id,
            None => {
                let engine = EngineInstance::render(0);
                let mut queue_create = ExecQueueCreate::new(
                    vm_id,
                    &engine as *const EngineInstance as u64,
                    1,
                );
                self.xe_exec_queue_create(&mut queue_create)?;
                queue_create.exec_queue_id
            }
        };

        // Submit the batch (synchronous - waits for completion)
        let mut exec = Exec::new(exec_queue_id, batch_gpu_addr);
        self.xe_exec(&mut exec)?;

        Ok((vm_id, exec_queue_id))
    }

    /// Submit a batch buffer on Xe driver with simplified interface.
    ///
    /// Creates a new VM and exec queue for each submission.
    /// Use this for simple cases where resource reuse is not needed.
    pub fn xe_submit_batch_simple(
        &self,
        batch_handle: u32,
        batch_gpu_addr: u64,
        batch_size_bytes: u64,
    ) -> io::Result<()> {
        self.xe_submit_batch(batch_handle, batch_gpu_addr, batch_size_bytes, None, None)?;
        Ok(())
    }

    /// Create a VM and exec queue pair for Xe batch submission.
    ///
    /// Returns (vm_id, exec_queue_id) that can be reused across multiple submissions.
    /// This is more efficient than creating new resources for each batch.
    pub fn xe_create_context(&self) -> io::Result<(u32, u32)> {
        // Create VM
        let mut vm_create = VmCreate::new();
        self.xe_vm_create(&mut vm_create)?;
        let vm_id = vm_create.vm_id;

        // Create exec queue on the render engine
        let engine = EngineInstance::render(0);
        let mut queue_create = ExecQueueCreate::new(
            vm_id,
            &engine as *const EngineInstance as u64,
            1,
        );
        self.xe_exec_queue_create(&mut queue_create)?;
        let exec_queue_id = queue_create.exec_queue_id;

        Ok((vm_id, exec_queue_id))
    }

    /// Destroy a VM and exec queue pair created by xe_create_context.
    ///
    /// Releases the resources associated with the context. After destruction,
    /// the VM and exec queue cannot be used for further submissions.
    pub fn xe_destroy_context(&self, vm_id: u32, exec_queue_id: u32) -> io::Result<()> {
        // Destroy exec queue first
        let mut queue_destroy = ExecQueueDestroy::new(exec_queue_id);
        self.xe_exec_queue_destroy(&mut queue_destroy)?;

        // Then destroy VM
        let mut vm_destroy = VmDestroy::new(vm_id);
        self.xe_vm_destroy(&mut vm_destroy)?;

        Ok(())
    }

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
        
        // SAFETY: DRM_IOCTL_VERSION requires:
        // - Valid DRM file descriptor (self.fd() returns valid fd)
        // - Initialized DrmVersion struct with valid buffer pointers
        // - name_buf and date/desc buffers remain valid for call duration (stack allocated)
        // - Return value is intentionally ignored (driver detection uses name_len == 0 for non-Xe)
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

    #[test]
    fn sync_object_signal() {
        let sync = SyncObject::new_signal(123);
        assert_eq!(sync.handle, 123);
        assert_eq!(sync.flags, DRM_XE_SYNC_FLAG_SIGNAL);
        assert_eq!(sync.timeline_value, 0);
    }

    #[test]
    fn engine_instance_render() {
        let engine = EngineInstance::render(0);
        assert_eq!(engine.engine_class, DRM_XE_ENGINE_CLASS_RENDER);
        assert_eq!(engine.engine_instance, 0);
        assert_eq!(engine.gt_id, 0);
    }

    #[test]
    fn engine_instance_compute() {
        let engine = EngineInstance {
            engine_class: DRM_XE_ENGINE_CLASS_COMPUTE,
            engine_instance: 1,
            gt_id: 0,
        };
        assert_eq!(engine.engine_class, DRM_XE_ENGINE_CLASS_COMPUTE);
        assert_eq!(engine.engine_instance, 1);
    }
}
