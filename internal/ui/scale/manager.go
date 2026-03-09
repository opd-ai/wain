// Package scale provides unified scale factor management for HiDPI/high-resolution displays.
//
// The scale package manages display scale factors across both Wayland and X11 backends,
// providing a consistent API for querying and updating the current scale factor.
package scale

import (
	"sync"
)

// Manager manages the current display scale factor.
type Manager struct {
	mu    sync.RWMutex
	scale float32
}

// NewManager creates a new scale manager with default scale 1.0.
func NewManager() *Manager {
	return &Manager{
		scale: 1.0,
	}
}

// Get returns the current scale factor.
func (m *Manager) Get() float32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scale
}

// Set updates the current scale factor.
func (m *Manager) Set(scale float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if scale <= 0 {
		scale = 1.0
	}
	m.scale = scale
}

// SetFromInt updates the scale factor from an integer value (Wayland scale).
func (m *Manager) SetFromInt(scale int32) {
	m.Set(float32(scale))
}

// SetFromDPI updates the scale factor from DPI value (X11).
// 96 DPI = 1.0 scale, 192 DPI = 2.0 scale, etc.
func (m *Manager) SetFromDPI(dpi int32) {
	m.Set(float32(dpi) / 96.0)
}

// ScaleInt scales an integer value by the current scale factor.
func (m *Manager) ScaleInt(value int) int {
	return int(float32(value) * m.Get())
}

// ScaleFloat scales a float value by the current scale factor.
func (m *Manager) ScaleFloat(value float32) float32 {
	return value * m.Get()
}
