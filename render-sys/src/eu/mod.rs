// Intel EU (Execution Unit) Backend - Phase 4.3
//
// This module implements the backend that lowers naga IR to Intel EU machine code.
// The EU backend compiles WGSL/GLSL shaders (after naga parsing) into binary kernel
// objects that can be uploaded to GPU memory and referenced by 3DSTATE_VS/3DSTATE_PS.
//
// Architecture:
// - Register allocation: Map naga IR SSA values to EU GRF (General Register File)
// - Instruction lowering: Translate naga ops to EU ALU, logic, and flow control
// - Texture sampling: Lower to EU SEND instructions targeting sampler shared function
// - I/O handling: URB reads/writes for vertex shader, render target writes for fragment
//
// Reference: Intel PRMs Volume 4 (EU ISA) and Volume 7 (3D Media GPGPU)
// Inspiration: Mesa's src/intel/compiler/ for lowering patterns

pub mod instruction;
pub mod regalloc;
pub mod encoding;
pub mod lower;

use crate::shader::ShaderModule;
use naga::{Module, ShaderStage};
use std::error::Error;
use std::fmt;

/// EU backend compilation error
#[derive(Debug)]
pub struct EUCompileError {
    message: String,
}

impl fmt::Display for EUCompileError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "EU compilation error: {}", self.message)
    }
}

impl Error for EUCompileError {}

impl From<String> for EUCompileError {
    fn from(msg: String) -> Self {
        EUCompileError { message: msg }
    }
}

impl From<&str> for EUCompileError {
    fn from(msg: &str) -> Self {
        EUCompileError {
            message: msg.to_string(),
        }
    }
}

/// Intel GPU generation for EU ISA selection
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IntelGen {
    /// Gen9 (Skylake, Kaby Lake, Coffee Lake)
    Gen9,
    /// Gen11 (Ice Lake)
    Gen11,
    /// Gen12 (Tiger Lake, Rocket Lake, Alder Lake)
    Gen12,
}

/// Compiled EU kernel binary
#[derive(Debug)]
pub struct EUKernel {
    /// Binary machine code ready for upload to GPU
    pub binary: Vec<u8>,
    /// GPU generation this kernel was compiled for
    pub gen: IntelGen,
    /// Shader stage
    pub stage: ShaderStage,
}

/// EU backend compiler
pub struct EUCompiler {
    gen: IntelGen,
}

impl EUCompiler {
    /// Create a new EU compiler for the specified GPU generation
    pub fn new(gen: IntelGen) -> Self {
        EUCompiler { gen }
    }

    /// Compile a shader module to EU machine code
    ///
    /// # Arguments
    /// * `module` - Validated naga IR module from ShaderModule
    ///
    /// # Returns
    /// Compiled binary kernel ready for GPU upload
    pub fn compile(&self, _module: &ShaderModule) -> Result<EUKernel, EUCompileError> {
        // Phase 4.3 implementation placeholder
        // This will contain the full compiler pipeline:
        // 1. Register allocation (regalloc module)
        // 2. Instruction lowering (instruction module)
        // 3. URB/texture/render target handling
        // 4. Binary encoding
        
        Err(EUCompileError::from("EU backend not yet implemented (Phase 4.3)"))
    }

    /// Get the target GPU generation
    pub fn gen(&self) -> IntelGen {
        self.gen
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::shader::ShaderModule;
    use naga::ShaderStage;

    #[test]
    fn test_eu_compiler_creation() {
        let compiler = EUCompiler::new(IntelGen::Gen9);
        assert_eq!(compiler.gen(), IntelGen::Gen9);
        
        let compiler = EUCompiler::new(IntelGen::Gen12);
        assert_eq!(compiler.gen(), IntelGen::Gen12);
    }

    #[test]
    fn test_eu_compile_placeholder() {
        // This test will be extended as we implement the EU backend
        let compiler = EUCompiler::new(IntelGen::Gen9);
        
        let shader_source = r#"
            @vertex
            fn main(@builtin(vertex_index) vertex_index: u32) -> @builtin(position) vec4<f32> {
                return vec4<f32>(0.0, 0.0, 0.0, 1.0);
            }
        "#;
        
        let module = ShaderModule::from_wgsl(shader_source, ShaderStage::Vertex).unwrap();
        let result = compiler.compile(&module);
        
        // Currently returns error since backend is not implemented
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("not yet implemented"));
    }
}
