// Shader-to-batch binding for the GPU rendering pipeline.
//
// This module connects compiled EU/RDNA shader kernels to the Intel/AMD batch
// buffer pipeline. It handles shader kernel upload, relocation registration,
// and pipeline state configuration for solid-colour, textured, and SDF-text
// rendering.
//
// The primary entry points are:
//   - `bind_eu_shader_to_batch`  — Intel Gen9/11/12 EU path
//   - `bind_rdna_shader_to_batch` — AMD RDNA1/2/3 path
//
// Both functions compile a WGSL shader, allocate a GEM buffer for the kernel
// binary, emit the appropriate pipeline-state commands into the provided
// `BatchBuilder`, and register a relocation entry so the kernel driver patches
// the shader address at submission time.

use crate::allocator::{BufferAllocator, TilingFormat};
use crate::batch::{BatchBuilder, Relocation};
use crate::detect::GpuGeneration;
use crate::eu::{EUCompiler, IntelGen};
use crate::pipeline::SolidColorPipeline;
use crate::rdna::{RDNACompiler, RDNAGen};
use crate::shader::ShaderModule;
use naga::ShaderStage;
use std::fmt;

// PM4 opcode constants for AMD RDNA shader program address registers.
const SPI_SHADER_PGM_LO_PS: u32 = 0x2C08;

/// Error produced while preparing a shader batch.
#[derive(Debug)]
pub struct ShaderBatchError {
    message: String,
}

impl fmt::Display for ShaderBatchError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "shader batch error: {}", self.message)
    }
}

impl std::error::Error for ShaderBatchError {}

impl From<String> for ShaderBatchError {
    fn from(msg: String) -> Self {
        ShaderBatchError { message: msg }
    }
}

impl From<&str> for ShaderBatchError {
    fn from(msg: &str) -> Self {
        ShaderBatchError {
            message: msg.to_string(),
        }
    }
}

/// Compile a WGSL shader to an Intel EU kernel and bind it to a batch buffer.
///
/// Allocates a GEM buffer for the compiled shader kernel, emits pipeline state
/// commands (via `SolidColorPipeline::emit_state`) into `batch`, and registers
/// a relocation entry so the kernel driver patches the shader address at
/// submit time.
///
/// # Arguments
/// * `allocator` — GEM buffer allocator for the target device
/// * `gen`       — GPU generation (must be Intel Gen9/11/12/Xe)
/// * `wgsl_source` — WGSL shader source text
/// * `stage`     — Shader stage (`Vertex` or `Fragment`)
/// * `batch`     — Batch builder to emit pipeline state commands into
///
/// # Returns
/// The GEM handle of the allocated shader kernel buffer, or an error if
/// compilation or allocation fails.
pub fn bind_eu_shader_to_batch(
    allocator: &BufferAllocator,
    gen: GpuGeneration,
    wgsl_source: &str,
    stage: ShaderStage,
    batch: &mut BatchBuilder,
) -> Result<u32, ShaderBatchError> {
    let intel_gen = gpu_gen_to_intel(gen)?;

    let module = ShaderModule::from_wgsl(wgsl_source, stage)
        .map_err(|e| ShaderBatchError::from(format!("WGSL parse error: {}", e)))?;

    let compiler = EUCompiler::new(intel_gen);
    let kernel = compiler
        .compile(&module)
        .map_err(|e| ShaderBatchError::from(format!("EU compile error: {}", e)))?;

    if kernel.binary.is_empty() {
        return Err(ShaderBatchError::from(
            "EU compiler produced empty kernel binary",
        ));
    }

    // Allocate a linear GEM buffer sized for the kernel binary.
    // width = kernel_size bytes, height = 1, bpp = 8 → stride = kernel_size.
    let kernel_size = kernel.binary.len() as u32;
    let shader_buf = allocator
        .allocate(kernel_size, 1, 8, TilingFormat::None)
        .map_err(|e| ShaderBatchError::from(format!("shader buffer allocation: {}", e)))?;

    let shader_handle = shader_buf.handle;

    // Emit pipeline state with a placeholder shader address (0).  The relocation
    // entry below causes the kernel driver to patch in the real GPU VA.
    let pipeline = SolidColorPipeline::new(gen);
    pipeline.emit_state(batch, 0u64);

    // The 3DSTATE_PS command emitted by emit_state writes a 64-bit address as
    // its last two dwords.  Record a relocation pointing at those dwords so the
    // kernel fills in the correct GPU virtual address at execbuffer time.
    let addr_dword_offset = batch.len_dwords().saturating_sub(2) as u32;
    batch.add_relocation(
        Relocation::new(addr_dword_offset, shader_handle, 0)
            .with_read_domains(0x4) // I915_GEM_DOMAIN_INSTRUCTION
            .with_write_domain(0),
    );

    Ok(shader_handle)
}

/// Compile a WGSL shader to an AMD RDNA kernel and bind it to a batch buffer.
///
/// Allocates a GEM buffer for the compiled shader kernel and emits PM4
/// `SET_SH_REG` commands that configure `SPI_SHADER_PGM_LO_PS` with the
/// shader address.  A relocation entry ensures the driver patches the address
/// at submission time.
///
/// # Arguments
/// * `allocator` — GEM buffer allocator for the target device
/// * `gen`       — GPU generation (must be RDNA1/2/3)
/// * `wgsl_source` — WGSL shader source text
/// * `stage`     — Shader stage (`Vertex` or `Fragment`)
/// * `batch`     — Batch builder to emit pipeline state commands into
///
/// # Returns
/// The GEM handle of the allocated shader kernel buffer, or an error.
pub fn bind_rdna_shader_to_batch(
    allocator: &BufferAllocator,
    gen: GpuGeneration,
    wgsl_source: &str,
    stage: ShaderStage,
    batch: &mut BatchBuilder,
) -> Result<u32, ShaderBatchError> {
    let rdna_gen = gpu_gen_to_rdna(gen)?;

    let module = ShaderModule::from_wgsl(wgsl_source, stage)
        .map_err(|e| ShaderBatchError::from(format!("WGSL parse error: {}", e)))?;

    let compiler = RDNACompiler::new(rdna_gen);
    let kernel = compiler
        .compile(&module)
        .map_err(|e| ShaderBatchError::from(format!("RDNA compile error: {}", e)))?;

    if kernel.binary.is_empty() {
        return Err(ShaderBatchError::from(
            "RDNA compiler produced empty kernel binary",
        ));
    }

    // Allocate a linear GEM buffer for the kernel (AMD kernels need 256-byte
    // alignment, but the GEM allocator handles alignment internally).
    let kernel_size = kernel.binary.len() as u32;
    let shader_buf = allocator
        .allocate(kernel_size, 1, 8, TilingFormat::None)
        .map_err(|e| ShaderBatchError::from(format!("shader buffer allocation: {}", e)))?;

    let shader_handle = shader_buf.handle;

    // Emit PM4 SET_SH_REG for SPI_SHADER_PGM_LO_PS (0x2C08).
    // The 64-bit shader address spans two consecutive 32-bit words.
    // TYPE3 opcode 0xC0 with 3 dwords (header + 2 address words).
    let header = 0xC002_0000u32 | (SPI_SHADER_PGM_LO_PS & 0xFFFF);
    batch.emit_dword(header);

    // Record the offset of the first address dword before emitting it.
    let addr_dword_offset = batch.len_dwords() as u32;
    batch.emit_dword(0); // low 32-bit of shader VA (patched by relocation)
    batch.emit_dword(0); // high 32-bit of shader VA

    batch.add_relocation(
        Relocation::new(addr_dword_offset, shader_handle, 0)
            .with_read_domains(0x4)
            .with_write_domain(0),
    );

    Ok(shader_handle)
}

/// Convert a `GpuGeneration` to an `IntelGen`, returning an error for non-Intel GPUs.
///
/// Xe is mapped to Gen12 as the closest supported generation.
pub fn gpu_gen_to_intel(gen: GpuGeneration) -> Result<IntelGen, ShaderBatchError> {
    match gen {
        GpuGeneration::Gen9 => Ok(IntelGen::Gen9),
        GpuGeneration::Gen11 => Ok(IntelGen::Gen11),
        GpuGeneration::Gen12 | GpuGeneration::Xe => Ok(IntelGen::Gen12),
        _ => Err(ShaderBatchError::from("GPU is not an Intel EU device")),
    }
}

/// Convert a `GpuGeneration` to a `RDNAGen`, returning an error for non-AMD GPUs.
pub fn gpu_gen_to_rdna(gen: GpuGeneration) -> Result<RDNAGen, ShaderBatchError> {
    match gen {
        GpuGeneration::AmdRdna1 => Ok(RDNAGen::RDNA1),
        GpuGeneration::AmdRdna2 => Ok(RDNAGen::RDNA2),
        GpuGeneration::AmdRdna3 => Ok(RDNAGen::RDNA3),
        _ => Err(ShaderBatchError::from("GPU is not an AMD RDNA device")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gpu_gen_to_intel_ok() {
        assert!(gpu_gen_to_intel(GpuGeneration::Gen9).is_ok());
        assert!(gpu_gen_to_intel(GpuGeneration::Gen11).is_ok());
        assert!(gpu_gen_to_intel(GpuGeneration::Gen12).is_ok());
        assert!(gpu_gen_to_intel(GpuGeneration::Xe).is_ok());
    }

    #[test]
    fn gpu_gen_to_intel_err_on_amd() {
        assert!(gpu_gen_to_intel(GpuGeneration::AmdRdna1).is_err());
        assert!(gpu_gen_to_intel(GpuGeneration::AmdRdna2).is_err());
        assert!(gpu_gen_to_intel(GpuGeneration::AmdRdna3).is_err());
    }

    #[test]
    fn gpu_gen_to_rdna_ok() {
        assert!(gpu_gen_to_rdna(GpuGeneration::AmdRdna1).is_ok());
        assert!(gpu_gen_to_rdna(GpuGeneration::AmdRdna2).is_ok());
        assert!(gpu_gen_to_rdna(GpuGeneration::AmdRdna3).is_ok());
    }

    #[test]
    fn gpu_gen_to_rdna_err_on_intel() {
        assert!(gpu_gen_to_rdna(GpuGeneration::Gen9).is_err());
        assert!(gpu_gen_to_rdna(GpuGeneration::Gen12).is_err());
    }

    #[test]
    fn shader_batch_error_display() {
        let err = ShaderBatchError::from("test error");
        assert_eq!(err.to_string(), "shader batch error: test error");
    }
}
