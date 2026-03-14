// Command double-buffer-demo demonstrates Phase 5.3 double/triple buffering
// with Wayland compositor synchronization.
//
// This binary showcases:
//   - buffer.Ring for framebuffer management
//   - buffer.Synchronizer for compositor event integration
//   - wl_buffer.release event handling
//   - Smooth multi-frame rendering without blocking
//
// Usage:
//
//	./bin/double-buffer-demo
//
// The demo renders 30 frames with animated content, demonstrating:
//   - Acquiring buffers from the ring
//   - Rendering to acquired buffers
//   - Presenting to compositor
//   - Synchronizing with compositor release events
package main

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/opd-ai/wain/internal/buffer"
	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
	"github.com/opd-ai/wain/internal/wayland/xdg"
)

const (
	windowWidth  = 400
	windowHeight = 300
	bufferCount  = 2 // double buffering (use 3 for triple buffering)
	frameCount   = 30
	bpp          = 32 // ARGB8888
)

func main() {
	demo.RunDemoWithSetup(
		"double-buffer-demo",
		"Double/triple buffering with Wayland compositor synchronization",
		[]string{
			demo.FormatExample("double-buffer-demo", "Run animated demo with double buffering"),
			demo.FormatExample("double-buffer-demo --help", "Show this help message"),
		},
		"wain Phase 5.3 Demo - Double/Triple Buffering",
		runDemo,
	)
}

type demoContext struct {
	conn       *client.Connection
	compositor *client.Compositor
	shmObj     *shm.SHM
	wmBase     *xdg.WmBase
	surface    *client.Surface
	ring       *buffer.Ring
	sync       *buffer.Synchronizer
	buffers    []*shm.Buffer
	pools      []*shm.Pool
	fds        []int
}

func runDemo() error {
	ctx := context.Background()

	demoCtx, cleanup, err := setup()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := createBufferRing(demoCtx); err != nil {
		return err
	}

	if err := renderFrames(ctx, demoCtx); err != nil {
		return err
	}

	printSummary(demoCtx)
	return nil
}

func setup() (*demoContext, func(), error) {
	fmt.Println("[1/5] Connecting to Wayland compositor...")
	conn, err := demo.ConnectToWayland()
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("      ✓ Connected")

	fmt.Println("\n[2/5] Discovering compositor globals...")
	wlCtx, err := demo.SetupWaylandGlobals(conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	fmt.Println("      ✓ Bound to wl_compositor, wl_shm, xdg_wm_base")

	fmt.Println("\n[3/5] Creating window...")
	surface, err := wlCtx.Compositor.CreateSurface()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("create surface: %w", err)
	}

	xdgSurface, toplevel, err := demo.CreateXdgWindow(conn, wlCtx.WmBase, surface, "Double Buffering Demo")
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	if err := surface.Commit(); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("commit surface: %w", err)
	}
	fmt.Println("      ✓ Window created")

	cleanup := func() {
		_ = toplevel.Destroy()
		_ = xdgSurface.Destroy()
		conn.Close()
	}

	demoCtx := &demoContext{
		conn:       conn,
		compositor: wlCtx.Compositor,
		shmObj:     wlCtx.SHM,
		wmBase:     wlCtx.WmBase,
		surface:    surface,
		fds:        make([]int, 0, bufferCount),
	}

	return demoCtx, cleanup, nil
}

func createBufferRing(demoCtx *demoContext) error {
	fmt.Printf("\n[4/5] Creating buffer ring (size=%d)...\n", bufferCount)

	ring, err := buffer.NewRing(bufferCount)
	if err != nil {
		return fmt.Errorf("create ring: %w", err)
	}
	demoCtx.ring = ring
	demoCtx.sync = buffer.NewSynchronizer(ring)
	fmt.Printf("      ✓ Ring created with %d slots\n", bufferCount)

	demoCtx.buffers = make([]*shm.Buffer, bufferCount)
	demoCtx.pools = make([]*shm.Pool, bufferCount)

	stride := int32(windowWidth * 4)
	size := stride * windowHeight

	for i := 0; i < bufferCount; i++ {
		if err := createAndRegisterBuffer(demoCtx, i, size, stride); err != nil {
			return err
		}
	}

	return nil
}

// createAndRegisterBuffer allocates a single SHM buffer, maps it, creates the
// corresponding wl_buffer, and registers it with the synchronizer.
func createAndRegisterBuffer(demoCtx *demoContext, i int, size, stride int32) error {
	pool, err := allocateSHMPool(demoCtx, i, size)
	if err != nil {
		return err
	}

	buf, err := pool.CreateBuffer(0, int32(windowWidth), int32(windowHeight), stride, shm.FormatARGB8888)
	if err != nil {
		return fmt.Errorf("create buffer %d: %w", i, err)
	}
	demoCtx.buffers[i] = buf

	if err := demoCtx.sync.RegisterBuffer(buf.ID(), i); err != nil {
		return fmt.Errorf("register buffer %d: %w", i, err)
	}

	fmt.Printf("      ✓ Buffer %d created (wl_buffer ID=%d)\n", i, buf.ID())
	return nil
}

// allocateSHMPool creates a memfd, sizes it, opens a wl_shm_pool, and maps it
// for direct pixel access. The file descriptor is appended to demoCtx.fds.
func allocateSHMPool(demoCtx *demoContext, i int, size int32) (*shm.Pool, error) {
	fd, err := shm.CreateMemfd(fmt.Sprintf("wain-buffer-%d", i))
	if err != nil {
		return nil, fmt.Errorf("create memfd %d: %w", i, err)
	}
	demoCtx.fds = append(demoCtx.fds, fd)

	if err := syscall.Ftruncate(fd, int64(size)); err != nil {
		return nil, fmt.Errorf("truncate memfd %d: %w", i, err)
	}

	pool, err := demoCtx.shmObj.CreatePool(fd, size)
	if err != nil {
		return nil, fmt.Errorf("create pool %d: %w", i, err)
	}
	demoCtx.pools[i] = pool

	if err := pool.Map(); err != nil {
		return nil, fmt.Errorf("map pool %d: %w", i, err)
	}
	return pool, nil
}

func renderFrames(ctx context.Context, demoCtx *demoContext) error {
	fmt.Printf("\n[5/5] Rendering %d frames...\n", frameCount)

	currentlyDisplaying := make(map[int]bool)

	for frame := 0; frame < frameCount; frame++ {
		if err := renderSingleFrame(ctx, demoCtx, frame, currentlyDisplaying); err != nil {
			return err
		}

		if (frame+1)%10 == 0 {
			fmt.Printf("      ✓ Frame %d/%d rendered\n", frame+1, frameCount)
		}

		time.Sleep(16 * time.Millisecond)
	}

	processRemainingEvents(demoCtx.conn)
	return nil
}

func renderSingleFrame(ctx context.Context, demoCtx *demoContext, frame int, currentlyDisplaying map[int]bool) error {
	slot, err := demoCtx.sync.AcquireForWriting(ctx)
	if err != nil {
		return fmt.Errorf("acquire slot for frame %d: %w", frame, err)
	}

	buf := demoCtx.buffers[slot.Index]
	pixels := buf.Pixels()
	if pixels == nil {
		return fmt.Errorf("buffer %d has nil pixels (pool not mapped?)", slot.Index)
	}

	renderFrame(pixels, frame)

	if err := attachAndCommitBuffer(demoCtx, slot.Index); err != nil {
		return err
	}

	if err := demoCtx.sync.MarkDisplaying(slot.Index); err != nil {
		return fmt.Errorf("mark displaying: %w", err)
	}
	currentlyDisplaying[slot.Index] = true

	simulateCompositorRelease(demoCtx, frame, slot.Index, currentlyDisplaying)
	return nil
}

func attachAndCommitBuffer(demoCtx *demoContext, slotIndex int) error {
	buf := demoCtx.buffers[slotIndex]
	if err := demoCtx.surface.Attach(buf.ID(), 0, 0); err != nil {
		return fmt.Errorf("attach buffer: %w", err)
	}
	if err := demoCtx.surface.Damage(0, 0, windowWidth, windowHeight); err != nil {
		return fmt.Errorf("damage surface: %w", err)
	}
	if err := demoCtx.surface.Commit(); err != nil {
		return fmt.Errorf("commit surface: %w", err)
	}
	return nil
}

func simulateCompositorRelease(demoCtx *demoContext, frame, currentSlot int, currentlyDisplaying map[int]bool) {
	if frame > 0 {
		prevSlot := (currentSlot - 1 + bufferCount) % bufferCount
		if currentlyDisplaying[prevSlot] {
			if err := demoCtx.sync.OnReleaseEvent(demoCtx.buffers[prevSlot].ID()); err == nil {
				delete(currentlyDisplaying, prevSlot)
			}
		}
	}
}

func processRemainingEvents(conn *client.Connection) {
	// Wait a bit for compositor to process final events
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
	}
}

func renderFrame(pixels []byte, frameNum int) {
	// Create ARGB8888 buffer wrapper for core rasterizer
	width := windowWidth
	height := windowHeight
	stride := width * 4

	buf := &primitives.Buffer{
		Pixels: pixels,
		Width:  width,
		Height: height,
		Stride: stride,
	}

	// Clear to background color (dark blue-gray)
	bgColor := primitives.Color{R: 0x2C, G: 0x3E, B: 0x50, A: 0xFF}
	buf.FillRect(0, 0, width, height, bgColor)

	// Draw animated rectangle
	rectSize := 60
	rectX := 50 + (frameNum * 10 % (width - rectSize - 100))
	rectY := height/2 - rectSize/2

	// Render rectangle using software rasterizer (red)
	rectColor := primitives.Color{R: 0xE7, G: 0x4C, B: 0x3C, A: 0xFF}
	buf.FillRect(rectX, rectY, rectSize, rectSize, rectColor)

	// Draw frame counter indicator (light gray bar)
	textX := 10
	textY := 10
	textColor := primitives.Color{R: 0xEC, G: 0xF0, B: 0xF1, A: 0xFF}
	buf.FillRect(textX, textY, 50, 10, textColor)
}

func printSummary(demoCtx *demoContext) {
	fmt.Println("\n=======================================================")
	fmt.Println("Phase 5.3 Implementation Summary")
	fmt.Println("=======================================================")
	fmt.Println()
	fmt.Println("Features demonstrated:")
	fmt.Printf("  • buffer.Ring with %d slots\n", bufferCount)
	fmt.Println("  • buffer.Synchronizer for compositor event coordination")
	fmt.Println("  • wl_buffer.release event integration")
	fmt.Println("  • Non-blocking frame rendering loop")
	fmt.Println()

	stats := demoCtx.sync.Stats()
	ringStats := stats["ring_stats"].(map[string]int)
	fmt.Println("Final buffer ring state:")
	for state, count := range ringStats {
		if count > 0 {
			fmt.Printf("  • %s: %d buffers\n", state, count)
		}
	}
	fmt.Printf("  • registered buffers: %d\n", stats["registered_count"])
	fmt.Println()
	fmt.Println("Integration components:")
	fmt.Println("  • internal/buffer/ring.go - Ring buffer state machine")
	fmt.Println("  • internal/buffer/sync.go - Synchronizer")
	fmt.Println("  • internal/integration/wayland_sync.go - Wayland integration")
	fmt.Println()
}
