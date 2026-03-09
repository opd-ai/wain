package wain

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
)

// mockRenderer is a test double for the backend.Renderer interface.
type mockRenderer struct {
	renderCalls           int
	renderWithDamageCalls int
	presentCalls          int
	destroyCalls          int
	lastDisplayList       *displaylist.DisplayList
	lastDamage            []displaylist.Rect
	renderError           error
	presentError          error
	presentFd             int
}

func (m *mockRenderer) Render(dl *displaylist.DisplayList) error {
	m.renderCalls++
	m.lastDisplayList = dl
	return m.renderError
}

func (m *mockRenderer) RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error {
	m.renderWithDamageCalls++
	m.lastDisplayList = dl
	m.lastDamage = damage
	return m.renderError
}

func (m *mockRenderer) Present() (int, error) {
	m.presentCalls++
	return m.presentFd, m.presentError
}

func (m *mockRenderer) Dimensions() (width, height int) {
	return 800, 600
}

func (m *mockRenderer) Destroy() error {
	m.destroyCalls++
	return nil
}

// mockWidget is a test widget that implements the Widget interface.
type mockWidget struct {
	BaseWidget
	children        []Widget
	emitError       error
	emitCalls       int
	emittedCommands int
}

func (mw *mockWidget) Children() []Widget {
	return mw.children
}

func (mw *mockWidget) EmitDisplayList(dl *displaylist.DisplayList) error {
	mw.emitCalls++
	if mw.emitError != nil {
		return mw.emitError
	}

	for i := 0; i < mw.emittedCommands; i++ {
		dl.AddFillRect(10+i*10, 10, 50, 30, primitives.Color{R: 255, G: 0, B: 0, A: 255})
	}
	return nil
}

func TestNewRenderBridge(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	if bridge == nil {
		t.Fatal("NewRenderBridge returned nil")
	}

	if bridge.renderer != renderer {
		t.Error("RenderBridge renderer not set correctly")
	}

	if bridge.displayList == nil {
		t.Error("RenderBridge displayList not initialized")
	}

	if !bridge.fullRedraw {
		t.Error("RenderBridge should start with fullRedraw=true")
	}
}

func TestRenderBridge_MarkDirty(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	bridge.dirty = false
	bridge.fullRedraw = false

	bridge.MarkDirty()

	if !bridge.dirty {
		t.Error("MarkDirty should set dirty=true")
	}

	if !bridge.fullRedraw {
		t.Error("MarkDirty should set fullRedraw=true")
	}
}

func TestRenderBridge_MarkRegionDirty(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	bridge.dirty = false
	bridge.fullRedraw = false

	bridge.MarkRegionDirty(10, 20, 100, 80)

	if !bridge.dirty {
		t.Error("MarkRegionDirty should set dirty=true")
	}

	if bridge.fullRedraw {
		t.Error("MarkRegionDirty should not set fullRedraw=true")
	}

	if bridge.dirtyRegion.X != 10 || bridge.dirtyRegion.Y != 20 ||
		bridge.dirtyRegion.Width != 100 || bridge.dirtyRegion.Height != 80 {
		t.Errorf("MarkRegionDirty set incorrect region: %+v", bridge.dirtyRegion)
	}
}

func TestRenderBridge_MarkRegionDirty_Union(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	bridge.dirty = false
	bridge.fullRedraw = false

	bridge.MarkRegionDirty(10, 10, 50, 50)
	bridge.MarkRegionDirty(40, 40, 50, 50)

	expected := displaylist.Rect{X: 10, Y: 10, Width: 80, Height: 80}
	if bridge.dirtyRegion != expected {
		t.Errorf("MarkRegionDirty union failed: got %+v, want %+v",
			bridge.dirtyRegion, expected)
	}
}

func TestRenderBridge_Render_NoRenderer(t *testing.T) {
	bridge := NewRenderBridge(nil)
	widget := &mockWidget{}

	err := bridge.Render(widget)
	if err != ErrNoRenderer {
		t.Errorf("Render with nil renderer: got error %v, want %v", err, ErrNoRenderer)
	}
}

func TestRenderBridge_Render_NoRootWidget(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	err := bridge.Render(nil)
	if err != ErrNoRootWidget {
		t.Errorf("Render with nil widget: got error %v, want %v", err, ErrNoRootWidget)
	}
}

func TestRenderBridge_Render_NotDirty(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)
	bridge.dirty = false

	widget := &mockWidget{emittedCommands: 1}

	err := bridge.Render(widget)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if renderer.renderCalls != 0 {
		t.Error("Render should not call renderer when not dirty")
	}

	if widget.emitCalls != 0 {
		t.Error("Render should not walk widgets when not dirty")
	}
}

func TestRenderBridge_Render_FullRedraw(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)
	bridge.MarkDirty()

	widget := &mockWidget{emittedCommands: 2}

	err := bridge.Render(widget)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if renderer.renderCalls != 1 {
		t.Errorf("Render calls: got %d, want 1", renderer.renderCalls)
	}

	if renderer.renderWithDamageCalls != 0 {
		t.Error("Full redraw should not call RenderWithDamage")
	}

	if widget.emitCalls != 1 {
		t.Errorf("Widget emit calls: got %d, want 1", widget.emitCalls)
	}

	if !bridge.dirty && bridge.fullRedraw {
		t.Error("Render should clear dirty and fullRedraw flags")
	}
}

func TestRenderBridge_Render_RegionalDamage(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	bridge.fullRedraw = false
	bridge.MarkRegionDirty(10, 10, 50, 50)

	widget := &mockWidget{emittedCommands: 1}

	err := bridge.Render(widget)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if renderer.renderWithDamageCalls != 1 {
		t.Errorf("RenderWithDamage calls: got %d, want 1", renderer.renderWithDamageCalls)
	}

	if renderer.renderCalls != 0 {
		t.Error("Regional damage should not call full Render")
	}

	if len(renderer.lastDamage) != 1 {
		t.Fatalf("Damage rects: got %d, want 1", len(renderer.lastDamage))
	}

	expectedRect := displaylist.Rect{X: 10, Y: 10, Width: 50, Height: 50}
	if renderer.lastDamage[0] != expectedRect {
		t.Errorf("Damage rect: got %+v, want %+v",
			renderer.lastDamage[0], expectedRect)
	}
}

func TestRenderBridge_Render_WalkChildren(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)
	bridge.MarkDirty()

	child1 := &mockWidget{emittedCommands: 1}
	child2 := &mockWidget{emittedCommands: 2}
	root := &mockWidget{
		children:        []Widget{child1, child2},
		emittedCommands: 1,
	}

	err := bridge.Render(root)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if root.emitCalls != 1 {
		t.Errorf("Root emit calls: got %d, want 1", root.emitCalls)
	}

	if child1.emitCalls != 1 {
		t.Errorf("Child1 emit calls: got %d, want 1", child1.emitCalls)
	}

	if child2.emitCalls != 1 {
		t.Errorf("Child2 emit calls: got %d, want 1", child2.emitCalls)
	}
}

func TestRenderBridge_Present(t *testing.T) {
	renderer := &mockRenderer{presentFd: 42}
	bridge := NewRenderBridge(renderer)

	fd, err := bridge.Present()
	if err != nil {
		t.Fatalf("Present failed: %v", err)
	}

	if fd != 42 {
		t.Errorf("Present fd: got %d, want 42", fd)
	}

	if renderer.presentCalls != 1 {
		t.Errorf("Present calls: got %d, want 1", renderer.presentCalls)
	}
}

func TestRenderBridge_Present_NoRenderer(t *testing.T) {
	bridge := NewRenderBridge(nil)

	fd, err := bridge.Present()
	if err != ErrNoRenderer {
		t.Errorf("Present with nil renderer: got error %v, want %v", err, ErrNoRenderer)
	}

	if fd != -1 {
		t.Errorf("Present fd: got %d, want -1", fd)
	}
}

func TestRenderBridge_Destroy(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	err := bridge.Destroy()
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	if renderer.destroyCalls != 1 {
		t.Errorf("Destroy calls: got %d, want 1", renderer.destroyCalls)
	}
}

func TestRenderBridge_ThreadSafety(t *testing.T) {
	renderer := &mockRenderer{}
	bridge := NewRenderBridge(renderer)

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			bridge.MarkDirty()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			bridge.MarkRegionDirty(i, i, 10, 10)
		}
		done <- true
	}()

	<-done
	<-done
}

func TestUnionRects(t *testing.T) {
	tests := []struct {
		name string
		a    displaylist.Rect
		b    displaylist.Rect
		want displaylist.Rect
	}{
		{
			name: "overlapping",
			a:    displaylist.Rect{X: 10, Y: 10, Width: 50, Height: 50},
			b:    displaylist.Rect{X: 40, Y: 40, Width: 50, Height: 50},
			want: displaylist.Rect{X: 10, Y: 10, Width: 80, Height: 80},
		},
		{
			name: "separate",
			a:    displaylist.Rect{X: 0, Y: 0, Width: 10, Height: 10},
			b:    displaylist.Rect{X: 20, Y: 20, Width: 10, Height: 10},
			want: displaylist.Rect{X: 0, Y: 0, Width: 30, Height: 30},
		},
		{
			name: "contained",
			a:    displaylist.Rect{X: 0, Y: 0, Width: 100, Height: 100},
			b:    displaylist.Rect{X: 25, Y: 25, Width: 50, Height: 50},
			want: displaylist.Rect{X: 0, Y: 0, Width: 100, Height: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unionRects(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("unionRects(%+v, %+v) = %+v, want %+v",
					tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// Verify that mockRenderer implements backend.Renderer
var _ backend.Renderer = (*mockRenderer)(nil)

// Verify that mockWidget implements Widget
var _ Widget = (*mockWidget)(nil)

// Verify that mockWidget implements DisplayListEmitter
var _ DisplayListEmitter = (*mockWidget)(nil)
