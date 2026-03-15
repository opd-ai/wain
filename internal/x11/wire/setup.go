package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrSetupFailed is returned when connection setup fails.
	ErrSetupFailed = errors.New("wire: setup failed")

	// ErrAuthFailed is returned when authentication fails.
	ErrAuthFailed = errors.New("wire: authentication failed")
)

const (
	// ByteOrderLSB indicates little-endian byte order.
	ByteOrderLSB = 0x6C

	// ByteOrderMSB indicates big-endian byte order.
	ByteOrderMSB = 0x42

	// ProtocolMajorVersion is the X11 protocol major version.
	ProtocolMajorVersion = 11

	// ProtocolMinorVersion is the X11 protocol minor version.
	ProtocolMinorVersion = 0
)

// SetupRequest represents the client's connection setup message.
type SetupRequest struct {
	ByteOrder            byte
	ProtocolMajorVersion uint16
	ProtocolMinorVersion uint16
	AuthName             string
	AuthData             []byte
}

// EncodeSetupRequest writes a setup request to w.
func EncodeSetupRequest(w io.Writer, req SetupRequest) error {
	var buf bytes.Buffer

	// Header
	buf.WriteByte(req.ByteOrder)
	buf.WriteByte(0) // Padding

	_ = binary.Write(&buf, binary.LittleEndian, req.ProtocolMajorVersion)
	_ = binary.Write(&buf, binary.LittleEndian, req.ProtocolMinorVersion)

	authNameLen := uint16(len(req.AuthName))
	authDataLen := uint16(len(req.AuthData))

	_ = binary.Write(&buf, binary.LittleEndian, authNameLen)
	_ = binary.Write(&buf, binary.LittleEndian, authDataLen)
	buf.WriteByte(0) // Padding
	buf.WriteByte(0) // Padding

	// Auth name with padding
	buf.WriteString(req.AuthName)
	if pad := Pad(len(req.AuthName)); pad > 0 {
		buf.Write(make([]byte, pad))
	}

	// Auth data with padding
	buf.Write(req.AuthData)
	if pad := Pad(len(req.AuthData)); pad > 0 {
		buf.Write(make([]byte, pad))
	}

	_, err := w.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("x11/wire: encode setup request: %w", err)
	}
	return nil
}

// SetupStatus represents the status of a setup reply.
type SetupStatus uint8

// X11 connection setup status constants.
const (
	// SetupStatusFailed indicates the connection setup failed.
	SetupStatusFailed SetupStatus = 0
	// SetupStatusSuccess indicates the connection setup succeeded.
	SetupStatusSuccess SetupStatus = 1
	// SetupStatusAuthenticate indicates authentication is required.
	SetupStatusAuthenticate SetupStatus = 2
)

// VisualClass represents the visual type.
type VisualClass uint8

// X11 visual class constants identifying color model types.
const (
	// VisualClassStaticGray represents a static grayscale visual.
	VisualClassStaticGray VisualClass = 0
	// VisualClassGrayScale represents a dynamic grayscale visual.
	VisualClassGrayScale VisualClass = 1
	// VisualClassStaticColor represents a static indexed color visual.
	VisualClassStaticColor VisualClass = 2
	// VisualClassPseudoColor represents a dynamic indexed color visual.
	VisualClassPseudoColor VisualClass = 3
	// VisualClassTrueColor represents a static RGB visual (most common).
	VisualClassTrueColor VisualClass = 4
	// VisualClassDirectColor represents a dynamic RGB visual.
	VisualClassDirectColor VisualClass = 5
)

// Visual represents a visual type configuration.
type Visual struct {
	ID         uint32
	Class      VisualClass
	BitsPerRGB uint8
	Colormap   uint16
	RedMask    uint32
	GreenMask  uint32
	BlueMask   uint32
}

// Depth represents a depth configuration.
type Depth struct {
	Depth   uint8
	Visuals []Visual
}

// Screen represents an X11 screen configuration.
type Screen struct {
	Root            uint32
	DefaultColormap uint32
	WhitePixel      uint32
	BlackPixel      uint32
	CurrentMasks    uint32
	WidthPixels     uint16
	HeightPixels    uint16
	WidthMM         uint16
	HeightMM        uint16
	MinMaps         uint16
	MaxMaps         uint16
	RootVisual      uint32
	BackingStores   uint8
	SaveUnders      bool
	RootDepth       uint8
	Depths          []Depth
}

// SetupReply represents the server's response to setup.
type SetupReply struct {
	Status               SetupStatus
	ProtocolMajorVersion uint16
	ProtocolMinorVersion uint16
	ReleaseNumber        uint32
	ResourceIDBase       uint32
	ResourceIDMask       uint32
	MotionBufferSize     uint32
	MaxRequestLength     uint16
	ImageByteOrder       uint8
	BitmapBitOrder       uint8
	BitmapScanlineUnit   uint8
	BitmapScanlinePad    uint8
	MinKeycode           uint8
	MaxKeycode           uint8
	Vendor               string
	PixmapFormats        []PixmapFormat
	Screens              []Screen
}

// PixmapFormat represents a pixmap format.
type PixmapFormat struct {
	Depth        uint8
	BitsPerPixel uint8
	ScanlinePad  uint8
}

// decodeSetupFailure reads and handles a setup failure response.
// readFailureReason reads the reason string from a setup failure response.
// Returns a wrapped ErrSetupFailed with the reason message, or nil if reasonLen is zero.
func readFailureReason(r io.Reader, reasonLen uint8) error {
	if reasonLen == 0 {
		return nil
	}
	reason := make([]byte, reasonLen)
	if _, err := io.ReadFull(r, reason); err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}
	return fmt.Errorf("%w: %s", ErrSetupFailed, string(reason))
}

// skipFailureData discards trailing failure data padding from the setup stream.
func skipFailureData(r io.Reader, failDataLen uint16) error {
	if failDataLen == 0 {
		return nil
	}
	if _, err := io.CopyN(io.Discard, r, int64(failDataLen)*4); err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}
	return nil
}

func decodeSetupFailure(r io.Reader, reply *SetupReply) error {
	reasonLen, err := DecodeUint8(r)
	if err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}
	if reply.ProtocolMajorVersion, err = DecodeUint16(r); err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}
	if reply.ProtocolMinorVersion, err = DecodeUint16(r); err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}
	failDataLen, err := DecodeUint16(r)
	if err != nil {
		return fmt.Errorf("setup failure decode: %w", err)
	}

	if err := readFailureReason(r, reasonLen); err != nil {
		return fmt.Errorf("x11/wire: decode setup failure reason: %w", err)
	}
	if err := skipFailureData(r, failDataLen); err != nil {
		return fmt.Errorf("x11/wire: decode setup failure data: %w", err)
	}
	return ErrSetupFailed
}

// decodePixmapFormats reads pixmap format entries.
func decodePixmapFormats(r io.Reader, count int) ([]PixmapFormat, error) {
	formats := make([]PixmapFormat, count)
	for i := 0; i < count; i++ {
		var err error
		if formats[i].Depth, err = DecodeUint8(r); err != nil {
			return nil, fmt.Errorf("pixmap format decode: %w", err)
		}
		if formats[i].BitsPerPixel, err = DecodeUint8(r); err != nil {
			return nil, fmt.Errorf("pixmap format decode: %w", err)
		}
		if formats[i].ScanlinePad, err = DecodeUint8(r); err != nil {
			return nil, fmt.Errorf("pixmap format decode: %w", err)
		}
		if _, err := io.CopyN(io.Discard, r, 5); err != nil {
			return nil, fmt.Errorf("pixmap format decode: %w", err)
		}
	}
	return formats, nil
}

// decodeSingleVisual reads one Visual entry from the X11 setup stream.
func decodeSingleVisual(r io.Reader) (Visual, error) {
	er := &seqReader{r: r}
	var v Visual
	v.ID = er.uint32()
	v.Class = VisualClass(er.uint8())
	v.BitsPerRGB = er.uint8()
	v.Colormap = er.uint16()
	v.RedMask = er.uint32()
	v.GreenMask = er.uint32()
	v.BlueMask = er.uint32()
	er.discard(4)
	if er.err != nil {
		return Visual{}, fmt.Errorf("visual decode: %w", er.err)
	}
	return v, nil
}

// seqReader accumulates the first error across sequential decode calls so that
// callers can defer error checking to the end of a field-by-field decode block.
type seqReader struct {
	r   io.Reader
	err error
}

func (s *seqReader) uint8() uint8 {
	if s.err != nil {
		return 0
	}
	v, err := DecodeUint8(s.r)
	s.err = err
	return v
}

func (s *seqReader) uint16() uint16 {
	if s.err != nil {
		return 0
	}
	v, err := DecodeUint16(s.r)
	s.err = err
	return v
}

func (s *seqReader) uint32() uint32 {
	if s.err != nil {
		return 0
	}
	v, err := DecodeUint32(s.r)
	s.err = err
	return v
}

func (s *seqReader) discard(n int64) {
	if s.err != nil {
		return
	}
	_, s.err = io.CopyN(io.Discard, s.r, n)
}

// decodeVisuals reads visual entries for a depth.
func decodeVisuals(r io.Reader, count int) ([]Visual, error) {
	visuals := make([]Visual, count)
	for k := 0; k < count; k++ {
		v, err := decodeSingleVisual(r)
		if err != nil {
			return nil, err
		}
		visuals[k] = v
	}
	return visuals, nil
}

// decodeDepths reads depth entries for a screen.
func decodeDepths(r io.Reader, count int) ([]Depth, error) {
	depths := make([]Depth, count)
	for j := 0; j < count; j++ {
		d, err := decodeSingleDepth(r)
		if err != nil {
			return nil, err
		}
		depths[j] = d
	}
	return depths, nil
}

// decodeSingleDepth reads one depth entry along with its associated visuals.
func decodeSingleDepth(r io.Reader) (Depth, error) {
	var d Depth
	var err error
	if d.Depth, err = DecodeUint8(r); err != nil {
		return d, fmt.Errorf("depth decode: %w", err)
	}
	if _, err = io.CopyN(io.Discard, r, 1); err != nil {
		return d, fmt.Errorf("depth decode: %w", err)
	}
	numVisuals, err := DecodeUint16(r)
	if err != nil {
		return d, fmt.Errorf("depth decode: %w", err)
	}
	if _, err = io.CopyN(io.Discard, r, 4); err != nil {
		return d, fmt.Errorf("depth decode: %w", err)
	}
	visuals, err := decodeVisuals(r, int(numVisuals))
	if err != nil {
		return d, err
	}
	d.Visuals = visuals
	return d, nil
}

// decodeScreen reads a single screen entry.
func decodeScreen(r io.Reader) (Screen, error) {
	var screen Screen
	if err := decodeScreenFields(r, &screen); err != nil {
		return screen, err
	}
	saveUnders, err := DecodeUint8(r)
	if err != nil {
		return screen, fmt.Errorf("screen decode: %w", err)
	}
	screen.SaveUnders = saveUnders != 0
	if screen.RootDepth, err = DecodeUint8(r); err != nil {
		return screen, fmt.Errorf("screen decode: %w", err)
	}
	numDepths, err := DecodeUint8(r)
	if err != nil {
		return screen, fmt.Errorf("screen decode: %w", err)
	}

	depths, err := decodeDepths(r, int(numDepths))
	if err != nil {
		return screen, err
	}
	screen.Depths = depths
	return screen, nil
}

// decodeScreenFields reads basic screen fields.
func decodeScreenFields(r io.Reader, s *Screen) error {
	if err := decode5Uint32(r, &s.Root, &s.DefaultColormap, &s.WhitePixel, &s.BlackPixel, &s.CurrentMasks); err != nil {
		return fmt.Errorf("x11/wire: decode screen fields 5uint32: %w", err)
	}
	if err := decode6Uint16(r, &s.WidthPixels, &s.HeightPixels, &s.WidthMM, &s.HeightMM, &s.MinMaps, &s.MaxMaps); err != nil {
		return fmt.Errorf("x11/wire: decode screen fields 6uint16: %w", err)
	}
	var err error
	if s.RootVisual, err = DecodeUint32(r); err != nil {
		return fmt.Errorf("screen fields decode: %w", err)
	}
	if s.BackingStores, err = DecodeUint8(r); err != nil {
		return fmt.Errorf("screen fields decode: %w", err)
	}
	return nil
}

// decodeValues reads consecutive values from r using the provided decode function,
// storing each result in the corresponding pointer.
func decodeValues[T uint8 | uint16 | uint32](r io.Reader, decode func(io.Reader) (T, error), vs ...*T) error {
	for _, v := range vs {
		val, err := decode(r)
		if err != nil {
			return fmt.Errorf("x11/wire: decode values: %w", err)
		}
		*v = val
	}
	return nil
}

// decode5Uint32 reads 5 consecutive uint32 values.
func decode5Uint32(r io.Reader, v1, v2, v3, v4, v5 *uint32) error {
	return decodeValues(r, DecodeUint32, v1, v2, v3, v4, v5)
}

// decode6Uint16 reads 6 consecutive uint16 values.
func decode6Uint16(r io.Reader, v1, v2, v3, v4, v5, v6 *uint16) error {
	return decodeValues(r, DecodeUint16, v1, v2, v3, v4, v5, v6)
}

// decode6Uint8 reads 6 consecutive uint8 values.
func decode6Uint8(r io.Reader, v1, v2, v3, v4, v5, v6 *uint8) error {
	return decodeValues(r, DecodeUint8, v1, v2, v3, v4, v5, v6)
}

// decodeScreenList reads numScreens Screen entries from the X11 setup stream.
func decodeScreenList(r io.Reader, numScreens uint8) ([]Screen, error) {
	screens := make([]Screen, numScreens)
	for i := 0; i < int(numScreens); i++ {
		screen, err := decodeScreen(r)
		if err != nil {
			return nil, err
		}
		screens[i] = screen
	}
	return screens, nil
}

// DecodeSetupReply reads the server's setup reply from r.
func DecodeSetupReply(r io.Reader) (SetupReply, error) {
	var reply SetupReply

	if err := decodeSetupHeader(r, &reply); err != nil {
		return reply, err
	}

	if reply.Status == SetupStatusFailed {
		return reply, decodeSetupFailure(r, &reply)
	}

	vendorLen, numScreens, numFormats, err := decodeSetupBody(r, &reply)
	if err != nil {
		return reply, err
	}

	if err := decodeVendorString(r, &reply, vendorLen); err != nil {
		return reply, err
	}

	formats, err := decodePixmapFormats(r, int(numFormats))
	if err != nil {
		return reply, err
	}
	reply.PixmapFormats = formats

	reply.Screens, err = decodeScreenList(r, numScreens)
	if err != nil {
		return reply, err
	}

	return reply, nil
}

// decodeSetupHeader reads and validates the initial setup response.
func decodeSetupHeader(r io.Reader, reply *SetupReply) error {
	status, err := DecodeUint8(r)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSetupFailed, err)
	}
	reply.Status = SetupStatus(status)
	return nil
}

// decodeSetupProtocolVersion reads protocol version fields from setup reply.
func decodeSetupProtocolVersion(r io.Reader, reply *SetupReply) error {
	if _, err := DecodeUint8(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	}
	if major, err := DecodeUint16(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.ProtocolMajorVersion = major
	}
	if minor, err := DecodeUint16(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.ProtocolMinorVersion = minor
	}
	if _, err := DecodeUint16(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	}
	return nil
}

// decodeSetupResourceFields reads resource ID and buffer size fields from setup reply.
func decodeSetupResourceFields(r io.Reader, reply *SetupReply) error {
	if release, err := DecodeUint32(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.ReleaseNumber = release
	}
	if base, err := DecodeUint32(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.ResourceIDBase = base
	}
	if mask, err := DecodeUint32(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.ResourceIDMask = mask
	}
	if bufSize, err := DecodeUint32(r); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.MotionBufferSize = bufSize
	}
	return nil
}

// decodeSetupCounts reads vendor length, max request length, and screen/format counts.
func decodeSetupCounts(r io.Reader, reply *SetupReply) (vendorLen uint16, numScreens, numFormats uint8, err error) {
	if vendorLen, err = DecodeUint16(r); err != nil {
		return 0, 0, 0, fmt.Errorf("setup body decode: %w", err)
	}
	if maxReq, err := DecodeUint16(r); err != nil {
		return 0, 0, 0, fmt.Errorf("setup body decode: %w", err)
	} else {
		reply.MaxRequestLength = maxReq
	}
	if numScreens, err = DecodeUint8(r); err != nil {
		return 0, 0, 0, fmt.Errorf("setup body decode: %w", err)
	}
	if numFormats, err = DecodeUint8(r); err != nil {
		return 0, 0, 0, fmt.Errorf("setup body decode: %w", err)
	}
	return vendorLen, numScreens, numFormats, nil
}

// decodeSetupByteOrderAndKeys reads byte order, bitmap, and keycode fields.
func decodeSetupByteOrderAndKeys(r io.Reader, reply *SetupReply) error {
	if err := decode6Uint8(r, &reply.ImageByteOrder, &reply.BitmapBitOrder, &reply.BitmapScanlineUnit,
		&reply.BitmapScanlinePad, &reply.MinKeycode, &reply.MaxKeycode); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	}
	if _, err := io.CopyN(io.Discard, r, 4); err != nil {
		return fmt.Errorf("setup body decode: %w", err)
	}
	return nil
}

// decodeSetupBody reads the main setup reply fields, returning vendor/screen/format counts.
func decodeSetupBody(r io.Reader, reply *SetupReply) (vendorLen uint16, numScreens, numFormats uint8, err error) {
	if err = decodeSetupProtocolVersion(r, reply); err != nil {
		return 0, 0, 0, err
	}
	if err = decodeSetupResourceFields(r, reply); err != nil {
		return 0, 0, 0, err
	}
	if vendorLen, numScreens, numFormats, err = decodeSetupCounts(r, reply); err != nil {
		return 0, 0, 0, err
	}
	if err = decodeSetupByteOrderAndKeys(r, reply); err != nil {
		return 0, 0, 0, err
	}
	return vendorLen, numScreens, numFormats, nil
}

// decodeVendorString reads the vendor string if present.
func decodeVendorString(r io.Reader, reply *SetupReply, vendorLen uint16) error {
	if vendorLen > 0 {
		vendor := make([]byte, vendorLen)
		if _, err := io.ReadFull(r, vendor); err != nil {
			return fmt.Errorf("vendor string decode: %w", err)
		}
		reply.Vendor = string(vendor)

		if pad := Pad(int(vendorLen)); pad > 0 {
			if _, err := io.CopyN(io.Discard, r, int64(pad)); err != nil {
				return fmt.Errorf("vendor string decode: %w", err)
			}
		}
	}
	return nil
}

// ReadAuthority reads the MIT-MAGIC-COOKIE-1 from .Xauthority file.
func ReadAuthority(display string) (string, []byte, error) {
	authFile, err := openAuthFile()
	if err != nil {
		return "", nil, err
	}
	defer authFile.Close()

	for {
		entry, err := readAuthEntry(authFile)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("wire: error reading authority: %w", err)
		}

		if entry.display == display && entry.name == "MIT-MAGIC-COOKIE-1" {
			return entry.name, entry.data, nil
		}
	}

	return "", nil, fmt.Errorf("%w: no authority found for display %s", ErrAuthFailed, display)
}

// openAuthFile opens the .Xauthority file.
func openAuthFile() (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("wire: cannot determine home directory: %w", err)
	}

	authFile := filepath.Join(home, ".Xauthority")
	f, err := os.Open(authFile)
	if err != nil {
		return nil, fmt.Errorf("wire: cannot open %s: %w", authFile, err)
	}
	return f, nil
}

// authEntry represents a single .Xauthority entry.
type authEntry struct {
	display string
	name    string
	data    []byte
}

// readAuthEntry reads a single authority entry from the file.
func readAuthEntry(f *os.File) (authEntry, error) {
	var entry authEntry
	var family uint16
	if err := binary.Read(f, binary.BigEndian, &family); err != nil {
		return entry, err
	}

	_, err := readLengthPrefixedBytes(f)
	if err != nil {
		return entry, err
	}

	disp, err := readLengthPrefixedBytes(f)
	if err != nil {
		return entry, err
	}
	entry.display = string(disp)

	name, err := readLengthPrefixedBytes(f)
	if err != nil {
		return entry, err
	}
	entry.name = string(name)

	data, err := readLengthPrefixedBytes(f)
	if err != nil {
		return entry, err
	}
	entry.data = data

	return entry, nil
}

// readLengthPrefixedBytes reads a uint16 length followed by that many bytes.
func readLengthPrefixedBytes(f *os.File) ([]byte, error) {
	var length uint16
	if err := binary.Read(f, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
