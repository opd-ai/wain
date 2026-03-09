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
// Software Rasterizer Limitations:
//
// The SoftwareConsumer does not implement the CmdDrawImage display list command.
// Image compositing is available through the composite package's Blit and BlitScaled
// functions, but DrawImage command execution requires a GPU backend. This is a
// deliberate design decision to keep the software rasterizer focused on vector
// primitives while GPU-accelerated texture sampling handles image operations.
//
// GPU Consumer:
//
// The GPUConsumer wraps the backend.GPUBackend to provide hardware-accelerated
// rendering of display lists. It converts display list commands into GPU batch
// buffers with vertex data, supporting damage tracking via scissor rectangles.
// The GPU consumer exports render targets as DMA-BUF file descriptors for
// zero-copy display on Wayland/X11.
package consumer
