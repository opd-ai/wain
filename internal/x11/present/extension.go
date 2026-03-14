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
	// PresentQueryVersion queries the extension version.
	PresentQueryVersion = 0
	// PresentPixmap presents a pixmap to a window.
	PresentPixmap = 1
	// PresentNotifyMSC requests notification at a specific MSC.
	PresentNotifyMSC = 2
	// PresentSelectInput selects input events to receive.
	PresentSelectInput = 3
	// PresentQueryCapables queries presentation capabilities (Present 1.2+).
	PresentQueryCapables = 4
)

// Present event codes (relative to extension base event).
const (
	// PresentConfigureNotify indicates a configure event occurred.
	PresentConfigureNotify = 0
	// PresentCompleteNotify indicates presentation completed.
	PresentCompleteNotify = 1
	// PresentIdleNotify indicates a pixmap became idle.
	PresentIdleNotify = 2
	// PresentRedirectNotify indicates a redirect occurred (Present 1.2+).
	PresentRedirectNotify = 3
)

// PresentOption flags for PresentPixmap request.
const (
	// PresentOptionNone uses default presentation behavior.
	PresentOptionNone = 0
	// PresentOptionAsync requests async flip if possible (Present 1.2+).
	PresentOptionAsync = 1 << 0
	// PresentOptionCopy requests copy instead of flip (Present 1.2+).
	PresentOptionCopy = 1 << 1
	// PresentOptionUST treats target_msc as UST (microseconds) (Present 1.2+).
	PresentOptionUST = 1 << 2
	// PresentOptionSuboptimal allows suboptimal presentation (Present 1.2+).
	PresentOptionSuboptimal = 1 << 3
)

// CompleteKind indicates how the pixmap was presented.
type CompleteKind uint8

// Present completion kind constants.
const (
	// CompleteKindPixmap indicates presentation via pixmap (flip or copy).
	CompleteKindPixmap CompleteKind = 0
	// CompleteKindNotifyMSC indicates a NotifyMSC event was delivered.
	CompleteKindNotifyMSC CompleteKind = 1
)

// CompleteMode indicates the presentation method used.
type CompleteMode uint8

// Present completion mode constants.
const (
	// CompleteModeFlip indicates a page flip was performed.
	CompleteModeFlip CompleteMode = 0
	// CompleteModeCopy indicates a blit/copy was performed.
	CompleteModeCopy CompleteMode = 1
	// CompleteModeSkip indicates the presentation was skipped (frame arrived too late).
	CompleteModeSkip CompleteMode = 2
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
	// Query extension using common helper
	info, err := wire.QueryExtensionVersion(conn, ExtensionName, PresentQueryVersion, 1, 0)
	if err != nil {
		return nil, err
	}

	// Verify version is at least 1.0
	if info.MajorVersion < 1 {
		return nil, ErrVersionTooOld
	}

	return &Extension{
		baseOpcode:   info.BaseOpcode,
		supported:    true,
		majorVersion: info.MajorVersion,
		minorVersion: info.MinorVersion,
		// Note: baseEvent would need to be queried from connection,
		// but for now we'll handle events by opcode only
	}, nil
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

// PixmapPresentOptions holds parameters for presenting a pixmap to a window.
//
// For typical double-buffering use:
//   - Serial: arbitrary client value for event tracking
//   - TargetMSC: 0 (present at next vblank)
//   - Options: PresentOptionNone
//   - All other fields: zero values
type PixmapPresentOptions struct {
	// Window is the target window for presentation
	Window XID
	// Pixmap is the pixmap to present (created via DRI3 or standard X11)
	Pixmap XID
	// Serial is a client serial number for matching with events (arbitrary)
	Serial uint32
	// ValidRegion is the region of pixmap with valid content (0 = entire pixmap)
	ValidRegion XID
	// UpdateRegion is the region that changed since last present (0 = unknown/entire)
	UpdateRegion XID
	// XOff is the X offset within window for pixmap placement
	XOff int16
	// YOff is the Y offset within window for pixmap placement
	YOff int16
	// TargetMSC is the target media stream counter value (0 = next vblank)
	TargetMSC uint64
	// Divisor is used for periodic updates (0 = ASAP)
	Divisor uint64
	// Remainder is used for periodic updates (0 = ASAP)
	Remainder uint64
	// Options contains PresentOption flags (typically PresentOptionNone)
	Options uint32
}

// PresentPixmap schedules a pixmap for presentation using the provided options.
//
// The server will present the pixmap synchronized to the specified MSC value.
// For typical double-buffering, use TargetMSC=0 to present at next vblank.
//
// After presentation, the server sends PresentCompleteNotify event with matching Serial.
// When the buffer is no longer needed, server sends PresentIdleNotify.
func (e *Extension) PresentPixmap(conn Connection, opts PixmapPresentOptions) error {
	var buf bytes.Buffer

	// PresentPixmap request (version 1.0):
	// header(4) + window(4) + pixmap(4) + serial(4) +
	// valid(4) + update(4) + x_off(2) + y_off(2) +
	// target_crtc(4) + wait_fence(4) + idle_fence(4) +
	// options(4) + target_msc(8) + divisor(8) + remainder(8) + notifies_len(4)
	//
	// Total: 68 bytes = 17 * 4-byte units
	wire.EncodeRequestHeader(&buf, e.baseOpcode+PresentPixmap, 0, 18)
	wire.EncodeUint32(&buf, uint32(opts.Window))
	wire.EncodeUint32(&buf, uint32(opts.Pixmap))
	wire.EncodeUint32(&buf, opts.Serial)
	wire.EncodeUint32(&buf, uint32(opts.ValidRegion))
	wire.EncodeUint32(&buf, uint32(opts.UpdateRegion))
	wire.EncodeInt16(&buf, opts.XOff)
	wire.EncodeInt16(&buf, opts.YOff)
	wire.EncodeUint32(&buf, 0) // target_crtc (0 = any CRTC)
	wire.EncodeUint32(&buf, 0) // wait_fence (0 = none)
	wire.EncodeUint32(&buf, 0) // idle_fence (0 = none)
	wire.EncodeUint32(&buf, opts.Options)
	_ = wire.EncodeUint64(&buf, opts.TargetMSC)
	_ = wire.EncodeUint64(&buf, opts.Divisor)
	wire.EncodeUint64(&buf, opts.Remainder)
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
	// PresentEventMaskNone selects no events.
	PresentEventMaskNone = 0
	// PresentEventMaskConfigureNotify selects configure events.
	PresentEventMaskConfigureNotify = 1 << 0
	// PresentEventMaskCompleteNotify selects completion events.
	PresentEventMaskCompleteNotify = 1 << 1
	// PresentEventMaskIdleNotify selects idle events.
	PresentEventMaskIdleNotify = 1 << 2
	// PresentEventMaskRedirectNotify selects redirect events (Present 1.2+).
	PresentEventMaskRedirectNotify = 1 << 3
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
	encodeNotifyMSCTiming(&buf, targetMSC, divisor, remainder)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("present: NotifyMSC failed: %w", err)
	}

	return nil
}

func encodeNotifyMSCTiming(buf *bytes.Buffer, targetMSC, divisor, remainder uint64) {
	wire.EncodeUint64(buf, targetMSC)
	wire.EncodeUint64(buf, divisor)
	wire.EncodeUint64(buf, remainder)
}

// CompleteNotifyEvent represents a PresentCompleteNotify event.
type CompleteNotifyEvent struct {
	Kind   CompleteKind // How presentation completed
	Mode   CompleteMode // Method used (flip/copy/skip)
	Serial uint32       // Client-provided serial from PresentPixmap
	UST    uint64       // Presentation timestamp (microseconds)
	MSC    uint64       // Media stream counter value
}

// IdleNotifyEvent represents a PresentIdleNotify event.
type IdleNotifyEvent struct {
	Pixmap XID // Pixmap that is now idle
}

// ParseCompleteNotify parses a PresentCompleteNotify event.
//
// Currently unused - reserved for advanced presentation timing analysis.
// Would enable tracking presentation completion callbacks for frame pacing optimization.
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
		UST: binary.LittleEndian.Uint64(data[24:32]),
	}

	// MSC is in next 8 bytes if present
	if len(data) >= 40 {
		evt.MSC = binary.LittleEndian.Uint64(data[32:40])
	}

	return evt, nil
}

// ParseIdleNotify parses a PresentIdleNotify event.
//
// Currently unused - reserved for advanced buffer synchronization.
// Would enable tracking pixmap idle notifications for precise buffer reuse timing.
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
