// decorations-demo demonstrates client-side window decorations on Wayland.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/ui/decorations"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/xdg"
)

func main() {
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		display = "wayland-0"
	}

	conn, err := client.Connect(display)
	if err != nil {
		log.Fatalf("Failed to connect to Wayland: %v", err)
	}
	defer conn.Close()

	registry := conn.Registry()

	// Bind compositor
	compGlobal := registry.FindGlobal("wl_compositor")
	if compGlobal == nil {
		log.Fatal("compositor not found")
	}
	compositor, err := registry.BindCompositor(compGlobal)
	if err != nil {
		log.Fatalf("Failed to bind compositor: %v", err)
	}

	// Bind XDG shell
	xdgGlobal := registry.FindGlobal("xdg_wm_base")
	if xdgGlobal == nil {
		log.Fatal("xdg_wm_base not found")
	}
	wmBaseID, wmBaseVersion, err := registry.BindXdgWmBase(xdgGlobal)
	if err != nil {
		log.Fatalf("Failed to bind xdg_wm_base: %v", err)
	}
	wmBase := xdg.NewWmBase(conn, wmBaseID, wmBaseVersion)

	// Try to bind decoration manager (optional)
	var decorationMgr *xdg.DecorationManager
	decoGlobal := registry.FindGlobal("zxdg_decoration_manager_v1")
	if decoGlobal != nil {
		decoMgrID, decoVersion, err := registry.BindXdgDecorationManager(decoGlobal)
		if err != nil {
			log.Printf("Warning: Failed to bind decoration manager: %v", err)
		} else {
			decorationMgr = xdg.NewDecorationManager(conn, decoMgrID, decoVersion)
			log.Printf("Decoration manager bound (version %d)", decoVersion)
		}
	} else {
		log.Println("Decoration manager not available - using client-side decorations")
	}

	// Create window
	width := 640
	height := 480
	theme := decorations.DefaultDecorationTheme()
	titleBarHeight := theme.TitleBarHeight

	surface, err := compositor.CreateSurface()
	if err != nil {
		log.Fatalf("Failed to create surface: %v", err)
	}

	xdgSurface, err := wmBase.GetXdgSurface(surface.ID())
	if err != nil {
		log.Fatalf("Failed to create XDG surface: %v", err)
	}

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		log.Fatalf("Failed to create toplevel: %v", err)
	}

	// Configure decoration mode
	if decorationMgr != nil {
		topLevelDecoration, err := decorationMgr.GetToplevelDecoration(toplevel)
		if err != nil {
			log.Printf("Warning: Failed to get toplevel decoration: %v", err)
		} else {
			// Request client-side decorations
			if err := topLevelDecoration.SetMode(xdg.DecorationModeClientSide); err != nil {
				log.Printf("Warning: Failed to set decoration mode: %v", err)
			} else {
				log.Println("Requested client-side decorations")
			}
		}
	}

	// Set window properties
	toplevel.SetTitle("Window Decorations Demo")
	toplevel.SetAppID("org.opd-ai.wain.decorations-demo")
	toplevel.SetMinSize(320, 240)

	// Create shared memory buffer
	totalHeight := height + titleBarHeight
	buf, err := core.NewBuffer(width, totalHeight)
	if err != nil {
		log.Fatalf("Failed to create buffer: %v", err)
	}

	// Create title bar
	titleBar := decorations.NewTitleBar("Window Decorations Demo", width, titleBarHeight)
	titleBar.SetTheme(theme)

	// Create text atlas for title rendering
	atlas := text.NewAtlas()
	titleBar.SetAtlas(atlas)

	// Render content
	renderFrame(buf, titleBar, width, height, titleBarHeight)

	// Display the buffer
	// (In a real application, this would be wired up to the Wayland surface)
	log.Printf("Rendered %dx%d window with %d pixel title bar", width, totalHeight, titleBarHeight)
	log.Println("Title bar contains: minimize, maximize, and close buttons")
	log.Println("Demo complete!")
}

func renderFrame(buf *core.Buffer, titleBar *decorations.TitleBar, width, height, titleBarHeight int) {
	// Clear background
	bgColor := core.Color{R: 255, G: 255, B: 255, A: 255}
	buf.FillRect(0, 0, width, height+titleBarHeight, bgColor)

	// Render title bar
	if err := titleBar.Draw(buf, 0, 0); err != nil {
		log.Printf("Warning: Failed to draw title bar: %v", err)
	}

	// Render window content
	contentColor := core.Color{R: 250, G: 250, B: 250, A: 255}
	buf.FillRect(0, titleBarHeight, width, height, contentColor)

	// Draw some example content
	exampleColor := core.Color{R: 100, G: 150, B: 200, A: 255}
	buf.FillRect(50, titleBarHeight+50, 200, 100, exampleColor)
}

func renderFrameWithDisplayList(titleBar *decorations.TitleBar, width, height, titleBarHeight int) *displaylist.DisplayList {
	dl := displaylist.New()

	// Background
	bgColor := core.Color{R: 255, G: 255, B: 255, A: 255}
	dl.AddFillRect(0, 0, width, height+titleBarHeight, bgColor)

	// Title bar
	titleBar.RenderToDisplayList(dl, 0, 0)

	// Content area
	contentColor := core.Color{R: 250, G: 250, B: 250, A: 255}
	dl.AddFillRect(0, titleBarHeight, width, height, contentColor)

	// Example content
	exampleColor := core.Color{R: 100, G: 150, B: 200, A: 255}
	dl.AddFillRect(50, titleBarHeight+50, 200, 100, exampleColor)

	fmt.Printf("Display list generated: %d commands\n", len(dl.Commands()))
	return dl
}
