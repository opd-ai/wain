// Package display provides GPU-to-Display integration for Wayland and X11.
//
// This package implements the pipeline that connects GPU-rendered output to
// display servers (Wayland compositors and X11 servers). It bridges the gap
// between GPU command submission (Phase 3) and production UI rendering.
//
// The display pipeline handles:
//   - DMA-BUF export from GPU render targets
//   - Wayland zwp_linux_dmabuf_v1 buffer creation and attachment
//   - X11 DRI3 pixmap creation and Present extension scheduling
//   - GPU framebuffer lifecycle management with triple buffering
//   - Compositor event handling (release/idle notifications)
//
// Usage:
//
//	// Wayland path
//	pipeline, err := display.NewWaylandPipeline(conn, surface, dmabuf, renderer)
//	defer pipeline.Close()
//	for {
//	    err := pipeline.RenderAndPresent(displayList)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
//	// X11 path
//	pipeline, err := display.NewX11Pipeline(conn, window, dri3, present, renderer)
//	defer pipeline.Close()
//	for {
//	    err := pipeline.RenderAndPresent(displayList)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	}
package display
