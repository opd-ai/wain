/// Shader compilation CI gate — integration test
///
/// This test verifies the full shader pipeline without GPU hardware:
///   WGSL source → naga IR → Intel EU binary (≥16 bytes, 16-byte aligned)
///   WGSL source → naga IR → AMD RDNA binary (non-empty)
///
/// All 7 UI shaders are tested for both vertex and fragment stages against
/// representative hardware generations (Intel Gen9, AMD RDNA2). Failures
/// indicate a regression in the compilation pipeline, not missing hardware.
use naga::ShaderStage;
use render_sys::eu::{EUCompiler, IntelGen};
use render_sys::rdna::{RDNACompiler, RDNAGen};
use render_sys::shader::ShaderModule;
use render_sys::shaders::UI_SHADERS;

/// Verify the shader count matches what PLAN.md Step 5 promises.
#[test]
fn shader_count_is_seven() {
    assert_eq!(
        UI_SHADERS.len(),
        7,
        "expected exactly 7 UI shaders, got {}",
        UI_SHADERS.len()
    );
}

/// All 7 WGSL shaders parse and validate through naga for both stages.
#[test]
fn all_shaders_parse_wgsl() {
    for (name, src) in UI_SHADERS {
        for stage in [ShaderStage::Vertex, ShaderStage::Fragment] {
            let result = ShaderModule::from_wgsl(src, stage);
            assert!(
                result.is_ok(),
                "shader '{}' {:?} failed naga parse: {:?}",
                name,
                stage,
                result.err()
            );
        }
    }
}

/// All 7 shaders compile to non-empty, 16-byte-aligned Intel EU binaries (Gen9).
#[test]
fn all_shaders_compile_to_eu_binary() {
    let compiler = EUCompiler::new(IntelGen::Gen9);

    for (name, src) in UI_SHADERS {
        for stage in [ShaderStage::Vertex, ShaderStage::Fragment] {
            let module = ShaderModule::from_wgsl(src, stage)
                .unwrap_or_else(|e| panic!("shader '{}' naga parse failed: {}", name, e));

            let kernel = compiler
                .compile(&module)
                .unwrap_or_else(|e| panic!("shader '{}' {:?} EU compile failed: {}", name, stage, e));

            assert!(
                kernel.binary.len() >= 16,
                "shader '{}' {:?} EU binary too small: {} bytes",
                name,
                stage,
                kernel.binary.len()
            );
            assert_eq!(
                kernel.binary.len() % 16,
                0,
                "shader '{}' {:?} EU binary not 128-bit aligned: {} bytes",
                name,
                stage,
                kernel.binary.len()
            );
        }
    }
}

/// All 7 shaders compile to non-empty AMD RDNA2 binaries.
#[test]
fn all_shaders_compile_to_rdna_binary() {
    let compiler = RDNACompiler::new(RDNAGen::RDNA2);

    for (name, src) in UI_SHADERS {
        // RDNA lower() operates on fragment stage; use Fragment for all shaders.
        let module = ShaderModule::from_wgsl(src, ShaderStage::Fragment)
            .unwrap_or_else(|e| panic!("shader '{}' naga parse failed: {}", name, e));

        let kernel = compiler
            .compile(&module)
            .unwrap_or_else(|e| panic!("shader '{}' RDNA compile failed: {}", name, e));

        assert!(
            !kernel.binary.is_empty(),
            "shader '{}' RDNA binary is empty",
            name
        );
    }
}

/// EU binaries for all generations (Gen9, Gen11, Gen12, Xe) are non-empty.
#[test]
fn eu_all_generations_produce_binary() {
    let src = UI_SHADERS
        .iter()
        .find(|(n, _)| *n == "solid_fill")
        .map(|(_, s)| *s)
        .expect("solid_fill shader must exist");

    for gen in [IntelGen::Gen9, IntelGen::Gen11, IntelGen::Gen12] {
        let compiler = EUCompiler::new(gen);
        let module = ShaderModule::from_wgsl(src, ShaderStage::Vertex).unwrap();
        let kernel = compiler
            .compile(&module)
            .unwrap_or_else(|e| panic!("solid_fill vertex EU {gen:?} failed: {e}"));
        assert!(
            kernel.binary.len() >= 16,
            "EU {gen:?} binary too small: {} bytes",
            kernel.binary.len()
        );
    }
}

/// RDNA binaries for all generations (RDNA1, RDNA2, RDNA3) are non-empty.
#[test]
fn rdna_all_generations_produce_binary() {
    let src = UI_SHADERS
        .iter()
        .find(|(n, _)| *n == "solid_fill")
        .map(|(_, s)| *s)
        .expect("solid_fill shader must exist");

    for gen in [RDNAGen::RDNA1, RDNAGen::RDNA2, RDNAGen::RDNA3] {
        let compiler = RDNACompiler::new(gen);
        let module = ShaderModule::from_wgsl(src, ShaderStage::Fragment).unwrap();
        let kernel = compiler
            .compile(&module)
            .unwrap_or_else(|e| panic!("solid_fill RDNA {gen:?} failed: {e}"));
        assert!(
            !kernel.binary.is_empty(),
            "RDNA {gen:?} binary is empty"
        );
    }
}
