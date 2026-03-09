package consumer

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

func TestNewSoftwareConsumer(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}
	sc := NewSoftwareConsumer(atlas)

	if sc == nil {
		t.Fatal("Expected non-nil consumer")
	}
	if sc.atlas != atlas {
		t.Error("Atlas not set correctly")
	}
}

func TestRenderNilInputs(t *testing.T) {
	sc := NewSoftwareConsumer(nil)

	tests := []struct {
		name string
		dl   *displaylist.DisplayList
		buf  *primitives.Buffer
	}{
		{"nil displaylist", nil, &primitives.Buffer{}},
		{"nil buffer", displaylist.New(), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.Render(tt.dl, tt.buf)
			if err == nil {
				t.Error("Expected error for nil input")
			}
		})
	}
}

func TestRenderFillRect(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl.AddFillRect(10, 10, 50, 30, color)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify pixel was written (simple smoke test)
	pixel := buf.At(20, 20)
	if pixel.R != 255 || pixel.A != 255 {
		t.Errorf("Expected red pixel at (20,20), got R=%d A=%d", pixel.R, pixel.A)
	}
}

func TestRenderFillRoundedRect(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color := primitives.Color{R: 0, G: 255, B: 0, A: 255}
	dl.AddFillRoundedRect(10, 10, 50, 30, 5, color)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify pixel in center was written
	pixel := buf.At(35, 25)
	if pixel.G == 0 || pixel.A == 0 {
		t.Errorf("Expected green pixel at center, got G=%d A=%d", pixel.G, pixel.A)
	}
}

func TestRenderDrawLine(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color := primitives.Color{R: 0, G: 0, B: 255, A: 255}
	dl.AddDrawLine(10, 10, 90, 90, 2, color)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify pixel along line was written
	pixel := buf.At(50, 50)
	if pixel.B == 0 || pixel.A == 0 {
		t.Errorf("Expected blue pixel on line, got B=%d A=%d", pixel.B, pixel.A)
	}
}

func TestRenderLinearGradient(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color0 := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	color1 := primitives.Color{R: 0, G: 0, B: 255, A: 255}
	dl.AddLinearGradient(10, 10, 80, 80, 10, 10, 90, 90, color0, color1)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify gradient was rendered
	pixel := buf.At(50, 50)
	if pixel.A == 0 {
		t.Error("Expected non-transparent pixel in gradient")
	}
}

func TestRenderRadialGradient(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color0 := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	color1 := primitives.Color{R: 0, G: 0, B: 0, A: 255}
	dl.AddRadialGradient(10, 10, 80, 80, 50, 50, 40, color0, color1)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify gradient was rendered
	pixel := buf.At(50, 50)
	if pixel.A == 0 {
		t.Error("Expected non-transparent pixel in gradient")
	}
}

func TestRenderBoxShadow(t *testing.T) {
	buf, err := primitives.NewBuffer(200, 200)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	color := primitives.Color{R: 0, G: 0, B: 0, A: 128}
	dl.AddBoxShadow(50, 50, 100, 100, 10, 0, color)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Box shadow renders outside the rect - verify pixels exist
	// (actual visual verification would require more complex testing)
}

func TestRenderDrawText(t *testing.T) {
	buf, err := primitives.NewBuffer(200, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}
	dl := displaylist.New()
	color := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	dl.AddDrawText("Test", 10, 50, 16, color, 0)

	sc := NewSoftwareConsumer(atlas)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Text rendering requires atlas to be populated
	// This is a smoke test - actual rendering tested in text package
}

func TestRenderMultipleCommands(t *testing.T) {
	buf, err := primitives.NewBuffer(200, 200)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	redColor := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	greenColor := primitives.Color{R: 0, G: 255, B: 0, A: 255}
	blueColor := primitives.Color{R: 0, G: 0, B: 255, A: 255}

	// Add multiple different commands
	dl.AddFillRect(10, 10, 50, 50, redColor)
	dl.AddFillRoundedRect(70, 10, 50, 50, 5, greenColor)
	dl.AddDrawLine(10, 70, 120, 70, 2, blueColor)

	sc := NewSoftwareConsumer(nil)
	if err := sc.Render(dl, buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify each command rendered
	red := buf.At(35, 35)
	if red.R != 255 || red.A != 255 {
		t.Errorf("Expected red rect, got R=%d A=%d", red.R, red.A)
	}

	green := buf.At(95, 35)
	if green.G == 0 || green.A == 0 {
		t.Errorf("Expected green rounded rect, got G=%d A=%d", green.G, green.A)
	}

	blue := buf.At(65, 70)
	if blue.B == 0 || blue.A == 0 {
		t.Errorf("Expected blue line, got B=%d A=%d", blue.B, blue.A)
	}
}

func TestRenderEmptyDisplayList(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	sc := NewSoftwareConsumer(nil)

	if err := sc.Render(dl, buf); err != nil {
		t.Errorf("Render failed on empty display list: %v", err)
	}
}
