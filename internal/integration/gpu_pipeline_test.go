//go:build integration

package integration

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
)

const (
	pipelineWidth  = 100
	pipelineHeight = 100
)

// TestGPUPipelineEndToEnd validates the full rendering pipeline without GPU hardware.
//
// This test verifies that:
//  1. The rendering pipeline initialises correctly via the software fallback path.
//  2. A display list containing a single CmdFillRect can be rendered to completion.
//  3. The rendered pixel data contains the expected color at the target rectangle.
//
// The test uses ForceSoftware:true so it runs on any Linux host (including CI
// runners without a GPU). It is gated by the integration build tag to allow
// the test suite to be partitioned from unit tests.
//
// Run with: go test -tags integration ./internal/integration/... -run TestGPUPipelineEndToEnd
func TestGPUPipelineEndToEnd(t *testing.T) {
	cfg := backend.AutoConfig{
		Width:         pipelineWidth,
		Height:        pipelineHeight,
		ForceSoftware: true,
	}

	renderer, btype, err := backend.NewRenderer(cfg)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	defer renderer.Destroy()

	if btype != backend.BackendSoftware {
		t.Fatalf("expected BackendSoftware, got %s", btype)
	}

	// Build a display list with a single red fill rectangle covering the whole surface.
	dl := displaylist.New()
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl.AddFillRect(0, 0, pipelineWidth, pipelineHeight, red)

	if err := renderer.Render(dl); err != nil {
		t.Fatalf("Render: %v", err)
	}

	sb, ok := renderer.(*backend.SoftwareBackend)
	if !ok {
		t.Skip("renderer is not a *SoftwareBackend — pixel verification requires direct buffer access")
	}

	pixels := sb.Pixels()
	if len(pixels) == 0 {
		t.Fatal("Render produced empty pixel buffer")
	}

	// Sample the center pixel. Buffer is ARGB8888 (little-endian: B, G, R, A per pixel).
	cx := pipelineWidth / 2
	cy := pipelineHeight / 2
	idx := cy*sb.Buffer().Stride + cx*4
	b, g, r, a := pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3]
	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Errorf("center pixel = (r=%d g=%d b=%d a=%d), want (255 0 0 255)", r, g, b, a)
	}
}
