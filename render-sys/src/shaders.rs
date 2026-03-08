// Pre-compiled UI shader sources
//
// This module provides access to the WGSL shader sources for all UI shaders.
// The sources are compiled at build time by build.rs and embedded as static
// strings in the binary.
//
// Per ROADMAP.md Phase 4.4: These shader sources will eventually be compiled
// to GPU binaries (Intel EU machine code) by the EU backend compiler and
// embedded as byte arrays. For now, we embed the WGSL sources which can be
// parsed via the shader module's naga frontend.

// Include the auto-generated shader constants
include!(concat!(env!("OUT_DIR"), "/compiled_shaders.rs"));

#[cfg(test)]
mod tests {
    use super::*;
    use crate::shader::ShaderModule;
    use naga::ShaderStage;

    #[test]
    fn test_all_shader_sources_exist() {
        assert_eq!(UI_SHADERS.len(), 7, "Should have exactly 7 UI shaders");
        
        for (name, source) in UI_SHADERS {
            assert!(!source.is_empty(), "Shader {} should not be empty", name);
            assert!(source.contains("@vertex"), "Shader {} should have vertex entry point", name);
            assert!(source.contains("@fragment"), "Shader {} should have fragment entry point", name);
        }
    }

    #[test]
    fn test_all_shaders_compile() {
        // Verify that all embedded shaders compile successfully via naga
        for (name, source) in UI_SHADERS {
            let vs_result = ShaderModule::from_wgsl(source, ShaderStage::Vertex);
            assert!(
                vs_result.is_ok(),
                "Shader {} vertex stage should compile: {:?}",
                name,
                vs_result.err()
            );
            
            let fs_result = ShaderModule::from_wgsl(source, ShaderStage::Fragment);
            assert!(
                fs_result.is_ok(),
                "Shader {} fragment stage should compile: {:?}",
                name,
                fs_result.err()
            );
        }
    }

    #[test]
    fn test_shader_lookup() {
        // Test that we can look up shaders by name
        let solid_fill = UI_SHADERS.iter()
            .find(|(name, _)| *name == "solid_fill")
            .map(|(_, source)| *source);
        
        assert!(solid_fill.is_some(), "Should be able to find solid_fill shader");
        assert_eq!(solid_fill.unwrap(), SOLID_FILL_WGSL);
    }

    #[test]
    fn test_individual_shader_constants() {
        // Verify all individual constants are accessible
        assert!(!SOLID_FILL_WGSL.is_empty());
        assert!(!TEXTURED_QUAD_WGSL.is_empty());
        assert!(!SDF_TEXT_WGSL.is_empty());
        assert!(!BOX_SHADOW_WGSL.is_empty());
        assert!(!ROUNDED_RECT_WGSL.is_empty());
        assert!(!LINEAR_GRADIENT_WGSL.is_empty());
        assert!(!RADIAL_GRADIENT_WGSL.is_empty());
    }
}
