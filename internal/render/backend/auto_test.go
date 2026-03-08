package backend

import (
	"testing"

	"github.com/opd-ai/wain/internal/render"
)

func TestGpuGenerationToBackendType(t *testing.T) {
	tests := []struct {
		gen      render.GpuGeneration
		wantType BackendType
	}{
		// Intel GPUs
		{render.GpuGen9, BackendIntelGPU},
		{render.GpuGen11, BackendIntelGPU},
		{render.GpuGen12, BackendIntelGPU},
		{render.GpuXe, BackendIntelGPU},

		// AMD GPUs
		{render.GpuAmdRdna1, BackendAMDGPU},
		{render.GpuAmdRdna2, BackendAMDGPU},
		{render.GpuAmdRdna3, BackendAMDGPU},

		// Unknown
		{render.GpuUnknown, BackendUnknown},
		{render.GpuGeneration(999), BackendUnknown},
	}

	for _, tt := range tests {
		got := gpuGenerationToBackendType(tt.gen)
		if got != tt.wantType {
			t.Errorf("gpuGenerationToBackendType(%v) = %v, want %v",
				tt.gen, got, tt.wantType)
		}
	}
}

func TestIsIntelGPU(t *testing.T) {
	intelGens := []render.GpuGeneration{
		render.GpuGen9,
		render.GpuGen11,
		render.GpuGen12,
		render.GpuXe,
	}

	for _, gen := range intelGens {
		if !IsIntelGPU(gen) {
			t.Errorf("IsIntelGPU(%v) = false, want true", gen)
		}
	}

	notIntel := []render.GpuGeneration{
		render.GpuAmdRdna1,
		render.GpuAmdRdna2,
		render.GpuAmdRdna3,
		render.GpuUnknown,
	}

	for _, gen := range notIntel {
		if IsIntelGPU(gen) {
			t.Errorf("IsIntelGPU(%v) = true, want false", gen)
		}
	}
}

func TestIsAMDGPU(t *testing.T) {
	amdGens := []render.GpuGeneration{
		render.GpuAmdRdna1,
		render.GpuAmdRdna2,
		render.GpuAmdRdna3,
	}

	for _, gen := range amdGens {
		if !IsAMDGPU(gen) {
			t.Errorf("IsAMDGPU(%v) = false, want true", gen)
		}
	}

	notAMD := []render.GpuGeneration{
		render.GpuGen9,
		render.GpuGen11,
		render.GpuGen12,
		render.GpuXe,
		render.GpuUnknown,
	}

	for _, gen := range notAMD {
		if IsAMDGPU(gen) {
			t.Errorf("IsAMDGPU(%v) = true, want false", gen)
		}
	}
}

func TestBackendTypeString(t *testing.T) {
	tests := []struct {
		bt   BackendType
		want string
	}{
		{BackendIntelGPU, "Intel GPU"},
		{BackendAMDGPU, "AMD GPU"},
		{BackendSoftware, "Software"},
		{BackendUnknown, "Unknown"},
		{BackendType(999), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.bt.String()
		if got != tt.want {
			t.Errorf("BackendType(%d).String() = %q, want %q", tt.bt, got, tt.want)
		}
	}
}

func TestNewRendererForceSoftware(t *testing.T) {
	cfg := DefaultAutoConfig()
	cfg.ForceSoftware = true
	cfg.Width = 320
	cfg.Height = 240

	renderer, backendType, err := NewRenderer(cfg)
	if err != nil {
		t.Fatalf("NewRenderer with ForceSoftware failed: %v", err)
	}
	defer renderer.Destroy()

	if backendType != BackendSoftware {
		t.Errorf("NewRenderer with ForceSoftware returned %v, want BackendSoftware", backendType)
	}

	// Verify dimensions
	w, h := renderer.Dimensions()
	if w != cfg.Width || h != cfg.Height {
		t.Errorf("renderer.Dimensions() = (%d, %d), want (%d, %d)",
			w, h, cfg.Width, cfg.Height)
	}
}

func TestNewRendererInvalidDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 600},
		{"zero height", 800, 0},
		{"negative width", -1, 600},
		{"negative height", 800, -1},
		{"both zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultAutoConfig()
			cfg.Width = tt.width
			cfg.Height = tt.height
			cfg.ForceSoftware = true

			_, _, err := NewRenderer(cfg)
			if err == nil {
				t.Errorf("NewRenderer with invalid dimensions %dx%d succeeded, want error",
					tt.width, tt.height)
			}
		})
	}
}

func TestDefaultAutoConfig(t *testing.T) {
	cfg := DefaultAutoConfig()

	if cfg.DRMPath != "/dev/dri/renderD128" {
		t.Errorf("DefaultAutoConfig().DRMPath = %q, want %q",
			cfg.DRMPath, "/dev/dri/renderD128")
	}

	if cfg.Width != 800 {
		t.Errorf("DefaultAutoConfig().Width = %d, want 800", cfg.Width)
	}

	if cfg.Height != 600 {
		t.Errorf("DefaultAutoConfig().Height = %d, want 600", cfg.Height)
	}

	if cfg.VertexBufferSize != 1024*1024 {
		t.Errorf("DefaultAutoConfig().VertexBufferSize = %d, want 1MB", cfg.VertexBufferSize)
	}

	if cfg.ForceSoftware {
		t.Errorf("DefaultAutoConfig().ForceSoftware = true, want false")
	}

	if cfg.Verbose {
		t.Errorf("DefaultAutoConfig().Verbose = true, want false")
	}
}
