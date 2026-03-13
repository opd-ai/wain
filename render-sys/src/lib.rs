/// C-ABI boundary functions exposed to Go via CGO.
///
/// This module validates the full Go → Rust build pipeline.
/// Start with trivial arithmetic to confirm the ABI works before
/// adding GPU/rendering functionality.

pub mod drm;
pub mod i915;
pub mod xe;
pub mod amd;
pub mod pm4;
pub mod allocator;
pub mod slab;
pub mod detect;
pub mod cmd;
pub mod batch;
pub mod pipeline;
pub mod surface;
pub mod shader;
pub mod shaders;
pub mod eu;
pub mod rdna;
pub mod submit;

#[cfg(test)]
pub mod gpu_test;

use std::os::unix::io::RawFd;
use drm::DrmDevice;
use allocator::{BufferAllocator, Buffer, TilingFormat, DriverType};
use detect::GpuGeneration;
use i915::RelocationEntry;

/// Maximum shader source size (1 MB). Inputs larger than this are rejected
/// before parsing to prevent compiler resource exhaustion.
const MAX_SHADER_SOURCE_SIZE: usize = 1_048_576;

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
        GpuGeneration::AmdRdna1 => 20,
        GpuGeneration::AmdRdna2 => 21,
        GpuGeneration::AmdRdna3 => 22,
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

    let generation = match device.detect_gpu_generation() {
        Ok(g) => g,
        Err(_) => return std::ptr::null_mut(),
    };

    let driver = match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => DriverType::I915,
        GpuGeneration::Xe => DriverType::Xe,
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => DriverType::Amdgpu,
        GpuGeneration::Unknown => return std::ptr::null_mut(),
    };

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

/// Get buffer GEM handle for GPU command submission.
///
/// # Safety
/// - buffer must be a valid pointer returned by buffer_allocate
#[no_mangle]
pub unsafe extern "C" fn buffer_get_handle(buffer: *mut Buffer) -> u32 {
    if buffer.is_null() {
        return 0;
    }
    let buf = &*buffer;
    buf.handle
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

/// Submit a batch buffer to the GPU and wait for completion (i915 driver).
///
/// # Arguments
/// - path: Path to the DRM device (e.g., "/dev/dri/renderD128")
/// - batch_handle: GEM buffer handle containing the command stream
/// - batch_len_bytes: Length of the command stream in bytes
/// - relocs: Array of relocation entries (null if relocs_count is 0)
/// - relocs_count: Number of relocation entries
/// - context_id: GPU context ID (0 for default context)
///
/// Returns 0 on success, -1 on error.
///
/// # Safety
/// - path must be a valid null-terminated C string
/// - relocs must be a valid array of RelocationEntry (or null if count is 0)
/// - batch_handle must be a valid GEM buffer handle
#[no_mangle]
pub unsafe extern "C" fn render_submit_batch(
    path: *const std::ffi::c_char,
    batch_handle: u32,
    batch_len_bytes: u32,
    relocs: *const RelocationEntry,
    relocs_count: usize,
    context_id: u32,
) -> i32 {
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

    // Convert C array to Rust slice
    let relocs_slice = if relocs.is_null() || relocs_count == 0 {
        &[]
    } else {
        std::slice::from_raw_parts(relocs, relocs_count)
    };

    // Detect driver type and submit accordingly
    let generation = match device.detect_gpu_generation() {
        Ok(g) => g,
        Err(_) => return -1,
    };

    let driver = match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => DriverType::I915,
        GpuGeneration::Xe => DriverType::Xe,
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => DriverType::Amdgpu,
        GpuGeneration::Unknown => return -1,
    };

    let allocator = BufferAllocator::new(device, driver);
    let dev = allocator.device();

    match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => {
            // Use i915 submission path
            match dev.i915_submit_batch(batch_handle, batch_len_bytes, relocs_slice, context_id) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::Xe => {
            // Use Xe submission path (simplified, no relocations)
            // Note: batch_gpu_addr of 0 means kernel will assign the address
            match dev.xe_submit_batch_simple(batch_handle, 0, batch_len_bytes as u64) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => {
            // Use AMD submission path with automatic VA management
            // VA base address starts at 1GB (0x4000_0000) to avoid low memory
            const AMD_VA_BASE: u64 = 0x4000_0000;
            let batch_size = ((batch_len_bytes + 4095) / 4096 * 4096) as u64; // Round up to 4K
            match dev.amdgpu_submit_with_va(
                batch_handle,
                batch_size,
                batch_len_bytes,
                context_id,
                AMD_VA_BASE,
            ) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::Unknown => -1,
    }
}

/// Create a GPU context and return its ID.
///
/// # Arguments
/// - path: Path to the DRM device (e.g., "/dev/dri/renderD128")
/// - out_context_id: Pointer to store the created context ID
/// - out_vm_id: Pointer to store the VM ID (Xe only, can be null for i915)
///
/// Returns 0 on success, -1 on error.
///
/// # Safety
/// - path must be a valid null-terminated C string
/// - out_context_id must be a valid pointer
/// - out_vm_id can be null if the caller doesn't need VM ID
#[no_mangle]
pub unsafe extern "C" fn render_create_context(
    path: *const std::ffi::c_char,
    out_context_id: *mut u32,
    out_vm_id: *mut u32,
) -> i32 {
    if path.is_null() || out_context_id.is_null() {
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

    let driver = match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => DriverType::I915,
        GpuGeneration::Xe => DriverType::Xe,
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => DriverType::Amdgpu,
        GpuGeneration::Unknown => return -1,
    };

    let allocator = BufferAllocator::new(device, driver);
    let dev = allocator.device();

    match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => {
            // Use i915 context creation
            match dev.i915_create_context() {
                Ok(ctx_id) => {
                    *out_context_id = ctx_id;
                    // i915 doesn't have separate VM ID
                    if !out_vm_id.is_null() {
                        *out_vm_id = 0;
                    }
                    0
                }
                Err(_) => -1,
            }
        }
        GpuGeneration::Xe => {
            // Use Xe context creation (returns VM and exec queue)
            match dev.xe_create_context() {
                Ok((vm_id, exec_queue_id)) => {
                    *out_context_id = exec_queue_id;
                    if !out_vm_id.is_null() {
                        *out_vm_id = vm_id;
                    }
                    0
                }
                Err(_) => -1,
            }
        }
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => {
            // AMD context creation
            match dev.amdgpu_create_context() {
                Ok(ctx_id) => {
                    *out_context_id = ctx_id;
                    // AMD doesn't have separate VM ID
                    if !out_vm_id.is_null() {
                        *out_vm_id = 0;
                    }
                    0
                }
                Err(_) => -1,
            }
        }
        GpuGeneration::Unknown => -1,
    }
}

/// Destroy a GPU context and release associated resources.
///
/// # Arguments
/// - path: Path to the DRM device (e.g., "/dev/dri/renderD128")
/// - context_id: Context ID to destroy (from render_create_context)
/// - vm_id: VM ID (Xe only, 0 for i915)
///
/// Returns 0 on success, -1 on error.
///
/// # Safety
/// - path must be a valid null-terminated C string
/// - context_id must be a valid context ID from render_create_context
#[no_mangle]
pub unsafe extern "C" fn render_destroy_context(
    path: *const std::ffi::c_char,
    context_id: u32,
    vm_id: u32,
) -> i32 {
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

    let driver = match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => DriverType::I915,
        GpuGeneration::Xe => DriverType::Xe,
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => DriverType::Amdgpu,
        GpuGeneration::Unknown => return -1,
    };

    let allocator = BufferAllocator::new(device, driver);
    let dev = allocator.device();

    match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => {
            // Use i915 context destruction
            match dev.i915_destroy_context(context_id) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::Xe => {
            // Use Xe context destruction (VM + exec queue)
            match dev.xe_destroy_context(vm_id, context_id) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3 => {
            // AMD context destruction
            match dev.amdgpu_destroy_context(context_id) {
                Ok(()) => 0,
                Err(_) => -1,
            }
        }
        GpuGeneration::Unknown => -1,
    }
}

/// Map a GPU buffer into CPU address space for reading/writing.
///
/// Returns a pointer to the mapped memory or null on error.
/// The caller must call buffer_munmap when done.
///
/// # Arguments
/// - allocator: Allocator that created the buffer
/// - buffer: Buffer to map
/// - out_size: Output parameter for the mapped region size in bytes
///
/// # Safety
/// - allocator must be a valid pointer returned by buffer_allocator_create
/// - buffer must be a valid pointer returned by buffer_allocate
/// - out_size must be a valid pointer
/// - The returned pointer must be unmapped with buffer_munmap
#[no_mangle]
pub unsafe extern "C" fn buffer_mmap(
    allocator: *mut BufferAllocator,
    buffer: *mut Buffer,
    out_size: *mut usize,
) -> *mut u8 {
    if allocator.is_null() || buffer.is_null() || out_size.is_null() {
        return std::ptr::null_mut();
    }

    let alloc = &*allocator;
    let buf = &*buffer;
    let dev = alloc.device();

    // Get mmap offset - driver-specific
    let offset = match alloc.driver() {
        DriverType::I915 | DriverType::Xe => {
            // Intel: use i915_gem_mmap_offset
            let mut mmap_req = i915::GemMmapOffset::new(buf.handle, i915::I915_MMAP_OFFSET_WB);
            match dev.i915_gem_mmap_offset(&mut mmap_req) {
                Ok(()) => mmap_req.offset,
                Err(_) => return std::ptr::null_mut(),
            }
        }
        DriverType::Amdgpu => {
            // AMD: use amdgpu_gem_mmap
            let mut mmap_req = amd::GemMmap::new(buf.handle);
            match dev.amdgpu_gem_mmap(&mut mmap_req) {
                Ok(()) => mmap_req.offset,
                Err(_) => return std::ptr::null_mut(),
            }
        }
    };

    // Calculate buffer size
    let size = (buf.stride * buf.height) as usize;
    *out_size = size;

    // Map the buffer into userspace using libc mmap
    let ptr = nix::libc::mmap(
        std::ptr::null_mut(),
        size,
        nix::libc::PROT_READ | nix::libc::PROT_WRITE,
        nix::libc::MAP_SHARED,
        dev.fd(),
        offset as i64,
    );

    if ptr == nix::libc::MAP_FAILED {
        return std::ptr::null_mut();
    }

    ptr as *mut u8
}

/// Unmap a previously mapped GPU buffer.
///
/// # Safety
/// - ptr must be a valid pointer returned by buffer_mmap
/// - size must be the size returned by buffer_mmap
/// - ptr must not be used after this call
#[no_mangle]
pub unsafe extern "C" fn buffer_munmap(ptr: *mut u8, size: usize) -> i32 {
    if ptr.is_null() {
        return -1;
    }

    let result = nix::libc::munmap(ptr as *mut nix::libc::c_void, size);
    if result == 0 {
        0
    } else {
        -1
    }
}

/// Compile a WGSL shader to Intel EU machine code.
///
/// Returns a pointer to the compiled binary and writes its size to out_size.
/// The caller must free the returned pointer with render_shader_free.
///
/// # Arguments
/// * wgsl_source - Null-terminated WGSL shader source code
/// * gpu_gen - GPU generation (9 = Gen9, 11 = Gen11, 12 = Gen12)
/// * is_fragment - 1 for fragment shader, 0 for vertex shader
/// * out_size - Pointer to write the binary size
///
/// # Returns
/// Pointer to binary data (must be freed), or null on error
///
/// # Safety
/// - wgsl_source must be a valid null-terminated C string
/// - out_size must be a valid pointer
/// - The returned pointer must be freed with render_shader_free
#[no_mangle]
pub unsafe extern "C" fn render_compile_shader(
    wgsl_source: *const std::ffi::c_char,
    gpu_gen: i32,
    is_fragment: i32,
    out_size: *mut usize,
) -> *mut u8 {
    use shader::ShaderModule;
    use eu::{EUCompiler, IntelGen};

    if wgsl_source.is_null() || out_size.is_null() {
        return std::ptr::null_mut();
    }

    // Convert C string to Rust string
    let source = match std::ffi::CStr::from_ptr(wgsl_source).to_str() {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    // Reject oversized inputs before parsing to prevent resource exhaustion
    if source.len() > MAX_SHADER_SOURCE_SIZE {
        return std::ptr::null_mut();
    }

    // Parse GPU generation
    let gen = match gpu_gen {
        9 => IntelGen::Gen9,
        11 => IntelGen::Gen11,
        12 => IntelGen::Gen12,
        _ => return std::ptr::null_mut(),
    };

    // Parse shader stage
    let stage = if is_fragment != 0 {
        naga::ShaderStage::Fragment
    } else {
        naga::ShaderStage::Vertex
    };

    // Compile WGSL to naga IR
    let module = match ShaderModule::from_wgsl(source, stage) {
        Ok(m) => m,
        Err(_) => return std::ptr::null_mut(),
    };

    // Compile to EU binary
    let compiler = EUCompiler::new(gen);
    let kernel = match compiler.compile(&module) {
        Ok(k) => k,
        Err(_) => return std::ptr::null_mut(),
    };

    // Allocate heap memory for the binary
    let binary_size = kernel.binary.len();
    
    // Guard against zero-size allocation
    if binary_size == 0 {
        return std::ptr::null_mut();
    }
    
    let layout = std::alloc::Layout::from_size_align(binary_size, 1)
        .expect("valid layout: align=1 is always a power of 2");
    let ptr = std::alloc::alloc(layout);
    
    if ptr.is_null() {
        return std::ptr::null_mut();
    }

    // Copy binary data to allocated memory
    std::ptr::copy_nonoverlapping(kernel.binary.as_ptr(), ptr, binary_size);
    
    // Write size to output parameter
    *out_size = binary_size;
    
    ptr
}

/// Free shader binary allocated by render_compile_shader.
///
/// # Safety
/// - ptr must be a pointer returned by render_compile_shader
/// - size must be the size returned by render_compile_shader
/// - ptr must not be used after this call
#[no_mangle]
pub unsafe extern "C" fn render_shader_free(ptr: *mut u8, size: usize) {
    if ptr.is_null() || size == 0 {
        return;
    }
    
    let layout = std::alloc::Layout::from_size_align(size, 1)
        .expect("valid layout: align=1 is always a power of 2");
    std::alloc::dealloc(ptr, layout);
}

/// Compile a WGSL shader to native GPU machine code and submit it as a batch.
///
/// This is the primary entry point for shader-driven GPU rendering. It compiles
/// the provided WGSL source to Intel EU or AMD RDNA machine code (selected
/// automatically from the detected GPU generation), builds a GPU command batch
/// that binds the compiled shader kernel, and submits that batch to the GPU.
///
/// # Arguments
/// * `path`        — Path to the DRM render node (e.g. `/dev/dri/renderD128`)
/// * `wgsl_source` — Null-terminated WGSL shader source
/// * `is_fragment` — 1 for fragment shader, 0 for vertex shader
/// * `context_id`  — GPU context ID (from `render_create_context`)
///
/// # Returns
/// 0 on success, −1 on any error (bad args, no GPU, compile failure, etc.).
///
/// # Safety
/// * `path` and `wgsl_source` must be valid null-terminated C strings.
/// * `wgsl_source` must not exceed `MAX_SHADER_SOURCE_SIZE` bytes.
#[no_mangle]
pub unsafe extern "C" fn render_submit_shader_batch(
    path: *const std::ffi::c_char,
    wgsl_source: *const std::ffi::c_char,
    is_fragment: i32,
    context_id: u32,
) -> i32 {
    use batch::BatchBuilder;
    use naga::ShaderStage;
    use submit::{bind_eu_shader_to_batch, bind_rdna_shader_to_batch};

    if path.is_null() || wgsl_source.is_null() {
        return -1;
    }

    let c_path = match std::ffi::CStr::from_ptr(path).to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    let source = match std::ffi::CStr::from_ptr(wgsl_source).to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    if source.len() > MAX_SHADER_SOURCE_SIZE {
        return -1;
    }

    let stage = if is_fragment != 0 {
        ShaderStage::Fragment
    } else {
        ShaderStage::Vertex
    };

    let device = match DrmDevice::open(c_path) {
        Ok(d) => d,
        Err(_) => return -1,
    };

    let generation = match device.detect_gpu_generation() {
        Ok(g) => g,
        Err(_) => return -1,
    };

    let driver = match generation {
        GpuGeneration::Gen9
        | GpuGeneration::Gen11
        | GpuGeneration::Gen12
        | GpuGeneration::Xe => DriverType::I915,
        GpuGeneration::AmdRdna1
        | GpuGeneration::AmdRdna2
        | GpuGeneration::AmdRdna3 => DriverType::Amdgpu,
        GpuGeneration::Unknown => return -1,
    };

    let allocator = BufferAllocator::new(device, driver);

    // BatchBuilder::new allocates the GEM batch buffer internally.
    const BATCH_SIZE: u32 = 64 * 1024;
    let mut builder = match BatchBuilder::new(&allocator, BATCH_SIZE, generation) {
        Ok(b) => b,
        Err(_) => return -1,
    };

    // Compile the WGSL shader and bind it to the batch (emits pipeline state).
    let bind_result = match generation {
        GpuGeneration::Gen9
        | GpuGeneration::Gen11
        | GpuGeneration::Gen12
        | GpuGeneration::Xe => {
            bind_eu_shader_to_batch(&allocator, generation, source, stage, &mut builder)
        }
        GpuGeneration::AmdRdna1
        | GpuGeneration::AmdRdna2
        | GpuGeneration::AmdRdna3 => {
            bind_rdna_shader_to_batch(&allocator, generation, source, stage, &mut builder)
        }
        GpuGeneration::Unknown => return -1,
    };

    if bind_result.is_err() {
        return -1;
    }

    // Emit batch-buffer-end marker.
    builder.emit_dword(0x0A000000); // MI_BATCH_BUFFER_END

    let submittable = builder.finalize();
    let batch_handle = submittable.buffer_handle;
    let batch_len = submittable.len_bytes() as u32;

    // Convert batch::Relocation entries to i915::RelocationEntry for submission.
    let relocs: Vec<RelocationEntry> = submittable
        .relocations
        .iter()
        .map(|r| RelocationEntry {
            target_handle: r.target_handle,
            delta: r.target_offset as u32,
            offset: r.offset_dwords as u64 * 4,
            presumed_offset: 0,
            read_domains: r.read_domains,
            write_domain: r.write_domain,
        })
        .collect();

    let dev = allocator.device();

    let result = match generation {
        GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 => {
            dev.i915_submit_batch(batch_handle, batch_len, &relocs, context_id)
        }
        GpuGeneration::Xe => dev.xe_submit_batch_simple(batch_handle, 0, batch_len as u64),
        GpuGeneration::AmdRdna1
        | GpuGeneration::AmdRdna2
        | GpuGeneration::AmdRdna3 => {
            const AMD_VA_BASE: u64 = 0x4000_0000;
            let batch_size = ((batch_len as u64 + 4095) / 4096) * 4096;
            dev.amdgpu_submit_with_va(batch_handle, batch_size, batch_len, context_id, AMD_VA_BASE)
        }
        GpuGeneration::Unknown => return -1,
    };

    match result {
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
