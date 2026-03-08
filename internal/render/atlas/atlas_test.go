package atlas

import (
	"testing"

	"github.com/opd-ai/wain/internal/render"
)

// TestNew validates atlas initialization
func TestNew(t *testing.T) {
	alloc := &render.Allocator{} // Minimal allocator for testing

	atlas := New(alloc, 0)
	if atlas == nil {
		t.Fatal("New returned nil")
	}

	if atlas.imagePageSize != DefaultImageAtlasSize {
		t.Errorf("expected default page size %d, got %d", DefaultImageAtlasSize, atlas.imagePageSize)
	}

	if atlas.imageRegions == nil {
		t.Error("imageRegions map not initialized")
	}

	if atlas.nextImageID != 1 {
		t.Errorf("expected nextImageID=1, got %d", atlas.nextImageID)
	}
}

// TestNewWithCustomPageSize validates custom page size
func TestNewWithCustomPageSize(t *testing.T) {
	alloc := &render.Allocator{}
	customSize := 1024

	atlas := New(alloc, customSize)
	if atlas.imagePageSize != customSize {
		t.Errorf("expected page size %d, got %d", customSize, atlas.imagePageSize)
	}
}

// TestUnionRect validates rectangle union calculation
func TestUnionRect(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Rect
		expected Rect
	}{
		{
			name:     "non-overlapping",
			a:        Rect{X: 0, Y: 0, Width: 10, Height: 10},
			b:        Rect{X: 20, Y: 20, Width: 10, Height: 10},
			expected: Rect{X: 0, Y: 0, Width: 30, Height: 30},
		},
		{
			name:     "overlapping",
			a:        Rect{X: 0, Y: 0, Width: 20, Height: 20},
			b:        Rect{X: 10, Y: 10, Width: 20, Height: 20},
			expected: Rect{X: 0, Y: 0, Width: 30, Height: 30},
		},
		{
			name:     "contained",
			a:        Rect{X: 0, Y: 0, Width: 50, Height: 50},
			b:        Rect{X: 10, Y: 10, Width: 10, Height: 10},
			expected: Rect{X: 0, Y: 0, Width: 50, Height: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unionRect(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("unionRect(%+v, %+v) = %+v, expected %+v",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestShelfPacking validates the shelf-packing algorithm
func TestShelfPacking(t *testing.T) {
	page := &ImagePage{
		ID:      1,
		Width:   100,
		Height:  100,
		Shelves: make([]*Shelf, 0),
	}

	atlas := &TextureAtlas{
		imagePageSize: 100,
	}

	// Allocate first region (creates first shelf)
	r1 := atlas.tryAllocateInPage(page, 30, 20)
	if r1 == nil {
		t.Fatal("failed to allocate first region")
	}
	if r1.Rect.X != 0 || r1.Rect.Y != 0 {
		t.Errorf("expected r1 at (0,0), got (%d,%d)", r1.Rect.X, r1.Rect.Y)
	}
	if r1.Rect.Width != 30 || r1.Rect.Height != 20 {
		t.Errorf("expected r1 size (30,20), got (%d,%d)", r1.Rect.Width, r1.Rect.Height)
	}

	// Allocate second region in same shelf
	r2 := atlas.tryAllocateInPage(page, 40, 15)
	if r2 == nil {
		t.Fatal("failed to allocate second region")
	}
	if r2.Rect.X != 30 || r2.Rect.Y != 0 {
		t.Errorf("expected r2 at (30,0), got (%d,%d)", r2.Rect.X, r2.Rect.Y)
	}

	// Allocate third region that doesn't fit in first shelf (creates second shelf)
	r3 := atlas.tryAllocateInPage(page, 40, 25)
	if r3 == nil {
		t.Fatal("failed to allocate third region")
	}
	if r3.Rect.X != 0 || r3.Rect.Y != 20 {
		t.Errorf("expected r3 at (0,20), got (%d,%d)", r3.Rect.X, r3.Rect.Y)
	}

	// Verify shelf count
	if len(page.Shelves) != 2 {
		t.Errorf("expected 2 shelves, got %d", len(page.Shelves))
	}

	// Verify dirty flag
	if !page.Dirty {
		t.Error("page should be marked dirty after allocations")
	}
}

// TestShelfPackingFull validates behavior when page is full
func TestShelfPackingFull(t *testing.T) {
	page := &ImagePage{
		ID:      1,
		Width:   100,
		Height:  100,
		Shelves: make([]*Shelf, 0),
	}

	atlas := &TextureAtlas{
		imagePageSize: 100,
	}

	// Fill the page both horizontally and vertically
	// Create 5 shelves of 20 pixels each, fully filling width (100x20 each)
	for i := 0; i < 5; i++ {
		r := atlas.tryAllocateInPage(page, 100, 20)
		if r == nil {
			t.Fatalf("failed to allocate region %d", i)
		}
	}

	// Page should now be full (100 width × 100 height used)
	// Try to allocate another region - should fail
	r := atlas.tryAllocateInPage(page, 100, 20)
	if r != nil {
		t.Error("allocation should fail when page is full vertically")
	}

	// Even a small region should fail if it doesn't fit in existing shelves
	r = atlas.tryAllocateInPage(page, 1, 21)
	if r != nil {
		t.Error("allocation should fail when height exceeds remaining vertical space")
	}
}

// TestGetFontAtlas validates font atlas getters
func TestGetFontAtlas(t *testing.T) {
	atlas := New(&render.Allocator{}, 0)

	// Before upload
	buf, w, h := atlas.GetFontAtlas()
	if buf != nil {
		t.Error("font atlas should be nil before upload")
	}
	if w != 0 || h != 0 {
		t.Errorf("expected dimensions (0,0) before upload, got (%d,%d)", w, h)
	}

	// After setting (simulated)
	atlas.fontWidth = 256
	atlas.fontHeight = 256
	atlas.fontAtlas = &render.BufferHandle{}

	buf, w, h = atlas.GetFontAtlas()
	if buf == nil {
		t.Error("font atlas should not be nil after upload")
	}
	if w != 256 || h != 256 {
		t.Errorf("expected dimensions (256,256), got (%d,%d)", w, h)
	}
}

// TestImagePageCount validates page count tracking
func TestImagePageCount(t *testing.T) {
	atlas := New(&render.Allocator{}, 100)

	if atlas.ImagePageCount() != 0 {
		t.Errorf("expected 0 pages initially, got %d", atlas.ImagePageCount())
	}

	// Add pages (simulated)
	atlas.imagePages = append(atlas.imagePages, &ImagePage{ID: 1})
	atlas.imagePages = append(atlas.imagePages, &ImagePage{ID: 2})

	if atlas.ImagePageCount() != 2 {
		t.Errorf("expected 2 pages, got %d", atlas.ImagePageCount())
	}
}

// TestGetImagePage validates page retrieval
func TestGetImagePage(t *testing.T) {
	atlas := New(&render.Allocator{}, 100)

	// Add test pages
	page1 := &ImagePage{ID: 1}
	page2 := &ImagePage{ID: 2}
	atlas.imagePages = append(atlas.imagePages, page1, page2)

	// Valid page ID
	p := atlas.GetImagePage(1)
	if p != page1 {
		t.Error("GetImagePage(1) returned wrong page")
	}

	p = atlas.GetImagePage(2)
	if p != page2 {
		t.Error("GetImagePage(2) returned wrong page")
	}

	// Invalid page IDs
	if atlas.GetImagePage(0) != nil {
		t.Error("GetImagePage(0) should return nil")
	}

	if atlas.GetImagePage(3) != nil {
		t.Error("GetImagePage(3) should return nil")
	}

	if atlas.GetImagePage(-1) != nil {
		t.Error("GetImagePage(-1) should return nil")
	}
}

// TestMinMax validates min/max helper functions
func TestMinMax(t *testing.T) {
	if min(5, 10) != 5 {
		t.Error("min(5, 10) should be 5")
	}
	if min(10, 5) != 5 {
		t.Error("min(10, 5) should be 5")
	}
	if min(7, 7) != 7 {
		t.Error("min(7, 7) should be 7")
	}

	if max(5, 10) != 10 {
		t.Error("max(5, 10) should be 10")
	}
	if max(10, 5) != 10 {
		t.Error("max(10, 5) should be 10")
	}
	if max(7, 7) != 7 {
		t.Error("max(7, 7) should be 7")
	}
}
