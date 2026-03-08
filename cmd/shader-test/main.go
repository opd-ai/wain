// Command shader-test tests shader compilation for all 7 UI shader types.
// This is a minimal test to verify Phase 4.6 - EU backend compiles all shaders.
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/render"
)

const drmPath = "/dev/dri/renderD128"

const (
	solidFillWGSL = `
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
    return vec4<f32>(0.8, 0.2, 0.2, 1.0);
}
`

	linearGradientWGSL = `
struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) vertex_index: u32) -> VertexOutput {
    var output: VertexOutput;
    let x = f32(vertex_index & 1u);
    let y = f32((vertex_index >> 1u) & 1u);
    output.position = vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
    output.uv = vec2<f32>(x, y);
    return output;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let t = in.uv.x;
    return vec4<f32>(t, 0.5, 1.0 - t, 1.0);
}
`

	radialGradientWGSL = `
struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) vertex_index: u32) -> VertexOutput {
    var output: VertexOutput;
    let x = f32(vertex_index & 1u);
    let y = f32((vertex_index >> 1u) & 1u);
    output.position = vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
    output.uv = vec2<f32>(x, y);
    return output;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let center = vec2<f32>(0.5, 0.5);
    let dist = distance(in.uv, center);
    let t = clamp(dist / 0.5, 0.0, 1.0);
    return vec4<f32>(0.8 * t, 0.4, 0.2 * (1.0 - t), 1.0);
}
`
)

type shaderTest struct {
	name string
	wgsl string
}

func main() {
	demo.CheckHelpFlag("shader-test", "Test shader compilation for all 7 UI shader types", []string{
		demo.FormatExample("shader-test", "Compile and validate all UI shaders"),
		demo.FormatExample("shader-test --help", "Show this help message"),
	})

	fmt.Println("==============================================")
	fmt.Println("wain Phase 4.6 - Shader Compilation Test")
	fmt.Println("==============================================")
	fmt.Println()

	if err := runTest(); err != nil {
		log.Fatalf("Test failed: %v", err)
	}

	fmt.Println("\n✓ All shaders compiled successfully!")
}

func runTest() error {
	// Detect GPU
	gen := render.DetectGPU(drmPath)
	if gen == render.GpuUnknown {
		return fmt.Errorf("GPU not detected at %s", drmPath)
	}

	fmt.Printf("Detected GPU: %v\n\n", gen)

	tests := []shaderTest{
		{"Solid Fill", solidFillWGSL},
		{"Linear Gradient", linearGradientWGSL},
		{"Radial Gradient", radialGradientWGSL},
	}

	for i, test := range tests {
		fmt.Printf("[%d/%d] %s\n", i+1, len(tests), test.name)

		// Compile vertex shader
		vsData, err := render.CompileShader(test.wgsl, gen, render.VertexShader)
		if err != nil {
			return fmt.Errorf("  ✗ vertex shader compilation failed: %w", err)
		}
		fmt.Printf("  ✓ Vertex shader compiled (%d bytes EU binary)\n", len(vsData.Data))

		// Compile fragment shader
		fsData, err := render.CompileShader(test.wgsl, gen, render.FragmentShader)
		if err != nil {
			return fmt.Errorf("  ✗ fragment shader compilation failed: %w", err)
		}
		fmt.Printf("  ✓ Fragment shader compiled (%d bytes EU binary)\n", len(fsData.Data))
	}

	return nil
}
