// Package consumer implements display list consumers for different rendering backends.
//
// A consumer takes a DisplayList and renders it to a target surface using either
// software (CPU) rasterization or GPU acceleration.
//
// Available Consumers:
//
//   - SoftwareConsumer: CPU-based rasterization for vector primitives
//   - GPUConsumer: GPU-accelerated rendering with batch submission
//
// The SoftwareConsumer handles all DisplayList command types, including
// CmdDrawImage. Image blitting uses bilinear scaling via
// internal/raster/composite.BlitScaled. If DrawImageData.Src is nil
// (GPU-only path), the call is silently skipped.
//
// GPU Consumer:
//
// The GPUConsumer wraps the backend.GPUBackend to provide hardware-accelerated
// rendering of display lists. It converts display list commands into GPU batch
// buffers with vertex data, supporting damage tracking via scissor rectangles.
// The GPU consumer exports render targets as DMA-BUF file descriptors for
// zero-copy display on Wayland/X11.
package consumer
