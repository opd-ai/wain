package demo

import (
	"fmt"

	"github.com/opd-ai/wain/internal/raster/primitives"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/wire"
)

// RunX11Demo executes the standard Phase 1 X11 demo with the given window dimensions.
// It connects to X11, creates a window, renders demo content, and prints feature summaries.
func RunX11Demo(width, height int) error {
	conn, err := connectX11()
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = createWindow(conn, width, height)
	if err != nil {
		return err
	}

	if err := setupEventLoop(width, height); err != nil {
		return err
	}

	return nil
}

// connectX11 establishes a connection to the X11 server.
func connectX11() (*x11client.Connection, error) {
	fmt.Println("[1/6] Connecting to X11 server...")
	conn, err := x11client.Connect("0")
	if err != nil {
		return nil, fmt.Errorf("connect to X11: %w", err)
	}
	fmt.Println("      ✓ Connected to :0")
	return conn, nil
}

// createWindow creates and maps an X11 window with the specified dimensions.
func createWindow(conn *x11client.Connection, width, height int) (x11client.XID, error) {
	fmt.Println("\n[2/6] Creating window...")
	root := conn.RootWindow()

	const (
		windowX     = 100
		windowY     = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress | wire.EventMaskButtonPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}

	wid, err := conn.CreateWindow(root, windowX, windowY, uint16(width), uint16(height), borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		return 0, fmt.Errorf("create window: %w", err)
	}
	fmt.Printf("      ✓ Created window XID %d\n", wid)

	fmt.Println("\n[3/6] Mapping window to display...")
	if err := conn.MapWindow(wid); err != nil {
		return 0, fmt.Errorf("map window: %w", err)
	}
	fmt.Println("      ✓ Window visible on screen")

	return wid, nil
}

// setupEventLoop creates widgets, renders content, and displays feature summaries.
func setupEventLoop(width, height int) error {
	fmt.Println("\n[4/6] Creating UI widgets...")
	btn, input := StandardWidgets()
	fmt.Println("      ✓ Created Button widget (120x40)")
	fmt.Println("      ✓ Created TextInput widget (200x30)")

	fmt.Println("\n[5/6] Rendering content to framebuffer...")
	renderBuffer, err := CreateDemoBuffer(width, height)
	if err != nil {
		return err
	}
	RenderDemoContent(renderBuffer, btn, input)
	fmt.Printf("      ✓ Rendered to %dx%d ARGB8888 buffer\n", width, height)

	fmt.Println("\n[6/6] Phase 1 Features Demonstrated:")
	PrintFeatureList("      PROTOCOL LAYER (X11)\n      -------------------", []string{
		"Connection setup and authentication",
		"Window creation (CreateWindow)",
		"Window mapping (MapWindow)",
		"Resource allocation (AllocXID)",
	})
	fmt.Println()
	PrintRenderingFeatures()
	fmt.Println()
	PrintUIFeatures()
	fmt.Println()

	PrintBufferStats(width, height, renderBuffer)

	return nil
}

// ConnectAndSetupX11Window creates an X11 connection and window, returning both.
// This is a lower-level helper for demos that need more control over the window lifecycle.
func ConnectAndSetupX11Window(width, height int) (*x11client.Connection, x11client.XID, error) {
	conn, err := x11client.Connect("0")
	if err != nil {
		return nil, 0, fmt.Errorf("connect to X11: %w", err)
	}

	root := conn.RootWindow()
	const (
		windowX     = 100
		windowY     = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0 // CopyFromParent
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress | wire.EventMaskButtonPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}

	wid, err := conn.CreateWindow(root, windowX, windowY, uint16(width), uint16(height), borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		conn.Close()
		return nil, 0, fmt.Errorf("create window: %w", err)
	}

	if err := conn.MapWindow(wid); err != nil {
		conn.Close()
		return nil, 0, fmt.Errorf("map window: %w", err)
	}

	return conn, wid, nil
}

// RenderAndDisplayDemo creates widgets, renders them to a buffer, and prints summaries.
// Returns the render buffer for further use.
func RenderAndDisplayDemo(width, height int) (*primitives.Buffer, error) {
	btn, input := StandardWidgets()
	renderBuffer, err := CreateDemoBuffer(width, height)
	if err != nil {
		return nil, err
	}
	RenderDemoContent(renderBuffer, btn, input)
	return renderBuffer, nil
}
