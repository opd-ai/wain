// Box Shadow Shader - separable Gaussian blur for shadows
// Phase 4.2 - UI Shader Authoring

// Input texture and sampler
@group(0) @binding(0) var input_tex: texture_2d<f32>;
@group(0) @binding(1) var tex_sampler: sampler;

// Blur parameters
struct BlurUniforms {
    // Blur direction: (1, 0) for horizontal, (0, 1) for vertical
    direction: vec2<f32>,
    // Blur radius in pixels
    radius: f32,
    // Texture size for offset calculation
    tex_size: vec2<f32>,
}
@group(0) @binding(2) var<uniform> blur: BlurUniforms;

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
    // Separable Gaussian blur - samples along one axis
    // 5-tap kernel with weights: 0.054, 0.244, 0.404, 0.244, 0.054
    let offset = blur.direction / blur.tex_size;
    
    var color = vec4<f32>(0.0);
    color += textureSample(input_tex, tex_sampler, in.uv - offset * 2.0) * 0.054;
    color += textureSample(input_tex, tex_sampler, in.uv - offset) * 0.244;
    color += textureSample(input_tex, tex_sampler, in.uv) * 0.404;
    color += textureSample(input_tex, tex_sampler, in.uv + offset) * 0.244;
    color += textureSample(input_tex, tex_sampler, in.uv + offset * 2.0) * 0.054;
    
    return color;
}
