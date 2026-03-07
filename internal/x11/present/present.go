// Package present implements the X11 Present extension for synchronized buffer presentation.
//
// The Present extension provides a way to display pixmaps at specific times with
// proper frame synchronization (vsync). It works in conjunction with DRI3 to enable
// zero-copy GPU rendering with tear-free presentation.
//
// # Extension Protocol
//
// Present adds several key operations:
//
//   - PresentQueryVersion: Query extension version and capabilities
//   - PresentPixmap: Schedule a pixmap for presentation at a specific time
//   - PresentNotifyMSC: Request notification at a specific media stream counter value
//   - PresentSelectInput: Register for present event notifications
//
// # Present Events
//
// The extension generates events for tracking presentation:
//   - PresentConfigureNotify: Window configuration changed
//   - PresentCompleteNotify: Pixmap presentation completed
//   - PresentIdleNotify: Pixmap is idle and can be reused
//
// # Typical Usage with DRI3
//
// 1. Client creates GPU buffer and pixmap via DRI3
// 2. Client renders to GPU buffer
// 3. Client calls PresentPixmap to schedule presentation
// 4. X server displays pixmap synchronized with vsync
// 5. Client receives PresentCompleteNotify when done
// 6. Client receives PresentIdleNotify when buffer can be reused
//
// This allows double/triple buffering with proper synchronization.
//
// # Version Support
//
// This implementation targets Present version 1.0, which provides:
//   - Basic synchronized presentation
//   - Event notifications for buffer lifecycle
//   - MSC (Media Stream Counter) based timing
//
// Version 1.2+ adds async flip support and other advanced features,
// but 1.0 is sufficient for basic tear-free rendering.
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations must be performed
// from the same goroutine that owns the X11 connection.
//
// Reference: https://gitlab.freedesktop.org/xorg/proto/xorgproto/-/blob/master/presentproto.txt
package present

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/x11/wire"
)

var (
	// ErrNotSupported is returned when Present extension is not available.
	ErrNotSupported = errors.New("present: extension not supported by X server")

	// ErrVersionTooOld is returned when server Present version is too old.
	ErrVersionTooOld = errors.New("present: server version too old (need 1.0+)")

	// ErrPresentFailed is returned when PresentPixmap operation fails.
	ErrPresentFailed = errors.New("present: pixmap presentation failed")

	// ErrInvalidSerial is returned when an invalid event serial is provided.
	ErrInvalidSerial = errors.New("present: invalid event serial")
)

const (
	// ExtensionName is the name as registered with X server.
	ExtensionName = "Present"
)

// Present request opcodes (relative to extension base opcode).
const (
	PresentQueryVersion   = 0
	PresentPixmap         = 1
	PresentNotifyMSC      = 2
	PresentSelectInput    = 3
	PresentQueryCapables  = 4 // Present 1.2+
)

// Present event codes (relative to extension base event).
const (
	PresentConfigureNotify = 0
	PresentCompleteNotify  = 1
	PresentIdleNotify      = 2
	PresentRedirectNotify  = 3 // Present 1.2+
)

// PresentOption flags for PresentPixmap request.
const (
	PresentOptionNone  = 0
	PresentOptionAsync = 1 << 0 // Present 1.2+: request async flip if possible
	PresentOptionCopy  = 1 << 1 // Present 1.2+: request copy instead of flip
	PresentOptionUST   = 1 << 2 // Present 1.2+: target_msc is in UST (microseconds)
	PresentOptionSuboptimal = 1 << 3 // Present 1.2+: presentation may be suboptimal
)

// CompleteKind indicates how the pixmap was presented.
type CompleteKind uint8

const (
	CompleteKindPixmap CompleteKind = 0 // Presented via pixmap (flip or copy)
	CompleteKindNotifyMSC CompleteKind = 1 // NotifyMSC event delivered
)

// CompleteMode indicates the presentation method used.
type CompleteMode uint8

const (
	CompleteModeFlip CompleteMode = 0 // Page flip
	CompleteModeCopy CompleteMode = 1 // Blit/copy
	CompleteModeSkip CompleteMode = 2 // Skipped (late)
)

// XID represents an X11 resource identifier.
type XID uint32

// Connection represents the minimal interface needed for Present operations.
type Connection interface {
	AllocXID() (XID, error)
	SendRequest(buf []byte) error
	SendRequestAndReply(req []byte) ([]byte, error)
	ExtensionOpcode(name string) (uint8, error)
}

// Extension represents the Present extension state.
type Extension struct {
	baseOpcode   uint8
	baseEvent    uint8
	supported    bool
	majorVersion uint32
	minorVersion uint32
}

// QueryExtension checks if Present extension is available on the X server.
// If available, it queries the extension version and capabilities.
func QueryExtension(conn Connection) (*Extension, error) {
	// Get extension opcode
	baseOpcode, err := conn.ExtensionOpcode(ExtensionName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotSupported, err)
	}

	ext := &Extension{
		baseOpcode: baseOpcode,
		supported:  true,
		// Note: baseEvent would need to be queried from connection,
		// but for now we'll handle events by opcode only
	}

	// Query extension version
	var buf bytes.Buffer
	wire.EncodeRequestHeader(&buf, baseOpcode+PresentQueryVersion, 0, 3)
	// Client version: major=1, minor=0
	wire.EncodeUint32(&buf, 1) // major version
	wire.EncodeUint32(&buf, 0) // minor version

	reply, err := conn.SendRequestAndReply(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("present: QueryVersion failed: %w", err)
	}

	// Parse reply: type(1) + pad(1) + sequence(2) + length(4) + major(4) + minor(4) + pad(16)
	if len(reply) < 32 {
		return nil, fmt.Errorf("present: invalid QueryVersion reply (got %d bytes)", len(reply))
	}

	ext.majorVersion = binary.LittleEndian.Uint32(reply[8:12])
	ext.minorVersion = binary.LittleEndian.Uint32(reply[12:16])

	// Verify version is at least 1.0
	if ext.majorVersion < 1 {
		return nil, ErrVersionTooOld
	}

	return ext, nil
}

// MajorVersion returns the negotiated Present major version.
func (e *Extension) MajorVersion() uint32 {
	return e.majorVersion
}

// MinorVersion returns the negotiated Present minor version.
func (e *Extension) MinorVersion() uint32 {
	return e.minorVersion
}

// SupportsAsync returns true if the server supports async flip mode (version 1.2+).
func (e *Extension) SupportsAsync() bool {
	return e.majorVersion > 1 || (e.majorVersion == 1 && e.minorVersion >= 2)
}

// PresentPixmap schedules a pixmap for presentation.
//
// Parameters:
//   - window: the target window for presentation
//   - pixmap: the pixmap to present (created via DRI3 or standard X11)
//   - serial: client serial number for matching with events (arbitrary, for client use)
//   - validRegion: region of pixmap with valid content (0 = entire pixmap)
//   - updateRegion: region that changed since last present (0 = unknown/entire)
//   - xOff, yOff: offset within window for pixmap placement
//   - targetMSC: target media stream counter value (0 = next vblank)
//   - divisor, remainder: for periodic updates (0,0 = ASAP)
//   - options: PresentOption flags (typically PresentOptionNone)
//
// The server will present the pixmap synchronized to the specified MSC value.
// For typical double-buffering, use targetMSC=0 to present at next vblank.
//
// After presentation, the server sends PresentCompleteNotify event with matching serial.
// When the buffer is no longer needed, server sends PresentIdleNotify.
func (e *Extension) PresentPixmap(conn Connection, window, pixmap XID,
	serial uint32, validRegion, updateRegion XID,
	xOff, yOff int16, targetMSC uint64,
	divisor, remainder uint64, options uint32,
) error {
	var buf bytes.Buffer

	// PresentPixmap request (version 1.0):
	// header(4) + window(4) + pixmap(4) + serial(4) +
	// valid(4) + update(4) + x_off(2) + y_off(2) +
	// target_crtc(4) + wait_fence(4) + idle_fence(4) +
	// options(4) + target_msc(8) + divisor(8) + remainder(8) + notifies_len(4)
	//
	// Total: 68 bytes = 17 * 4-byte units
	wire.EncodeRequestHeader(&buf, e.baseOpcode+PresentPixmap, 0, 18)
	wire.EncodeUint32(&buf, uint32(window))
	wire.EncodeUint32(&buf, uint32(pixmap))
	wire.EncodeUint32(&buf, serial)
	wire.EncodeUint32(&buf, uint32(validRegion))
	wire.EncodeUint32(&buf, uint32(updateRegion))
	wire.EncodeInt16(&buf, xOff)
	wire.EncodeInt16(&buf, yOff)
	wire.EncodeUint32(&buf, 0) // target_crtc (0 = any CRTC)
	wire.EncodeUint32(&buf, 0) // wait_fence (0 = none)
	wire.EncodeUint32(&buf, 0) // idle_fence (0 = none)
	wire.EncodeUint32(&buf, options)
	wire.EncodeUint64(&buf, targetMSC)
	wire.EncodeUint64(&buf, divisor)
	wire.EncodeUint64(&buf, remainder)
	wire.EncodeUint32(&buf, 0) // notifies_len (no notify list)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("%w: %v", ErrPresentFailed, err)
	}

	return nil
}

// SelectInput registers the client to receive Present events for a window.
//
// Parameters:
//   - eid: event ID to use for notifications (from AllocXID)
//   - window: the window to monitor
//   - eventMask: bitmask of desired events (see PresentEventMask constants)
//
// After calling SelectInput, the client will receive PresentCompleteNotify,
// PresentIdleNotify, and other events as they occur.
func (e *Extension) SelectInput(conn Connection, eid, window XID, eventMask uint32) error {
	var buf bytes.Buffer

	// PresentSelectInput request: header(4) + eid(4) + window(4) + event_mask(4)
	// Total: 16 bytes = 4 * 4-byte units
	wire.EncodeRequestHeader(&buf, e.baseOpcode+PresentSelectInput, 0, 4)
	wire.EncodeUint32(&buf, uint32(eid))
	wire.EncodeUint32(&buf, uint32(window))
	wire.EncodeUint32(&buf, eventMask)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("present: SelectInput failed: %w", err)
	}

	return nil
}

// PresentEventMask flags for SelectInput.
const (
	PresentEventMaskNone            = 0
	PresentEventMaskConfigureNotify = 1 << 0
	PresentEventMaskCompleteNotify  = 1 << 1
	PresentEventMaskIdleNotify      = 1 << 2
	PresentEventMaskRedirectNotify  = 1 << 3 // Present 1.2+
)

// NotifyMSC requests notification when the display reaches a specific MSC value.
//
// Parameters:
//   - window: the window to monitor
//   - serial: client serial for matching with event
//   - targetMSC: MSC value to wait for
//   - divisor, remainder: for periodic notifications (0,0 = one-shot)
//
// When the MSC reaches the target, server sends PresentCompleteNotify with
// kind=CompleteKindNotifyMSC and the specified serial.
func (e *Extension) NotifyMSC(conn Connection, window XID,
	serial uint32, targetMSC, divisor, remainder uint64,
) error {
	var buf bytes.Buffer

	// PresentNotifyMSC request: header(4) + window(4) + serial(4) + pad(4) +
	// target_msc(8) + divisor(8) + remainder(8)
	// Total: 40 bytes = 10 * 4-byte units
	wire.EncodeRequestHeader(&buf, e.baseOpcode+PresentNotifyMSC, 0, 10)
	wire.EncodeUint32(&buf, uint32(window))
	wire.EncodeUint32(&buf, serial)
	wire.EncodeUint32(&buf, 0) // pad
	wire.EncodeUint64(&buf, targetMSC)
	wire.EncodeUint64(&buf, divisor)
	wire.EncodeUint64(&buf, remainder)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("present: NotifyMSC failed: %w", err)
	}

	return nil
}

// CompleteNotifyEvent represents a PresentCompleteNotify event.
type CompleteNotifyEvent struct {
	Kind     CompleteKind // How presentation completed
	Mode     CompleteMode // Method used (flip/copy/skip)
	Serial   uint32       // Client-provided serial from PresentPixmap
	UST      uint64       // Presentation timestamp (microseconds)
	MSC      uint64       // Media stream counter value
}

// IdleNotifyEvent represents a PresentIdleNotify event.
type IdleNotifyEvent struct {
	Pixmap XID // Pixmap that is now idle
}

// ParseCompleteNotify parses a PresentCompleteNotify event.
//
// Event format (32 bytes):
//   - type(1) + extension(1) + sequence(2) + length(4)
//   - kind(1) + mode(1) + pad(2) + serial(4)
//   - window(4) + pixmap(4)
//   - ust(8) + msc(8)
func ParseCompleteNotify(data []byte) (*CompleteNotifyEvent, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("present: invalid CompleteNotify event (got %d bytes)", len(data))
	}

	evt := &CompleteNotifyEvent{
		Kind:   CompleteKind(data[8]),
		Mode:   CompleteMode(data[9]),
		Serial: binary.LittleEndian.Uint32(data[12:16]),
		// window: data[16:20] (not extracted)
		// pixmap: data[20:24] (not extracted)
		UST:    binary.LittleEndian.Uint64(data[24:32]),
	}

	// MSC is in next 8 bytes if present
	if len(data) >= 40 {
		evt.MSC = binary.LittleEndian.Uint64(data[32:40])
	}

	return evt, nil
}

// ParseIdleNotify parses a PresentIdleNotify event.
//
// Event format (32 bytes):
//   - type(1) + extension(1) + sequence(2) + length(4)
//   - event(4) + window(4) + serial(4) + pixmap(4)
//   - idle_fence(4) + pad(8)
func ParseIdleNotify(data []byte) (*IdleNotifyEvent, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("present: invalid IdleNotify event (got %d bytes)", len(data))
	}

	evt := &IdleNotifyEvent{
		Pixmap: XID(binary.LittleEndian.Uint32(data[20:24])),
	}

	return evt, nil
}
