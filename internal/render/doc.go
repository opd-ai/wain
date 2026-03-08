// Package render provides GPU abstraction, buffer management, and shader compilation
// for GPU-accelerated rendering. It handles GPU detection across Intel and AMD GPUs,
// manages GPU memory allocation with DMA-BUF export support, and compiles WGSL/GLSL
// shaders to GPU-specific binaries (Intel EU ISA or AMD RDNA ISA).
package render
