package demo

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
)

// AttachAndDisplayBuffer creates a shared memory buffer and displays it on a surface.
// This helper encapsulates the common pattern of:
//  1. Creating a memfd
//  2. Creating a wl_shm pool
//  3. Creating a wl_buffer
//  4. Copying pixel data
//  5. Attaching and committing to the surface
func AttachAndDisplayBuffer(shmObj *shm.SHM, surface *client.Surface, renderBuffer *core.Buffer, width, height int) error {
	fd, err := shm.CreateMemfd("wain-demo-buffer")
	if err != nil {
		return fmt.Errorf("create memfd: %w", err)
	}

	bufferSize := int32(width * height * 4)
	if err := syscall.Ftruncate(fd, int64(bufferSize)); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("truncate memfd: %w", err)
	}

	pool, err := shmObj.CreatePool(fd, bufferSize)
	if err != nil {
		syscall.Close(fd)
		return fmt.Errorf("create shm pool: %w", err)
	}
	defer pool.Destroy()

	if err := pool.Map(); err != nil {
		return fmt.Errorf("map pool: %w", err)
	}

	buffer, err := pool.CreateBuffer(0, int32(width), int32(height), int32(width*4), shm.FormatARGB8888)
	if err != nil {
		return fmt.Errorf("create buffer: %w", err)
	}

	copy(buffer.Pixels(), renderBuffer.Pixels)

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
