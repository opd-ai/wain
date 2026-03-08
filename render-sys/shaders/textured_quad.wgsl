// Textured Quad Shader - renders a quad with a texture
// Phase 4.2 - UI Shader Authoring

// Texture and sampler bindings
@group(0) @binding(0) var tex: texture_2d<f32>;
@group(0) @binding(1) var samp: sampler;

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
    // Sample the texture with bilinear filtering
    return textureSample(tex, samp, in.uv);
}
