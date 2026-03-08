package render

// #include <stdint.h>
// #include <stdlib.h>
//
// // Buffer allocator opaque handle
// typedef void* BufferAllocator;
// typedef void* Buffer;
//
// BufferAllocator buffer_allocator_create(const char* path);
// void buffer_allocator_destroy(BufferAllocator allocator);
//
// Buffer buffer_allocate(BufferAllocator allocator, uint32_t width, uint32_t height, uint32_t bpp, uint32_t tiling);
// int32_t buffer_export_dmabuf(BufferAllocator allocator, Buffer buffer);
// int32_t buffer_get_info(Buffer buffer, uint32_t* out_width, uint32_t* out_height, uint32_t* out_stride);
// uint32_t buffer_get_handle(Buffer buffer);
// int32_t buffer_destroy(BufferAllocator allocator, Buffer buffer);
// uint8_t* buffer_mmap(BufferAllocator allocator, Buffer buffer, size_t* out_size);
// int32_t buffer_munmap(uint8_t* ptr, size_t size);
import "C"

import (
	"fmt"
	"unsafe"
)

// TilingFormat represents GPU buffer tiling modes.
type TilingFormat uint32

const (
	// TilingNone means linear (no tiling) layout.
	TilingNone TilingFormat = 0
	// TilingX means X-tiled layout (Intel GPUs).
	TilingX TilingFormat = 1
	// TilingY means Y-tiled layout (Intel GPUs).
	TilingY TilingFormat = 2
)

// Allocator manages GPU buffer allocation and export.
type Allocator struct {
	handle C.BufferAllocator
}

// BufferHandle represents an allocated GPU buffer.
type BufferHandle struct {
	handle    C.Buffer
	allocator *Allocator
	Width     uint32
	Height    uint32
	Stride    uint32
}

// NewAllocator creates a buffer allocator for the DRM device at path.
// Typical path is "/dev/dri/renderD128".
func NewAllocator(path string) (*Allocator, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.buffer_allocator_create(cPath)
	if handle == nil {
		return nil, fmt.Errorf("failed to create buffer allocator for %s", path)
	}

	return &Allocator{handle: handle}, nil
}

// Close destroys the allocator and releases resources.
func (a *Allocator) Close() {
	if a.handle != nil {
		C.buffer_allocator_destroy(a.handle)
		a.handle = nil
	}
}

// Allocate creates a new GPU buffer with the specified dimensions and format.
//
// Parameters:
//   - width, height: buffer dimensions in pixels
//   - bpp: bits per pixel (typically 32 for ARGB8888)
//   - tiling: tiling format (TilingNone, TilingX, TilingY)
func (a *Allocator) Allocate(width, height, bpp uint32, tiling TilingFormat) (*BufferHandle, error) {
	if a.handle == nil {
		return nil, fmt.Errorf("allocator is closed")
	}

	handle := C.buffer_allocate(a.handle, C.uint32_t(width), C.uint32_t(height), C.uint32_t(bpp), C.uint32_t(tiling))
	if handle == nil {
		return nil, fmt.Errorf("failed to allocate buffer %dx%d", width, height)
	}

	var w, h, stride C.uint32_t
	if C.buffer_get_info(handle, &w, &h, &stride) != 0 {
		C.buffer_destroy(a.handle, handle)
		return nil, fmt.Errorf("failed to get buffer info")
	}

	return &BufferHandle{
		handle:    handle,
		allocator: a,
		Width:     uint32(w),
		Height:    uint32(h),
		Stride:    uint32(stride),
	}, nil
}

// ExportDmabuf exports the buffer as a DMA-BUF file descriptor.
// The caller owns the fd and must close it when done (using syscall.Close).
func (a *Allocator) ExportDmabuf(buffer *BufferHandle) (int, error) {
	if a.handle == nil {
		return -1, fmt.Errorf("allocator is closed")
	}
	if buffer.handle == nil {
		return -1, fmt.Errorf("buffer is destroyed")
	}

	fd := C.buffer_export_dmabuf(a.handle, buffer.handle)
	if fd < 0 {
		return -1, fmt.Errorf("failed to export buffer as dmabuf")
	}

	return int(fd), nil
}

// GemHandle returns the GEM buffer handle for GPU command submission.
//
// This handle can be used with render.SubmitBatch to reference the buffer
// in GPU commands (e.g., as a render target or vertex buffer).
func (b *BufferHandle) GemHandle() uint32 {
	if b.handle == nil {
		return 0
	}
	return uint32(C.buffer_get_handle(b.handle))
}

// Destroy frees the buffer and releases GPU memory.
func (b *BufferHandle) Destroy() error {
	if b.handle == nil {
		return nil // already destroyed
	}
	if b.allocator.handle == nil {
		return fmt.Errorf("allocator is closed")
	}

	if C.buffer_destroy(b.allocator.handle, b.handle) != 0 {
		return fmt.Errorf("failed to destroy buffer")
	}

	b.handle = nil
	return nil
}

// Mmap maps the buffer into CPU address space for reading/writing.
//
// Returns a byte slice pointing to the mapped memory. The caller must call
// Munmap when done to avoid leaking memory mappings.
//
// Example:
//
//	data, err := buffer.Mmap()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer buffer.Munmap(data)
//	// Write to data...
//	copy(data, pixels)
func (b *BufferHandle) Mmap() ([]byte, error) {
	if b.handle == nil {
		return nil, fmt.Errorf("buffer is destroyed")
	}
	if b.allocator.handle == nil {
		return nil, fmt.Errorf("allocator is closed")
	}

	var size C.size_t
	ptr := C.buffer_mmap(b.allocator.handle, b.handle, &size)
	if ptr == nil {
		return nil, fmt.Errorf("failed to mmap buffer")
	}

	// Convert C pointer to Go slice
	// Note: This creates a slice that points to mmap'd memory
	return unsafe.Slice((*byte)(ptr), int(size)), nil
}

// Munmap unmaps a previously mapped buffer.
//
// The data slice must be the one returned by Mmap. After calling Munmap,
// the data slice must not be used.
func (b *BufferHandle) Munmap(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	size := C.size_t(len(data))

	if C.buffer_munmap(ptr, size) != 0 {
		return fmt.Errorf("failed to munmap buffer")
	}

	return nil
}
