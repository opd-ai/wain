// Package output implements the Wayland wl_output interface for display information.
//
// The wl_output interface provides information about physical displays, including
// geometry, mode, scale factor, and other properties. This is essential for HiDPI
// support and multi-monitor setups.
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_output
package output

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Output represents a wl_output object for a physical display.
type Output struct {
	id      uint32
	conn    Connection
	version uint32

	// Display properties
	geometry Geometry
	mode     Mode
	scale    int32
	done     bool
}

// Connection interface for sending Wayland protocol messages.
type Connection interface {
	sendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// Geometry contains physical display properties.
type Geometry struct {
	X         int32  // X position in compositor space
	Y         int32  // Y position in compositor space
	PhysicalW int32  // Width in millimeters
	PhysicalH int32  // Height in millimeters
	Subpixel  int32  // Subpixel orientation
	Make      string // Display manufacturer
	Model     string // Display model
	Transform int32  // Output transform
}

// Mode contains display mode information.
type Mode struct {
	Flags   uint32 // Mode flags (current, preferred)
	Width   int32  // Width in pixels
	Height  int32  // Height in pixels
	Refresh int32  // Refresh rate in mHz
}

// Mode flags for output modes.
const (
	// ModeFlagCurrent indicates the mode is currently active on the output.
	ModeFlagCurrent uint32 = 0x1
	// ModeFlagPreferred indicates the mode is preferred by the output.
	ModeFlagPreferred uint32 = 0x2
)

// Subpixel orientations describe how subpixels are arranged on the display.
const (
	// SubpixelUnknown indicates the subpixel arrangement is unknown.
	SubpixelUnknown int32 = 0
	// SubpixelNone indicates no subpixel arrangement.
	SubpixelNone int32 = 1
	// SubpixelHorizontalRGB indicates horizontal RGB subpixel arrangement.
	SubpixelHorizontalRGB int32 = 2
	// SubpixelHorizontalBGR indicates horizontal BGR subpixel arrangement.
	SubpixelHorizontalBGR int32 = 3
	// SubpixelVerticalRGB indicates vertical RGB subpixel arrangement.
	SubpixelVerticalRGB int32 = 4
	// SubpixelVerticalBGR indicates vertical BGR subpixel arrangement.
	SubpixelVerticalBGR int32 = 5
)

// Transform values describe how the output content is transformed.
const (
	// TransformNormal indicates no transformation is applied.
	TransformNormal int32 = 0
	// Transform90 indicates the output is rotated 90 degrees clockwise.
	Transform90 int32 = 1
	// Transform180 indicates the output is rotated 180 degrees.
	Transform180 int32 = 2
	// Transform270 indicates the output is rotated 270 degrees clockwise.
	Transform270 int32 = 3
	// TransformFlipped indicates the output is flipped horizontally.
	TransformFlipped int32 = 4
	// TransformFlipped90 indicates the output is flipped and rotated 90 degrees.
	TransformFlipped90 int32 = 5
	// TransformFlipped180 indicates the output is flipped and rotated 180 degrees.
	TransformFlipped180 int32 = 6
	// TransformFlipped270 indicates the output is flipped and rotated 270 degrees.
	TransformFlipped270 int32 = 7
)

const (
	outputOpcodeRelease uint16 = 0
)

const (
	outputEventGeometry uint16 = 0
	outputEventMode     uint16 = 1
	outputEventDone     uint16 = 2
	outputEventScale    uint16 = 3
)

// New creates a new Output object.
func New(id uint32, conn Connection, version uint32) *Output {
	return &Output{
		id:      id,
		conn:    conn,
		version: version,
		scale:   1, // Default scale is 1
	}
}

// ID returns the object ID.
func (o *Output) ID() uint32 {
	return o.id
}

// Interface returns the interface name.
func (o *Output) Interface() string {
	return "wl_output"
}

// Scale returns the current display scale factor.
func (o *Output) Scale() int32 {
	return o.scale
}

// Geometry returns the current display geometry.
func (o *Output) Geometry() Geometry {
	return o.geometry
}

// Mode returns the current display mode.
func (o *Output) Mode() Mode {
	return o.mode
}

// Release releases the output object.
func (o *Output) Release() error {
	if err := o.conn.sendRequest(o.id, outputOpcodeRelease, nil); err != nil {
		return fmt.Errorf("output: release failed: %w", err)
	}
	return nil
}

// HandleEvent processes events from the compositor.
func (o *Output) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case outputEventGeometry:
		return o.handleGeometry(args)
	case outputEventMode:
		return o.handleMode(args)
	case outputEventDone:
		return o.handleDone(args)
	case outputEventScale:
		return o.handleScale(args)
	default:
		return fmt.Errorf("output: unknown event opcode %d", opcode)
	}
}

// handleGeometry processes a geometry event.
func (o *Output) handleGeometry(args []wire.Argument) error {
	if len(args) != 8 {
		return fmt.Errorf("output: geometry event requires 8 args, got %d", len(args))
	}

	geom, err := parseGeometryArgs(args)
	if err != nil {
		return err
	}

	o.geometry = geom
	return nil
}

// geomDecoder provides sequential argument decoding with accumulated error state.
// If any decode call fails, subsequent calls become no-ops and the error is
// preserved for retrieval via err().
type geomDecoder struct {
	args []wire.Argument
	idx  int
	err  error
}

// int32Field decodes the next argument as int32, recording any type mismatch.
func (d *geomDecoder) int32Field(name string) int32 {
	if d.err != nil {
		return 0
	}
	val, err := getInt32Arg(d.args[d.idx], name)
	d.idx++
	d.err = err
	return val
}

// strField decodes the next argument as string, recording any type mismatch.
func (d *geomDecoder) strField(name string) string {
	if d.err != nil {
		return ""
	}
	val, err := getStringArg(d.args[d.idx], name)
	d.idx++
	d.err = err
	return val
}

// parseGeometryArgs extracts and validates geometry arguments.
func parseGeometryArgs(args []wire.Argument) (Geometry, error) {
	dec := &geomDecoder{args: args}
	g := Geometry{
		X:         dec.int32Field("x"),
		Y:         dec.int32Field("y"),
		PhysicalW: dec.int32Field("physical_width"),
		PhysicalH: dec.int32Field("physical_height"),
		Subpixel:  dec.int32Field("subpixel"),
		Make:      dec.strField("make"),
		Model:     dec.strField("model"),
		Transform: dec.int32Field("transform"),
	}
	return g, dec.err
}

// getInt32Arg extracts an int32 argument with error handling.
func getInt32Arg(arg wire.Argument, name string) (int32, error) {
	val, ok := arg.Value.(int32)
	if !ok {
		return 0, fmt.Errorf("output: invalid %s type", name)
	}
	return val, nil
}

// getStringArg extracts a string argument with error handling.
func getStringArg(arg wire.Argument, name string) (string, error) {
	val, ok := arg.Value.(string)
	if !ok {
		return "", fmt.Errorf("output: invalid %s type", name)
	}
	return val, nil
}

// handleMode processes a mode event.
func (o *Output) handleMode(args []wire.Argument) error {
	if len(args) != 4 {
		return fmt.Errorf("output: mode event requires 4 args, got %d", len(args))
	}

	flags, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("output: invalid flags type")
	}
	width, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("output: invalid width type")
	}
	height, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("output: invalid height type")
	}
	refresh, ok := args[3].Value.(int32)
	if !ok {
		return fmt.Errorf("output: invalid refresh type")
	}

	o.mode = Mode{
		Flags:   flags,
		Width:   width,
		Height:  height,
		Refresh: refresh,
	}

	return nil
}

// handleDone processes a done event.
func (o *Output) handleDone(args []wire.Argument) error {
	if len(args) != 0 {
		return fmt.Errorf("output: done event requires 0 args, got %d", len(args))
	}
	o.done = true
	return nil
}

// handleScale processes a scale event.
func (o *Output) handleScale(args []wire.Argument) error {
	if len(args) != 1 {
		return fmt.Errorf("output: scale event requires 1 arg, got %d", len(args))
	}

	scale, ok := args[0].Value.(int32)
	if !ok {
		return fmt.Errorf("output: invalid scale type")
	}

	o.scale = scale
	return nil
}
