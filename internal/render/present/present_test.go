package present_test

import (
	"context"
	"errors"
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render/present"
)

// mockFramebuffer is an opaque handle used by mock implementations.
type mockFramebuffer struct{ id int }

// mockPool records Acquire and MarkDisplaying calls for verification.
type mockPool struct {
	fb            *mockFramebuffer
	acquireErr    error
	markErr       error
	acquireCalled bool
	markCalled    bool
}

func (p *mockPool) Acquire(_ context.Context) (present.FramebufferHandle, error) {
	p.acquireCalled = true
	if p.acquireErr != nil {
		return nil, p.acquireErr
	}
	return p.fb, nil
}

func (p *mockPool) MarkDisplaying(_ present.FramebufferHandle) error {
	p.markCalled = true
	return p.markErr
}

// mockPresenter records all method calls and can inject errors.
type mockPresenter struct {
	closed        bool
	renderErr     error
	ensureErr     error
	presentErr    error
	renderCalled  bool
	ensureCalled  bool
	presentCalled bool
	releaseCalled bool
}

func (p *mockPresenter) RenderToFramebuffer(_ *displaylist.DisplayList, _ present.FramebufferHandle) error {
	p.renderCalled = true
	return p.renderErr
}

func (p *mockPresenter) EnsurePlatformBuffer(_ present.FramebufferHandle) error {
	p.ensureCalled = true
	return p.ensureErr
}

func (p *mockPresenter) PresentBuffer(_ present.FramebufferHandle) error {
	p.presentCalled = true
	return p.presentErr
}

func (p *mockPresenter) ReleaseFramebuffer(_ present.FramebufferHandle) {
	p.releaseCalled = true
}

func (p *mockPresenter) IsClosed() bool { return p.closed }

func TestRenderAndPresent_HappyPath(t *testing.T) {
	t.Parallel()

	fb := &mockFramebuffer{id: 1}
	pool := &mockPool{fb: fb}
	pres := &mockPresenter{}
	dl := displaylist.New()

	err := present.RenderAndPresent(context.Background(), dl, pool, pres)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !pool.acquireCalled {
		t.Error("expected Acquire to be called")
	}
	if !pres.renderCalled {
		t.Error("expected RenderToFramebuffer to be called")
	}
	if !pres.ensureCalled {
		t.Error("expected EnsurePlatformBuffer to be called")
	}
	if !pres.presentCalled {
		t.Error("expected PresentBuffer to be called")
	}
	if !pool.markCalled {
		t.Error("expected MarkDisplaying to be called")
	}
	if pres.releaseCalled {
		t.Error("expected ReleaseFramebuffer NOT to be called on success")
	}
}

func TestRenderAndPresent_PresenterClosed(t *testing.T) {
	t.Parallel()

	pool := &mockPool{fb: &mockFramebuffer{}}
	pres := &mockPresenter{closed: true}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if !errors.Is(err, present.ErrPresenterClosed) {
		t.Fatalf("expected ErrPresenterClosed, got %v", err)
	}
	if pool.acquireCalled {
		t.Error("Acquire should not be called when presenter is closed")
	}
}

func TestRenderAndPresent_AcquireError(t *testing.T) {
	t.Parallel()

	acquireErr := errors.New("pool exhausted")
	pool := &mockPool{acquireErr: acquireErr}
	pres := &mockPresenter{}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if err == nil {
		t.Fatal("expected error when Acquire fails")
	}
	if !errors.Is(err, acquireErr) {
		t.Errorf("expected wrapped acquire error, got %v", err)
	}
	if pres.renderCalled {
		t.Error("RenderToFramebuffer should not be called after Acquire error")
	}
}

func TestRenderAndPresent_RenderError(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("render failed")
	pool := &mockPool{fb: &mockFramebuffer{}}
	pres := &mockPresenter{renderErr: renderErr}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if err == nil {
		t.Fatal("expected error when render fails")
	}
	if !errors.Is(err, renderErr) {
		t.Errorf("expected wrapped render error, got %v", err)
	}
	if !pres.releaseCalled {
		t.Error("expected ReleaseFramebuffer to be called on render error")
	}
	if pres.presentCalled {
		t.Error("PresentBuffer should not be called after render error")
	}
}

func TestRenderAndPresent_EnsureError(t *testing.T) {
	t.Parallel()

	ensureErr := errors.New("ensure failed")
	pool := &mockPool{fb: &mockFramebuffer{}}
	pres := &mockPresenter{ensureErr: ensureErr}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if err == nil {
		t.Fatal("expected error when EnsurePlatformBuffer fails")
	}
	if !errors.Is(err, ensureErr) {
		t.Errorf("expected wrapped ensure error, got %v", err)
	}
	if !pres.releaseCalled {
		t.Error("expected ReleaseFramebuffer to be called on ensure error")
	}
	if pres.presentCalled {
		t.Error("PresentBuffer should not be called after ensure error")
	}
}

func TestRenderAndPresent_PresentError(t *testing.T) {
	t.Parallel()

	presentErr := errors.New("present failed")
	pool := &mockPool{fb: &mockFramebuffer{}}
	pres := &mockPresenter{presentErr: presentErr}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if err == nil {
		t.Fatal("expected error when PresentBuffer fails")
	}
	if !errors.Is(err, presentErr) {
		t.Errorf("expected wrapped present error, got %v", err)
	}
	if !pres.releaseCalled {
		t.Error("expected ReleaseFramebuffer to be called on present error")
	}
	if pool.markCalled {
		t.Error("MarkDisplaying should not be called after present error")
	}
}

func TestRenderAndPresent_MarkDisplayingError(t *testing.T) {
	t.Parallel()

	markErr := errors.New("mark failed")
	pool := &mockPool{fb: &mockFramebuffer{}, markErr: markErr}
	pres := &mockPresenter{}

	err := present.RenderAndPresent(context.Background(), displaylist.New(), pool, pres)
	if err == nil {
		t.Fatal("expected error when MarkDisplaying fails")
	}
	if !errors.Is(err, markErr) {
		t.Errorf("expected wrapped mark error, got %v", err)
	}
	if !pres.releaseCalled {
		t.Error("expected ReleaseFramebuffer to be called on mark error")
	}
}
