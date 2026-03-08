package integration

import (
	"github.com/opd-ai/wain/internal/buffer"
)

// X11BufferHandler wraps buffer release event handling with synchronizer integration
// for X11 Present extension PresentIdleNotify events.
type X11BufferHandler struct {
	pixmapXID uint32
	sync      *buffer.Synchronizer
	slotIndex int
}

// NewX11BufferHandler creates a handler for X11 Present IdleNotify events.
// The pixmapXID is the X11 Pixmap ID that will be used to identify the buffer
// in PresentIdleNotify events.
func NewX11BufferHandler(pixmapXID uint32, sync *buffer.Synchronizer, slotIndex int) (*X11BufferHandler, error) {
	// Register the pixmap XID with the synchronizer
	if err := sync.RegisterBuffer(pixmapXID, slotIndex); err != nil {
		return nil, err
	}

	return &X11BufferHandler{
		pixmapXID: pixmapXID,
		sync:      sync,
		slotIndex: slotIndex,
	}, nil
}

// HandleIdleNotify processes a PresentIdleNotify event and integrates with the synchronizer.
// Call this from your X11 event loop when receiving a PresentIdleNotify event.
func (x *X11BufferHandler) HandleIdleNotify(pixmapXID uint32) error {
	// Verify this is for our pixmap
	if pixmapXID != x.pixmapXID {
		return nil // Not our pixmap, ignore
	}

	// Notify the synchronizer that the buffer has been released
	return x.sync.OnReleaseEvent(x.pixmapXID)
}

// Cleanup unregisters the pixmap from the synchronizer.
// Call this before destroying the pixmap.
func (x *X11BufferHandler) Cleanup() {
	x.sync.UnregisterBuffer(x.pixmapXID)
}

// PixmapXID returns the X11 pixmap ID.
func (x *X11BufferHandler) PixmapXID() uint32 {
	return x.pixmapXID
}

// SlotIndex returns the ring slot index for this buffer.
func (x *X11BufferHandler) SlotIndex() int {
	return x.slotIndex
}
