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

    // Phase 4.2 - UI Shader Authoring validation tests
    // These tests validate that all UI shaders in render-sys/shaders/ compile correctly

    #[test]
    fn test_solid_fill_shader() {
        let source = include_str!("../shaders/solid_fill.wgsl");
        
        // Test vertex shader compilation
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Solid fill vertex shader should compile: {:?}", vs_result.err());
        
        // Test fragment shader compilation
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Solid fill fragment shader should compile: {:?}", fs_result.err());
        
        // Verify entry points exist
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Solid fill shader should have entry points");
    }

    #[test]
    fn test_textured_quad_shader() {
        let source = include_str!("../shaders/textured_quad.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Textured quad vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Textured quad fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Textured quad shader should have entry points");
    }

    #[test]
    fn test_sdf_text_shader() {
        let source = include_str!("../shaders/sdf_text.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "SDF text vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "SDF text fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "SDF text shader should have entry points");
    }

    #[test]
    fn test_box_shadow_shader() {
        let source = include_str!("../shaders/box_shadow.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Box shadow vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Box shadow fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Box shadow shader should have entry points");
    }

    #[test]
    fn test_rounded_rect_shader() {
        let source = include_str!("../shaders/rounded_rect.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Rounded rect vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Rounded rect fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Rounded rect shader should have entry points");
    }

    #[test]
    fn test_linear_gradient_shader() {
        let source = include_str!("../shaders/linear_gradient.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Linear gradient vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Linear gradient fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Linear gradient shader should have entry points");
    }

    #[test]
    fn test_radial_gradient_shader() {
        let source = include_str!("../shaders/radial_gradient.wgsl");
        
        let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
        assert!(vs_result.is_ok(), "Radial gradient vertex shader should compile: {:?}", vs_result.err());
        
        let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
        assert!(fs_result.is_ok(), "Radial gradient fragment shader should compile: {:?}", fs_result.err());
        
        let module = vs_result.unwrap();
        assert!(!module.ir().entry_points.is_empty(), "Radial gradient shader should have entry points");
    }

    #[test]
    fn test_all_shaders_count() {
        // Verify we have all 7 required shaders by attempting to load them
        // This validates that all shader files exist and are accessible via include_str!
        let shaders = vec![
            ("solid_fill", include_str!("../shaders/solid_fill.wgsl")),
            ("textured_quad", include_str!("../shaders/textured_quad.wgsl")),
            ("sdf_text", include_str!("../shaders/sdf_text.wgsl")),
            ("box_shadow", include_str!("../shaders/box_shadow.wgsl")),
            ("rounded_rect", include_str!("../shaders/rounded_rect.wgsl")),
            ("linear_gradient", include_str!("../shaders/linear_gradient.wgsl")),
            ("radial_gradient", include_str!("../shaders/radial_gradient.wgsl")),
        ];
        
        assert_eq!(shaders.len(), 7, "Should have exactly 7 UI shaders");
        
        // Verify each shader is non-empty
        for (name, source) in shaders.iter() {
            assert!(!source.is_empty(), "Shader {} should not be empty", name);
            assert!(source.contains("@vertex"), "Shader {} should have a vertex entry point", name);
            assert!(source.contains("@fragment"), "Shader {} should have a fragment entry point", name);
        }
    }
}
