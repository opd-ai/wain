package demo

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/render"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
	"github.com/opd-ai/wain/internal/x11/wire"
)

// SetupDisplay provides a unified display setup helper for X11 demos with DRI3/Present support.
// It encapsulates the common 26-line boilerplate repeated across 8+ demo binaries.
//
// Returns the connection, DRI3 extension, Present extension, window XID, and a cleanup function.
// The cleanup function should be called when the demo completes.
func SetupDisplay(width, height int) (*x11client.Connection, *dri3.Extension, *present.Extension, x11client.XID, func(), error) {
	conn, err := x11client.Connect("0")
	if err != nil {
		return nil, nil, nil, 0, nil, fmt.Errorf("connect to X11: %w", err)
	}

	dri3Ext, presentExt, renderFd, err := QueryDRI3AndPresentExtensions(conn)
	if err != nil {
		conn.Close()
		return nil, nil, nil, 0, nil, err
	}

	wid, err := CreateX11WindowWithDefaults(conn, width, height)
	if err != nil {
		syscall.Close(renderFd)
		conn.Close()
		return nil, nil, nil, 0, nil, err
	}

	cleanup := func() {
		syscall.Close(renderFd)
		conn.Close()
	}

	return conn, dri3Ext, presentExt, wid, cleanup, nil
}

// QueryDRI3AndPresentExtensions queries both DRI3 and Present extensions from the X11 server.
// Returns the DRI3 extension, Present extension, and the render node file descriptor.
func QueryDRI3AndPresentExtensions(conn *x11client.Connection) (*dri3.Extension, *present.Extension, int, error) {
	dri3Adapter := NewDRI3ConnectionAdapter(conn)
	dri3Ext, err := dri3.QueryExtension(dri3Adapter)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("query DRI3: %w", err)
	}

	presentAdapter := NewPresentConnectionAdapter(conn)
	presentExt, err := present.QueryExtension(presentAdapter)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("query Present: %w", err)
	}

	root := conn.RootWindow()
	renderFd, err := dri3Ext.Open(dri3Adapter, dri3.XID(root), 0)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("open DRI3: %w", err)
	}

	return dri3Ext, presentExt, renderFd, nil
}

// CreateX11WindowWithDefaults creates an X11 window with standard demo settings.
// Window is positioned at (100, 100) with Exposure and KeyPress event masks.
func CreateX11WindowWithDefaults(conn *x11client.Connection, width, height int) (x11client.XID, error) {
	const (
		windowX     = 100
		windowY     = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0 // CopyFromParent
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}
	root := conn.RootWindow()

	wid, err := conn.CreateWindow(root, windowX, windowY, uint16(width), uint16(height), borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		return 0, fmt.Errorf("create window: %w", err)
	}

	if err := conn.MapWindow(wid); err != nil {
		return 0, fmt.Errorf("map window: %w", err)
	}

	return wid, nil
}

// CreatePixmapFromBuffer creates an X11 pixmap from a GPU buffer via DMA-BUF.
// This 35-line pattern is duplicated identically across 3 demos.
func CreatePixmapFromBuffer(conn *x11client.Connection, dri3Ext *dri3.Extension, window x11client.XID, buffer *render.BufferHandle, allocator *render.Allocator, depth, bpp int) (x11client.XID, error) {
	fd, err := allocator.ExportDmabuf(buffer)
	if err != nil {
		return 0, fmt.Errorf("export dmabuf: %w", err)
	}
	defer syscall.Close(fd)

	pixmapXID, err := conn.AllocXID()
	if err != nil {
		return 0, fmt.Errorf("allocate pixmap XID: %w", err)
	}

	size := buffer.Stride * buffer.Height
	dri3Adapter := NewDRI3ConnectionAdapter(conn)
	err = dri3Ext.PixmapFromBuffer(
		dri3Adapter,
		dri3.XID(pixmapXID),
		dri3.XID(window),
		size,
		uint16(buffer.Width),
		uint16(buffer.Height),
		uint16(buffer.Stride),
		uint8(depth),
		uint8(bpp),
		fd,
	)
	if err != nil {
		return 0, fmt.Errorf("create pixmap from buffer: %w", err)
	}

	return pixmapXID, nil
}

// PresentPixmapToWindow presents a pixmap to the specified window using the Present extension.
// This 10-line pattern is byte-for-byte identical across 3 demos.
func PresentPixmapToWindow(conn *x11client.Connection, presentExt *present.Extension, window, pixmap x11client.XID) error {
	presentAdapter := NewPresentConnectionAdapter(conn)
	err := presentExt.PresentPixmap(presentAdapter, present.PixmapPresentOptions{
		Window:  present.XID(window),
		Pixmap:  present.XID(pixmap),
		Options: present.PresentOptionNone,
	})
	if err != nil {
		return fmt.Errorf("present pixmap: %w", err)
	}

	return nil
}
