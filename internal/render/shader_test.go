package render

import (
	"testing"
)

const solidFillWGSL = `
struct VertexOutput {
    @builtin(position) position: vec4<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) vertex_index: u32) -> VertexOutput {
    var output: VertexOutput;
    let x = f32(vertex_index & 1u);
    let y = f32((vertex_index >> 1u) & 1u);
    output.position = vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
    return output;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    return vec4<f32>(1.0, 1.0, 1.0, 1.0);
}
`

func TestCompileShader(t *testing.T) {
	// Test vertex shader compilation
	vs, err := CompileShader(solidFillWGSL, GpuGen9, VertexShader)
	if err != nil {
		t.Fatalf("Failed to compile vertex shader: %v", err)
	}
	if len(vs.Data) == 0 {
		t.Fatal("Compiled vertex shader is empty")
	}
	if vs.Gen != GpuGen9 {
		t.Errorf("Expected Gen9, got %v", vs.Gen)
	}
	if vs.Stage != VertexShader {
		t.Errorf("Expected VertexShader, got %v", vs.Stage)
	}
	t.Logf("Compiled vertex shader: %d bytes", len(vs.Data))

	// Test fragment shader compilation
	fs, err := CompileShader(solidFillWGSL, GpuGen9, FragmentShader)
	if err != nil {
		t.Fatalf("Failed to compile fragment shader: %v", err)
	}
	if len(fs.Data) == 0 {
		t.Fatal("Compiled fragment shader is empty")
	}
	t.Logf("Compiled fragment shader: %d bytes", len(fs.Data))
}

func TestCompileShaderInvalidInput(t *testing.T) {
	// Test empty source
	_, err := CompileShader("", GpuGen9, VertexShader)
	if err == nil {
		t.Fatal("Expected error for empty source")
	}

	// Test invalid WGSL
	_, err = CompileShader("this is not valid WGSL", GpuGen9, VertexShader)
	if err == nil {
		t.Fatal("Expected error for invalid WGSL")
	}
}
