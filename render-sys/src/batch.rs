/// Batch buffer builder for Intel GPU command submission.
///
/// This module provides a typed interface for constructing GPU command buffers
/// (batch buffers) that can be submitted to the i915 or Xe drivers. The batch
/// builder manages buffer allocation, command emission, and relocation entries.
///
/// References:
/// - Intel PRMs Volume 1 (Graphics Memory Management)
/// - Mesa iris driver (src/gallium/drivers/iris/iris_batch.c)

use std::io;
use crate::allocator::{BufferAllocator, Buffer, TilingFormat};
use crate::cmd::GpuCommand;
use crate::detect::GpuGeneration;

/// Relocation entry for patching GPU addresses in batch buffers.
///
/// When a batch buffer references another buffer (e.g., a render target or
/// vertex buffer), the actual GPU virtual address is not known until command
/// submission. Relocation entries tell the kernel to patch these addresses.
#[derive(Debug, Clone)]
pub struct Relocation {
    /// Offset in the batch buffer where the address needs to be written (in dwords)
    pub offset_dwords: u32,
    /// Target buffer handle to be referenced
    pub target_handle: u32,
    /// Read/write domain flags (for cache coherency)
    pub read_domains: u32,
    pub write_domain: u32,
    /// Offset within the target buffer
    pub target_offset: u64,
}

impl Relocation {
    /// Create a new relocation entry.
    pub fn new(offset_dwords: u32, target_handle: u32, target_offset: u64) -> Self {
        Self {
            offset_dwords,
            target_handle,
            read_domains: 0,
            write_domain: 0,
            target_offset,
        }
    }
    
    /// Set read domains for cache coherency (I915_GEM_DOMAIN_*).
    pub fn with_read_domains(mut self, domains: u32) -> Self {
        self.read_domains = domains;
        self
    }
    
    /// Set write domain for cache coherency (I915_GEM_DOMAIN_*).
    pub fn with_write_domain(mut self, domain: u32) -> Self {
        self.write_domain = domain;
        self
    }
}

/// Batch buffer builder for GPU command streams.
///
/// The BatchBuilder allocates a GEM buffer object and provides typed methods
/// for emitting GPU commands. Commands are serialized to binary and written
/// sequentially. Relocation entries track buffer references that need address
/// patching before submission.
pub struct BatchBuilder {
    /// The underlying GEM buffer holding the command stream
    buffer: Buffer,
    /// Command stream data (dwords)
    commands: Vec<u32>,
    /// Relocation entries for address patching
    relocations: Vec<Relocation>,
    /// GPU generation (for generation-specific command variants)
    generation: GpuGeneration,
}

impl BatchBuilder {
    /// Create a new batch buffer builder.
    ///
    /// Allocates a GEM buffer of the specified size for command storage.
    /// The buffer uses linear (no tiling) format since it's not a render target.
    pub fn new(
        allocator: &BufferAllocator,
        size_bytes: u32,
        generation: GpuGeneration,
    ) -> io::Result<Self> {
        // Batch buffers are always linear (not tiled)
        let buffer = allocator.allocate(size_bytes, 1, 8, TilingFormat::None)?;
        
        Ok(Self {
            buffer,
            commands: Vec::with_capacity((size_bytes / 4) as usize),
            relocations: Vec::new(),
            generation,
        })
    }
    
    /// Get the GPU generation this batch is targeting.
    pub fn generation(&self) -> GpuGeneration {
        self.generation
    }
    
    /// Emit a GPU command to the batch buffer.
    ///
    /// Serializes the command to dwords and appends to the command stream.
    pub fn emit<T: GpuCommand>(&mut self, command: T) {
        let dwords = command.serialize();
        self.commands.extend_from_slice(&dwords);
    }
    
    /// Emit a raw dword to the batch buffer.
    ///
    /// For low-level command construction or padding.
    pub fn emit_dword(&mut self, dword: u32) {
        self.commands.push(dword);
    }
    
    /// Emit multiple raw dwords to the batch buffer.
    pub fn emit_dwords(&mut self, dwords: &[u32]) {
        self.commands.extend_from_slice(dwords);
    }
    
    /// Add a relocation entry for address patching.
    ///
    /// Call this after emitting a command that references another buffer.
    /// The offset should point to the dword(s) that will hold the GPU address.
    pub fn add_relocation(&mut self, relocation: Relocation) {
        self.relocations.push(relocation);
    }
    
    /// Emit a GPU address reference with automatic relocation.
    ///
    /// Emits placeholder dwords (64-bit address) and registers a relocation
    /// entry. The kernel will patch the actual GPU virtual address at submit time.
    pub fn emit_reloc(&mut self, target_handle: u32, target_offset: u64, read_domains: u32, write_domain: u32) {
        let offset_dwords = self.commands.len() as u32;
        
        // Emit placeholder address (will be patched by kernel)
        self.commands.push(0); // Low 32 bits
        self.commands.push(0); // High 32 bits
        
        let reloc = Relocation::new(offset_dwords, target_handle, target_offset)
            .with_read_domains(read_domains)
            .with_write_domain(write_domain);
        
        self.relocations.push(reloc);
    }
    
    /// Get the current command stream length in dwords.
    pub fn len_dwords(&self) -> usize {
        self.commands.len()
    }
    
    /// Check if the batch buffer is empty.
    pub fn is_empty(&self) -> bool {
        self.commands.is_empty()
    }
    
    /// Get the current command stream length in bytes.
    pub fn len_bytes(&self) -> usize {
        self.commands.len() * 4
    }
    
    /// Get the underlying GEM buffer handle.
    pub fn buffer_handle(&self) -> u32 {
        self.buffer.handle
    }
    
    /// Get a reference to the command stream.
    pub fn commands(&self) -> &[u32] {
        &self.commands
    }
    
    /// Get a reference to the relocation entries.
    pub fn relocations(&self) -> &[Relocation] {
        &self.relocations
    }
    
    /// Finalize the batch buffer and return a submittable batch.
    ///
    /// This consumes the builder and returns a `SubmittableBatch` containing
    /// the buffer handle, command data, and relocations. The caller can then
    /// submit this via execbuffer2 (i915) or xe_exec (Xe).
    pub fn finalize(self) -> SubmittableBatch {
        SubmittableBatch {
            buffer_handle: self.buffer.handle,
            commands: self.commands,
            relocations: self.relocations,
        }
    }
}

/// A finalized batch buffer ready for submission.
///
/// Contains the GEM buffer handle, serialized command stream, and relocation
/// entries. This structure can be passed to execbuffer2 or xe_exec for GPU
/// execution.
#[derive(Debug)]
pub struct SubmittableBatch {
    /// GEM buffer handle containing the command stream
    pub buffer_handle: u32,
    /// Serialized command data (dwords)
    pub commands: Vec<u32>,
    /// Relocation entries for address patching
    pub relocations: Vec<Relocation>,
}

impl SubmittableBatch {
    /// Get the command stream length in bytes.
    pub fn len_bytes(&self) -> usize {
        self.commands.len() * 4
    }
    
    /// Check if the batch is empty.
    pub fn is_empty(&self) -> bool {
        self.commands.is_empty()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cmd::{MiNoop, PipeControl};
    use crate::drm::DrmDevice;
    use crate::allocator::DriverType;

    #[test]
    fn batch_builder_creation() {
        // Skip if no DRM device available
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen9);
        
        assert!(builder.is_ok());
        let builder = builder.unwrap();
        assert_eq!(builder.len_dwords(), 0);
        assert!(builder.is_empty());
    }

    #[test]
    fn batch_builder_emit_command() {
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let mut builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen9).unwrap();
        
        builder.emit(MiNoop);
        assert_eq!(builder.len_dwords(), 1);
        assert_eq!(builder.commands()[0], 0x0);
    }

    #[test]
    fn batch_builder_emit_multiple_commands() {
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let mut builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen12).unwrap();
        
        builder.emit(MiNoop);
        builder.emit(PipeControl::full_flush());
        builder.emit(MiNoop);
        
        assert_eq!(builder.len_dwords(), 1 + 6 + 1); // noop + pipe_control + noop
    }

    #[test]
    fn batch_builder_relocation() {
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let mut builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen9).unwrap();
        
        // Emit a relocation (simulating a buffer reference)
        builder.emit_reloc(42, 0x1000, 0x1, 0x2);
        
        assert_eq!(builder.len_dwords(), 2); // 64-bit address = 2 dwords
        assert_eq!(builder.relocations().len(), 1);
        
        let reloc = &builder.relocations()[0];
        assert_eq!(reloc.target_handle, 42);
        assert_eq!(reloc.target_offset, 0x1000);
        assert_eq!(reloc.read_domains, 0x1);
        assert_eq!(reloc.write_domain, 0x2);
    }

    #[test]
    fn batch_builder_finalize() {
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let mut builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen11).unwrap();
        
        builder.emit(MiNoop);
        builder.emit(PipeControl::new());
        
        let batch = builder.finalize();
        
        assert_eq!(batch.len_bytes(), (1 + 6) * 4); // 1 noop + 6 pipe_control dwords
        assert!(!batch.is_empty());
    }

    #[test]
    fn batch_builder_raw_dwords() {
        let device = match DrmDevice::open("/dev/dri/renderD128") {
            Ok(d) => d,
            Err(_) => return,
        };
        
        let allocator = BufferAllocator::new(device, DriverType::I915);
        let mut builder = BatchBuilder::new(&allocator, 4096, GpuGeneration::Gen9).unwrap();
        
        builder.emit_dword(0xDEADBEEF);
        builder.emit_dwords(&[0xCAFE, 0xBABE]);
        
        assert_eq!(builder.len_dwords(), 3);
        assert_eq!(builder.commands()[0], 0xDEADBEEF);
        assert_eq!(builder.commands()[1], 0xCAFE);
        assert_eq!(builder.commands()[2], 0xBABE);
    }
}
