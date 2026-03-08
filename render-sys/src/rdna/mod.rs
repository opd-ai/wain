// AMD RDNA Backend - Phase 6.3
//
// This module implements the backend that lowers naga IR to AMD RDNA machine code.
// The RDNA backend compiles WGSL/GLSL shaders (after naga parsing) into binary kernel
// objects that can be uploaded to GPU memory and executed on AMD GPUs.
//
// Architecture:
// - Register allocation: Map naga IR SSA values to RDNA VGPR/SGPR files
// - Instruction lowering: Translate naga ops to RDNA VOP, SOP, MIMG, and EXP instructions
// - Texture sampling: Lower to MIMG instructions for sampler access
// - I/O handling: Export instructions for render target writes
//
// Reference: AMD RDNA ISA documentation, Mesa RADV compiler (src/amd/compiler/)
// Similar structure to Intel EU backend (../eu/)

pub mod instruction;
pub mod regalloc;
pub mod encoding;
pub mod lower;
pub mod types;

use crate::shader::ShaderModule;
use naga::ShaderStage;
use std::error::Error;
use std::fmt;

/// RDNA backend compilation error
#[derive(Debug)]
pub struct RDNACompileError {
    message: String,
}

impl fmt::Display for RDNACompileError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "RDNA compilation error: {}", self.message)
    }
}

impl Error for RDNACompileError {}

impl From<String> for RDNACompileError {
    fn from(msg: String) -> Self {
        RDNACompileError { message: msg }
    }
}

impl From<&str> for RDNACompileError {
    fn from(msg: &str) -> Self {
        RDNACompileError {
            message: msg.to_string(),
        }
    }
}

/// AMD GPU generation for RDNA ISA selection
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RDNAGen {
    /// RDNA1 (RX 5000 series)
    RDNA1,
    /// RDNA2 (RX 6000 series, Steam Deck)
    RDNA2,
    /// RDNA3 (RX 7000 series)
    RDNA3,
}

/// Compiled RDNA kernel binary
#[derive(Debug)]
pub struct RDNAKernel {
    /// Binary machine code ready for upload to GPU
    pub binary: Vec<u8>,
    /// GPU generation this kernel was compiled for
    pub gen: RDNAGen,
    /// Shader stage
    pub stage: ShaderStage,
    /// VGPR count used
    pub vgpr_count: u8,
    /// SGPR count used
    pub sgpr_count: u8,
}

/// RDNA backend compiler
pub struct RDNACompiler {
    gen: RDNAGen,
}

impl RDNACompiler {
    /// Create a new RDNA compiler for the specified GPU generation
    pub fn new(gen: RDNAGen) -> Self {
        RDNACompiler { gen }
    }

    /// Compile a shader module to RDNA machine code
    ///
    /// # Arguments
    /// * `module` - Validated naga IR module from ShaderModule
    ///
    /// # Returns
    /// Compiled binary kernel ready for GPU upload
    pub fn compile(&self, module: &ShaderModule) -> Result<RDNAKernel, RDNACompileError> {
        use crate::rdna::lower::LoweringContext;
        
        // Get the naga module and entry point info
        let naga_module = module.ir();
        let stage = module.stage();

        // Find the entry point function
        let entry_point = naga_module
            .entry_points
            .iter()
            .find(|ep| ep.stage == stage)
            .ok_or("No entry point found for shader stage")?;

        // Create lowering context and compile
        let mut ctx = LoweringContext::new(naga_module, &entry_point.function, stage);
        let binary = ctx.lower();

        Ok(RDNAKernel {
            binary,
            gen: self.gen,
            stage,
            vgpr_count: ctx.vgpr_count(),
            sgpr_count: ctx.sgpr_count(),
        })
    }

    /// Get the target GPU generation
    pub fn generation(&self) -> RDNAGen {
        self.gen
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rdna_compiler_creation() {
        let compiler = RDNACompiler::new(RDNAGen::RDNA2);
        assert_eq!(compiler.generation(), RDNAGen::RDNA2);
    }

    #[test]
    fn test_rdna_gen_variants() {
        assert_ne!(RDNAGen::RDNA1, RDNAGen::RDNA2);
        assert_ne!(RDNAGen::RDNA2, RDNAGen::RDNA3);
    }

    #[test]
    fn test_compile_simple_shader() {
        use crate::shader::ShaderModule;

        // Simple fragment shader that outputs red
        let wgsl = r#"
            @fragment
            fn main() -> @location(0) vec4<f32> {
                return vec4<f32>(1.0, 0.0, 0.0, 1.0);
            }
        "#;

        let module = ShaderModule::from_wgsl(wgsl, ShaderStage::Fragment)
            .expect("Failed to parse shader");
        
        let compiler = RDNACompiler::new(RDNAGen::RDNA2);
        let kernel = compiler.compile(&module).expect("Failed to compile shader");

        assert_eq!(kernel.stage, ShaderStage::Fragment);
        assert_eq!(kernel.gen, RDNAGen::RDNA2);
        assert!(!kernel.binary.is_empty());
        assert!(kernel.vgpr_count > 0);
    }

    #[test]
    fn test_compile_vertex_shader() {
        use crate::shader::ShaderModule;

        // Simple vertex shader
        let wgsl = r#"
            struct VertexOutput {
                @builtin(position) position: vec4<f32>,
            }

            @vertex
            fn main(@location(0) pos: vec2<f32>) -> VertexOutput {
                var output: VertexOutput;
                output.position = vec4<f32>(pos, 0.0, 1.0);
                return output;
            }
        "#;

        let module = ShaderModule::from_wgsl(wgsl, ShaderStage::Vertex)
            .expect("Failed to parse shader");
        
        let compiler = RDNACompiler::new(RDNAGen::RDNA2);
        let kernel = compiler.compile(&module).expect("Failed to compile shader");

        assert_eq!(kernel.stage, ShaderStage::Vertex);
        assert!(!kernel.binary.is_empty());
    }

    #[test]
    fn test_compile_all_rdna_generations() {
        use crate::shader::ShaderModule;

        let wgsl = r#"
            @fragment
            fn main() -> @location(0) vec4<f32> {
                return vec4<f32>(1.0, 0.0, 0.0, 1.0);
            }
        "#;

        let module = ShaderModule::from_wgsl(wgsl, ShaderStage::Fragment)
            .expect("Failed to parse shader");

        // Test compilation for all RDNA generations
        for gen in [RDNAGen::RDNA1, RDNAGen::RDNA2, RDNAGen::RDNA3] {
            let compiler = RDNACompiler::new(gen);
            let kernel = compiler.compile(&module).expect("Failed to compile");
            assert_eq!(kernel.gen, gen);
            assert!(!kernel.binary.is_empty());
        }
    }
}
