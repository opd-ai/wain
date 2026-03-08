// SDF Text Shader - renders text using signed distance fields
// Phase 4.2 - UI Shader Authoring

// SDF atlas texture and sampler
@group(0) @binding(0) var sdf_atlas: texture_2d<f32>;
@group(0) @binding(1) var sdf_sampler: sampler;

// Uniforms for text rendering
struct TextUniforms {
    color: vec4<f32>,
    smoothing: f32,
    threshold: f32,
}
@group(0) @binding(2) var<uniform> uniforms: TextUniforms;

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(
    @builtin(vertex_index) vertex_index: u32,
) -> VertexOutput {
    var output: VertexOutput;
    
    // Generate triangle strip quad from vertex index
    let x = f32(vertex_index & 1u);
    let y = f32((vertex_index >> 1u) & 1u);
    
    output.position = vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
    output.uv = vec2<f32>(x, y);
    
    return output;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    // Sample the SDF value from the atlas
    let sdf_value = textureSample(sdf_atlas, sdf_sampler, in.uv).r;
    
    // Convert SDF to alpha with smoothstep for anti-aliasing
    // SDF stores signed distance: 0.5 = edge, >0.5 = inside, <0.5 = outside
    let edge_distance = sdf_value - uniforms.threshold;
    let alpha = smoothstep(-uniforms.smoothing, uniforms.smoothing, edge_distance);
    
    // Return the text color with SDF-based alpha
    return vec4<f32>(uniforms.color.rgb, uniforms.color.a * alpha);
}
