package displaylist

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

func TestNew(t *testing.T) {
	dl := New()
	if dl == nil {
		t.Fatal("New() returned nil")
	}
	if dl.Len() != 0 {
		t.Errorf("Expected new display list to have 0 commands, got %d", dl.Len())
	}
}

func TestAddFillRect(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 255, G: 0, B: 0, A: 255}

	dl.AddFillRect(10, 20, 100, 50, color)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdFillRect {
		t.Errorf("Expected CmdFillRect, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(FillRectData)
	if !ok {
		t.Fatalf("Expected FillRectData, got %T", cmd.Data)
	}

	if data.X != 10 || data.Y != 20 || data.Width != 100 || data.Height != 50 {
		t.Errorf("Expected rect (10,20,100,50), got (%d,%d,%d,%d)",
			data.X, data.Y, data.Width, data.Height)
	}

	if data.Color != color {
		t.Errorf("Expected color %v, got %v", color, data.Color)
	}
}

func TestAddFillRoundedRect(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 0, G: 255, B: 0, A: 255}

	dl.AddFillRoundedRect(5, 10, 200, 100, 15, color)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdFillRoundedRect {
		t.Errorf("Expected CmdFillRoundedRect, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(FillRoundedRectData)
	if !ok {
		t.Fatalf("Expected FillRoundedRectData, got %T", cmd.Data)
	}

	if data.Radius != 15 {
		t.Errorf("Expected radius 15, got %d", data.Radius)
	}
}

func TestAddDrawLine(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 0, G: 0, B: 255, A: 255}

	dl.AddDrawLine(0, 0, 100, 100, 2, color)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdDrawLine {
		t.Errorf("Expected CmdDrawLine, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(DrawLineData)
	if !ok {
		t.Fatalf("Expected DrawLineData, got %T", cmd.Data)
	}

	if data.X0 != 0 || data.Y0 != 0 || data.X1 != 100 || data.Y1 != 100 {
		t.Errorf("Expected line (0,0)-(100,100), got (%d,%d)-(%d,%d)",
			data.X0, data.Y0, data.X1, data.Y1)
	}

	if data.Width != 2 {
		t.Errorf("Expected width 2, got %d", data.Width)
	}
}

func TestAddDrawText(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 0, G: 0, B: 0, A: 255}

	dl.AddDrawText("Hello, World!", 10, 20, 16, color, 0)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdDrawText {
		t.Errorf("Expected CmdDrawText, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(DrawTextData)
	if !ok {
		t.Fatalf("Expected DrawTextData, got %T", cmd.Data)
	}

	if data.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", data.Text)
	}

	if data.FontSize != 16 {
		t.Errorf("Expected font size 16, got %d", data.FontSize)
	}
}

func TestAddLinearGradient(t *testing.T) {
	dl := New()
	color0 := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	color1 := primitives.Color{R: 0, G: 0, B: 255, A: 255}

	dl.AddLinearGradient(0, 0, 200, 100, 0, 0, 200, 0, color0, color1)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdLinearGradient {
		t.Errorf("Expected CmdLinearGradient, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(LinearGradientData)
	if !ok {
		t.Fatalf("Expected LinearGradientData, got %T", cmd.Data)
	}

	if data.Color0 != color0 || data.Color1 != color1 {
		t.Errorf("Gradient colors don't match")
	}
}

func TestAddRadialGradient(t *testing.T) {
	dl := New()
	color0 := primitives.Color{R: 255, G: 255, B: 0, A: 255}
	color1 := primitives.Color{R: 255, G: 0, B: 255, A: 255}

	dl.AddRadialGradient(0, 0, 200, 200, 100, 100, 80, color0, color1)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdRadialGradient {
		t.Errorf("Expected CmdRadialGradient, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(RadialGradientData)
	if !ok {
		t.Fatalf("Expected RadialGradientData, got %T", cmd.Data)
	}

	if data.Radius != 80 {
		t.Errorf("Expected radius 80, got %d", data.Radius)
	}
}

func TestAddBoxShadow(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 0, G: 0, B: 0, A: 128}

	dl.AddBoxShadow(10, 10, 100, 50, 5, 2, color)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdBoxShadow {
		t.Errorf("Expected CmdBoxShadow, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(BoxShadowData)
	if !ok {
		t.Fatalf("Expected BoxShadowData, got %T", cmd.Data)
	}

	if data.BlurRadius != 5 || data.SpreadRadius != 2 {
		t.Errorf("Expected blur radius 5, spread radius 2, got %d, %d",
			data.BlurRadius, data.SpreadRadius)
	}
}

func TestAddDrawImage(t *testing.T) {
	dl := New()

	dl.AddDrawImage(10, 20, 100, 80, 1, 0.0, 0.0, 1.0, 1.0)

	if dl.Len() != 1 {
		t.Fatalf("Expected 1 command, got %d", dl.Len())
	}

	cmd := dl.Commands()[0]
	if cmd.Type != CmdDrawImage {
		t.Errorf("Expected CmdDrawImage, got %v", cmd.Type)
	}

	data, ok := cmd.Data.(DrawImageData)
	if !ok {
		t.Fatalf("Expected DrawImageData, got %T", cmd.Data)
	}

	if data.TextureID != 1 {
		t.Errorf("Expected texture ID 1, got %d", data.TextureID)
	}

	if data.U0 != 0.0 || data.V0 != 0.0 || data.U1 != 1.0 || data.V1 != 1.0 {
		t.Errorf("Expected UV coords (0,0)-(1,1), got (%f,%f)-(%f,%f)",
			data.U0, data.V0, data.U1, data.V1)
	}
}

func TestMultipleCommands(t *testing.T) {
	dl := New()
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	green := primitives.Color{R: 0, G: 255, B: 0, A: 255}
	blue := primitives.Color{R: 0, G: 0, B: 255, A: 255}

	dl.AddFillRect(0, 0, 100, 100, red)
	dl.AddFillRect(50, 50, 100, 100, green)
	dl.AddDrawLine(0, 0, 150, 150, 1, blue)

	if dl.Len() != 3 {
		t.Fatalf("Expected 3 commands, got %d", dl.Len())
	}

	cmds := dl.Commands()
	if cmds[0].Type != CmdFillRect || cmds[1].Type != CmdFillRect || cmds[2].Type != CmdDrawLine {
		t.Errorf("Command types don't match expected sequence")
	}
}

func TestReset(t *testing.T) {
	dl := New()
	color := primitives.Color{R: 255, G: 255, B: 255, A: 255}

	dl.AddFillRect(0, 0, 100, 100, color)
	dl.AddFillRect(10, 10, 50, 50, color)

	if dl.Len() != 2 {
		t.Fatalf("Expected 2 commands before reset, got %d", dl.Len())
	}

	dl.Reset()

	if dl.Len() != 0 {
		t.Errorf("Expected 0 commands after reset, got %d", dl.Len())
	}

	// Verify we can reuse after reset
	dl.AddFillRect(5, 5, 20, 20, color)
	if dl.Len() != 1 {
		t.Errorf("Expected 1 command after adding to reset list, got %d", dl.Len())
	}
}
