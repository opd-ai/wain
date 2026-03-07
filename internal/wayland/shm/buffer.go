package shm

import (
	"context"
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Buffer represents a wl_buffer object for drawable content.
type Buffer struct {
	objectBase
	pool        *Pool
	offset      int32
	width       int32
	height      int32
	stride      int32
	format      uint32
	pixels      []byte
	releaseChan chan struct{}
}

// NewBuffer creates a new Buffer object.
func NewBuffer(conn Conn, id uint32, pool *Pool, offset, width, height, stride int32, format uint32, pixels []byte) *Buffer {
	return &Buffer{
		objectBase: objectBase{
			id:    id,
			iface: "wl_buffer",
			conn:  conn,
		},
		pool:        pool,
		offset:      offset,
		width:       width,
		height:      height,
		stride:      stride,
		format:      format,
		pixels:      pixels,
		releaseChan: make(chan struct{}, 1),
	}
}

// Pixels returns the pixel data for this buffer.
// Returns nil if the pool is not currently mapped.
func (b *Buffer) Pixels() []byte {
	return b.pixels
}

// Width returns the buffer width in pixels.
func (b *Buffer) Width() int32 {
	return b.width
}

// Height returns the buffer height in pixels.
func (b *Buffer) Height() int32 {
	return b.height
}

// Stride returns the buffer stride in bytes.
func (b *Buffer) Stride() int32 {
	return b.stride
}

// Format returns the pixel format.
func (b *Buffer) Format() uint32 {
	return b.format
}

// WaitRelease waits for the compositor to release this buffer.
// The buffer should not be modified until it is released.
func (b *Buffer) WaitRelease(ctx context.Context) error {
	select {
	case <-b.releaseChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Destroy destroys the buffer and releases its client-side resources.
// Note: This does not free the underlying shared memory in the pool.
func (b *Buffer) Destroy() error {
	if err := b.conn.SendRequest(b.id, 0, nil); err != nil {
		return fmt.Errorf("destroy request failed: %w", err)
	}

	// Remove from pool's buffer registry.
	if b.pool != nil {
		delete(b.pool.buffers, b.id)
	}

	return nil
}

// HandleEvent processes events from the compositor for this buffer.
func (b *Buffer) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // release event
		select {
		case b.releaseChan <- struct{}{}:
		default:
			// Channel already has a release signal, don't block.
		}
		return nil
	default:
		return fmt.Errorf("unknown wl_buffer event opcode: %d", opcode)
	}
}
