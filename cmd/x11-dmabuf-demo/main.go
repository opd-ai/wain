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
	"log"
	"syscall"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/render"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	windowWidth  = 800
	windowHeight = 600
	bpp          = 32 // ARGB8888
	depth        = 24
)

func main() {
	demo.CheckHelpFlag("x11-dmabuf-demo", "GPU buffer sharing with X11 using DRI3/Present", []string{
		demo.FormatExample("x11-dmabuf-demo", "Run DRI3 GPU buffer demo"),
		demo.FormatExample("x11-dmabuf-demo --help", "Show this help message"),
	})

	fmt.Println("==============================================")
	fmt.Println("wain Phase 2.4 Demo - DRI3/Present + X11")
	fmt.Println("==============================================")
	fmt.Println()

	if err := runDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
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
	fmt.Println("[1/9] Connecting to X11 server...")
	conn, err := x11client.Connect("0")
	if err != nil {
		return nil, nil, fmt.Errorf("connect to X11: %w", err)
	}
	fmt.Println("      ✓ Connected to :0")

	fmt.Println("\n[2/9] Querying DRI3 extension...")
	dri3Adapter := demo.NewDRI3ConnectionAdapter(conn)
	dri3Ext, err := dri3.QueryExtension(dri3Adapter)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("query DRI3: %w", err)
	}
	fmt.Printf("      ✓ DRI3 version %d.%d\n", dri3Ext.MajorVersion(), dri3Ext.MinorVersion())

	fmt.Println("\n[3/9] Querying Present extension...")
	presentAdapter := demo.NewPresentConnectionAdapter(conn)
	presentExt, err := present.QueryExtension(presentAdapter)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("query Present: %w", err)
	}
	fmt.Printf("      ✓ Present version %d.%d\n", presentExt.MajorVersion(), presentExt.MinorVersion())

	fmt.Println("\n[4/9] Opening DRI3 render node...")
	root := conn.RootWindow()
	renderFd, err := dri3Ext.Open(dri3Adapter, dri3.XID(root), 0)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("open DRI3: %w", err)
	}
	fmt.Printf("      ✓ Opened render node (fd %d)\n", renderFd)

	fmt.Println("\n[5/9] Creating GPU buffer allocator...")
	drmPath := "/dev/dri/renderD128"
	allocator, err := render.NewAllocator(drmPath)
	if err != nil {
		syscall.Close(renderFd)
		conn.Close()
		return nil, nil, fmt.Errorf("create allocator: %w (is %s accessible?)", err, drmPath)
	}
	fmt.Printf("      ✓ Opened %s\n", drmPath)

	fmt.Println("\n[6/9] Creating X11 window...")
	const (
		x           = 100
		y           = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0 // CopyFromParent
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}

	wid, err := conn.CreateWindow(root, x, y, windowWidth, windowHeight, borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		allocator.Close()
		syscall.Close(renderFd)
		conn.Close()
		return nil, nil, fmt.Errorf("create window: %w", err)
	}
	fmt.Printf("      ✓ Created window XID %d\n", wid)

	if err := conn.MapWindow(wid); err != nil {
		allocator.Close()
		syscall.Close(renderFd)
		conn.Close()
		return nil, nil, fmt.Errorf("map window: %w", err)
	}
	fmt.Println("      ✓ Window mapped to display")

	ctx := &demoContext{
		conn:       conn,
		dri3Ext:    dri3Ext,
		presentExt: presentExt,
		allocator:  allocator,
		window:     wid,
	}
	cleanup := func() {
		allocator.Close()
		syscall.Close(renderFd)
		conn.Close()
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

	fd, err := ctx.allocator.ExportDmabuf(buffer)
	if err != nil {
		return 0, fmt.Errorf("export dmabuf: %w", err)
	}
	defer syscall.Close(fd)
	fmt.Printf("      ✓ Exported as DMA-BUF fd %d\n", fd)
	fmt.Println("      Note: Buffer content is uninitialized (CPU mmap TBD, GPU rendering in Phase 3+)")

	pixmapXID, err := ctx.conn.AllocXID()
	if err != nil {
		return 0, fmt.Errorf("allocate pixmap XID: %w", err)
	}

	size := buffer.Stride * buffer.Height
	dri3Adapter := demo.NewDRI3ConnectionAdapter(ctx.conn)
	err = ctx.dri3Ext.PixmapFromBuffer(
		dri3Adapter,
		dri3.XID(pixmapXID),
		dri3.XID(ctx.window),
		size,
		uint16(buffer.Width),
		uint16(buffer.Height),
		uint16(buffer.Stride),
		depth,
		bpp,
		fd,
	)
	if err != nil {
		return 0, fmt.Errorf("create pixmap from buffer: %w", err)
	}
	fmt.Printf("      ✓ Created pixmap XID %d from DMA-BUF\n", pixmapXID)

	return pixmapXID, nil
}

func presentPixmap(ctx *demoContext, pixmap x11client.XID) error {
	fmt.Println("\n[9/9] Presenting pixmap to window...")

	presentAdapter := demo.NewPresentConnectionAdapter(ctx.conn)
	err := ctx.presentExt.PresentPixmap(presentAdapter, present.PixmapPresentOptions{
		Window:  present.XID(ctx.window),
		Pixmap:  present.XID(pixmap),
		Options: present.PresentOptionNone,
	})
	if err != nil {
		return fmt.Errorf("present pixmap: %w", err)
	}
	fmt.Println("      ✓ Pixmap presented to window")

	return nil
}
