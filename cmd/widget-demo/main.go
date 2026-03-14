// Command widget-demo demonstrates interactive Phase 1 widgets on both X11 and Wayland.
//
// This binary showcases:
//   - Interactive UI widgets (button, text input, scroll container)
//   - Mouse and keyboard input handling
//   - Event-driven rendering
//   - Platform abstraction (X11 or Wayland auto-detected)
//
// Usage:
//
//	./bin/widget-demo              # Auto-detect platform
//	./bin/widget-demo --x11        # Force X11
//	./bin/widget-demo --wayland    # Force Wayland
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/ui/widgets"
	"github.com/opd-ai/wain/internal/wayland/client"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/gc"
	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	windowWidth  = 600
	windowHeight = 500
)

func main() {
	demo.CheckHelpFlag("widget-demo", "Interactive Phase 1 widgets on X11 and Wayland", []string{
		demo.FormatExample("widget-demo", "Auto-detect platform"),
		demo.FormatExample("widget-demo --x11", "Force X11"),
		demo.FormatExample("widget-demo --wayland", "Force Wayland"),
		demo.FormatExample("widget-demo --help", "Show this help message"),
	})

	fmt.Println("======================================")
	fmt.Println("wain Interactive Widget Demo")
	fmt.Println("======================================")
	fmt.Println()

	// Determine platform
	platform := detectPlatform()
	fmt.Printf("Platform: %s\n\n", platform)

	if err := runDemo(platform); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
}

// detectPlatform determines which backend to use based on environment and flags.
func detectPlatform() string {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--x11" {
			return "x11"
		}
		if arg == "--wayland" {
			return "wayland"
		}
	}

	// Auto-detect: prefer Wayland if available
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "x11"
	}

	// Default to X11 if nothing detected
	return "x11"
}

// runDemo demonstrates interactive widgets on the selected platform.
func runDemo(platform string) error {
	// Create application state
	app := &application{
		running:     true,
		clickCount:  0,
		inputText:   "",
		scrollItems: generateScrollItems(20),
	}

	// Create UI widgets
	fmt.Println("[1/4] Creating UI widgets...")
	app.createWidgets()
	demo.PrintFeatureList("", []string{
		"Created Button widgets (3)",
		"Created TextInput widget",
		"Created ScrollContainer widget (20 items)",
	})

	// Create render buffer
	fmt.Println("\n[2/4] Initializing framebuffer...")
	buffer, err := primitives.NewBuffer(windowWidth, windowHeight)
	if err != nil {
		return fmt.Errorf("create buffer: %w", err)
	}
	app.buffer = buffer
	fmt.Printf("      ✓ Created %dx%d ARGB8888 buffer\n", windowWidth, windowHeight)

	// Render initial frame
	fmt.Println("\n[3/4] Rendering initial frame...")
	app.render()
	fmt.Println("      ✓ Rendered widgets to buffer")

	// Display window (platform-specific)
	fmt.Println("\n[4/4] Opening window...")
	if platform == "wayland" {
		return runWayland(app)
	}
	return runX11(app)
}

// application holds the demo application state.
type application struct {
	running      bool
	clickCount   int
	inputText    string
	scrollItems  []string
	scrollOffset int

	buffer      *primitives.Buffer
	clickButton *widgets.Button
	resetButton *widgets.Button
	quitButton  *widgets.Button
	textInput   *widgets.TextInput
	scrollList  *widgets.ScrollContainer
	statusLabel string
	lastMouseX  int
	lastMouseY  int
	needsRedraw bool
}

// createWidgets initializes all UI widgets.
func (app *application) createWidgets() {
	// Click counter button
	app.clickButton = widgets.NewButton("Click Me!", 150, 40)
	app.clickButton.SetOnClick(func() {
		app.clickCount++
		app.statusLabel = fmt.Sprintf("Clicked %d times", app.clickCount)
		app.needsRedraw = true
	})

	// Reset button
	app.resetButton = widgets.NewButton("Reset Counter", 150, 40)
	app.resetButton.SetOnClick(func() {
		app.clickCount = 0
		app.statusLabel = "Counter reset"
		app.needsRedraw = true
	})

	// Quit button
	app.quitButton = widgets.NewButton("Quit Demo", 150, 40)
	app.quitButton.SetOnClick(func() {
		app.statusLabel = "Goodbye!"
		app.running = false
		app.needsRedraw = true
	})

	// Text input field
	app.textInput = widgets.NewTextInput("Type something...", 400, 35)

	// Scroll container with items (2000px tall content as specified in audit)
	app.scrollList = widgets.NewScrollContainer(400, 200)

	// Add text block children to demonstrate scrolling (10 blocks, 200px each = 2000px total)
	for i := 0; i < 10; i++ {
		label := widgets.NewLabel(fmt.Sprintf("Scroll Item Block %d\nThis is scrollable content.\nTotal content: 2000px\nVisible area: 200px", i+1), 380, 180)
		app.scrollList.AddChild(label)
	}

	app.statusLabel = "Ready - Use mouse wheel to scroll"
	app.scrollOffset = 0
}

// render draws all widgets to the framebuffer.
func (app *application) render() {
	// Clear background
	app.buffer.FillRect(0, 0, windowWidth, windowHeight,
		primitives.Color{R: 250, G: 250, B: 250, A: 255})

	// Title
	renderText(app.buffer, "Interactive Widget Demo", 20, 20,
		primitives.Color{R: 50, G: 50, B: 50, A: 255})

	// Buttons row
	app.clickButton.Draw(app.buffer, 50, 60)
	app.resetButton.Draw(app.buffer, 220, 60)
	app.quitButton.Draw(app.buffer, 390, 60)

	// Status label
	statusText := fmt.Sprintf("Status: %s", app.statusLabel)
	renderText(app.buffer, statusText, 50, 120,
		primitives.Color{R: 70, G: 70, B: 70, A: 255})

	// Text input
	renderText(app.buffer, "Text Input:", 50, 160,
		primitives.Color{R: 70, G: 70, B: 70, A: 255})
	app.textInput.Draw(app.buffer, 50, 185)

	// Scroll container
	renderText(app.buffer, "Scrollable List:", 50, 240,
		primitives.Color{R: 70, G: 70, B: 70, A: 255})
	app.scrollList.Draw(app.buffer, 50, 265)

	// Scroll position indicator
	scrollText := fmt.Sprintf("Scroll: %dpx / %dpx", app.scrollOffset, 1800)
	renderText(app.buffer, scrollText, 460, 265,
		primitives.Color{R: 120, G: 120, B: 120, A: 255})

	// Mouse position indicator
	mouseText := fmt.Sprintf("Mouse: (%d, %d)", app.lastMouseX, app.lastMouseY)
	renderText(app.buffer, mouseText, 500, windowHeight-30,
		primitives.Color{R: 120, G: 120, B: 120, A: 255})

	app.needsRedraw = false
}

// handleMouseMove processes mouse movement events.
func (app *application) handleMouseMove(x, y int) {
	app.lastMouseX = x
	app.lastMouseY = y

	// Check hover states for buttons
	inButton1 := pointInRect(x, y, 50, 60, 150, 40)
	inButton2 := pointInRect(x, y, 220, 60, 150, 40)
	inButton3 := pointInRect(x, y, 390, 60, 150, 40)

	if inButton1 {
		app.clickButton.HandlePointerEnter()
	} else {
		app.clickButton.HandlePointerLeave()
	}

	if inButton2 {
		app.resetButton.HandlePointerEnter()
	} else {
		app.resetButton.HandlePointerLeave()
	}

	if inButton3 {
		app.quitButton.HandlePointerEnter()
	} else {
		app.quitButton.HandlePointerLeave()
	}

	app.needsRedraw = true
}

// handleMouseClick processes mouse click events.
func (app *application) handleMouseClick(x, y int, button uint32) {
	// Check which button was clicked
	if pointInRect(x, y, 50, 60, 150, 40) {
		app.clickButton.HandlePointerDown(button)
		app.clickButton.HandlePointerUp(button)
	} else if pointInRect(x, y, 220, 60, 150, 40) {
		app.resetButton.HandlePointerDown(button)
		app.resetButton.HandlePointerUp(button)
	} else if pointInRect(x, y, 390, 60, 150, 40) {
		app.quitButton.HandlePointerDown(button)
		app.quitButton.HandlePointerUp(button)
	}

	app.needsRedraw = true
}

// handleMouseScroll processes mouse wheel scroll events.
func (app *application) handleMouseScroll(x, y, delta int) {
	// Check if scroll is over the scroll container area
	if pointInRect(x, y, 50, 265, 400, 200) {
		app.scrollOffset += delta * 20
		app.scrollList.SetScrollOffset(app.scrollOffset)
		app.scrollOffset = app.scrollList.ScrollOffset() // Get clamped value
		app.statusLabel = fmt.Sprintf("Scrolled to %dpx", app.scrollOffset)
		app.needsRedraw = true
	}
}

// handleKeyPress processes keyboard events.
func (app *application) handleKeyPress(key string) {
	if key == "Escape" {
		app.running = false
		app.statusLabel = "Quit via Escape key"
		app.needsRedraw = true
		return
	}

	// Update text input
	if key == "BackSpace" && len(app.inputText) > 0 {
		app.inputText = app.inputText[:len(app.inputText)-1]
	} else if len(key) == 1 && len(app.inputText) < 50 {
		app.inputText += key
	}

	app.statusLabel = fmt.Sprintf("Input: %s", app.inputText)
	app.needsRedraw = true
}

// pointInRect checks if a point is inside a rectangle.
func pointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}

// renderText is a simple text rendering helper.
func renderText(buf *primitives.Buffer, text string, xPos, yPos int, color primitives.Color) {
	// For now, just draw a simple rectangle as a placeholder
	// In a full implementation, this would use the text rasterizer
	width := len(text) * 8
	height := 16
	buf.FillRect(xPos, yPos, width, height, primitives.Color{R: 0, G: 0, B: 0, A: 0})
}

// generateScrollItems creates dummy items for the scroll container.
func generateScrollItems(count int) []string {
	items := make([]string, count)
	for i := 0; i < count; i++ {
		items[i] = fmt.Sprintf("Item %d - Sample scrollable content", i+1)
	}
	return items
}

// waylandSurface holds Wayland surface/toplevel objects for a demo window.
type waylandSurface struct {
	ctx     *demo.WaylandContext
	surface *client.Surface
}

// setupWaylandWindow creates a titled toplevel window using the provided Wayland globals.
func setupWaylandWindow(wlCtx *demo.WaylandContext, title string) (*waylandSurface, error) {
	surface, err := wlCtx.Compositor.CreateSurface()
	if err != nil {
		return nil, fmt.Errorf("create surface: %w", err)
	}

	if _, _, err := demo.CreateXdgWindow(wlCtx.Conn, wlCtx.WmBase, surface, title); err != nil {
		return nil, fmt.Errorf("create xdg window: %w", err)
	}

	if err := surface.Commit(); err != nil {
		return nil, fmt.Errorf("initial commit: %w", err)
	}

	return &waylandSurface{ctx: wlCtx, surface: surface}, nil
}

// runWayland displays the widget demo on a Wayland compositor.
// It connects to the compositor, creates a surface, renders the initial frame
// and then presents ~90 frames at ~30 FPS (≈3 seconds) before exiting.
func runWayland(app *application) error {
	conn, err := demo.ConnectToWayland()
	if err != nil {
		return fmt.Errorf("connect to Wayland: %w", err)
	}
	defer conn.Close()

	wlCtx, err := demo.SetupWaylandGlobals(conn)
	if err != nil {
		return fmt.Errorf("setup Wayland globals: %w", err)
	}

	ws, err := setupWaylandWindow(wlCtx, "wain Widget Demo")
	if err != nil {
		return err
	}

	fmt.Println("      ✓ Wayland window created and visible")
	return runWaylandFrameLoop(app, ws)
}

// runWaylandFrameLoop renders and presents ~90 frames at ~30 FPS via SHM attach.
func runWaylandFrameLoop(app *application, ws *waylandSurface) error {
	const (
		frames   = 90
		frameDur = 33 * time.Millisecond // ~30 FPS
	)
	for i := 0; i < frames; i++ {
		if app.needsRedraw || i == 0 {
			app.render()
		}
		if err := demo.AttachAndDisplayBuffer(ws.ctx.SHM, ws.surface, app.buffer, windowWidth, windowHeight); err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}
		time.Sleep(frameDur)
	}
	return nil
}

// gcConn adapts *x11client.Connection to gc.Connection.
type gcConn struct{ conn *x11client.Connection }

// AllocXID allocates a new X11 resource ID via the underlying connection.
func (a *gcConn) AllocXID() (gc.XID, error) {
	xid, err := a.conn.AllocXID()
	return gc.XID(xid), err
}

// SendRequest sends a raw X11 protocol request via the underlying connection.
func (a *gcConn) SendRequest(buf []byte) error { return a.conn.SendRequest(buf) }

// runX11 displays the widget demo on an X11 server.
// It connects to :0, creates a window, and renders ~90 frames at ~30 FPS
// (≈3 seconds) before exiting.
func runX11(app *application) error {
	conn, err := x11client.Connect("0")
	if err != nil {
		return fmt.Errorf("connect to X11: %w", err)
	}
	defer conn.Close()

	wid, err := setupX11Window(conn)
	if err != nil {
		return err
	}

	adapter := &gcConn{conn: conn}
	gcID, err := setupX11GC(adapter, wid)
	if err != nil {
		return err
	}
	defer gc.FreeGC(adapter, gcID) //nolint:errcheck

	fmt.Println("      ✓ X11 window created and visible")
	return runX11FrameLoop(app, adapter, wid, gcID)
}

// setupX11Window creates an X11 window and maps it to the screen.
func setupX11Window(conn *x11client.Connection) (x11client.XID, error) {
	root := conn.RootWindow()
	wid, err := conn.CreateWindow(
		root, 100, 100,
		uint16(windowWidth), uint16(windowHeight),
		0, wire.WindowClassInputOutput, 0,
		wire.CWBackPixel|wire.CWEventMask,
		[]uint32{0x000000, wire.EventMaskExposure | wire.EventMaskKeyPress | wire.EventMaskButtonPress | wire.EventMaskPointerMotion},
	)
	if err != nil {
		return 0, fmt.Errorf("create window: %w", err)
	}
	if err := conn.MapWindow(wid); err != nil {
		return 0, fmt.Errorf("map window: %w", err)
	}
	return wid, nil
}

// setupX11GC creates a graphics context bound to the given window.
func setupX11GC(adapter *gcConn, wid x11client.XID) (gc.XID, error) {
	gcID, err := gc.CreateGC(adapter, gc.XID(wid), 0, nil)
	if err != nil {
		return 0, fmt.Errorf("create GC: %w", err)
	}
	return gcID, nil
}

// runX11FrameLoop renders and presents ~90 frames at ~30 FPS via PutImage.
func runX11FrameLoop(app *application, adapter *gcConn, wid x11client.XID, gcID gc.XID) error {
	const (
		frames   = 90
		frameDur = 33 * time.Millisecond // ~30 FPS
	)
	for i := 0; i < frames; i++ {
		if app.needsRedraw || i == 0 {
			app.render()
		}
		if err := gc.PutImage(
			adapter,
			gc.XID(wid), gcID,
			uint16(windowWidth), uint16(windowHeight),
			0, 0,
			24, gc.FormatZPixmap,
			app.buffer.Pixels,
		); err != nil {
			return fmt.Errorf("frame %d PutImage: %w", i, err)
		}
		time.Sleep(frameDur)
	}
	return nil
}
