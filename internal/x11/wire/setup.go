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

	binary.Write(&buf, binary.LittleEndian, req.ProtocolMajorVersion)
	binary.Write(&buf, binary.LittleEndian, req.ProtocolMinorVersion)

	authNameLen := uint16(len(req.AuthName))
	authDataLen := uint16(len(req.AuthData))

	binary.Write(&buf, binary.LittleEndian, authNameLen)
	binary.Write(&buf, binary.LittleEndian, authDataLen)
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
	return err
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
func decodeSetupFailure(r io.Reader, reply *SetupReply) error {
	reasonLen, _ := DecodeUint8(r)
	reply.ProtocolMajorVersion, _ = DecodeUint16(r)
	reply.ProtocolMinorVersion, _ = DecodeUint16(r)
	failDataLen, _ := DecodeUint16(r)

	if reasonLen > 0 {
		reason := make([]byte, reasonLen)
		io.ReadFull(r, reason)
		return fmt.Errorf("%w: %s", ErrSetupFailed, string(reason))
	}

	if failDataLen > 0 {
		io.CopyN(io.Discard, r, int64(failDataLen)*4)
	}
	return ErrSetupFailed
}

// decodePixmapFormats reads pixmap format entries.
func decodePixmapFormats(r io.Reader, count int) ([]PixmapFormat, error) {
	formats := make([]PixmapFormat, count)
	for i := 0; i < count; i++ {
		formats[i].Depth, _ = DecodeUint8(r)
		formats[i].BitsPerPixel, _ = DecodeUint8(r)
		formats[i].ScanlinePad, _ = DecodeUint8(r)
		io.CopyN(io.Discard, r, 5)
	}
	return formats, nil
}

// decodeVisuals reads visual entries for a depth.
func decodeVisuals(r io.Reader, count int) ([]Visual, error) {
	visuals := make([]Visual, count)
	for k := 0; k < count; k++ {
		visuals[k].ID, _ = DecodeUint32(r)
		class, _ := DecodeUint8(r)
		visuals[k].Class = VisualClass(class)
		visuals[k].BitsPerRGB, _ = DecodeUint8(r)
		visuals[k].Colormap, _ = DecodeUint16(r)
		visuals[k].RedMask, _ = DecodeUint32(r)
		visuals[k].GreenMask, _ = DecodeUint32(r)
		visuals[k].BlueMask, _ = DecodeUint32(r)
		io.CopyN(io.Discard, r, 4)
	}
	return visuals, nil
}

// decodeDepths reads depth entries for a screen.
func decodeDepths(r io.Reader, count int) ([]Depth, error) {
	depths := make([]Depth, count)
	for j := 0; j < count; j++ {
		depths[j].Depth, _ = DecodeUint8(r)
		io.CopyN(io.Discard, r, 1)
		numVisuals, _ := DecodeUint16(r)
		io.CopyN(io.Discard, r, 4)

		visuals, err := decodeVisuals(r, int(numVisuals))
		if err != nil {
			return nil, err
		}
		depths[j].Visuals = visuals
	}
	return depths, nil
}

// decodeScreen reads a single screen entry.
func decodeScreen(r io.Reader) (Screen, error) {
	var screen Screen
	decodeScreenFields(r, &screen)
	saveUnders, _ := DecodeUint8(r)
	screen.SaveUnders = saveUnders != 0
	screen.RootDepth, _ = DecodeUint8(r)
	numDepths, _ := DecodeUint8(r)

	depths, err := decodeDepths(r, int(numDepths))
	if err != nil {
		return screen, err
	}
	screen.Depths = depths
	return screen, nil
}

// decodeScreenFields reads basic screen fields, ignoring errors for performance.
func decodeScreenFields(r io.Reader, s *Screen) {
	decode5Uint32(r, &s.Root, &s.DefaultColormap, &s.WhitePixel, &s.BlackPixel, &s.CurrentMasks)
	decode6Uint16(r, &s.WidthPixels, &s.HeightPixels, &s.WidthMM, &s.HeightMM, &s.MinMaps, &s.MaxMaps)
	s.RootVisual, _ = DecodeUint32(r)
	s.BackingStores, _ = DecodeUint8(r)
}

// decode5Uint32 reads 5 consecutive uint32 values.
func decode5Uint32(r io.Reader, v1, v2, v3, v4, v5 *uint32) {
	*v1, _ = DecodeUint32(r)
	*v2, _ = DecodeUint32(r)
	*v3, _ = DecodeUint32(r)
	*v4, _ = DecodeUint32(r)
	*v5, _ = DecodeUint32(r)
}

// decode6Uint16 reads 6 consecutive uint16 values.
func decode6Uint16(r io.Reader, v1, v2, v3, v4, v5, v6 *uint16) {
	*v1, _ = DecodeUint16(r)
	*v2, _ = DecodeUint16(r)
	*v3, _ = DecodeUint16(r)
	*v4, _ = DecodeUint16(r)
	*v5, _ = DecodeUint16(r)
	*v6, _ = DecodeUint16(r)
}

// decode6Uint8 reads 6 consecutive uint8 values.
func decode6Uint8(r io.Reader, v1, v2, v3, v4, v5, v6 *uint8) {
	*v1, _ = DecodeUint8(r)
	*v2, _ = DecodeUint8(r)
	*v3, _ = DecodeUint8(r)
	*v4, _ = DecodeUint8(r)
	*v5, _ = DecodeUint8(r)
	*v6, _ = DecodeUint8(r)
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

	vendorLen, numScreens, numFormats := decodeSetupBody(r, &reply)

	if err := decodeVendorString(r, &reply, vendorLen); err != nil {
		return reply, err
	}

	formats, err := decodePixmapFormats(r, int(numFormats))
	if err != nil {
		return reply, err
	}
	reply.PixmapFormats = formats

	screens := make([]Screen, numScreens)
	for i := 0; i < int(numScreens); i++ {
		screen, err := decodeScreen(r)
		if err != nil {
			return reply, err
		}
		screens[i] = screen
	}
	reply.Screens = screens

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

// decodeSetupBody reads the main setup reply fields, returning vendor/screen/format counts.
func decodeSetupBody(r io.Reader, reply *SetupReply) (vendorLen uint16, numScreens, numFormats uint8) {
	_, _ = DecodeUint8(r)
	reply.ProtocolMajorVersion, _ = DecodeUint16(r)
	reply.ProtocolMinorVersion, _ = DecodeUint16(r)
	_, _ = DecodeUint16(r)

	reply.ReleaseNumber, _ = DecodeUint32(r)
	reply.ResourceIDBase, _ = DecodeUint32(r)
	reply.ResourceIDMask, _ = DecodeUint32(r)
	reply.MotionBufferSize, _ = DecodeUint32(r)

	vendorLen, _ = DecodeUint16(r)
	reply.MaxRequestLength, _ = DecodeUint16(r)

	numScreens, _ = DecodeUint8(r)
	numFormats, _ = DecodeUint8(r)

	decode6Uint8(r, &reply.ImageByteOrder, &reply.BitmapBitOrder, &reply.BitmapScanlineUnit,
		&reply.BitmapScanlinePad, &reply.MinKeycode, &reply.MaxKeycode)

	io.CopyN(io.Discard, r, 4)
	return
}

// decodeVendorString reads the vendor string if present.
func decodeVendorString(r io.Reader, reply *SetupReply, vendorLen uint16) error {
	if vendorLen > 0 {
		vendor := make([]byte, vendorLen)
		io.ReadFull(r, vendor)
		reply.Vendor = string(vendor)

		if pad := Pad(int(vendorLen)); pad > 0 {
			io.CopyN(io.Discard, r, int64(pad))
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
