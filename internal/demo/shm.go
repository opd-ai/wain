package demo

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
)

// prepareShmBuffer allocates, maps, and creates a wl_buffer from a memfd.
// On error, fd is closed. On success, the returned pool owns fd and must be
// destroyed by the caller (pool.Destroy also closes fd).
func prepareShmBuffer(shmObj *shm.SHM, fd int, width, height int) (*shm.Pool, *shm.Buffer, error) {
	bufferSize := int32(width * height * 4)
	if err := syscall.Ftruncate(fd, int64(bufferSize)); err != nil {
		syscall.Close(fd)
		return nil, nil, fmt.Errorf("truncate memfd: %w", err)
	}
	pool, err := shmObj.CreatePool(fd, bufferSize)
	if err != nil {
		syscall.Close(fd)
		return nil, nil, fmt.Errorf("create shm pool: %w", err)
	}
	if err := pool.Map(); err != nil {
		pool.Destroy() //nolint:errcheck
		return nil, nil, fmt.Errorf("map pool: %w", err)
	}
	buffer, err := pool.CreateBuffer(0, int32(width), int32(height), int32(width*4), shm.FormatARGB8888)
	if err != nil {
		pool.Destroy() //nolint:errcheck
		return nil, nil, fmt.Errorf("create buffer: %w", err)
	}
	return pool, buffer, nil
}

// commitBufferToSurface attaches the buffer, marks the full surface as damaged,
// and commits the surface state to the compositor.
func commitBufferToSurface(surface *client.Surface, buffer *shm.Buffer, width, height int) error {
	if err := surface.Attach(buffer.ID(), 0, 0); err != nil {
		return fmt.Errorf("attach buffer: %w", err)
	}
	if err := surface.Damage(0, 0, int32(width), int32(height)); err != nil {
		return fmt.Errorf("damage surface: %w", err)
	}
	if err := surface.Commit(); err != nil {
		return fmt.Errorf("commit surface: %w", err)
	}
	return nil
}

// AttachAndDisplayBuffer creates a shared memory buffer and displays it on a surface.
// This helper encapsulates the common pattern of:
//  1. Creating a memfd
//  2. Creating a wl_shm pool
//  3. Creating a wl_buffer
//  4. Copying pixel data
//  5. Attaching and committing to the surface
func AttachAndDisplayBuffer(shmObj *shm.SHM, surface *client.Surface, renderBuffer *primitives.Buffer, width, height int) error {
	fd, err := shm.CreateMemfd("wain-demo-buffer")
	if err != nil {
		return fmt.Errorf("create memfd: %w", err)
	}

	pool, buffer, err := prepareShmBuffer(shmObj, fd, width, height)
	if err != nil {
		return err
	}
	defer pool.Destroy() //nolint:errcheck

	copy(buffer.Pixels(), renderBuffer.Pixels)

	return commitBufferToSurface(surface, buffer, width, height)
}
