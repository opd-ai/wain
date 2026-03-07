// Command wayland-demo demonstrates Phase 1 features of the wain UI toolkit on Wayland.
//
// This binary showcases:
//   - Wayland protocol client
//   - Software rasterizer (rectangles, rounded rects, lines)
//   - UI widgets (button, text input)
//   - Complete rendering pipeline
//
// Usage:
//
//	./bin/wayland-demo
package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/ui/widgets"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
	"github.com/opd-ai/wain/internal/wayland/xdg"
)

const (
	windowWidth  = 400
	windowHeight = 300
)

func main() {
	fmt.Println("======================================")
	fmt.Println("wain Phase 1 Demo - Wayland Backend")
	fmt.Println("======================================")
	fmt.Println()

	if err := runDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
}

type demoContext struct {
	conn       *client.Connection
	compositor *client.Compositor
	shmObj     *shm.SHM
	wmBase     *xdg.WmBase
}

// runDemo demonstrates the Phase 1 feature stack on Wayland.
func runDemo() error {
	conn, err := connectToCompositor()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, err := discoverGlobals(conn)
	if err != nil {
		return err
	}

	surface, err := createWindow(ctx)
	if err != nil {
		return err
	}

	btn, input := createWidgets()

	renderBuffer, err := renderContent(btn, input)
	if err != nil {
		return err
	}

	if err := displayBuffer(ctx, surface, renderBuffer); err != nil {
		return err
	}

	printFeatureSummary(renderBuffer)
	return nil
}

// connectToCompositor establishes connection to the Wayland compositor.
func connectToCompositor() (*client.Connection, error) {
	fmt.Println("[1/6] Connecting to Wayland compositor...")
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		display = "wayland-0"
	}

	conn, err := client.Connect(display)
	if err != nil {
		return nil, fmt.Errorf("connect to Wayland: %w", err)
	}
	fmt.Printf("      ✓ Connected to %s\n", display)
	return conn, nil
}

// discoverGlobals binds to required Wayland global objects.
func discoverGlobals(conn *client.Connection) (*demoContext, error) {
	fmt.Println("\n[2/6] Discovering compositor globals...")
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		return nil, fmt.Errorf("get registry: %w", err)
	}

	compositorGlobal := registry.FindGlobal("wl_compositor")
	if compositorGlobal == nil {
		return nil, fmt.Errorf("wl_compositor not found")
	}
	compositor, err := registry.BindCompositor(compositorGlobal)
	if err != nil {
		return nil, fmt.Errorf("bind compositor: %w", err)
	}
	fmt.Println("      ✓ Bound to wl_compositor")

	shmGlobal := registry.FindGlobal("wl_shm")
	if shmGlobal == nil {
		return nil, fmt.Errorf("wl_shm not found")
	}
	shmID, err := registry.Bind(shmGlobal.Name, "wl_shm", shmGlobal.Version)
	if err != nil {
		return nil, fmt.Errorf("bind shm: %w", err)
	}
	shmObj := shm.NewSHM(conn, shmID)
	fmt.Println("      ✓ Bound to wl_shm")

	xdgGlobal := registry.FindGlobal("xdg_wm_base")
	if xdgGlobal == nil {
		return nil, fmt.Errorf("xdg_wm_base not found")
	}
	wmBaseID, _, err := registry.BindXdgWmBase(xdgGlobal)
	if err != nil {
		return nil, fmt.Errorf("bind xdg_wm_base: %w", err)
	}
	wmBase := xdg.NewWmBase(conn, wmBaseID, xdgGlobal.Version)
	fmt.Println("      ✓ Bound to xdg_wm_base")

	return &demoContext{
		conn:       conn,
		compositor: compositor,
		shmObj:     shmObj,
		wmBase:     wmBase,
	}, nil
}

// createWindow creates and configures the application window.
func createWindow(ctx *demoContext) (*client.Surface, error) {
	fmt.Println("\n[3/6] Creating surface and window...")
	surface, err := ctx.compositor.CreateSurface()
	if err != nil {
		return nil, fmt.Errorf("create surface: %w", err)
	}
	fmt.Printf("      ✓ Created wl_surface (ID %d)\n", surface.ID())

	xdgSurface, err := ctx.wmBase.GetXdgSurface(surface.ID())
	if err != nil {
		return nil, fmt.Errorf("get xdg_surface: %w", err)
	}
	fmt.Printf("      ✓ Created xdg_surface (ID %d)\n", xdgSurface.ID())

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		return nil, fmt.Errorf("get toplevel: %w", err)
	}
	fmt.Printf("      ✓ Created xdg_toplevel (ID %d)\n", toplevel.ID())

	if err := toplevel.SetTitle("wain Wayland Demo"); err != nil {
		return nil, fmt.Errorf("set title: %w", err)
	}
	fmt.Println("      ✓ Set window title")

	if err := surface.Commit(); err != nil {
		return nil, fmt.Errorf("commit surface: %w", err)
	}
	fmt.Println("      ✓ Window visible on screen")

	return surface, nil
}

// createWidgets instantiates the demo UI widgets.
func createWidgets() (*widgets.Button, *widgets.TextInput) {
	fmt.Println("\n[4/6] Creating UI widgets...")
	btn := widgets.NewButton("Click Me!", 120, 40)
	input := widgets.NewTextInput("Type here...", 200, 30)
	fmt.Println("      ✓ Created Button widget (120x40)")
	fmt.Println("      ✓ Created TextInput widget (200x30)")
	return btn, input
}

// renderContent renders UI content to a buffer using the software rasterizer.
func renderContent(btn *widgets.Button, input *widgets.TextInput) (*core.Buffer, error) {
	fmt.Println("\n[5/6] Rendering content to framebuffer...")
	renderBuffer, err := core.NewBuffer(windowWidth, windowHeight)
	if err != nil {
		return nil, fmt.Errorf("create buffer: %w", err)
	}
	renderDemoContent(renderBuffer, btn, input)
	fmt.Printf("      ✓ Rendered to %dx%d ARGB8888 buffer\n", windowWidth, windowHeight)
	return renderBuffer, nil
}

// displayBuffer copies the render buffer to shared memory and displays it.
func displayBuffer(ctx *demoContext, surface *client.Surface, renderBuffer *core.Buffer) error {
	fd, err := shm.CreateMemfd("wain-wayland-demo")
	if err != nil {
		return fmt.Errorf("create memfd: %w", err)
	}

	bufferSize := int32(windowWidth * windowHeight * 4)
	if err := syscall.Ftruncate(fd, int64(bufferSize)); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("truncate memfd: %w", err)
	}

	pool, err := ctx.shmObj.CreatePool(fd, bufferSize)
	if err != nil {
		syscall.Close(fd)
		return fmt.Errorf("create shm pool: %w", err)
	}
	defer pool.Destroy()

	if err := pool.Map(); err != nil {
		return fmt.Errorf("map pool: %w", err)
	}

	buffer, err := pool.CreateBuffer(0, int32(windowWidth), int32(windowHeight), int32(windowWidth*4), shm.FormatARGB8888)
	if err != nil {
		return fmt.Errorf("create buffer: %w", err)
	}

	copy(buffer.Pixels(), renderBuffer.Pixels)
	fmt.Println("      ✓ Copied to shared memory buffer")

	if err := surface.Attach(buffer.ID(), 0, 0); err != nil {
		return fmt.Errorf("attach buffer: %w", err)
	}

	if err := surface.Damage(0, 0, int32(windowWidth), int32(windowHeight)); err != nil {
		return fmt.Errorf("damage surface: %w", err)
	}

	if err := surface.Commit(); err != nil {
		return fmt.Errorf("commit surface: %w", err)
	}
	fmt.Println("      ✓ Buffer attached and displayed")

	return nil
}

// printFeatureSummary displays a summary of demonstrated Phase 1 features.
func printFeatureSummary(renderBuffer *core.Buffer) {
	fmt.Println("\n[6/6] Phase 1 Features Demonstrated:")
	fmt.Println()
	fmt.Println("      PROTOCOL LAYER (Wayland)")
	fmt.Println("      ------------------------")
	fmt.Println("      • Connection to compositor via Unix socket")
	fmt.Println("      • Global discovery (wl_registry)")
	fmt.Println("      • Surface creation (wl_compositor)")
	fmt.Println("      • Shared memory buffers (wl_shm)")
	fmt.Println("      • Window management (xdg_wm_base)")
	fmt.Println("      • XDG surface and toplevel creation")
	fmt.Println()
	fmt.Println("      RENDERING LAYER (Software Rasterizer)")
	fmt.Println("      -------------------------------------")
	fmt.Println("      • Filled rectangles (title bar)")
	fmt.Println("      • Rounded rectangles (radius=8px, anti-aliased)")
	fmt.Println("      • Alpha gradient (manual alpha blending)")
	fmt.Println("      • Anti-aliased lines (3px width)")
	fmt.Println("      • Color grid (8 unique colors)")
	fmt.Println()
	fmt.Println("      UI LAYER (Widgets)")
	fmt.Println("      ------------------")
	fmt.Println("      • Button widget with state management")
	fmt.Println("      • TextInput widget with placeholder")
	fmt.Println()

	fmt.Println("Buffer Stats:")
	fmt.Printf("  Pixels rendered: %d\n", windowWidth*windowHeight)
	fmt.Printf("  Buffer size: %d bytes\n", len(renderBuffer.Pixels))
	fmt.Printf("  Stride: %d bytes/row\n", renderBuffer.Stride)
}

// renderDemoContent renders Phase 1 features to the buffer.
func renderDemoContent(buf *core.Buffer, btn *widgets.Button, input *widgets.TextInput) {
	// Feature 1: Clear with solid color
	buf.Clear(core.Color{R: 240, G: 240, B: 245, A: 255})

	// Feature 2: Filled rectangle (title bar)
	titleColor := core.Color{R: 60, G: 60, B: 80, A: 255}
	buf.FillRect(10, 10, 380, 50, titleColor)

	// Feature 3: Button widget with rounded corners
	if err := btn.Draw(buf, 140, 100); err != nil {
		log.Printf("Warning: button draw failed: %v", err)
	}

	// Feature 4: TextInput widget
	if err := input.Draw(buf, 100, 170); err != nil {
		log.Printf("Warning: input draw failed: %v", err)
	}

	// Feature 5: Showcase rasterizer primitives
	showcaseY := 220

	// 5a. Rounded rectangle with anti-aliased corners
	buf.FillRoundedRect(10, showcaseY, 60, 40, 8,
		core.Color{R: 100, G: 200, B: 150, A: 255})

	// 5b. Alpha gradient (manual blending demonstration)
	for i := 0; i < 60; i++ {
		alpha := uint8(255 - (i * 4))
		buf.FillRect(80+i, showcaseY, 1, 40,
			core.Color{R: 200, G: 100, B: 150, A: alpha})
	}

	// 5c. Anti-aliased line (3px width)
	buf.DrawLine(160, showcaseY, 220, showcaseY+40, 3,
		core.Color{R: 150, G: 150, B: 200, A: 255})

	// 5d. Grid of colored rectangles
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			x := 240 + i*20
			y := showcaseY + j*20
			c := core.Color{
				R: uint8(50 + i*40),
				G: uint8(50 + j*80),
				B: uint8(200 - i*30),
				A: 255,
			}
			buf.FillRect(x, y, 15, 15, c)
		}
	}
}
