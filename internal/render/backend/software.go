package backend

import (
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/raster/consumer"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// ErrSoftwareNoDmabuf is returned when Present() is called on software backend.
var ErrSoftwareNoDmabuf = errors.New("backend: software renderer does not export DMA-BUF")

// SoftwareBackend renders display lists using CPU rasterization.
//
// This is the fallback path when GPU rendering is unavailable or fails.
// It provides pixel-perfect output matching the GPU backends.
type SoftwareBackend struct {
	consumer *consumer.SoftwareConsumer
	buffer   *primitives.Buffer
	width    int
	height   int
}

// SoftwareConfig contains configuration for the software backend.
type SoftwareConfig struct {
	// Width is the render target width in pixels
	Width int

	// Height is the render target height in pixels
	Height int

	// Atlas is the font atlas for text rendering (optional)
	Atlas *text.Atlas
}

// NewSoftwareBackend creates a new software rendering backend.
func NewSoftwareBackend(cfg SoftwareConfig) (*SoftwareBackend, error) {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("backend: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}

	buffer, err := primitives.NewBuffer(cfg.Width, cfg.Height)
	if err != nil {
		return nil, fmt.Errorf("backend: buffer creation failed: %w", err)
	}

	consumer := consumer.NewSoftwareConsumer(cfg.Atlas)

	return &SoftwareBackend{
		consumer: consumer,
		buffer:   buffer,
		width:    cfg.Width,
		height:   cfg.Height,
	}, nil
}

// Render renders a display list to the software buffer.
func (sb *SoftwareBackend) Render(dl *displaylist.DisplayList) error {
	return sb.RenderWithDamage(dl, nil)
}

// RenderWithDamage renders a display list with optional damage tracking.
// For software rendering, damage tracking is implemented by clearing and
// re-rendering only the damaged regions.
func (sb *SoftwareBackend) RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error {
	if dl == nil {
		return ErrNilDisplayList
	}

	// If no damage specified, render full frame
	if len(damage) == 0 {
		return sb.consumer.Render(dl, sb.buffer)
	}

	// Filter commands to those intersecting damage regions and render each one.
	// This replaces the previous default: branch that fell back to rendering the
	// entire display list whenever a text, gradient, shadow, or image command was
	// present.
	commands := displaylist.FilterCommandsByDamage(dl.Commands(), damage)
	if len(commands) == 0 {
		return nil // No commands intersect damage regions
	}

	return sb.consumer.RenderCommands(commands, sb.buffer)
}

// Present returns -1 and an error since software rendering does not export DMA-BUF.
// The caller should access the pixel data directly via Buffer() method instead.
func (sb *SoftwareBackend) Present() (int, error) {
	return -1, ErrSoftwareNoDmabuf
}

// Buffer returns the underlying pixel buffer for direct access.
// This is the software equivalent of Present() for GPU backends.
func (sb *SoftwareBackend) Buffer() *primitives.Buffer {
	return sb.buffer
}

// Pixels returns the raw ARGB8888 pixel data from the software render buffer.
// Returns nil if the backend has not been initialized or has been destroyed.
func (sb *SoftwareBackend) Pixels() []byte {
	if sb.buffer == nil {
		return nil
	}
	return sb.buffer.Pixels
}

// Dimensions returns the render target dimensions.
func (sb *SoftwareBackend) Dimensions() (width, height int) {
	return sb.width, sb.height
}

// Destroy frees the software rendering resources.
func (sb *SoftwareBackend) Destroy() error {
	// Software buffer is GC'd automatically
	sb.buffer = nil
	sb.consumer = nil
	return nil
}
