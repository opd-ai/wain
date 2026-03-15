// Command gpu-display-demo demonstrates end-to-end GPU rendering to display.
//
// This binary showcases the complete GPU-to-Display pipeline integration:
//   - GPU backend initialization (Intel or AMD)
//   - Display list rendering to GPU
//   - DMA-BUF export from GPU render target
//   - Wayland/X11 compositor integration via display pipeline
//   - Event-driven rendering loop with proper frame synchronization
//
// This demo proves that GPU-rendered output can reach the screen via
// zero-copy DMA-BUF sharing with the compositor.
//
// Usage:
//
//	./bin/gpu-display-demo           # auto-detect Wayland or X11
//	./bin/gpu-display-demo -wayland  # force Wayland
//	./bin/gpu-display-demo -x11      # force X11
//
// Requirements:
//   - Wayland: compositor with zwp_linux_dmabuf_v1 support
//   - X11: server with DRI3 and Present extensions
//   - Intel GPU (Gen9-Gen12/Xe) or AMD GPU (RDNA1/2/3)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/render/display"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/dmabuf"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	windowWidth  = 800
	windowHeight = 600
	drmPath      = "/dev/dri/renderD128"
	numFrames    = 60 // render 60 frames then exit
)

var (
	forceWayland = flag.Bool("wayland", false, "force Wayland (ignore X11)")
	forceX11     = flag.Bool("x11", false, "force X11 (ignore Wayland)")
	showHelp     = flag.Bool("help", false, "show help message")
)

func main() {
	flag.Parse()

	if *showHelp {
		demo.PrintUsageAndExit("gpu-display-demo", "End-to-end GPU rendering to display", []string{
			demo.FormatExample("gpu-display-demo", "Auto-detect Wayland or X11"),
			demo.FormatExample("gpu-display-demo -wayland", "Force Wayland"),
			demo.FormatExample("gpu-display-demo -x11", "Force X11"),
			demo.FormatExample("gpu-display-demo -help", "Show this help message"),
		})
	}

	fmt.Println("==============================================")
	fmt.Println("wain GPU Display Integration Demo")
	fmt.Println("==============================================")
	fmt.Println()

	if err := runDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
	fmt.Println("GPU-rendered output successfully reached the display compositor!")
}

func runDemo() error {
	// Auto-detect display server
	useWayland, err := shouldUseWayland()
	if err != nil {
		return fmt.Errorf("display detection failed: %w", err)
	}

	if useWayland {
		fmt.Println("Display server: Wayland")
		return runWaylandDemo()
	}
	fmt.Println("Display server: X11")
	return runX11Demo()
}

func shouldUseWayland() (bool, error) {
	if *forceX11 {
		return false, nil
	}
	if *forceWayland {
		return true, nil
	}

	// Check for Wayland display
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true, nil
	}

	// Check for X11 display
	if os.Getenv("DISPLAY") != "" {
		return false, nil
	}

	return false, fmt.Errorf("no WAYLAND_DISPLAY or DISPLAY environment variable set")
}

func runWaylandDemo() error {
	fmt.Println("Initializing Wayland connection...")

	conn, err := demo.ConnectToWayland()
	if err != nil {
		return fmt.Errorf("failed to connect to Wayland: %w", err)
	}
	defer conn.Close()

	dmabufObj, surface, err := setupWaylandContext(conn)
	if err != nil {
		return err
	}

	renderer, err := createGPUBackend()
	if err != nil {
		return err
	}
	defer func() { _ = renderer.Destroy() }()

	pipeline, err := createWaylandPipeline(surface, dmabufObj, renderer)
	if err != nil {
		return err
	}
	defer pipeline.Close()

	return renderFrameLoop(pipeline)
}

func setupWaylandContext(conn *client.Connection) (*dmabuf.Dmabuf, *client.Surface, error) {
	wlCtx, err := demo.SetupWaylandGlobals(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup globals: %w", err)
	}

	dmabufObj, err := bindDmabuf(conn)
	if err != nil {
		return nil, nil, err
	}

	surface, err := createWaylandWindow(wlCtx)
	if err != nil {
		return nil, nil, err
	}

	return dmabufObj, surface, nil
}

func bindDmabuf(conn *client.Connection) (*dmabuf.Dmabuf, error) {
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	return demo.BindDmabuf(conn, registry)
}

func createWaylandWindow(wlCtx *demo.WaylandContext) (*client.Surface, error) {
	surface, err := wlCtx.Compositor.CreateSurface()
	if err != nil {
		return nil, fmt.Errorf("failed to create surface: %w", err)
	}

	if _, _, err := demo.CreateXdgWindow(wlCtx.Conn, wlCtx.WmBase, surface, "wain GPU Display Demo (Wayland)"); err != nil {
		return nil, fmt.Errorf("failed to create xdg window: %w", err)
	}

	if err := surface.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit surface: %w", err)
	}

	return surface, nil
}

func createGPUBackend() (*backend.GPUBackend, error) {
	fmt.Println("Initializing GPU backend...")
	renderer, err := backend.New(backend.Config{
		DRMPath:          drmPath,
		Width:            windowWidth,
		Height:           windowHeight,
		VertexBufferSize: 1024 * 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GPU backend: %w", err)
	}
	return renderer, nil
}

func createWaylandPipeline(surface *client.Surface, dmabufObj *dmabuf.Dmabuf, renderer *backend.GPUBackend) (*display.WaylandPipeline, error) {
	fmt.Println("Creating GPU→Wayland display pipeline...")
	pipeline, err := display.NewWaylandPipeline(surface, dmabufObj, renderer)
	if err != nil {
		return nil, fmt.Errorf("failed to create display pipeline: %w", err)
	}
	return pipeline, nil
}

func renderFrameLoop(pipeline interface {
	RenderAndPresent(context.Context, *displaylist.DisplayList) error
},
) error {
	fmt.Printf("Rendering %d frames...\n", numFrames)
	for i := 0; i < numFrames; i++ {
		dl := createAnimatedDisplayList(i, numFrames)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err := pipeline.RenderAndPresent(ctx, dl); err != nil {
			cancel()
			return fmt.Errorf("frame %d failed: %w", i, err)
		}
		cancel()

		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}
	return nil
}

func runX11Demo() error {
	fmt.Println("Initializing X11 connection...")

	conn, err := x11client.Connect("0")
	if err != nil {
		return fmt.Errorf("failed to connect to X11: %w", err)
	}
	defer conn.Close()

	dri3Ext, presentExt, window, err := setupX11Context(conn)
	if err != nil {
		return err
	}

	renderer, err := createGPUBackend()
	if err != nil {
		return err
	}
	defer func() { _ = renderer.Destroy() }()

	pipeline, err := createX11Pipeline(conn, window, dri3Ext, presentExt, renderer)
	if err != nil {
		return err
	}
	defer pipeline.Close()

	return renderFrameLoop(pipeline)
}

func setupX11Context(conn *x11client.Connection) (*dri3.Extension, *present.Extension, x11client.XID, error) {
	dri3Ext, presentExt, err := queryX11Extensions(conn)
	if err != nil {
		return nil, nil, 0, err
	}

	window, err := createX11Window(conn)
	if err != nil {
		return nil, nil, 0, err
	}

	return dri3Ext, presentExt, window, nil
}

func queryX11Extensions(conn *x11client.Connection) (*dri3.Extension, *present.Extension, error) {
	dri3Ext, err := dri3.QueryExtension(newDRI3Adapter(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("DRI3 extension not available: %w", err)
	}

	presentExt, err := present.QueryExtension(newPresentAdapter(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("present extension not available: %w", err)
	}

	return dri3Ext, presentExt, nil
}

func createX11Window(conn *x11client.Connection) (x11client.XID, error) {
	fmt.Println("Creating window...")
	root := conn.RootWindow()

	const (
		windowX     = 100
		windowY     = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0 // CopyFromParent
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}

	window, err := conn.CreateWindow(root, windowX, windowY, windowWidth, windowHeight, borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		return 0, fmt.Errorf("failed to create window: %w", err)
	}

	if err := conn.MapWindow(window); err != nil {
		return 0, fmt.Errorf("failed to map window: %w", err)
	}

	return window, nil
}

func createX11Pipeline(conn *x11client.Connection, window x11client.XID, dri3Ext *dri3.Extension, presentExt *present.Extension, renderer *backend.GPUBackend) (*display.X11Pipeline, error) {
	fmt.Println("Creating GPU→X11 display pipeline...")
	pipeline, err := display.NewX11Pipeline(conn, window, dri3Ext, presentExt, renderer)
	if err != nil {
		return nil, fmt.Errorf("failed to create display pipeline: %w", err)
	}
	return pipeline, nil
}

// createAnimatedDisplayList creates a simple animated display list for testing.
func createAnimatedDisplayList(frame, totalFrames int) *displaylist.DisplayList {
	dl := displaylist.New()

	// Background: interpolate from blue to purple
	progress := float32(frame) / float32(totalFrames)
	bg := primitives.Color{
		R: uint8(progress * 128),
		G: 0,
		B: 255,
		A: 255,
	}

	// Fill background
	dl.AddFillRect(0, 0, windowWidth, windowHeight, bg)

	// Draw a rectangle that moves across the screen
	x := int(progress * float32(windowWidth-200))
	y := windowHeight/2 - 50

	dl.AddFillRect(x, y, 200, 100, primitives.Color{R: 255, G: 255, B: 255, A: 255})

	return dl
}

// DRI3/Present adapters (similar to gpu-triangle-demo)

type dri3Adapter struct {
	*x11client.Connection
}

func newDRI3Adapter(conn *x11client.Connection) *dri3Adapter {
	return &dri3Adapter{Connection: conn}
}

// AllocXID allocates a new X11 identifier from the connection's XID pool.
func (a *dri3Adapter) AllocXID() (dri3.XID, error) {
	xid, err := a.Connection.AllocXID()
	return dri3.XID(xid), err
}

// SendRequestAndReplyWithFDs sends an X11 request with file descriptors and waits for the reply.
func (a *dri3Adapter) SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error) {
	return a.Connection.SendRequestAndReplyWithFDs(req, fds)
}

// SendRequestWithFDs sends an X11 request with file descriptors without waiting for a reply.
func (a *dri3Adapter) SendRequestWithFDs(req []byte, fds []int) error {
	return a.Connection.SendRequestWithFDs(req, fds)
}

type presentAdapter struct {
	*x11client.Connection
}

func newPresentAdapter(conn *x11client.Connection) *presentAdapter {
	return &presentAdapter{Connection: conn}
}

// AllocXID allocates a new X11 identifier from the connection's XID pool.
func (a *presentAdapter) AllocXID() (present.XID, error) {
	xid, err := a.Connection.AllocXID()
	return present.XID(xid), err
}
