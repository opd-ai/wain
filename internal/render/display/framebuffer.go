package display

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
)

var (
	// ErrPoolClosed is returned when the framebuffer pool is closed.
	ErrPoolClosed = errors.New("display: framebuffer pool is closed")

	// ErrNoAvailableFramebuffer is returned when all framebuffers are in use.
	ErrNoAvailableFramebuffer = errors.New("display: no available framebuffer")

	// ErrFramebufferNotFound is returned when a framebuffer ID is not found.
	ErrFramebufferNotFound = errors.New("display: framebuffer not found")
)

// FramebufferState represents the lifecycle state of a GPU framebuffer.
type FramebufferState int

const (
	// FramebufferAvailable indicates the framebuffer is ready for writing.
	FramebufferAvailable FramebufferState = iota
	// FramebufferRendering indicates GPU rendering is in progress.
	FramebufferRendering
	// FramebufferDisplaying indicates the compositor is displaying the framebuffer.
	FramebufferDisplaying
)

// String returns the string representation of the framebuffer state.
func (s FramebufferState) String() string {
	switch s {
	case FramebufferAvailable:
		return "available"
	case FramebufferRendering:
		return "rendering"
	case FramebufferDisplaying:
		return "displaying"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// Framebuffer represents a GPU framebuffer with compositor-specific metadata.
type Framebuffer struct {
	// Index is the framebuffer's position in the pool (0-based).
	Index int

	// Fd is the DMA-BUF file descriptor. -1 if not yet exported.
	Fd int

	// BufferID is the compositor buffer ID (wl_buffer for Wayland, Pixmap XID for X11).
	BufferID uint32

	// Width, Height, Stride are the buffer dimensions.
	Width  uint32
	Height uint32
	Stride uint32

	// state tracks the current lifecycle state.
	state FramebufferState

	// releasedAt tracks when the compositor released the buffer.
	releasedAt time.Time

	// releaseChan signals when the framebuffer transitions to available state.
	// Buffered to prevent blocking the event handler.
	releaseChan chan struct{}

	// mu protects state transitions.
	mu sync.Mutex
}

// State returns the current state of the framebuffer.
func (f *Framebuffer) State() FramebufferState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

// setState transitions the framebuffer to a new state.
func (f *Framebuffer) setState(newState FramebufferState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = newState
	if newState == FramebufferAvailable {
		f.releasedAt = time.Now()
	}
}

// WaitRelease blocks until the framebuffer is released or the context is canceled.
func (f *Framebuffer) WaitRelease(ctx context.Context) error {
	select {
	case <-f.releaseChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// signalRelease sends a non-blocking release notification.
func (f *Framebuffer) signalRelease() {
	select {
	case f.releaseChan <- struct{}{}:
	default:
	}
}

// FramebufferPool manages a ring of GPU framebuffers for triple buffering.
type FramebufferPool struct {
	framebuffers []*Framebuffer
	idToIndex    map[uint32]int // bufferID → index mapping
	closed       bool
	mu           sync.Mutex
}

// NewFramebufferPool creates a new framebuffer pool with the specified size.
// Typical size is 3 for triple buffering.
func NewFramebufferPool(size int) (*FramebufferPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("display: invalid pool size %d", size)
	}

	framebuffers := make([]*Framebuffer, size)
	for i := 0; i < size; i++ {
		framebuffers[i] = &Framebuffer{
			Index:       i,
			Fd:          -1,
			state:       FramebufferAvailable,
			releaseChan: make(chan struct{}, 1),
		}
		// Signal initially available
		framebuffers[i].signalRelease()
	}

	return &FramebufferPool{
		framebuffers: framebuffers,
		idToIndex:    make(map[uint32]int),
	}, nil
}

// Acquire blocks until a framebuffer is available for rendering.
// Returns ErrPoolClosed if the pool is closed.
func (p *FramebufferPool) Acquire(ctx context.Context) (*Framebuffer, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrPoolClosed
		}

		// Find the oldest released framebuffer
		var oldest *Framebuffer
		var oldestTime time.Time
		for _, fb := range p.framebuffers {
			if fb.State() == FramebufferAvailable {
				if oldest == nil || fb.releasedAt.Before(oldestTime) {
					oldest = fb
					oldestTime = fb.releasedAt
				}
			}
		}

		if oldest != nil {
			oldest.setState(FramebufferRendering)
			p.mu.Unlock()
			return oldest, nil
		}
		p.mu.Unlock()

		// Wait for any framebuffer to be released
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Retry
		}
	}
}

// MarkDisplaying marks a framebuffer as being displayed by the compositor.
func (p *FramebufferPool) MarkDisplaying(fb *Framebuffer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	fb.setState(FramebufferDisplaying)
	return nil
}

// OnRelease is called when the compositor releases a buffer.
// This marks the buffer as available for reuse.
func (p *FramebufferPool) OnRelease(bufferID uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	index, ok := p.idToIndex[bufferID]
	if !ok {
		return ErrFramebufferNotFound
	}

	fb := p.framebuffers[index]
	fb.setState(FramebufferAvailable)
	fb.signalRelease()
	return nil
}

// Register associates a compositor buffer ID with a framebuffer.
func (p *FramebufferPool) Register(fb *Framebuffer, bufferID uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fb.BufferID = bufferID
	p.idToIndex[bufferID] = fb.Index
}

// Close closes the pool and releases all framebuffer file descriptors.
func (p *FramebufferPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	for _, fb := range p.framebuffers {
		if fb.Fd >= 0 {
			syscall.Close(fb.Fd)
			fb.Fd = -1
		}
	}

	p.closed = true
	return nil
}

// Size returns the number of framebuffers in the pool.
func (p *FramebufferPool) Size() int {
	return len(p.framebuffers)
}
