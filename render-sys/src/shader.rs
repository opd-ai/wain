// Shader frontend module - parses GLSL/WGSL into naga IR
//
// This module provides the frontend IR infrastructure for Phase 4 of the shader
// compiler pipeline. It uses naga to parse GLSL and WGSL shader sources into
// a typed intermediate representation (IR) that can be validated and lowered
// to GPU-specific machine code.

use naga::front::{glsl, wgsl};
use naga::valid::{Capabilities, ValidationFlags, Validator};
use naga::{Module, ShaderStage};
use std::error::Error;
use std::fmt;

/// Shader source language
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ShaderLanguage {
    /// WGSL (WebGPU Shading Language) - preferred for new shaders
    WGSL,
    /// GLSL (OpenGL Shading Language) - legacy support
    GLSL,
}

/// Shader compilation error
#[derive(Debug)]
pub struct ShaderError {
    message: String,
}

impl fmt::Display for ShaderError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Shader compilation error: {}", self.message)
    }
}

impl Error for ShaderError {}

impl From<String> for ShaderError {
    fn from(msg: String) -> Self {
        ShaderError { message: msg }
    }
}

impl From<&str> for ShaderError {
    fn from(msg: &str) -> Self {
        ShaderError {
            message: msg.to_string(),
        }
    }
}

/// Compiled shader module containing validated naga IR
pub struct ShaderModule {
    /// The naga IR module
    pub module: Module,
    /// Shader stage (vertex, fragment, compute)
    pub stage: ShaderStage,
}

impl ShaderModule {
    /// Create a new shader module from WGSL source
    ///
    /// # Arguments
    /// * `source` - WGSL shader source code
    /// * `stage` - Shader stage (Vertex, Fragment, or Compute)
    ///
    /// # Returns
    /// Validated shader module or compilation error
    pub fn from_wgsl(source: &str, stage: ShaderStage) -> Result<Self, ShaderError> {
        // Parse WGSL source
        let module = wgsl::parse_str(source).map_err(|e| ShaderError {
            message: format!("WGSL parse error: {:?}", e),
        })?;

        // Validate the module
        let mut validator = Validator::new(ValidationFlags::all(), Capabilities::all());
        validator
            .validate(&module)
            .map_err(|e| ShaderError {
                message: format!("Validation error: {:?}", e),
            })?;

        Ok(ShaderModule { module, stage })
    }

    /// Create a new shader module from GLSL source
    ///
    /// # Arguments
    /// * `source` - GLSL shader source code
    /// * `stage` - Shader stage (Vertex or Fragment)
    ///
    /// # Returns
    /// Validated shader module or compilation error
    pub fn from_glsl(source: &str, stage: ShaderStage) -> Result<Self, ShaderError> {
        // GLSL parsing requires explicit version directive and entry point
        // The parser infers the stage from #version directive and shader content
        
        // Configure GLSL parser options
        let options = glsl::Options::from(stage);

        // Parse GLSL source
        let mut parser = glsl::Frontend::default();
        let module = parser
            .parse(&options, source)
            .map_err(|errors| ShaderError {
                message: format!("GLSL parse errors: {:?}", errors),
            })?;

        // Validate the module
        let mut validator = Validator::new(ValidationFlags::all(), Capabilities::all());
        validator
            .validate(&module)
            .map_err(|e| ShaderError {
                message: format!("Validation error: {:?}", e),
            })?;

        Ok(ShaderModule { module, stage })
    }

    /// Get the shader stage
    pub fn stage(&self) -> ShaderStage {
        self.stage
    }

    /// Get the naga IR module
    pub fn ir(&self) -> &Module {
        &self.module
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_wgsl_vertex_shader() {
        let source = r#"
            @vertex
            fn main(@builtin(vertex_index) vertex_index: u32) -> @builtin(position) vec4<f32> {
                let x = f32(vertex_index & 1u);
                let y = f32((vertex_index >> 1u) & 1u);
                return vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
            }
        "#;

        let result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(result.is_ok(), "WGSL vertex shader should compile");

        let module = result.unwrap();
        assert_eq!(module.stage(), ShaderStage::Vertex);
        assert!(!module.ir().entry_points.is_empty());
    }

    #[test]
    fn test_wgsl_fragment_shader() {
        let source = r#"
            @fragment
            fn main() -> @location(0) vec4<f32> {
                return vec4<f32>(1.0, 0.0, 0.0, 1.0);
            }
        "#;

        let result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(result.is_ok(), "WGSL fragment shader should compile");

        let module = result.unwrap();
        assert_eq!(module.stage(), ShaderStage::Fragment);
    }

    #[test]
    fn test_glsl_vertex_shader() {
        let source = r#"
            #version 450
            layout(location = 0) in vec2 position;
            layout(location = 1) in vec4 color;
            layout(location = 0) out vec4 v_color;
            
            void main() {
                gl_Position = vec4(position, 0.0, 1.0);
                v_color = color;
            }
        "#;

        let result = ShaderModule::from_glsl(source, ShaderStage::Vertex);
        assert!(result.is_ok(), "GLSL vertex shader should compile");

        let module = result.unwrap();
        assert_eq!(module.stage(), ShaderStage::Vertex);
    }

    #[test]
    fn test_glsl_fragment_shader() {
        let source = r#"
            #version 450
            layout(location = 0) in vec4 v_color;
            layout(location = 0) out vec4 f_color;
            
            void main() {
                f_color = v_color;
            }
        "#;

        let result = ShaderModule::from_glsl(source, ShaderStage::Fragment);
        assert!(result.is_ok(), "GLSL fragment shader should compile");

        let module = result.unwrap();
        assert_eq!(module.stage(), ShaderStage::Fragment);
    }

    #[test]
    fn test_invalid_wgsl() {
        let source = "this is not valid WGSL code";
        let result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(result.is_err(), "Invalid WGSL should fail compilation");
    }

    #[test]
    fn test_invalid_glsl() {
        let source = "this is not valid GLSL code";
        let result = ShaderModule::from_glsl(source, ShaderStage::Vertex);
        assert!(result.is_err(), "Invalid GLSL should fail compilation");
    }
}
