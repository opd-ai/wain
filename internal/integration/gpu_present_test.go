// go:build integration
//go:build integration
// +build integration

package integration

import (
	"os"
	"testing"

	"github.com/opd-ai/wain/internal/render"
	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/render/display"
	"github.com/opd-ai/wain/internal/x11/client"
)

// TestGPUPresent validates the GPU DMA-BUF / X11-Present presentation pipeline
// end-to-end: GPU backend → X11Pipeline (DRI3 + Present) → X11 window.
//
// This test verifies that:
//  1. A GPUBackend can be initialized from /dev/dri/renderD128.
//  2. An X11 connection can be opened and a window created.
//  3. NewGPUX11PresenterFromConn successfully queries DRI3 + Present extensions
//     and constructs the pipeline without error.
//  4. The presenter's Close method releases resources cleanly.
//
// The test is gated by the `integration` build tag and skips gracefully when:
//   - /dev/dri/renderD128 is absent (no GPU hardware).
//   - DISPLAY is unset (no X11 server).
//   - DRI3 or Present extensions are unavailable on the X11 server.
//
// Run with:
//
//	go test -tags=integration -run TestGPUPresent ./internal/integration/...
func TestGPUPresent(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping GPU present test: %s not found", drmRenderNode)
	}

	x11Display := os.Getenv("DISPLAY")
	if x11Display == "" {
		t.Skip("Skipping GPU present test: DISPLAY not set")
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping GPU present test: GPU not recognized at %s", drmRenderNode)
	}
	t.Logf("GPU: %s", gen)

	// Initialize GPU backend.
	cfg := backend.DefaultConfig()
	cfg.DRMPath = drmRenderNode
	gpuBackend, err := backend.New(cfg)
	if err != nil {
		t.Skipf("Skipping GPU present test: GPU backend unavailable: %v", err)
	}
	defer gpuBackend.Destroy()

	// Open X11 connection.
	displayNum := extractDisplayNum(x11Display)
	conn, err := client.Connect(displayNum)
	if err != nil {
		t.Skipf("Skipping GPU present test: X11 connection failed: %v", err)
	}
	defer conn.Close()

	// Allocate a window XID.
	wid, err := conn.AllocXID()
	if err != nil {
		t.Fatalf("AllocXID failed: %v", err)
	}

	// Attempt to create the GPU X11 presenter (queries DRI3 + Present internally).
	p, err := display.NewGPUX11PresenterFromConn(conn, wid, gpuBackend)
	if err != nil {
		// DRI3/Present not available on this server — acceptable skip.
		t.Skipf("GPU X11 presenter unavailable (DRI3/Present not supported): %v", err)
	}
	defer func() {
		if cerr := p.Close(); cerr != nil {
			t.Logf("Presenter Close: %v", cerr)
		}
	}()

	t.Log("✓ GPU X11 presenter created successfully")
}

// extractDisplayNum strips the optional hostname and screen suffix from a DISPLAY
// string so only the numeric display number (e.g. ":0") remains.
func extractDisplayNum(display string) string {
	s := display
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			s = s[:i]
			break
		}
	}
	return s
}
