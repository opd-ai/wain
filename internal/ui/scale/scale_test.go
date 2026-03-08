package scale

import (
	"sync"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m.Get() != 1.0 {
		t.Errorf("expected default scale 1.0, got %f", m.Get())
	}
}

func TestSetGet(t *testing.T) {
	m := NewManager()
	m.Set(2.0)
	if m.Get() != 2.0 {
		t.Errorf("expected scale 2.0, got %f", m.Get())
	}
}

func TestSetZeroScale(t *testing.T) {
	m := NewManager()
	m.Set(0.0)
	if m.Get() != 1.0 {
		t.Errorf("expected scale reset to 1.0 for zero input, got %f", m.Get())
	}
}

func TestSetNegativeScale(t *testing.T) {
	m := NewManager()
	m.Set(-1.0)
	if m.Get() != 1.0 {
		t.Errorf("expected scale reset to 1.0 for negative input, got %f", m.Get())
	}
}

func TestSetFromInt(t *testing.T) {
	m := NewManager()
	m.SetFromInt(2)
	if m.Get() != 2.0 {
		t.Errorf("expected scale 2.0, got %f", m.Get())
	}
}

func TestSetFromDPI(t *testing.T) {
	tests := []struct {
		dpi      int32
		expected float32
	}{
		{96, 1.0},
		{192, 2.0},
		{144, 1.5},
		{288, 3.0},
	}

	for _, tc := range tests {
		m := NewManager()
		m.SetFromDPI(tc.dpi)
		if m.Get() != tc.expected {
			t.Errorf("DPI %d: expected scale %f, got %f", tc.dpi, tc.expected, m.Get())
		}
	}
}

func TestScaleInt(t *testing.T) {
	m := NewManager()
	m.Set(2.0)

	result := m.ScaleInt(100)
	if result != 200 {
		t.Errorf("expected 200, got %d", result)
	}

	m.Set(1.5)
	result = m.ScaleInt(100)
	if result != 150 {
		t.Errorf("expected 150, got %d", result)
	}
}

func TestScaleFloat(t *testing.T) {
	m := NewManager()
	m.Set(2.0)

	result := m.ScaleFloat(100.0)
	if result != 200.0 {
		t.Errorf("expected 200.0, got %f", result)
	}

	m.Set(1.5)
	result = m.ScaleFloat(100.0)
	if result != 150.0 {
		t.Errorf("expected 150.0, got %f", result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup

	// Multiple goroutines reading
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = m.Get()
			}
		}()
	}

	// Multiple goroutines writing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(scale float32) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.Set(scale)
			}
		}(float32(i+1) * 0.5)
	}

	wg.Wait()

	// Just ensure no race conditions - final value doesn't matter
	_ = m.Get()
}
