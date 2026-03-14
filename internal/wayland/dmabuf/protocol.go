// Package dmabuf implements the zwp_linux_dmabuf_v1 Wayland protocol extension.
//
// This extension allows clients to create wl_buffers backed by DMA-BUF file descriptors,
// enabling zero-copy sharing of GPU-allocated buffers with the compositor.
//
// Protocol reference: https://wayland.app/protocols/linux-dmabuf-unstable-v1
package dmabuf

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// DRM fourcc format codes (subset relevant for UI rendering).
const (
	// FormatARGB8888 is 'AR24' - ARGB 8:8:8:8.
	FormatARGB8888 = 0x34325241
	// FormatXRGB8888 is 'XR24' - XRGB 8:8:8:8 (no alpha).
	FormatXRGB8888 = 0x34325258
	// FormatABGR8888 is 'AB24' - ABGR 8:8:8:8.
	FormatABGR8888 = 0x34324241
	// FormatXBGR8888 is 'XB24' - XBGR 8:8:8:8.
	FormatXBGR8888 = 0x34324258
)

// Modifier constants.
const (
	// ModifierLinear indicates no tiling.
	ModifierLinear = 0x0000000000000000
	// ModifierInvalid is an invalid modifier sentinel.
	ModifierInvalid = 0x00ffffffffffffff
)

// Flags for buffer creation.
const (
	// FlagYInvert indicates Y-axis is inverted (bottom-up).
	FlagYInvert = 1
	// FlagInterlaced indicates buffer contains interlaced data.
	FlagInterlaced = 2
	// FlagBottomFirst indicates bottom field first.
	FlagBottomFirst = 4
)

// Conn represents the subset of client.Connection methods needed by dmabuf objects.
type Conn interface {
	AllocID() uint32
	RegisterObject(obj interface{})
	SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// objectBase provides common fields for dmabuf Wayland objects.
type objectBase struct {
	id    uint32
	iface string
	conn  Conn
}

// ID returns the object's unique identifier.
func (o *objectBase) ID() uint32 {
	return o.id
}

// Interface returns the Wayland interface name.
func (o *objectBase) Interface() string {
	return o.iface
}

// Dmabuf represents the zwp_linux_dmabuf_v1 global interface.
type Dmabuf struct {
	objectBase
	formats map[uint32][]uint64 // format -> list of modifiers
}

// NewDmabuf creates a new Dmabuf object from a registry binding.
func NewDmabuf(conn Conn, id uint32) *Dmabuf {
	return &Dmabuf{
		objectBase: objectBase{
			id:    id,
			iface: "zwp_linux_dmabuf_v1",
			conn:  conn,
		},
		formats: make(map[uint32][]uint64),
	}
}

// CreateParams creates a temporary object for constructing buffer parameters.
//
// Returns a BufferParams object that can be used to add planes and create
// a wl_buffer from DMA-BUF file descriptors.
func (d *Dmabuf) CreateParams() (*BufferParams, error) {
	paramsID := d.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: paramsID},
	}

	if err := d.conn.SendRequest(d.id, 2, args); err != nil {
		return nil, fmt.Errorf("dmabuf: create_params failed: %w", err)
	}

	params := &BufferParams{
		objectBase: objectBase{
			id:    paramsID,
			iface: "zwp_linux_buffer_params_v1",
			conn:  d.conn,
		},
	}

	d.conn.RegisterObject(params)

	return params, nil
}

// HandleEvent processes events from the compositor for this dmabuf object.
func (d *Dmabuf) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0:
		return d.handleFormatEvent(args)
	case 1:
		return d.handleModifierEvent(args)
	default:
		return fmt.Errorf("dmabuf: unknown event opcode: %d", opcode)
	}
}

// handleFormatEvent processes the deprecated format event (zwp_linux_dmabuf_v1 v1/v2).
// This handler is retained for backward compatibility with compositors that have not yet
// upgraded to v3. Removing it would silently break DMA-BUF support on older wlroots,
// Mutter/GNOME Shell < 44, and any compositor still advertising zwp_linux_dmabuf_v1
// below version 3. Remove only after the minimum required compositor version is formally
// raised to v3 across all supported distributions.
func (d *Dmabuf) handleFormatEvent(args []wire.Argument) error {
	if len(args) != 1 {
		return fmt.Errorf("dmabuf: format event: expected 1 argument, got %d", len(args))
	}
	if args[0].Type != wire.ArgTypeUint32 {
		return fmt.Errorf("dmabuf: format event: expected uint32 argument")
	}
	format := args[0].Value.(uint32)
	if _, exists := d.formats[format]; !exists {
		d.formats[format] = []uint64{ModifierLinear}
	}
	return nil
}

// handleModifierEvent processes the modifier event (zwp_linux_dmabuf_v1 v3+).
func (d *Dmabuf) handleModifierEvent(args []wire.Argument) error {
	if len(args) != 3 {
		return fmt.Errorf("dmabuf: modifier event: expected 3 arguments, got %d", len(args))
	}
	if args[0].Type != wire.ArgTypeUint32 || args[1].Type != wire.ArgTypeUint32 || args[2].Type != wire.ArgTypeUint32 {
		return fmt.Errorf("dmabuf: modifier event: invalid argument types")
	}
	format := args[0].Value.(uint32)
	modHi := uint64(args[1].Value.(uint32))
	modLo := uint64(args[2].Value.(uint32))
	d.formats[format] = append(d.formats[format], (modHi<<32)|modLo)
	return nil
}

// HasFormat checks if a format is supported by the compositor.
func (d *Dmabuf) HasFormat(format uint32) bool {
	_, ok := d.formats[format]
	return ok
}

// HasFormatModifier checks if a specific format+modifier combination is supported.
func (d *Dmabuf) HasFormatModifier(format uint32, modifier uint64) bool {
	modifiers, ok := d.formats[format]
	if !ok {
		return false
	}
	for _, m := range modifiers {
		if m == modifier {
			return true
		}
	}
	return false
}

// BufferParams represents zwp_linux_buffer_params_v1, used to construct DMA-BUF backed buffers.
type BufferParams struct {
	objectBase
}

// Add adds a plane to this buffer.
//
// Parameters:
//   - fd: DMA-BUF file descriptor for this plane
//   - planeIdx: plane index (0 for single-plane formats like ARGB8888)
//   - offset: offset in bytes within the DMA-BUF
//   - stride: stride in bytes
//   - modifierHi: upper 32 bits of format modifier
//   - modifierLo: lower 32 bits of format modifier
//
// The file descriptor is consumed by this call and must not be closed by the client.
func (p *BufferParams) Add(fd int32, planeIdx, offset, stride, modifierHi, modifierLo uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeFD, Value: fd},
		{Type: wire.ArgTypeUint32, Value: planeIdx},
		{Type: wire.ArgTypeUint32, Value: offset},
		{Type: wire.ArgTypeUint32, Value: stride},
		{Type: wire.ArgTypeUint32, Value: modifierHi},
		{Type: wire.ArgTypeUint32, Value: modifierLo},
	}

	if err := p.conn.SendRequest(p.id, 0, args); err != nil {
		return fmt.Errorf("buffer_params: add failed: %w", err)
	}

	return nil
}

// Create creates a wl_buffer from the added planes.
//
// Parameters:
//   - width: buffer width in pixels
//   - height: buffer height in pixels
//   - format: DRM fourcc format code
//   - flags: buffer creation flags (FlagYInvert, etc.)
//
// Returns the wl_buffer object ID. The BufferParams object is destroyed after this call.
func (p *BufferParams) Create(width, height int32, format, flags uint32) (uint32, error) {
	bufferID := p.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: bufferID},
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
		{Type: wire.ArgTypeUint32, Value: format},
		{Type: wire.ArgTypeUint32, Value: flags},
	}

	if err := p.conn.SendRequest(p.id, 1, args); err != nil {
		return 0, fmt.Errorf("buffer_params: create failed: %w", err)
	}

	return bufferID, nil
}

// CreateImmed immediately creates a wl_buffer from the added planes.
//
// This is a synchronous variant of Create that doesn't wait for the compositor
// to validate the parameters. Use with caution - invalid parameters will cause
// a protocol error.
func (p *BufferParams) CreateImmed(width, height int32, format, flags uint32) (uint32, error) {
	bufferID := p.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: bufferID},
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
		{Type: wire.ArgTypeUint32, Value: format},
		{Type: wire.ArgTypeUint32, Value: flags},
	}

	if err := p.conn.SendRequest(p.id, 2, args); err != nil {
		return 0, fmt.Errorf("buffer_params: create_immed failed: %w", err)
	}

	return bufferID, nil
}

// Destroy destroys this params object without creating a buffer.
func (p *BufferParams) Destroy() error {
	args := []wire.Argument{}

	if err := p.conn.SendRequest(p.id, 3, args); err != nil {
		return fmt.Errorf("buffer_params: destroy failed: %w", err)
	}

	return nil
}

// HandleEvent processes events for BufferParams.
func (p *BufferParams) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // created event
		// Buffer was created successfully
		// The wl_buffer object ID was already allocated in Create()
		return nil

	case 1: // failed event
		// Buffer creation failed due to invalid parameters
		return fmt.Errorf("buffer_params: buffer creation failed")

	default:
		return fmt.Errorf("buffer_params: unknown event opcode: %d", opcode)
	}
}
