package wain

import (
	"errors"
	"sync"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render/backend"
)

var (
	// ErrNoRenderer is returned when rendering is attempted without a renderer.
	ErrNoRenderer = errors.New("wain: no renderer available")

	// ErrNoRootWidget is returned when rendering is attempted without a root widget.
	ErrNoRootWidget = errors.New("wain: no root widget set")
)

// RenderBridge connects the widget tree to the rendering pipeline.
// It manages the frame lifecycle:
//  1. Walk the widget tree, collect dirty widgets
//  2. Emit DisplayList commands for dirty regions
//  3. Submit to renderer with damage rects
//  4. Present to compositor
type RenderBridge struct {
	mu sync.Mutex

	renderer    backend.Renderer
	displayList *displaylist.DisplayList
	damageRects []displaylist.Rect

	// Track dirty state
	dirty       bool
	fullRedraw  bool
	dirtyRegion displaylist.Rect
}

// NewRenderBridge creates a new render bridge with the specified renderer.
func NewRenderBridge(renderer backend.Renderer) *RenderBridge {
	return &RenderBridge{
		renderer:    renderer,
		displayList: displaylist.New(),
		fullRedraw:  true,
	}
}

// MarkDirty marks the entire rendering surface as needing a redraw.
func (rb *RenderBridge) MarkDirty() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.dirty = true
	rb.fullRedraw = true
}

// MarkRegionDirty marks a specific region as needing a redraw.
func (rb *RenderBridge) MarkRegionDirty(x, y, width, height int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.dirty = true

	if rb.fullRedraw {
		return
	}

	newRect := displaylist.Rect{X: x, Y: y, Width: width, Height: height}

	if rb.dirtyRegion.Width == 0 && rb.dirtyRegion.Height == 0 {
		rb.dirtyRegion = newRect
	} else {
		rb.dirtyRegion = unionRects(rb.dirtyRegion, newRect)
	}
}

// Render performs a rendering pass if the bridge is marked dirty.
// Returns true if a frame was rendered, false if no rendering was needed.
func (rb *RenderBridge) Render(rootWidget Widget) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.renderer == nil {
		return ErrNoRenderer
	}

	if rootWidget == nil {
		return ErrNoRootWidget
	}

	if !rb.dirty {
		return nil
	}

	rb.displayList.Reset()

	if err := rb.walkWidget(rootWidget); err != nil {
		return err
	}

	var err error
	if rb.fullRedraw {
		err = rb.renderer.Render(rb.displayList)
	} else {
		damage := []displaylist.Rect{rb.dirtyRegion}
		err = rb.renderer.RenderWithDamage(rb.displayList, damage)
	}

	if err != nil {
		return err
	}

	rb.dirty = false
	rb.fullRedraw = false
	rb.dirtyRegion = displaylist.Rect{}

	return nil
}

// Present presents the rendered frame to the display.
// Returns a file descriptor (DMA-BUF for GPU, or -1 for software).
func (rb *RenderBridge) Present() (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.renderer == nil {
		return -1, ErrNoRenderer
	}

	return rb.renderer.Present()
}

// Destroy frees all resources.
func (rb *RenderBridge) Destroy() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.renderer != nil {
		return rb.renderer.Destroy()
	}
	return nil
}

// walkWidget recursively walks the widget tree and emits display list commands.
func (rb *RenderBridge) walkWidget(w Widget) error {
	if w == nil {
		return nil
	}

	if dw, ok := w.(DisplayListEmitter); ok {
		if err := dw.EmitDisplayList(rb.displayList); err != nil {
			return err
		}
	}

	for _, child := range w.Children() {
		if err := rb.walkWidget(child); err != nil {
			return err
		}
	}

	return nil
}

// DisplayListEmitter is an optional interface that widgets can implement
// to emit display list commands for GPU rendering.
type DisplayListEmitter interface {
	EmitDisplayList(dl *displaylist.DisplayList) error
}

// unionRects returns the bounding box that contains both rectangles.
func unionRects(a, b displaylist.Rect) displaylist.Rect {
	x1 := min(a.X, b.X)
	y1 := min(a.Y, b.Y)
	x2 := max(a.X+a.Width, b.X+b.Width)
	y2 := max(a.Y+a.Height, b.Y+b.Height)

	return displaylist.Rect{
		X:      x1,
		Y:      y1,
		Width:  x2 - x1,
		Height: y2 - y1,
	}
}
