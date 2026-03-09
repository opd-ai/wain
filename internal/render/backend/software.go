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

	// Filter commands to those intersecting damage regions
	commands := displaylist.FilterCommandsByDamage(dl.Commands(), damage)
	if len(commands) == 0 {
		return nil // No commands intersect damage regions
	}

	// Create a temporary display list with filtered commands
	// and render it to the buffer
	for _, cmd := range commands {
		// Manually reconstruct the display list by examining command types
		switch cmd.Type {
		case displaylist.CmdFillRect:
			data := cmd.Data.(displaylist.FillRectData)
			sb.buffer.FillRect(data.X, data.Y, data.Width, data.Height, data.Color)
		case displaylist.CmdFillRoundedRect:
			data := cmd.Data.(displaylist.FillRoundedRectData)
			sb.buffer.FillRoundedRect(data.X, data.Y, data.Width, data.Height, float64(data.Radius), data.Color)
		case displaylist.CmdDrawLine:
			data := cmd.Data.(displaylist.DrawLineData)
			sb.buffer.DrawLine(data.X0, data.Y0, data.X1, data.Y1, float64(data.Width), data.Color)
		default:
			// Other command types would need more complex filtering
			// For now, just render the full display list
			return sb.consumer.Render(dl, sb.buffer)
		}
	}

	return nil
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
