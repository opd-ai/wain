// Command dmabuf-demo demonstrates GPU buffer sharing with Wayland using DMA-BUF.
//
// This binary showcases Phase 2.3 features:
//   - zwp_linux_dmabuf_v1 protocol implementation
//   - GPU buffer allocation via Rust DRM/GEM API
//   - DMA-BUF file descriptor export
//   - Zero-copy buffer sharing with compositor
//
// Usage:
//
//	./bin/dmabuf-demo
//
// Requirements:
//   - Wayland compositor with linux-dmabuf support
//   - /dev/dri/renderD128 (Intel GPU)
package main

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/render"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/dmabuf"
	"github.com/opd-ai/wain/internal/wayland/xdg"
)

const (
	windowWidth  = 800
	windowHeight = 600
	bpp          = 32 // ARGB8888
)

func main() {
	demo.RunDemoWithSetup(
		"dmabuf-demo",
		"Wayland DMA-BUF GPU buffer sharing demonstration",
		[]string{
			demo.FormatExample("dmabuf-demo", "Run demo on Wayland compositor"),
			demo.FormatExample("dmabuf-demo --help", "Show this help message"),
		},
		"wain Phase 2.3 Demo - DMA-BUF + Wayland",
		runDemo,
	)
}

type demoContext struct {
	conn       *client.Connection
	compositor *client.Compositor
	dmabufObj  *dmabuf.Dmabuf
	wmBase     *xdg.WmBase
	allocator  *render.Allocator
}

func runDemo() error {
	ctx, cleanup, err := setupWaylandContext()
	if err != nil {
		return err
	}
	defer cleanup()

	buffer, bufferCleanup, err := createGPUBuffer(ctx)
	if err != nil {
		return err
	}
	defer bufferCleanup()

	wlBufferID, err := createWaylandBuffer(ctx, buffer)
	if err != nil {
		return err
	}

	if err := createAndDisplayWindow(ctx, wlBufferID); err != nil {
		return err
	}

	fmt.Println("\nWindow should now be visible with GPU-allocated buffer!")
	fmt.Println("Note: This is a demonstration of DMA-BUF protocol implementation.")
	fmt.Println("The window will close immediately as we don't have a full event loop.")
	return nil
}

func setupWaylandContext() (*demoContext, func(), error) {
	fmt.Println("[1/8] Connecting to Wayland compositor...")
	conn, err := demo.ConnectToWayland()
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("      ✓ Connected")

	fmt.Println("\n[2/8] Discovering compositor globals...")
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("get registry: %w", err)
	}

	compositor, dmabufObj, wmBase, err := bindGlobals(conn, registry)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	allocator, err := demo.SetupGPUAllocatorSimple(demo.DefaultDRMPath, 3, 8)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	ctx := &demoContext{
		conn:       conn,
		compositor: compositor,
		dmabufObj:  dmabufObj,
		wmBase:     wmBase,
		allocator:  allocator,
	}
	cleanup := func() {
		allocator.Close()
		conn.Close()
	}
	return ctx, cleanup, nil
}

func bindGlobals(conn *client.Connection, registry *client.Registry) (*client.Compositor, *dmabuf.Dmabuf, *xdg.WmBase, error) {
	compositor, err := bindCompositorGlobal(registry)
	if err != nil {
		return nil, nil, nil, err
	}
	dmabufObj, err := bindDmabufGlobal(conn, registry)
	if err != nil {
		return nil, nil, nil, err
	}
	wmBase, err := bindXdgWmBaseGlobal(conn, registry)
	if err != nil {
		return nil, nil, nil, err
	}
	return compositor, dmabufObj, wmBase, nil
}

// bindCompositorGlobal finds and binds the wl_compositor global.
func bindCompositorGlobal(registry *client.Registry) (*client.Compositor, error) {
	g := registry.FindGlobal("wl_compositor")
	if g == nil {
		return nil, fmt.Errorf("wl_compositor not found")
	}
	compositor, err := registry.BindCompositor(g)
	if err != nil {
		return nil, fmt.Errorf("bind compositor: %w", err)
	}
	fmt.Println("      ✓ Bound to wl_compositor")
	return compositor, nil
}

// bindDmabufGlobal finds and binds the zwp_linux_dmabuf_v1 global.
func bindDmabufGlobal(conn *client.Connection, registry *client.Registry) (*dmabuf.Dmabuf, error) {
	obj, err := demo.BindDmabuf(conn, registry)
	if err != nil {
		return nil, err
	}
	fmt.Println("      ✓ Bound to zwp_linux_dmabuf_v1")
	return obj, nil
}

// bindXdgWmBaseGlobal finds and binds the xdg_wm_base global.
func bindXdgWmBaseGlobal(conn *client.Connection, registry *client.Registry) (*xdg.WmBase, error) {
	wmBase, err := demo.BindXdgWmBase(conn, registry)
	if err != nil {
		return nil, err
	}
	fmt.Println("      ✓ Bound to xdg_wm_base")
	return wmBase, nil
}

func createGPUBuffer(ctx *demoContext) (*render.BufferHandle, func(), error) {
	fmt.Println("\n[4/8] Allocating GPU buffer...")
	buffer, cleanup, err := demo.AllocateBuffer(ctx.allocator, windowWidth, windowHeight, bpp)
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("      ✓ Allocated %dx%d buffer (stride: %d)\n", buffer.Width, buffer.Height, buffer.Stride)
	return buffer, cleanup, nil
}

func createWaylandBuffer(ctx *demoContext, buffer *render.BufferHandle) (uint32, error) {
	fmt.Println("\n[5/8] Exporting buffer as DMA-BUF...")
	fd, err := ctx.allocator.ExportDmabuf(buffer)
	if err != nil {
		return 0, fmt.Errorf("export dmabuf: %w", err)
	}
	defer syscall.Close(fd)
	fmt.Printf("      ✓ Exported as DMA-BUF fd %d\n", fd)
	fmt.Println("      Note: Buffer content is uninitialized (CPU mmap TBD, GPU rendering in Phase 3+)")

	fmt.Println("\n[6/8] Creating wl_buffer from DMA-BUF...")
	params, err := ctx.dmabufObj.CreateParams()
	if err != nil {
		return 0, fmt.Errorf("create params: %w", err)
	}

	if err := params.Add(int32(fd), 0, 0, buffer.Stride, 0, 0); err != nil {
		return 0, fmt.Errorf("add plane: %w", err)
	}

	bufferID, err := params.CreateImmed(
		int32(buffer.Width),
		int32(buffer.Height),
		dmabuf.FormatARGB8888,
		0,
	)
	if err != nil {
		return 0, fmt.Errorf("create wl_buffer: %w", err)
	}
	fmt.Printf("      ✓ Created wl_buffer (object ID %d)\n", bufferID)
	return bufferID, nil
}

func createAndDisplayWindow(ctx *demoContext, bufferID uint32) error {
	fmt.Println("\n[7/8] Creating window...")
	surface, err := ctx.compositor.CreateSurface()
	if err != nil {
		return fmt.Errorf("create surface: %w", err)
	}

	_, _, err = demo.CreateXdgWindow(ctx.conn, ctx.wmBase, surface, "wain DMA-BUF Demo")
	if err != nil {
		return err
	}

	if err := surface.Commit(); err != nil {
		return fmt.Errorf("commit surface: %w", err)
	}
	fmt.Println("      ✓ Window created")

	return attachDMABUFBuffer(surface, bufferID)
}

func attachDMABUFBuffer(surface *client.Surface, bufferID uint32) error {
	fmt.Println("\n[8/8] Attaching DMA-BUF buffer to window...")
	if err := surface.Attach(bufferID, 0, 0); err != nil {
		return fmt.Errorf("attach buffer: %w", err)
	}

	if err := surface.Damage(0, 0, int32(windowWidth), int32(windowHeight)); err != nil {
		return fmt.Errorf("damage surface: %w", err)
	}

	if err := surface.Commit(); err != nil {
		return fmt.Errorf("commit surface: %w", err)
	}

	fmt.Println("      ✓ Buffer attached")
	return nil
}
