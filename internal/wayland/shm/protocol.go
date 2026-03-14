package shm

import (
	"context"
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Pixel format constants for wl_shm.
const (
	// FormatARGB8888 is the ARGB8888 pixel format.
	FormatARGB8888 = 0
	// FormatXRGB8888 is the XRGB8888 pixel format (no alpha).
	FormatXRGB8888 = 1
)

// Conn represents the subset of client.Connection methods needed by SHM objects.
// This interface allows SHM objects to interact with the Wayland connection
// without creating a circular dependency on the client package.
type Conn interface {
	AllocID() uint32
	RegisterObject(obj interface{})
	SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// objectBase provides common fields for SHM-related Wayland objects.
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

// SHM represents the wl_shm global interface for shared memory support.
type SHM struct {
	objectBase
	formats     map[uint32]bool
	formatsChan chan uint32
}

// NewSHM creates a new SHM object from a registry binding.
func NewSHM(conn Conn, id uint32) *SHM {
	return &SHM{
		objectBase: objectBase{
			id:    id,
			iface: "wl_shm",
			conn:  conn,
		},
		formats:     make(map[uint32]bool),
		formatsChan: make(chan uint32, 16),
	}
}

// WaitForFormats waits for the compositor to announce supported pixel formats.
func (s *SHM) WaitForFormats(ctx context.Context) error {
	for {
		select {
		case format := <-s.formatsChan:
			s.formats[format] = true
		case <-ctx.Done():
			return ctx.Err()
		default:
			return s.checkFormatsAvailable()
		}
	}
}

// checkFormatsAvailable returns nil if at least one format has been received,
// or an error if none have arrived yet.
func (s *SHM) checkFormatsAvailable() error {
	if len(s.formats) > 0 {
		return nil
	}
	return fmt.Errorf("no formats received")
}

// HasFormat checks if a pixel format is supported by the compositor.
func (s *SHM) HasFormat(format uint32) bool {
	return s.formats[format]
}

// HandleEvent processes events from the compositor for this SHM object.
func (s *SHM) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // format event
		if len(args) != 1 {
			return fmt.Errorf("format event: expected 1 argument, got %d", len(args))
		}
		if args[0].Type != wire.ArgTypeUint32 {
			return fmt.Errorf("format event: expected uint32 argument")
		}
		format := args[0].Value.(uint32)
		s.formatsChan <- format
		return nil
	default:
		return fmt.Errorf("unknown wl_shm event opcode: %d", opcode)
	}
}

// CreatePool creates a new shared memory pool.
func (s *SHM) CreatePool(fd int, size int32) (*Pool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid pool size: %d", size)
	}

	poolID := s.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: poolID},
		{Type: wire.ArgTypeFD, Value: int32(fd)},
		{Type: wire.ArgTypeInt32, Value: size},
	}

	if err := s.conn.SendRequest(s.id, 0, args); err != nil {
		return nil, fmt.Errorf("create_pool request failed: %w", err)
	}

	pool := NewPool(s.conn, poolID, fd, size)
	s.conn.RegisterObject(pool)

	return pool, nil
}
