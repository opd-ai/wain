package render

// #include <stdint.h>
// #include <stdlib.h>
// #include <string.h>
//
// // Compile a WGSL shader to Intel EU machine code
// uint8_t* render_compile_shader(const char* wgsl_source, int32_t gpu_gen, int32_t is_fragment, size_t* out_size);
// void render_shader_free(uint8_t* ptr, size_t size);
import "C"

import (
	"errors"
	"unsafe"
)

// ShaderStage represents the shader pipeline stage.
type ShaderStage int

const (
	// VertexShader is the vertex shader stage
	VertexShader ShaderStage = 0
	// FragmentShader is the fragment/pixel shader stage
	FragmentShader ShaderStage = 1
)

// ShaderBinary represents a compiled shader binary ready for GPU upload.
type ShaderBinary struct {
	// Data is the binary machine code
	Data []byte
	// Gen is the GPU generation this shader was compiled for
	Gen GpuGeneration
	// Stage is the shader pipeline stage
	Stage ShaderStage
}

// Free releases the shader binary memory.
func (s *ShaderBinary) Free() {
	if len(s.Data) > 0 {
		// Note: Go manages the memory for s.Data slice
		// No need to call C free since we copied the data
		s.Data = nil
	}
}

// CompileShader compiles WGSL source code to Intel EU machine code.
//
// Parameters:
//   - wgslSource: WGSL shader source code
//   - gen: Target GPU generation (GpuGen9, GpuGen11, or GpuGen12)
//   - stage: Shader stage (VertexShader or FragmentShader)
//
// Returns:
//   - Compiled shader binary ready for GPU upload
//   - Error if compilation fails
func CompileShader(wgslSource string, gen GpuGeneration, stage ShaderStage) (*ShaderBinary, error) {
	if wgslSource == "" {
		return nil, errors.New("shader source cannot be empty")
	}

	// Convert Go string to C string
	cSource := C.CString(wgslSource)
	defer C.free(unsafe.Pointer(cSource))

	// Convert stage to is_fragment flag
	isFragment := C.int32_t(0)
	if stage == FragmentShader {
		isFragment = 1
	}

	// Call C function to compile shader
	var outSize C.size_t
	ptr := C.render_compile_shader(cSource, C.int32_t(gen), isFragment, &outSize)
	if ptr == nil {
		return nil, errors.New("shader compilation failed")
	}

	// Copy binary data to Go slice
	size := int(outSize)
	data := make([]byte, size)
	C.memcpy(
		unsafe.Pointer(&data[0]),
		unsafe.Pointer(ptr),
		outSize,
	)

	// Free C-allocated memory
	C.render_shader_free(ptr, outSize)

	return &ShaderBinary{
		Data:  data,
		Gen:   gen,
		Stage: stage,
	}, nil
}
