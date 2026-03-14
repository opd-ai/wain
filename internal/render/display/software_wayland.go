package display

import (
	"context"
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
)

// SoftwareWaylandPresenter presents software-rendered frames to a Wayland compositor via wl_shm.
// It maintains a single SHM pool and buffer, re-allocating when dimensions change.
type SoftwareWaylandPresenter struct {
	shmGlobal *shm.SHM
	surface   *client.Surface
	renderer  *backend.SoftwareBackend

	pool   *shm.Pool
	buf    *shm.Buffer
	poolFd int
	width  int
	height int
}

// NewSoftwareWaylandPresenter creates a presenter that writes pixels into a wl_shm
// pool and commits each frame to the given surface.
func NewSoftwareWaylandPresenter(shmGlobal *shm.SHM, surface *client.Surface, renderer *backend.SoftwareBackend) *SoftwareWaylandPresenter {
	return &SoftwareWaylandPresenter{
		shmGlobal: shmGlobal,
		surface:   surface,
		renderer:  renderer,
		poolFd:    -1,
	}
}

// Present copies the software-rendered pixels into an SHM buffer and commits it to the
// Wayland surface. It is safe to call every frame; the pool is recreated only when
// dimensions change.
func (p *SoftwareWaylandPresenter) Present(_ context.Context) error {
	pixels := p.renderer.Pixels()
	if pixels == nil {
		return nil
	}

	width, height := p.renderer.Dimensions()
	if width <= 0 || height <= 0 {
		return nil
	}

	if err := p.ensureBuffer(width, height); err != nil {
		return fmt.Errorf("display/shm: ensure buffer: %w", err)
	}

	copy(p.buf.Pixels(), pixels)

	if err := p.surface.Attach(p.buf.ID(), 0, 0); err != nil {
		return fmt.Errorf("display/shm: attach: %w", err)
	}
	if err := p.surface.Damage(0, 0, int32(width), int32(height)); err != nil {
		return fmt.Errorf("display/shm: damage: %w", err)
	}
	if err := p.surface.Commit(); err != nil {
		return fmt.Errorf("display/shm: commit: %w", err)
	}

	return nil
}

// Close releases the SHM pool and file descriptor.
func (p *SoftwareWaylandPresenter) Close() error {
	if p.pool != nil {
		if err := p.pool.Destroy(); err != nil {
			return fmt.Errorf("display/shm: destroy pool: %w", err)
		}
		p.pool = nil
		p.buf = nil
	}
	if p.poolFd >= 0 {
		_ = syscall.Close(p.poolFd)
		p.poolFd = -1
	}
	return nil
}

// ensureBuffer creates or recreates the SHM pool+buffer when dimensions change.
func (p *SoftwareWaylandPresenter) ensureBuffer(width, height int) error {
	if p.buf != nil && p.width == width && p.height == height {
		return nil
	}

	// Destroy the old pool before re-creating.
	if p.pool != nil {
		_ = p.pool.Destroy()
		p.pool = nil
		p.buf = nil
	}
	if p.poolFd >= 0 {
		_ = syscall.Close(p.poolFd)
		p.poolFd = -1
	}

	stride := width * 4
	size := stride * height

	fd, err := shm.CreateMemfd("wain-shm")
	if err != nil {
		return fmt.Errorf("memfd_create: %w", err)
	}
	if err := syscall.Ftruncate(fd, int64(size)); err != nil {
		_ = syscall.Close(fd)
		return fmt.Errorf("ftruncate: %w", err)
	}
	p.poolFd = fd

	pool, err := p.shmGlobal.CreatePool(fd, int32(size))
	if err != nil {
		return fmt.Errorf("create_pool: %w", err)
	}
	if err := pool.Map(); err != nil {
		_ = pool.Destroy()
		return fmt.Errorf("mmap pool: %w", err)
	}
	p.pool = pool

	buf, err := pool.CreateBuffer(0, int32(width), int32(height), int32(stride), shm.FormatARGB8888)
	if err != nil {
		_ = pool.Destroy()
		return fmt.Errorf("create_buffer: %w", err)
	}

	p.buf = buf
	p.width = width
	p.height = height
	return nil
}
