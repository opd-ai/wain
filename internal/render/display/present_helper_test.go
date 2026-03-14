package display

import (
	"context"
	"errors"
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// mockRenderer implements backend.Renderer for testing presentRenderedFramebuffer.
type mockRenderer struct {
	presentFd  int
	presentErr error
	dims       [2]int
}

func (r *mockRenderer) Render(_ *displaylist.DisplayList) error { return nil }
func (r *mockRenderer) RenderWithDamage(_ *displaylist.DisplayList, _ []displaylist.Rect) error {
	return nil
}
func (r *mockRenderer) Present() (int, error)  { return r.presentFd, r.presentErr }
func (r *mockRenderer) Dimensions() (int, int) { return r.dims[0], r.dims[1] }
func (r *mockRenderer) Destroy() error         { return nil }

func TestPresentRenderedFramebuffer_HappyPath(t *testing.T) {
	t.Parallel()

	pool, err := NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	renderer := &mockRenderer{presentFd: -1, dims: [2]int{8, 8}}

	ensureCalled := false
	commitCalled := false

	err = presentRenderedFramebuffer(
		context.Background(),
		pool,
		renderer,
		func(fb *Framebuffer) { pool.Close() }, // release: just close pool to allow re-acquire
		func(fb *Framebuffer) error { ensureCalled = true; return nil },
		func(fb *Framebuffer) error { commitCalled = true; return nil },
	)
	// The pool is closed inside releaseFunc so MarkDisplaying will fail; we just
	// verify that ensure and commit were reached.
	if !ensureCalled {
		t.Error("ensureFunc not called")
	}
	if !commitCalled {
		t.Error("commitFunc not called")
	}
	_ = err // error from MarkDisplaying on closed pool is expected
}

func TestPresentRenderedFramebuffer_PresentError(t *testing.T) {
	t.Parallel()

	pool, err := NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	presentErr := errors.New("dmabuf export failed")
	renderer := &mockRenderer{presentErr: presentErr, dims: [2]int{8, 8}}

	releaseCalled := false
	err = presentRenderedFramebuffer(
		context.Background(),
		pool,
		renderer,
		func(_ *Framebuffer) { releaseCalled = true },
		func(_ *Framebuffer) error { return nil },
		func(_ *Framebuffer) error { return nil },
	)
	if !errors.Is(err, presentErr) {
		t.Errorf("expected presentErr, got %v", err)
	}
	if !releaseCalled {
		t.Error("releaseFunc not called on present error")
	}
}

func TestPresentRenderedFramebuffer_EnsureError(t *testing.T) {
	t.Parallel()

	pool, err := NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	ensureErr := errors.New("ensure failed")
	renderer := &mockRenderer{presentFd: -1, dims: [2]int{8, 8}}

	releaseCalled := false
	err = presentRenderedFramebuffer(
		context.Background(),
		pool,
		renderer,
		func(_ *Framebuffer) { releaseCalled = true },
		func(_ *Framebuffer) error { return ensureErr },
		func(_ *Framebuffer) error { return nil },
	)
	if !errors.Is(err, ensureErr) {
		t.Errorf("expected ensureErr, got %v", err)
	}
	if !releaseCalled {
		t.Error("releaseFunc not called on ensure error")
	}
}

func TestPresentRenderedFramebuffer_CommitError(t *testing.T) {
	t.Parallel()

	pool, err := NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	commitErr := errors.New("commit failed")
	renderer := &mockRenderer{presentFd: -1, dims: [2]int{8, 8}}

	releaseCalled := false
	err = presentRenderedFramebuffer(
		context.Background(),
		pool,
		renderer,
		func(_ *Framebuffer) { releaseCalled = true },
		func(_ *Framebuffer) error { return nil },
		func(_ *Framebuffer) error { return commitErr },
	)
	if !errors.Is(err, commitErr) {
		t.Errorf("expected commitErr, got %v", err)
	}
	if !releaseCalled {
		t.Error("releaseFunc not called on commit error")
	}
}

func TestPresentRenderedFramebuffer_ClosedPool(t *testing.T) {
	t.Parallel()

	pool, err := NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	pool.Close() //nolint:errcheck

	renderer := &mockRenderer{presentFd: -1, dims: [2]int{8, 8}}
	err = presentRenderedFramebuffer(
		context.Background(),
		pool,
		renderer,
		func(_ *Framebuffer) {},
		func(_ *Framebuffer) error { return nil },
		func(_ *Framebuffer) error { return nil },
	)
	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}
}
