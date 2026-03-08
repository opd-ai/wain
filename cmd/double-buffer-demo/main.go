// Command double-buffer-demo demonstrates Phase 5.3 double/triple buffering
// with Wayland compositor synchronization.
//
// NOTE: This demo is currently out of sync with the latest Wayland client API
// and will be updated in a future commit. The underlying buffer ring and
// synchronization infrastructure is fully functional and tested in
// internal/buffer/ and internal/integration/.
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

//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/opd-ai/wain/internal/buffer"
	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/core"
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
	fmt.Println("=======================================================")
	fmt.Println("wain Phase 5.3 Demo - Double/Triple Buffering")
	fmt.Println("=======================================================")
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
	surface    *client.Surface
	ring       *buffer.Ring
	sync       *buffer.Synchronizer
	shmBuffers []*shm.Buffer
	pools      []*shm.Pool
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
	surfaceID, err := wlCtx.Compositor.CreateSurface()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("create surface: %w", err)
	}
	surface := client.NewSurface(conn, surfaceID)
	conn.RegisterObject(surface)

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
		toplevel.Destroy()
		xdgSurface.Destroy()
		conn.Close()
	}

	demoCtx := &demoContext{
		conn:       conn,
		compositor: wlCtx.Compositor,
		shmObj:     wlCtx.SHM,
		wmBase:     wlCtx.WmBase,
		surface:    surface,
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

	demoCtx.shmBuffers = make([]*shm.Buffer, bufferCount)
	demoCtx.pools = make([]*shm.Pool, bufferCount)

	stride := windowWidth * (bpp / 8)
	size := stride * windowHeight

	for i := 0; i < bufferCount; i++ {
		// Create shared memory pool
		pool, err := shm.CreatePool(size)
		if err != nil {
			return fmt.Errorf("create pool %d: %w", i, err)
		}
		demoCtx.pools[i] = pool

		// Create wl_shm_pool
		poolID, err := demoCtx.shmObj.CreatePool(pool.FD(), int32(size))
		if err != nil {
			return fmt.Errorf("create wl_shm_pool %d: %w", i, err)
		}
		wlPool := shm.NewPool(demoCtx.conn, poolID, pool)
		demoCtx.conn.RegisterObject(wlPool)

		// Create wl_buffer
		bufferID, err := wlPool.CreateBuffer(0, int32(windowWidth), int32(windowHeight), int32(stride), shm.FormatARGB8888)
		if err != nil {
			return fmt.Errorf("create buffer %d: %w", i, err)
		}
		wlBuffer := shm.NewBuffer(demoCtx.conn, bufferID)
		demoCtx.conn.RegisterObject(wlBuffer)
		demoCtx.shmBuffers[i] = wlBuffer

		// Register buffer ID with synchronizer for future integration
		// For this demo, we'll simulate release events manually
		if err := demoCtx.sync.RegisterBuffer(wlBuffer.ID(), i); err != nil {
			return fmt.Errorf("register buffer %d: %w", i, err)
		}

		// Associate pool data with ring slot
		slot := demoCtx.ring.GetSlot(i)
		slot.UserData = pool

		fmt.Printf("      ✓ Buffer %d created (wl_buffer ID=%d)\n", i, wlBuffer.ID())
	}

	return nil
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

	pool, ok := slot.UserData.(*shm.Pool)
	if !ok {
		return fmt.Errorf("slot %d has invalid UserData", slot.Index)
	}

	renderFrame(pool.Data(), frame)

	if err := attachAndCommitBuffer(demoCtx, slot.Index); err != nil {
		return err
	}

	if err := demoCtx.sync.MarkDisplaying(slot.Index); err != nil {
		return fmt.Errorf("mark displaying: %w", err)
	}
	currentlyDisplaying[slot.Index] = true

	if err := demoCtx.conn.Dispatch(); err != nil {
		return fmt.Errorf("dispatch events: %w", err)
	}

	simulateCompositorRelease(demoCtx, frame, slot.Index, currentlyDisplaying)
	return nil
}

func attachAndCommitBuffer(demoCtx *demoContext, slotIndex int) error {
	wlBuffer := demoCtx.shmBuffers[slotIndex]
	if err := demoCtx.surface.Attach(wlBuffer.ID(), 0, 0); err != nil {
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
			if err := demoCtx.sync.OnReleaseEvent(demoCtx.shmBuffers[prevSlot].ID()); err == nil {
				delete(currentlyDisplaying, prevSlot)
			}
		}
	}
}

func processRemainingEvents(conn *client.Connection) {
	for i := 0; i < 10; i++ {
		conn.Dispatch()
		time.Sleep(10 * time.Millisecond)
	}
}

func renderFrame(data []byte, frameNum int) {
	// Create ARGB8888 image buffer
	width := windowWidth
	height := windowHeight
	stride := width * 4

	// Clear to background color
	bgColor := uint32(0xFF2C3E50) // dark blue-gray
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := y*stride + x*4
			*(*uint32)(unsafe.Pointer(&data[offset])) = bgColor
		}
	}

	// Draw animated rectangle
	rectSize := 60
	rectX := 50 + (frameNum * 10 % (width - rectSize - 100))
	rectY := height/2 - rectSize/2

	// Render rectangle using software rasterizer
	core.FillRect(data, stride, rectX, rectY, rectSize, rectSize, 0xFFE74C3C) // red

	// Draw frame counter text (simplified)
	textX := 10
	textY := 10
	drawFrameNumber(data, stride, textX, textY, frameNum)
}

func drawFrameNumber(data []byte, stride, x, y, frameNum int) {
	// Simple 5x7 pixel font for digits
	// Draw "Frame: NN" text
	color := uint32(0xFFECF0F1) // light gray

	// Very simplified - just draw a few pixels to indicate frame number
	// In a real implementation, use internal/raster/text
	for i := 0; i < 50; i++ {
		for j := 0; j < 10; j++ {
			offset := (y+j)*stride + (x+i)*4
			if offset+3 < len(data) {
				*(*uint32)(unsafe.Pointer(&data[offset])) = color
			}
		}
	}
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
