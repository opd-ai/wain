package client

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Compositor represents a wl_compositor object.
//
// The compositor is a singleton object that provides the ability to create
// surfaces for rendering. It is the primary factory for wl_surface objects.
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_compositor
type Compositor struct {
	baseObject
	version uint32
}

const (
	compositorOpcodeCreateSurface uint16 = 0
	compositorOpcodeCreateRegion  uint16 = 1
)

// CreateSurface creates a new wl_surface object.
// Surfaces are the primary drawable objects in Wayland.
func (c *Compositor) CreateSurface() (*Surface, error) {
	surfaceID := c.conn.allocID()

	surface := &Surface{
		baseObject: baseObject{
			id:    surfaceID,
			iface: "wl_surface",
			conn:  c.conn,
		},
	}

	c.conn.registerObject(surface)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: surfaceID},
	}

	if err := c.conn.sendRequest(c.id, compositorOpcodeCreateSurface, args); err != nil {
		return nil, fmt.Errorf("compositor: create_surface failed: %w", err)
	}

	return surface, nil
}

// Surface represents a wl_surface object.
//
// A surface is a rectangular area that can be displayed on screen. It has a
// double-buffered state that is applied atomically on commit. Content is
// attached to surfaces via buffers.
//
// Basic rendering flow:
//  1. Attach a buffer (from wl_shm or dmabuf)
//  2. Mark damaged regions
//  3. Commit to apply changes
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_surface
type Surface struct {
	baseObject
}

const (
	surfaceOpcodeDestroy            uint16 = 0
	surfaceOpcodeAttach             uint16 = 1
	surfaceOpcodeDamage             uint16 = 2
	surfaceOpcodeFrame              uint16 = 3
	surfaceOpcodeSetOpaqueRegion    uint16 = 4
	surfaceOpcodeSetInputRegion     uint16 = 5
	surfaceOpcodeCommit             uint16 = 6
	surfaceOpcodeSetBufferTransform uint16 = 7
	surfaceOpcodeSetBufferScale     uint16 = 8
	surfaceOpcodeDamageBuffer       uint16 = 9
)

// Attach attaches a buffer to this surface.
//
// The buffer provides the pixel content for the surface. Attaching a buffer
// does not make it visible immediately; Commit must be called to apply the
// pending state.
//
// Parameters:
//   - buffer: object ID of the wl_buffer (0 to detach)
//   - x, y: buffer attachment offset (usually 0, 0)
func (s *Surface) Attach(buffer uint32, x, y int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeObject, Value: buffer},
		{Type: wire.ArgTypeInt32, Value: x},
		{Type: wire.ArgTypeInt32, Value: y},
	}

	if err := s.conn.sendRequest(s.id, surfaceOpcodeAttach, args); err != nil {
		return fmt.Errorf("surface: attach failed: %w", err)
	}

	return nil
}

// Damage marks a region of the surface as damaged.
//
// Damaged regions indicate areas that have changed since the last commit and
// need to be redrawn. Coordinates are in surface-local space.
//
// Parameters:
//   - x, y: top-left corner of the damaged region
//   - width, height: size of the damaged region
func (s *Surface) Damage(x, y, width, height int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: x},
		{Type: wire.ArgTypeInt32, Value: y},
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
	}

	if err := s.conn.sendRequest(s.id, surfaceOpcodeDamage, args); err != nil {
		return fmt.Errorf("surface: damage failed: %w", err)
	}

	return nil
}

// Frame requests a frame callback to be notified when the compositor is ready
// for the next frame.
//
// The compositor will send a done event on the returned callback when it's
// time to render the next frame. This is used to synchronize rendering with
// the compositor's refresh cycle.
func (s *Surface) Frame() (*Callback, error) {
	callbackID := s.conn.allocID()

	cb := &Callback{
		baseObject: baseObject{
			id:    callbackID,
			iface: "wl_callback",
			conn:  s.conn,
		},
		doneChan: make(chan uint32, 1),
	}

	s.conn.registerObject(cb)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: callbackID},
	}

	if err := s.conn.sendRequest(s.id, surfaceOpcodeFrame, args); err != nil {
		return nil, fmt.Errorf("surface: frame failed: %w", err)
	}

	return cb, nil
}

// Commit atomically applies all pending surface state.
//
// Commit makes all pending state (attached buffers, damage regions, etc.)
// take effect. The surface state is double-buffered: changes are accumulated
// and only become visible when committed.
func (s *Surface) Commit() error {
	if err := s.conn.sendRequest(s.id, surfaceOpcodeCommit, nil); err != nil {
		return fmt.Errorf("surface: commit failed: %w", err)
	}

	return nil
}

// Destroy destroys the surface and releases its resources.
func (s *Surface) Destroy() error {
	if err := s.conn.sendRequest(s.id, surfaceOpcodeDestroy, nil); err != nil {
		return fmt.Errorf("surface: destroy failed: %w", err)
	}

	// Remove from connection's object registry.
	delete(s.conn.objects, s.id)

	return nil
}

// SetBufferScale sets the buffer scale for this surface.
//
// The buffer scale indicates the ratio between buffer pixels and surface
// coordinates. Used for HiDPI support.
//
// Parameters:
//   - scale: buffer scale factor (typically 1, 2, or 3)
func (s *Surface) SetBufferScale(scale int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: scale},
	}

	if err := s.conn.sendRequest(s.id, surfaceOpcodeSetBufferScale, args); err != nil {
		return fmt.Errorf("surface: set_buffer_scale failed: %w", err)
	}

	return nil
}
