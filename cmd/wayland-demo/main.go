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

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/primitives"
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
	demo.CheckHelpFlag("wayland-demo", "Wayland protocol client with software rasterizer", []string{
		demo.FormatExample("wayland-demo", "Run demo on Wayland compositor"),
		demo.FormatExample("wayland-demo --help", "Show this help message"),
	})

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
	conn, err := demo.ConnectToWayland()
	if err != nil {
		return nil, err
	}
	fmt.Println("      ✓ Connected")
	return conn, nil
}

// discoverGlobals binds to required Wayland global objects.
func discoverGlobals(conn *client.Connection) (*demoContext, error) {
	fmt.Println("\n[2/6] Discovering compositor globals...")
	wlCtx, err := demo.SetupWaylandGlobals(conn)
	if err != nil {
		return nil, err
	}
	fmt.Println("      ✓ Bound to wl_compositor, wl_shm, xdg_wm_base")

	return &demoContext{
		conn:       wlCtx.Conn,
		compositor: wlCtx.Compositor,
		shmObj:     wlCtx.SHM,
		wmBase:     wlCtx.WmBase,
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

	xdgSurface, toplevel, err := demo.CreateXdgWindow(ctx.conn, ctx.wmBase, surface, "wain Wayland Demo")
	if err != nil {
		return nil, err
	}
	fmt.Printf("      ✓ Created xdg_surface (ID %d)\n", xdgSurface.ID())
	fmt.Printf("      ✓ Created xdg_toplevel (ID %d)\n", toplevel.ID())
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
	btn, input := demo.StandardWidgets()
	fmt.Println("      ✓ Created Button widget (120x40)")
	fmt.Println("      ✓ Created TextInput widget (200x30)")
	return btn, input
}

// renderContent renders UI content to a buffer using the software rasterizer.
func renderContent(btn *widgets.Button, input *widgets.TextInput) (*primitives.Buffer, error) {
	fmt.Println("\n[5/6] Rendering content to framebuffer...")
	renderBuffer, err := demo.CreateDemoBuffer(windowWidth, windowHeight)
	if err != nil {
		return nil, err
	}
	demo.RenderDemoContent(renderBuffer, btn, input)
	fmt.Printf("      ✓ Rendered to %dx%d ARGB8888 buffer\n", windowWidth, windowHeight)
	return renderBuffer, nil
}

// displayBuffer copies the render buffer to shared memory and displays it.
func displayBuffer(ctx *demoContext, surface *client.Surface, renderBuffer *primitives.Buffer) error {
	if err := demo.AttachAndDisplayBuffer(ctx.shmObj, surface, renderBuffer, windowWidth, windowHeight); err != nil {
		return err
	}
	fmt.Println("      ✓ Buffer attached and displayed")
	return nil
}

// printFeatureSummary displays a summary of demonstrated Phase 1 features.
func printFeatureSummary(renderBuffer *primitives.Buffer) {
	fmt.Println("\n[6/6] Phase 1 Features Demonstrated:")
	demo.PrintFeatureList("      PROTOCOL LAYER (Wayland)\n      ------------------------", []string{
		"Connection to compositor via Unix socket",
		"Global discovery (wl_registry)",
		"Surface creation (wl_compositor)",
		"Shared memory buffers (wl_shm)",
		"Window management (xdg_wm_base)",
		"XDG surface and toplevel creation",
	})
	fmt.Println()
	demo.PrintRenderingFeatures()
	fmt.Println()
	demo.PrintUIFeatures()
	fmt.Println()

	demo.PrintBufferStats(windowWidth, windowHeight, renderBuffer)
}
