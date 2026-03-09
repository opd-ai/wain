package consumer

import (
	"os"
	"syscall"
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
)

// mockGPURenderer is a test implementation of GPURenderer.
type mockGPURenderer struct {
	width, height int
	renderCalled  bool
	presentCalled bool
}

func (m *mockGPURenderer) Render(dl *displaylist.DisplayList) error {
	m.renderCalled = true
	return nil
}

func (m *mockGPURenderer) RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error {
	m.renderCalled = true
	return nil
}

func (m *mockGPURenderer) Present() (int, error) {
	m.presentCalled = true
	// Return a dummy fd (we can't create a real DMA-BUF without GPU)
	return -1, nil
}

func (m *mockGPURenderer) Dimensions() (width, height int) {
	return m.width, m.height
}

func (m *mockGPURenderer) Destroy() error {
	return nil
}

// TestGPUConsumerCreation verifies GPU consumer creation and cleanup.
func TestGPUConsumerCreation(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}

	if consumer == nil {
		t.Fatal("NewGPUConsumer returned nil consumer")
	}

	if err := consumer.Destroy(); err != nil {
		t.Errorf("Destroy failed: %v", err)
	}
}

// TestGPUConsumerNilRenderer verifies error handling for nil renderer.
func TestGPUConsumerNilRenderer(t *testing.T) {
	_, err := NewGPUConsumer(nil)
	if err == nil {
		t.Error("NewGPUConsumer with nil renderer should return error")
	}
}

// TestGPUConsumerRender verifies basic display list rendering.
func TestGPUConsumerRender(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	// Create a simple display list with a filled rectangle
	dl := displaylist.New()
	dl.AddFillRect(100, 100, 200, 150, primitives.Color{R: 255, G: 0, B: 0, A: 255})

	if err := consumer.Render(dl, nil); err != nil {
		t.Errorf("Render failed: %v", err)
	}

	if !renderer.renderCalled {
		t.Error("Renderer.Render was not called")
	}
}

// TestGPUConsumerRenderNilDisplayList verifies error handling for nil display list.
func TestGPUConsumerRenderNilDisplayList(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	err = consumer.Render(nil, nil)
	if err == nil {
		t.Error("Render with nil display list should return error")
	}
}

// TestGPUConsumerRenderMultipleCommands tests rendering with multiple draw commands.
func TestGPUConsumerRenderMultipleCommands(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	// Create display list with various command types
	dl := displaylist.New()
	dl.AddFillRect(10, 10, 50, 50, primitives.Color{R: 255, G: 0, B: 0, A: 255})
	dl.AddFillRoundedRect(100, 10, 50, 50, 10, primitives.Color{R: 0, G: 255, B: 0, A: 255})
	dl.AddDrawLine(200, 10, 250, 60, 2, primitives.Color{R: 0, G: 0, B: 255, A: 255})
	dl.AddLinearGradient(10, 100, 100, 50, 10, 100, 110, 150,
		primitives.Color{R: 255, G: 0, B: 0, A: 255},
		primitives.Color{R: 0, G: 0, B: 255, A: 255})

	if err := consumer.Render(dl, nil); err != nil {
		t.Errorf("Render with multiple commands failed: %v", err)
	}
}

// TestGPUConsumerRenderWithDamage tests damage-based incremental rendering.
func TestGPUConsumerRenderWithDamage(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	dl := displaylist.New()
	dl.AddFillRect(50, 50, 100, 100, primitives.Color{R: 255, G: 0, B: 0, A: 255})
	dl.AddFillRect(200, 200, 100, 100, primitives.Color{R: 0, G: 255, B: 0, A: 255})

	// Render only the first rectangle's region
	damage := []displaylist.Rect{
		{X: 50, Y: 50, Width: 100, Height: 100},
	}

	if err := consumer.RenderWithDamage(dl, damage); err != nil {
		t.Errorf("RenderWithDamage failed: %v", err)
	}

	if !renderer.renderCalled {
		t.Error("Renderer.RenderWithDamage was not called")
	}
}

// TestGPUConsumerPresent tests DMA-BUF export for display.
func TestGPUConsumerPresent(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	// Render something first
	dl := displaylist.New()
	dl.AddFillRect(0, 0, 100, 100, primitives.Color{R: 128, G: 128, B: 128, A: 255})
	if err := consumer.Render(dl, nil); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Export as DMA-BUF
	fd, err := consumer.Present()
	if err != nil {
		// Mock returns -1 with nil error
		if fd != -1 {
			t.Errorf("Present returned unexpected fd: %d", fd)
		}
	}

	if !renderer.presentCalled {
		t.Error("Renderer.Present was not called")
	}

	// Only close valid file descriptors
	if fd >= 0 {
		syscall.Close(fd)
	}
}

// TestGPUConsumerDimensions verifies dimension reporting.
func TestGPUConsumerDimensions(t *testing.T) {
	width, height := 1024, 768
	renderer := &mockGPURenderer{width: width, height: height}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	w, h := consumer.Dimensions()
	if w != width || h != height {
		t.Errorf("Dimensions mismatch: got (%d, %d), want (%d, %d)", w, h, width, height)
	}
}

// TestGPUConsumerEmptyDisplayList tests rendering an empty display list.
func TestGPUConsumerEmptyDisplayList(t *testing.T) {
	renderer := &mockGPURenderer{width: 800, height: 600}
	consumer, err := NewGPUConsumer(renderer)
	if err != nil {
		t.Fatalf("NewGPUConsumer failed: %v", err)
	}
	defer consumer.Destroy()

	dl := displaylist.New()
	if err := consumer.Render(dl, nil); err != nil {
		t.Errorf("Render with empty display list failed: %v", err)
	}
}

// gpuAvailable checks if a GPU device is available for testing.
func gpuAvailable() bool {
	_, err := os.Stat("/dev/dri/renderD128")
	return err == nil
}
