/// AMD AMDGPU DRM driver IOCTL wrappers.
///
/// This module provides safe Rust wrappers around AMD-specific ioctls
/// for GEM buffer management and GPU command submission via the AMDGPU driver.
///
/// Target: RDNA2+ (RX 6000 series and newer, Steam Deck APU)
/// Reference: Mesa src/amd/common/ and Linux kernel drm/amd/amdgpu/

use std::io;
use crate::drm::DrmDevice;

/// AMDGPU_GEM_CREATE: Allocate a GEM buffer object.
///
/// AMDGPU uses a domain-based allocation model: buffers can be placed
/// in VRAM, GTT (system memory accessible by GPU), or CPU.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemCreate {
    pub size: u64,              // in: size in bytes
    pub alignment: u64,         // in: alignment requirement
    pub domains: u64,           // in: placement domains (VRAM, GTT, etc.)
    pub flags: u64,             // in: creation flags
    pub handle: u32,            // out: GEM handle
    pub pad: u32,
}

/// GEM domain flags for AMDGPU_GEM_CREATE.
pub const AMDGPU_GEM_DOMAIN_CPU: u64 = 0x1;      // CPU accessible memory
pub const AMDGPU_GEM_DOMAIN_GTT: u64 = 0x2;      // GTT (system memory, GPU accessible)
pub const AMDGPU_GEM_DOMAIN_VRAM: u64 = 0x4;     // VRAM (dedicated GPU memory)
pub const AMDGPU_GEM_DOMAIN_GDS: u64 = 0x8;      // Global Data Share
pub const AMDGPU_GEM_DOMAIN_GWS: u64 = 0x10;     // Global Wave Sync
pub const AMDGPU_GEM_DOMAIN_OA: u64 = 0x20;      // Ordered Append

/// GEM creation flags for AMDGPU_GEM_CREATE.
pub const AMDGPU_GEM_CREATE_CPU_ACCESS_REQUIRED: u64 = 1 << 0;
pub const AMDGPU_GEM_CREATE_NO_CPU_ACCESS: u64 = 1 << 1;
pub const AMDGPU_GEM_CREATE_CPU_GTT_USWC: u64 = 1 << 2;  // Write-combining
pub const AMDGPU_GEM_CREATE_VRAM_CLEARED: u64 = 1 << 3;
pub const AMDGPU_GEM_CREATE_VM_ALWAYS_VALID: u64 = 1 << 4;
pub const AMDGPU_GEM_CREATE_EXPLICIT_SYNC: u64 = 1 << 5;

impl GemCreate {
    /// Create a new GemCreate request.
    ///
    /// # Arguments
    ///
    /// * `size` - Buffer size in bytes
    /// * `domains` - Placement domains (VRAM, GTT, etc.)
    /// * `flags` - Creation flags
    pub fn new(size: u64, domains: u64, flags: u64) -> Self {
        Self {
            size,
            alignment: 4096,  // Default 4KB alignment
            domains,
            flags,
            handle: 0,
            pad: 0,
        }
    }

    /// Set alignment requirement.
    pub fn with_alignment(mut self, alignment: u64) -> Self {
        self.alignment = alignment;
        self
    }
}

/// AMDGPU_GEM_MMAP: Get mmap offset for a GEM buffer.
///
/// Used to map a GEM buffer into CPU address space via mmap().
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemMmap {
    pub handle: u32,           // in: GEM handle
    pub pad: u32,
    pub offset: u64,           // out: offset for mmap()
}

impl GemMmap {
    /// Create a new GemMmap request.
    pub fn new(handle: u32) -> Self {
        Self {
            handle,
            pad: 0,
            offset: 0,
        }
    }
}

/// AMDGPU_GEM_WAIT_IDLE: Wait for a GEM buffer to become idle.
///
/// Used to synchronize CPU access after GPU operations.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemWaitIdle {
    pub handle: u32,           // in: GEM handle
    pub flags: u32,
    pub timeout: u64,          // in: timeout in nanoseconds
}

impl GemWaitIdle {
    /// Create a new GemWaitIdle request with infinite timeout.
    pub fn new(handle: u32) -> Self {
        Self {
            handle,
            flags: 0,
            timeout: !0u64,    // Infinite timeout
        }
    }

    /// Set a timeout in nanoseconds.
    pub fn with_timeout(mut self, timeout_ns: u64) -> Self {
        self.timeout = timeout_ns;
        self
    }
}

/// AMDGPU_GEM_VA: Map a GEM buffer into GPU virtual address space.
///
/// AMDGPU uses explicit GPU virtual memory management. Buffers must be
/// explicitly mapped to GPU VA ranges before they can be accessed.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GemVa {
    pub handle: u32,           // in: GEM handle
    pub operation: u32,        // in: operation (map/unmap/replace)
    pub flags: u32,            // in: mapping flags
    pub va_address: u64,       // in: GPU virtual address
    pub offset_in_bo: u64,     // in: offset within buffer
    pub map_size: u64,         // in: size to map
}

/// VA operation types for AMDGPU_GEM_VA.
pub const AMDGPU_VA_OP_MAP: u32 = 1;
pub const AMDGPU_VA_OP_UNMAP: u32 = 2;
pub const AMDGPU_VA_OP_REPLACE: u32 = 3;
pub const AMDGPU_VA_OP_CLEAR: u32 = 4;

/// VA mapping flags for AMDGPU_GEM_VA.
pub const AMDGPU_VM_PAGE_READABLE: u32 = 1 << 0;
pub const AMDGPU_VM_PAGE_WRITEABLE: u32 = 1 << 1;
pub const AMDGPU_VM_PAGE_EXECUTABLE: u32 = 1 << 2;

impl GemVa {
    /// Create a new GemVa map request.
    ///
    /// # Arguments
    ///
    /// * `handle` - GEM buffer handle
    /// * `va_address` - GPU virtual address to map to
    /// * `size` - Size to map
    pub fn map(handle: u32, va_address: u64, size: u64) -> Self {
        Self {
            handle,
            operation: AMDGPU_VA_OP_MAP,
            flags: AMDGPU_VM_PAGE_READABLE | AMDGPU_VM_PAGE_WRITEABLE | AMDGPU_VM_PAGE_EXECUTABLE,
            va_address,
            offset_in_bo: 0,
            map_size: size,
        }
    }

    /// Create a new GemVa unmap request.
    pub fn unmap(va_address: u64, size: u64) -> Self {
        Self {
            handle: 0,
            operation: AMDGPU_VA_OP_UNMAP,
            flags: 0,
            va_address,
            offset_in_bo: 0,
            map_size: size,
        }
    }

    /// Set offset within the buffer.
    pub fn with_offset(mut self, offset: u64) -> Self {
        self.offset_in_bo = offset;
        self
    }

    /// Set custom mapping flags.
    pub fn with_flags(mut self, flags: u32) -> Self {
        self.flags = flags;
        self
    }
}

/// AMDGPU_CTX_OP: Context operation (create/destroy/query).
///
/// Contexts isolate GPU state and allow multiple independent command streams.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ContextOp {
    pub op: u32,               // in: operation type (create/destroy/query)
    pub flags: u32,            // in: context flags
    pub ctx_id: u32,           // in/out: context ID
    pub pad: u32,
}

/// Context operation types for AMDGPU_CTX_OP.
pub const AMDGPU_CTX_OP_ALLOC_CTX: u32 = 1;
pub const AMDGPU_CTX_OP_FREE_CTX: u32 = 2;
pub const AMDGPU_CTX_OP_QUERY_STATE: u32 = 3;
pub const AMDGPU_CTX_OP_QUERY_STATE2: u32 = 4;

/// Context priority levels.
pub const AMDGPU_CTX_PRIORITY_UNSET: u32 = 0;
pub const AMDGPU_CTX_PRIORITY_NORMAL: u32 = 1;
pub const AMDGPU_CTX_PRIORITY_HIGH: u32 = 2;
pub const AMDGPU_CTX_PRIORITY_VERY_HIGH: u32 = 3;

impl ContextOp {
    /// Create a new context creation request.
    pub fn alloc() -> Self {
        Self {
            op: AMDGPU_CTX_OP_ALLOC_CTX,
            flags: 0,
            ctx_id: 0,
            pad: 0,
        }
    }

    /// Create a new context destruction request.
    pub fn free(ctx_id: u32) -> Self {
        Self {
            op: AMDGPU_CTX_OP_FREE_CTX,
            flags: 0,
            ctx_id,
            pad: 0,
        }
    }

    /// Create a new context query request.
    pub fn query(ctx_id: u32) -> Self {
        Self {
            op: AMDGPU_CTX_OP_QUERY_STATE,
            flags: 0,
            ctx_id,
            pad: 0,
        }
    }
}

/// AMDGPU_CS: Command submission structure.
///
/// Submits GPU work via indirect buffers (IBs) and handles synchronization
/// via fences.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct CommandSubmission {
    pub ctx_id: u32,           // in: context ID
    pub bo_list_handle: u32,   // in: handle to buffer list
    pub num_chunks: u32,       // in: number of chunks
    pub pad: u32,
    pub chunks: u64,           // in: pointer to chunk array
    pub fence_info: u64,       // out: fence info for synchronization
}

/// Chunk types for AMDGPU_CS.
pub const AMDGPU_CHUNK_ID_IB: u32 = 0x01;            // Indirect Buffer
pub const AMDGPU_CHUNK_ID_FENCE: u32 = 0x02;         // Fence
pub const AMDGPU_CHUNK_ID_DEPENDENCIES: u32 = 0x03;  // Dependencies
pub const AMDGPU_CHUNK_ID_SYNCOBJ_IN: u32 = 0x04;    // Sync object input
pub const AMDGPU_CHUNK_ID_SYNCOBJ_OUT: u32 = 0x05;   // Sync object output
pub const AMDGPU_CHUNK_ID_BO_HANDLES: u32 = 0x06;    // BO handles

/// CS chunk header.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct ChunkData {
    pub chunk_id: u32,         // Chunk type ID
    pub length_dw: u32,        // Length in DWords
    pub chunk_data: u64,       // Pointer to chunk-specific data
}

/// Indirect Buffer chunk data.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct IBChunk {
    pub va_start: u64,         // GPU virtual address of IB
    pub ib_bytes: u32,         // Size of IB in bytes
    pub ip_type: u32,          // IP block type (GFX, COMPUTE, etc.)
    pub ip_instance: u32,      // IP instance
    pub ring: u32,             // Ring index
    pub flags: u32,            // IB flags
    pub pad: u32,
}

/// IP block types for IBChunk.
pub const AMDGPU_HW_IP_GFX: u32 = 0;       // Graphics pipeline
pub const AMDGPU_HW_IP_COMPUTE: u32 = 1;   // Compute pipeline
pub const AMDGPU_HW_IP_DMA: u32 = 2;       // DMA engine
pub const AMDGPU_HW_IP_UVD: u32 = 3;       // Video decode
pub const AMDGPU_HW_IP_VCE: u32 = 4;       // Video encode
pub const AMDGPU_HW_IP_UVD_ENC: u32 = 5;   // UVD encode
pub const AMDGPU_HW_IP_VCN_DEC: u32 = 6;   // VCN decode
pub const AMDGPU_HW_IP_VCN_ENC: u32 = 7;   // VCN encode
pub const AMDGPU_HW_IP_VCN_JPEG: u32 = 8;  // VCN JPEG

impl IBChunk {
    /// Create a new graphics IB chunk.
    pub fn gfx(va_start: u64, ib_bytes: u32) -> Self {
        Self {
            va_start,
            ib_bytes,
            ip_type: AMDGPU_HW_IP_GFX,
            ip_instance: 0,
            ring: 0,
            flags: 0,
            pad: 0,
        }
    }

    /// Set ring index.
    pub fn with_ring(mut self, ring: u32) -> Self {
        self.ring = ring;
        self
    }
}

impl CommandSubmission {
    /// Create a new command submission request.
    pub fn new(ctx_id: u32, num_chunks: u32, chunks: u64) -> Self {
        Self {
            ctx_id,
            bo_list_handle: 0,
            num_chunks,
            pad: 0,
            chunks,
            fence_info: 0,
        }
    }

    /// Set buffer list handle.
    pub fn with_bo_list(mut self, bo_list_handle: u32) -> Self {
        self.bo_list_handle = bo_list_handle;
        self
    }
}

/// AMDGPU_WAIT_CS: Wait for command submission to complete.
///
/// Synchronization primitive for waiting on GPU work.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct WaitCommandSubmission {
    pub ctx_id: u32,           // in: context ID
    pub ip_type: u32,          // in: IP block type
    pub ip_instance: u32,      // in: IP instance
    pub ring: u32,             // in: ring index
    pub handle: u64,           // in: fence handle
    pub timeout: u64,          // in: timeout in nanoseconds
    pub flags: u32,            // in: wait flags
    pub busy: u32,             // out: busy indicator
}

impl WaitCommandSubmission {
    /// Create a new wait request.
    pub fn new(ctx_id: u32, ip_type: u32, handle: u64) -> Self {
        Self {
            ctx_id,
            ip_type,
            ip_instance: 0,
            ring: 0,
            handle,
            timeout: !0u64,    // Infinite timeout
            flags: 0,
            busy: 0,
        }
    }

    /// Set timeout in nanoseconds.
    pub fn with_timeout(mut self, timeout_ns: u64) -> Self {
        self.timeout = timeout_ns;
        self
    }

    /// Set ring index.
    pub fn with_ring(mut self, ring: u32) -> Self {
        self.ring = ring;
        self
    }
}

/// AMDGPU_INFO: Query device information.
///
/// Used to detect GPU properties, memory info, firmware versions, etc.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct DeviceInfo {
    pub return_pointer: u64,   // in: pointer to return data
    pub return_size: u32,      // in: size of return data buffer
    pub query: u32,            // in: query type
    pub value: u64,            // in: query-specific value
}

/// Query types for AMDGPU_INFO.
pub const AMDGPU_INFO_ACCEL_WORKING: u32 = 0x00;
pub const AMDGPU_INFO_CRTC_FROM_ID: u32 = 0x01;
pub const AMDGPU_INFO_HW_IP_INFO: u32 = 0x02;
pub const AMDGPU_INFO_HW_IP_COUNT: u32 = 0x03;
pub const AMDGPU_INFO_TIMESTAMP: u32 = 0x05;
pub const AMDGPU_INFO_FW_VERSION: u32 = 0x0e;
pub const AMDGPU_INFO_NUM_BYTES_MOVED: u32 = 0x0f;
pub const AMDGPU_INFO_VRAM_USAGE: u32 = 0x10;
pub const AMDGPU_INFO_GTT_USAGE: u32 = 0x11;
pub const AMDGPU_INFO_GDS_CONFIG: u32 = 0x13;
pub const AMDGPU_INFO_VRAM_GTT: u32 = 0x14;
pub const AMDGPU_INFO_READ_MMR_REG: u32 = 0x15;
pub const AMDGPU_INFO_DEV_INFO: u32 = 0x16;
pub const AMDGPU_INFO_VIS_VRAM_USAGE: u32 = 0x17;

impl DeviceInfo {
    /// Create a new device info query.
    pub fn new(query: u32, return_pointer: u64, return_size: u32) -> Self {
        Self {
            return_pointer,
            return_size,
            query,
            value: 0,
        }
    }

    /// Set query-specific value.
    pub fn with_value(mut self, value: u64) -> Self {
        self.value = value;
        self
    }
}

/// Device info structure returned by AMDGPU_INFO_DEV_INFO.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub struct GpuDevInfo {
    pub device_id: u32,                // PCI device ID
    pub chip_rev: u32,                 // Chip revision
    pub external_rev: u32,             // External revision ID
    pub pci_rev: u32,                  // PCI revision ID
    pub family: u32,                   // GPU family
    pub num_shader_engines: u32,       // Number of shader engines
    pub num_shader_arrays_per_engine: u32,
    pub gpu_counter_freq: u64,         // GPU timestamp counter frequency
    pub max_engine_clk: u64,           // Max engine clock
    pub max_memory_clk: u64,           // Max memory clock
    pub cu_active_number: u32,         // Number of active CUs
    pub cu_ao_mask: u32,               // Always-on CU mask
    pub cu_bitmap: [[u32; 4]; 4],      // CU bitmap per shader array
    pub enabled_rb_pipes_mask: u32,    // Render backend pipe mask
    pub num_rb_pipes: u32,             // Number of RB pipes
    pub num_hw_gfx_contexts: u32,      // Number of hardware GFX contexts
    pub padding: u32,
    pub ids_flags: u64,                // IDs flags
    pub virtual_address_offset: u64,   // Virtual address offset
    pub virtual_address_max: u64,      // Max virtual address
    pub virtual_address_alignment: u32, // VA alignment
    pub pte_fragment_size: u32,        // Page table entry fragment size
    pub gart_page_size: u32,           // GART page size
    pub ce_ram_size: u32,              // CE RAM size
    pub vram_type: u32,                // VRAM type
    pub vram_bit_width: u32,           // VRAM bit width
    pub vce_harvest_config: u32,       // VCE harvest config
    pub gc_double_offchip_lds_buf: u32, // GC double offchip LDS buffer
    pub prim_buf_gpu_addr: u64,        // Primitive buffer GPU address
    pub pos_buf_gpu_addr: u64,         // Position buffer GPU address
    pub cntl_sb_buf_gpu_addr: u64,     // Control SB buffer GPU address
    pub param_buf_gpu_addr: u64,       // Parameter buffer GPU address
    pub prim_buf_size: u32,            // Primitive buffer size
    pub pos_buf_size: u32,             // Position buffer size
    pub cntl_sb_buf_size: u32,         // Control SB buffer size
    pub param_buf_size: u32,           // Parameter buffer size
    pub wave_front_size: u32,          // Wave front size
    pub num_shader_visible_vgprs: u32, // Number of shader visible VGPRs
    pub num_cu_per_sh: u32,            // Number of CUs per shader array
    pub num_tcc_blocks: u32,           // Number of TCC blocks
    pub gs_vgt_table_depth: u32,       // GS VGT table depth
    pub gs_prim_buffer_depth: u32,     // GS primitive buffer depth
    pub max_gs_waves_per_vgt: u32,     // Max GS waves per VGT
    pub padding2: u32,
    pub cu_ao_bitmap: [[u32; 4]; 4],   // Always-on CU bitmap
    pub high_va_offset: u64,           // High VA offset
    pub high_va_max: u64,              // High VA max
    pub pa_sc_tile_steering_override: u32, // PA SC tile steering override
    pub tcc_disabled_mask: u64,        // TCC disabled mask
}

/// GPU family IDs for AMDGPU.
pub const AMDGPU_FAMILY_SI: u32 = 110;        // Southern Islands (GCN1)
pub const AMDGPU_FAMILY_CI: u32 = 120;        // Sea Islands (GCN2)
pub const AMDGPU_FAMILY_KV: u32 = 125;        // Kaveri APU (GCN2)
pub const AMDGPU_FAMILY_VI: u32 = 130;        // Volcanic Islands (GCN3)
pub const AMDGPU_FAMILY_CZ: u32 = 135;        // Carrizo APU (GCN3)
pub const AMDGPU_FAMILY_AI: u32 = 141;        // Arctic Islands (GCN4/Polaris)
pub const AMDGPU_FAMILY_RV: u32 = 142;        // Raven (GCN5 APU)
pub const AMDGPU_FAMILY_NV: u32 = 143;        // Navi (RDNA1)
pub const AMDGPU_FAMILY_VGH: u32 = 144;       // Van Gogh (RDNA2 APU, Steam Deck)
pub const AMDGPU_FAMILY_YC: u32 = 146;        // Yellow Carp (RDNA2 APU)
pub const AMDGPU_FAMILY_GC_11_0_0: u32 = 148; // RDNA3 (Navi 3x)
pub const AMDGPU_FAMILY_GC_11_0_1: u32 = 149; // RDNA3 APU (Phoenix)

// IOCTL number construction using standard Linux ioctl macros pattern
const DRM_IOCTL_BASE: u8 = b'd';
const DRM_COMMAND_BASE: u32 = 0x40;

// AMDGPU DRM command numbers (DRM_COMMAND_BASE + offset)
const DRM_AMDGPU_GEM_CREATE: u32 = DRM_COMMAND_BASE + 0x00;
const DRM_AMDGPU_GEM_MMAP: u32 = DRM_COMMAND_BASE + 0x01;
const DRM_AMDGPU_CTX: u32 = DRM_COMMAND_BASE + 0x02;
const DRM_AMDGPU_BO_LIST: u32 = DRM_COMMAND_BASE + 0x03;
const DRM_AMDGPU_CS: u32 = DRM_COMMAND_BASE + 0x04;
const DRM_AMDGPU_INFO: u32 = DRM_COMMAND_BASE + 0x05;
const DRM_AMDGPU_GEM_METADATA: u32 = DRM_COMMAND_BASE + 0x06;
const DRM_AMDGPU_GEM_WAIT_IDLE: u32 = DRM_COMMAND_BASE + 0x07;
const DRM_AMDGPU_GEM_VA: u32 = DRM_COMMAND_BASE + 0x08;
const DRM_AMDGPU_WAIT_CS: u32 = DRM_COMMAND_BASE + 0x09;

// IOCTL request codes
const DRM_IOCTL_AMDGPU_GEM_CREATE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_GEM_CREATE, std::mem::size_of::<GemCreate>());
const DRM_IOCTL_AMDGPU_GEM_MMAP: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_GEM_MMAP, std::mem::size_of::<GemMmap>());
const DRM_IOCTL_AMDGPU_GEM_WAIT_IDLE: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_GEM_WAIT_IDLE, std::mem::size_of::<GemWaitIdle>());
const DRM_IOCTL_AMDGPU_GEM_VA: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_GEM_VA, std::mem::size_of::<GemVa>());
const DRM_IOCTL_AMDGPU_CTX: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_CTX, std::mem::size_of::<ContextOp>());
const DRM_IOCTL_AMDGPU_CS: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_CS, std::mem::size_of::<CommandSubmission>());
const DRM_IOCTL_AMDGPU_WAIT_CS: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_WAIT_CS, std::mem::size_of::<WaitCommandSubmission>());
const DRM_IOCTL_AMDGPU_INFO: nix::libc::Ioctl = nix::request_code_readwrite!(DRM_IOCTL_BASE, DRM_AMDGPU_INFO, std::mem::size_of::<DeviceInfo>());

impl DrmDevice {
    /// Execute AMDGPU_GEM_CREATE ioctl.
    pub fn amdgpu_gem_create(&self, req: &mut GemCreate) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_GEM_CREATE as _, req as *mut GemCreate)
        };
        Ok(())
    }

    /// Execute AMDGPU_GEM_MMAP ioctl.
    pub fn amdgpu_gem_mmap(&self, req: &mut GemMmap) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_GEM_MMAP as _, req as *mut GemMmap)
        };
        Ok(())
    }

    /// Execute AMDGPU_GEM_WAIT_IDLE ioctl.
    pub fn amdgpu_gem_wait_idle(&self, req: &mut GemWaitIdle) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_GEM_WAIT_IDLE as _, req as *mut GemWaitIdle)
        };
        Ok(())
    }

    /// Execute AMDGPU_GEM_VA ioctl.
    pub fn amdgpu_gem_va(&self, req: &mut GemVa) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_GEM_VA as _, req as *mut GemVa)
        };
        Ok(())
    }

    /// Execute AMDGPU_CTX ioctl.
    pub fn amdgpu_ctx(&self, req: &mut ContextOp) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_CTX as _, req as *mut ContextOp)
        };
        Ok(())
    }

    /// Execute AMDGPU_CS ioctl.
    pub fn amdgpu_cs(&self, req: &mut CommandSubmission) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_CS as _, req as *mut CommandSubmission)
        };
        Ok(())
    }

    /// Execute AMDGPU_WAIT_CS ioctl.
    pub fn amdgpu_wait_cs(&self, req: &mut WaitCommandSubmission) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_WAIT_CS as _, req as *mut WaitCommandSubmission)
        };
        Ok(())
    }

    /// Execute AMDGPU_INFO ioctl.
    pub fn amdgpu_info(&self, req: &mut DeviceInfo) -> io::Result<()> {
        unsafe {
            nix::libc::ioctl(self.fd(), DRM_IOCTL_AMDGPU_INFO as _, req as *mut DeviceInfo)
        };
        Ok(())
    }

    /// Submit a batch buffer (PM4 commands) to the GPU and wait for completion.
    ///
    /// This is the high-level AMD equivalent of i915_submit_batch.
    ///
    /// # Arguments
    ///
    /// * `batch_handle` - GEM handle of buffer containing PM4 commands
    /// * `batch_va` - GPU virtual address where batch buffer is mapped
    /// * `batch_len_bytes` - Length of PM4 command stream in bytes
    /// * `context_id` - AMD GPU context ID
    ///
    /// # Returns
    ///
    /// Ok(()) on successful submission and completion, or io::Error on failure.
    ///
    /// # Note
    ///
    /// This method assumes the batch buffer is already mapped to GPU virtual
    /// address space via amdgpu_gem_va. For now, it uses a simple single-IB
    /// submission without BO lists or dependencies.
    pub fn amdgpu_submit_batch(
        &self,
        batch_handle: u32,
        batch_va: u64,
        batch_len_bytes: u32,
        context_id: u32,
    ) -> io::Result<()> {
        // Build the indirect buffer chunk
        let ib_chunk = IBChunk::gfx(batch_va, batch_len_bytes);
        
        // Build the chunk data header
        let chunk_data = ChunkData {
            chunk_id: AMDGPU_CHUNK_ID_IB,
            length_dw: (std::mem::size_of::<IBChunk>() / 4) as u32,
            chunk_data: &ib_chunk as *const IBChunk as u64,
        };
        
        // Build command submission request
        let mut cs = CommandSubmission::new(
            context_id,
            1, // Single chunk (IB only)
            &chunk_data as *const ChunkData as u64,
        );
        
        // Submit to GPU
        self.amdgpu_cs(&mut cs)?;
        
        // Wait for completion using the fence returned by CS
        let mut wait = WaitCommandSubmission::new(
            context_id,
            AMDGPU_HW_IP_GFX,
            cs.fence_info,
        );
        self.amdgpu_wait_cs(&mut wait)?;
        
        Ok(())
    }

    /// Submit a batch buffer without explicit VA (simplified for testing).
    ///
    /// This is a simplified version that assumes the kernel will handle VA mapping.
    /// Used for basic smoke tests before full VA management is implemented.
    pub fn amdgpu_submit_batch_simple(
        &self,
        batch_handle: u32,
        batch_len_bytes: u32,
        context_id: u32,
    ) -> io::Result<()> {
        // For testing, use the GEM handle as a placeholder VA
        // In production, this should be a proper mapped VA from amdgpu_gem_va
        let batch_va = (batch_handle as u64) << 12; // Simple mapping: handle * 4K
        self.amdgpu_submit_batch(batch_handle, batch_va, batch_len_bytes, context_id)
    }

    /// Create an AMD GPU context and return its ID.
    ///
    /// Contexts isolate GPU state and allow independent workloads.
    pub fn amdgpu_create_context(&self) -> io::Result<u32> {
        let mut req = ContextOp::alloc();
        self.amdgpu_ctx(&mut req)?;
        Ok(req.ctx_id)
    }

    /// Destroy an AMD GPU context.
    pub fn amdgpu_destroy_context(&self, ctx_id: u32) -> io::Result<()> {
        let mut req = ContextOp::free(ctx_id);
        self.amdgpu_ctx(&mut req)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gem_create_new() {
        let req = GemCreate::new(4096, AMDGPU_GEM_DOMAIN_VRAM, 0);
        assert_eq!(req.size, 4096);
        assert_eq!(req.domains, AMDGPU_GEM_DOMAIN_VRAM);
        assert_eq!(req.alignment, 4096);
    }

    #[test]
    fn test_gem_create_with_alignment() {
        let req = GemCreate::new(8192, AMDGPU_GEM_DOMAIN_GTT, 0)
            .with_alignment(8192);
        assert_eq!(req.alignment, 8192);
    }

    #[test]
    fn test_gem_va_map() {
        let va_addr = 0x1000_0000;
        let req = GemVa::map(42, va_addr, 4096);
        assert_eq!(req.handle, 42);
        assert_eq!(req.operation, AMDGPU_VA_OP_MAP);
        assert_eq!(req.va_address, va_addr);
        assert_eq!(req.map_size, 4096);
    }

    #[test]
    fn test_gem_va_unmap() {
        let va_addr = 0x1000_0000;
        let req = GemVa::unmap(va_addr, 4096);
        assert_eq!(req.operation, AMDGPU_VA_OP_UNMAP);
        assert_eq!(req.va_address, va_addr);
    }

    #[test]
    fn test_context_alloc() {
        let req = ContextOp::alloc();
        assert_eq!(req.op, AMDGPU_CTX_OP_ALLOC_CTX);
        assert_eq!(req.ctx_id, 0);
    }

    #[test]
    fn test_context_free() {
        let req = ContextOp::free(123);
        assert_eq!(req.op, AMDGPU_CTX_OP_FREE_CTX);
        assert_eq!(req.ctx_id, 123);
    }

    #[test]
    fn test_ib_chunk_gfx() {
        let chunk = IBChunk::gfx(0x2000_0000, 1024);
        assert_eq!(chunk.va_start, 0x2000_0000);
        assert_eq!(chunk.ib_bytes, 1024);
        assert_eq!(chunk.ip_type, AMDGPU_HW_IP_GFX);
    }

    #[test]
    fn test_wait_cs_new() {
        let req = WaitCommandSubmission::new(1, AMDGPU_HW_IP_GFX, 0x1234);
        assert_eq!(req.ctx_id, 1);
        assert_eq!(req.ip_type, AMDGPU_HW_IP_GFX);
        assert_eq!(req.handle, 0x1234);
        assert_eq!(req.timeout, !0u64);
    }

    #[test]
    fn test_device_info_new() {
        let mut buffer: [u8; 256] = [0; 256];
        let req = DeviceInfo::new(
            AMDGPU_INFO_DEV_INFO,
            buffer.as_mut_ptr() as u64,
            buffer.len() as u32
        );
        assert_eq!(req.query, AMDGPU_INFO_DEV_INFO);
        assert_eq!(req.return_size, 256);
    }
}
