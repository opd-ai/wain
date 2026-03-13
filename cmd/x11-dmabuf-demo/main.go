// Command x11-dmabuf-demo demonstrates GPU buffer sharing with X11 using DRI3.
//
// This binary showcases Phase 2.4 features:
//   - DRI3 extension implementation
//   - Present extension for frame synchronization
//   - GPU buffer allocation via Rust DRM/GEM API
//   - DMA-BUF file descriptor export
//   - Zero-copy buffer sharing with X server
//
// Usage:
//
//	./bin/x11-dmabuf-demo
//
// Requirements:
//   - X11 server with DRI3 and Present support
//   - /dev/dri/renderD128 (Intel GPU)
package main

import (
	"fmt"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/render"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
)

const (
	windowWidth  = 800
	windowHeight = 600
	bpp          = 32 // ARGB8888
	depth        = 24
)

func main() {
	demo.RunDemoWithSetup(
		"x11-dmabuf-demo",
		"GPU buffer sharing with X11 using DRI3/Present",
		[]string{
			demo.FormatExample("x11-dmabuf-demo", "Run DRI3 GPU buffer demo"),
			demo.FormatExample("x11-dmabuf-demo --help", "Show this help message"),
		},
		"wain Phase 2.4 Demo - DRI3/Present + X11",
		runDemo,
	)
}

type demoContext struct {
	conn       *x11client.Connection
	dri3Ext    *dri3.Extension
	presentExt *present.Extension
	allocator  *render.Allocator
	window     x11client.XID
}

func runDemo() error {
	ctx, cleanup, err := setupX11Context()
	if err != nil {
		return err
	}
	defer cleanup()

	buffer, bufferCleanup, err := createGPUBuffer(ctx)
	if err != nil {
		return err
	}
	defer bufferCleanup()

	pixmap, err := createPixmapFromBuffer(ctx, buffer)
	if err != nil {
		return err
	}

	if err := presentPixmap(ctx, pixmap); err != nil {
		return err
	}

	fmt.Println("\nWindow should now be visible with GPU-allocated buffer!")
	fmt.Println("Note: This is a demonstration of DRI3/Present protocol implementation.")
	fmt.Println("The window will close immediately as we don't have a full event loop.")
	return nil
}

func setupX11Context() (*demoContext, func(), error) {
	fmt.Println("[1/9] Connecting to X11 server and setting up display...")
	conn, dri3Ext, presentExt, wid, displayCleanup, err := demo.SetupDisplay(windowWidth, windowHeight)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("      ✓ Connected to :0")
	fmt.Printf("      ✓ DRI3 version %d.%d\n", dri3Ext.MajorVersion(), dri3Ext.MinorVersion())
	fmt.Printf("      ✓ Present version %d.%d\n", presentExt.MajorVersion(), presentExt.MinorVersion())
	fmt.Printf("      ✓ Created and mapped window XID %d\n", wid)

	fmt.Println("\n[5/9] Creating GPU buffer allocator...")
	drmPath := "/dev/dri/renderD128"
	allocator, err := render.NewAllocator(drmPath)
	if err != nil {
		displayCleanup()
		return nil, nil, fmt.Errorf("create allocator: %w (is %s accessible?)", err, drmPath)
	}
	fmt.Printf("      ✓ Opened %s\n", drmPath)

	ctx := &demoContext{
		conn:       conn,
		dri3Ext:    dri3Ext,
		presentExt: presentExt,
		allocator:  allocator,
		window:     wid,
	}
	cleanup := func() {
		allocator.Close()
		displayCleanup()
	}
	return ctx, cleanup, nil
}

func createGPUBuffer(ctx *demoContext) (*render.BufferHandle, func(), error) {
	fmt.Println("\n[7/9] Allocating GPU buffer...")
	buffer, err := ctx.allocator.Allocate(windowWidth, windowHeight, bpp, render.TilingNone)
	if err != nil {
		return nil, nil, fmt.Errorf("allocate buffer: %w", err)
	}
	fmt.Printf("      ✓ Allocated %dx%d buffer (stride: %d)\n", buffer.Width, buffer.Height, buffer.Stride)

	cleanup := func() { buffer.Destroy() }
	return buffer, cleanup, nil
}

func createPixmapFromBuffer(ctx *demoContext, buffer *render.BufferHandle) (x11client.XID, error) {
	fmt.Println("\n[8/9] Creating X11 pixmap from DMA-BUF...")

	pixmapXID, err := demo.CreatePixmapFromBuffer(ctx.conn, ctx.dri3Ext, ctx.window, buffer, ctx.allocator, depth, bpp)
	if err != nil {
		return 0, err
	}

	fmt.Printf("      ✓ Exported buffer as DMA-BUF and created pixmap XID %d\n", pixmapXID)
	fmt.Println("      Note: Buffer content is uninitialized (CPU mmap TBD, GPU rendering in Phase 3+)")

	return pixmapXID, nil
}

func presentPixmap(ctx *demoContext, pixmap x11client.XID) error {
	fmt.Println("\n[9/9] Presenting pixmap to window...")

	if err := demo.PresentPixmapToWindow(ctx.conn, ctx.presentExt, ctx.window, pixmap); err != nil {
		return err
	}

	fmt.Println("      ✓ Pixmap presented to window")
	return nil
}
