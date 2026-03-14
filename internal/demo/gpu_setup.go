package demo

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/render"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	dri3pkg "github.com/opd-ai/wain/internal/x11/dri3"
	presentpkg "github.com/opd-ai/wain/internal/x11/present"
)

// GPUTriangleSetup bundles the X11 connection, GPU allocator, and GPU context
// that are needed by GPU triangle demo binaries. Obtain a value via
// NewGPUTriangleSetup and release all resources via Cleanup.
type GPUTriangleSetup struct {
	Conn       *x11client.Connection
	DRI3Ext    *dri3pkg.Extension
	PresentExt *presentpkg.Extension
	Allocator  *render.Allocator
	GPUCtx     *render.GpuContext
	Window     x11client.XID

	renderFd int
}

// NewGPUTriangleSetup initialises the full X11+GPU stack required by GPU
// triangle demo binaries. It connects to the X11 server, queries DRI3 and
// Present extensions, opens the DRM device at devicePath, detects the GPU
// generation, creates a GPU allocator and context, and maps an X11 window.
//
// On success callers must call Cleanup when they are done. On failure all
// partially-initialised resources are released before returning.
func NewGPUTriangleSetup(devicePath string, windowWidth, windowHeight int) (*GPUTriangleSetup, error) {
	conn, err := x11client.Connect("0")
	if err != nil {
		return nil, fmt.Errorf("connect to X11: %w", err)
	}

	dri3Ext, presentExt, renderFd, err := QueryDRI3AndPresentExtensions(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	allocator, gpuCtx, err := SetupGPUAllocator(devicePath, 1, 3)
	if err != nil {
		syscall.Close(renderFd)
		conn.Close()
		return nil, err
	}

	window, err := CreateX11WindowWithDefaults(conn, windowWidth, windowHeight)
	if err != nil {
		allocator.Close()
		syscall.Close(renderFd)
		conn.Close()
		return nil, fmt.Errorf("create X11 window: %w", err)
	}

	return &GPUTriangleSetup{
		Conn:       conn,
		DRI3Ext:    dri3Ext,
		PresentExt: presentExt,
		Allocator:  allocator,
		GPUCtx:     gpuCtx,
		Window:     window,
		renderFd:   renderFd,
	}, nil
}

// Cleanup releases all resources owned by the setup.
func (s *GPUTriangleSetup) Cleanup() {
	s.Allocator.Close()
	syscall.Close(s.renderFd)
	s.Conn.Close()
}
