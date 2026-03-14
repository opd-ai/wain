package shm

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Pool represents a wl_shm_pool object for shared memory management.
type Pool struct {
	objectBase
	fd      int
	size    int32
	mapping []byte
	buffers map[uint32]*Buffer
}

// NewPool creates a new Pool object.
func NewPool(conn Conn, id uint32, fd int, size int32) *Pool {
	return &Pool{
		objectBase: objectBase{
			id:    id,
			iface: "wl_shm_pool",
			conn:  conn,
		},
		fd:      fd,
		size:    size,
		buffers: make(map[uint32]*Buffer),
	}
}

// Map memory-maps the shared memory pool for read/write access.
func (p *Pool) Map() error {
	if p.mapping != nil {
		return fmt.Errorf("pool already mapped")
	}

	data, err := MmapFile(p.fd, int(p.size))
	if err != nil {
		return fmt.Errorf("wayland/shm: map pool: %w", err)
	}

	p.mapping = data
	return nil
}

// Unmap unmaps the shared memory pool.
func (p *Pool) Unmap() error {
	if p.mapping == nil {
		return nil
	}

	if err := MunmapFile(p.mapping); err != nil {
		return fmt.Errorf("wayland/shm: unmap pool: %w", err)
	}

	p.mapping = nil
	return nil
}

// CreateBuffer creates a wl_buffer from this pool.
func (p *Pool) CreateBuffer(offset, width, height, stride int32, format uint32) (*Buffer, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid buffer dimensions: %dx%d", width, height)
	}

	if stride <= 0 {
		return nil, fmt.Errorf("invalid stride: %d", stride)
	}

	if offset < 0 || offset >= p.size {
		return nil, fmt.Errorf("invalid offset: %d (pool size: %d)", offset, p.size)
	}

	bufferSize := int32(height) * stride
	if offset+bufferSize > p.size {
		return nil, fmt.Errorf("buffer extends beyond pool (offset %d + size %d > pool size %d)", offset, bufferSize, p.size)
	}

	bufferID := p.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: bufferID},
		{Type: wire.ArgTypeInt32, Value: offset},
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
		{Type: wire.ArgTypeInt32, Value: stride},
		{Type: wire.ArgTypeUint32, Value: format},
	}

	if err := p.conn.SendRequest(p.id, 0, args); err != nil {
		return nil, fmt.Errorf("create_buffer request failed: %w", err)
	}

	var pixels []byte
	if p.mapping != nil {
		pixels = p.mapping[offset : offset+bufferSize]
	}

	buffer := NewBuffer(p.conn, bufferID, p, offset, width, height, stride, format, pixels)
	p.conn.RegisterObject(buffer)
	p.buffers[bufferID] = buffer

	return buffer, nil
}

// Resize changes the size of the pool.
func (p *Pool) Resize(size int32) error {
	if size <= 0 {
		return fmt.Errorf("invalid pool size: %d", size)
	}

	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: size},
	}

	if err := p.conn.SendRequest(p.id, 1, args); err != nil {
		return fmt.Errorf("resize request failed: %w", err)
	}

	// Update the fd size.
	if err := syscall.Ftruncate(p.fd, int64(size)); err != nil {
		return fmt.Errorf("ftruncate failed: %w", err)
	}

	return p.remapAfterResize(size)
}

// remapAfterResize updates the memory mapping to reflect the new pool size.
func (p *Pool) remapAfterResize(size int32) error {
	if p.mapping == nil {
		p.size = size
		return nil
	}
	if err := p.Unmap(); err != nil {
		return fmt.Errorf("unmap before resize failed: %w", err)
	}
	p.size = size
	if err := p.Map(); err != nil {
		return fmt.Errorf("remap after resize failed: %w", err)
	}
	return nil
}

// Destroy destroys the pool and releases its resources.
func (p *Pool) Destroy() error {
	// Send destroy request.
	if err := p.conn.SendRequest(p.id, 2, nil); err != nil {
		return fmt.Errorf("destroy request failed: %w", err)
	}

	// Unmap and close fd.
	if err := p.Unmap(); err != nil {
		return fmt.Errorf("wayland/shm: destroy pool: unmap: %w", err)
	}

	if err := syscall.Close(p.fd); err != nil {
		return fmt.Errorf("close fd failed: %w", err)
	}

	return nil
}

// HandleEvent processes events from the compositor for this pool.
// wl_shm_pool has no events, so this always returns an error.
func (p *Pool) HandleEvent(opcode uint16, args []wire.Argument) error {
	return fmt.Errorf("wl_shm_pool has no events (opcode %d)", opcode)
}
