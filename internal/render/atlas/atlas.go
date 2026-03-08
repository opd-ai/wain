// Package atlas provides GPU texture atlas management for fonts and images.
//
// A texture atlas packs multiple smaller textures (glyphs, images) into larger
// GPU textures to minimize bind operations and improve performance. The package
// supports:
//   - Font SDF atlas: static allocation for the embedded font atlas
//   - Image atlas: dynamic allocation with shelf-packing and LRU eviction
//   - Dirty region tracking for efficient GPU uploads
package atlas

import (
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/render"
)

var (
	// ErrAtlasNotInitialized is returned when operations are attempted on an
	// uninitialized atlas.
	ErrAtlasNotInitialized = errors.New("atlas: not initialized")

	// ErrRegionTooLarge is returned when a requested region doesn't fit in
	// available atlas pages.
	ErrRegionTooLarge = errors.New("atlas: region too large for atlas pages")

	// ErrOutOfSpace is returned when the atlas is full and eviction is disabled
	// or cannot free enough space.
	ErrOutOfSpace = errors.New("atlas: out of space")
)

const (
	// FontAtlasID is the reserved texture ID for the font SDF atlas.
	// Texture ID 0 is always the font atlas.
	FontAtlasID = 0

	// DefaultImageAtlasSize is the default size for image atlas pages (2048x2048).
	DefaultImageAtlasSize = 2048

	// MaxImageAtlasPages is the maximum number of image atlas pages.
	MaxImageAtlasPages = 16
)

// TextureAtlas manages GPU texture atlases for fonts and images.
type TextureAtlas struct {
	allocator *render.Allocator

	// Font atlas (texture ID 0)
	fontAtlas     *render.BufferHandle
	fontWidth     int
	fontHeight    int
	fontDirty     bool
	fontDirtyRect Rect

	// Image atlases (texture IDs 1+)
	imagePages    []*ImagePage
	imagePageSize int

	// Image → region mapping
	imageRegions map[int]*Region
	nextImageID  int
}

// Rect represents a rectangular region.
type Rect struct {
	X, Y, Width, Height int
}

// ImagePage represents a single texture atlas page for images.
type ImagePage struct {
	ID        int
	Buffer    *render.BufferHandle
	Width     int
	Height    int
	Shelves   []*Shelf
	DirtyRect Rect
	Dirty     bool
}

// Shelf represents a horizontal shelf in the shelf-packing algorithm.
type Shelf struct {
	Y       int
	Height  int
	X       int
	Regions []*Region
}

// Region represents an allocated region in an atlas.
type Region struct {
	ImageID    int
	PageID     int
	Rect       Rect
	LRUCounter uint64
}

// New creates a new texture atlas manager.
//
// The allocator is used to allocate GPU buffers for atlas textures.
// The font atlas is not initialized until UploadFontAtlas is called.
func New(allocator *render.Allocator, imagePageSize int) *TextureAtlas {
	if imagePageSize == 0 {
		imagePageSize = DefaultImageAtlasSize
	}

	return &TextureAtlas{
		allocator:     allocator,
		imagePageSize: imagePageSize,
		imagePages:    make([]*ImagePage, 0, MaxImageAtlasPages),
		imageRegions:  make(map[int]*Region),
		nextImageID:   1,
	}
}

// UploadFontAtlas uploads the font SDF atlas to the GPU.
//
// The font atlas is always texture ID 0. This must be called before any
// text rendering operations.
//
// Parameters:
//   - sdfData: Raw SDF texture data (8-bit grayscale)
//   - width, height: Dimensions of the SDF atlas in pixels
func (ta *TextureAtlas) UploadFontAtlas(sdfData []uint8, width, height int) error {
	if ta.allocator == nil {
		return ErrAtlasNotInitialized
	}

	// Allocate GPU buffer for font atlas (8-bit format, linear tiling)
	buf, err := ta.allocator.Allocate(uint32(width), uint32(height), 1, render.TilingNone)
	if err != nil {
		return fmt.Errorf("atlas: failed to allocate font atlas buffer: %w", err)
	}

	ta.fontAtlas = buf
	ta.fontWidth = width
	ta.fontHeight = height

	// Mark entire font atlas as dirty for initial upload
	ta.fontDirty = true
	ta.fontDirtyRect = Rect{X: 0, Y: 0, Width: width, Height: height}

	// Upload SDF data to GPU buffer via mmap
	data, err := buf.Mmap()
	if err != nil {
		return fmt.Errorf("atlas: failed to mmap font atlas buffer: %w", err)
	}
	defer buf.Munmap(data)

	// Copy SDF data to mapped GPU memory
	if len(sdfData) > len(data) {
		return fmt.Errorf("atlas: SDF data size %d exceeds buffer size %d", len(sdfData), len(data))
	}
	copy(data, sdfData)

	// Munmap automatically syncs changes to GPU

	return nil
}

// GetFontAtlas returns the font atlas buffer and dimensions.
//
// Returns nil if the font atlas has not been uploaded.
func (ta *TextureAtlas) GetFontAtlas() (buffer *render.BufferHandle, width, height int) {
	return ta.fontAtlas, ta.fontWidth, ta.fontHeight
}

// AllocateImageRegion allocates a region in the image atlas for an image.
//
// Returns the image ID and UV coordinates, or an error if allocation fails.
// The image ID can be used in display list commands to reference the texture.
//
// The shelf-packing algorithm is used to pack images efficiently:
//  1. Try to find an existing shelf that can fit the image
//  2. If no shelf fits, create a new shelf
//  3. If no page has space, allocate a new page
//  4. If all pages are full, evict least-recently-used images (if enabled)
func (ta *TextureAtlas) AllocateImageRegion(width, height int) (imageID int, u0, v0, u1, v1 float32, err error) {
	if ta.allocator == nil {
		return 0, 0, 0, 0, 0, ErrAtlasNotInitialized
	}

	if width > ta.imagePageSize || height > ta.imagePageSize {
		return 0, 0, 0, 0, 0, ErrRegionTooLarge
	}

	// Try to allocate in existing pages
	for _, page := range ta.imagePages {
		if region := ta.tryAllocateInPage(page, width, height); region != nil {
			imageID := ta.nextImageID
			ta.nextImageID++
			region.ImageID = imageID
			ta.imageRegions[imageID] = region

			// Calculate UV coordinates
			u0 := float32(region.Rect.X) / float32(page.Width)
			v0 := float32(region.Rect.Y) / float32(page.Height)
			u1 := float32(region.Rect.X+region.Rect.Width) / float32(page.Width)
			v1 := float32(region.Rect.Y+region.Rect.Height) / float32(page.Height)

			return imageID, u0, v0, u1, v1, nil
		}
	}

	// Need a new page
	if len(ta.imagePages) >= MaxImageAtlasPages {
		// Try LRU eviction to free space
		if evicted := ta.evictLRURegions(width, height); evicted {
			// Retry allocation after eviction
			for _, page := range ta.imagePages {
				if region := ta.tryAllocateInPage(page, width, height); region != nil {
					imageID := ta.nextImageID
					ta.nextImageID++
					region.ImageID = imageID
					ta.imageRegions[imageID] = region

					// Calculate UV coordinates
					u0 := float32(region.Rect.X) / float32(page.Width)
					v0 := float32(region.Rect.Y) / float32(page.Height)
					u1 := float32(region.Rect.X+region.Rect.Width) / float32(page.Width)
					v1 := float32(region.Rect.Y+region.Rect.Height) / float32(page.Height)

					return imageID, u0, v0, u1, v1, nil
				}
			}
		}
		return 0, 0, 0, 0, 0, ErrOutOfSpace
	}

	page, err := ta.allocateNewPage()
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	region := ta.tryAllocateInPage(page, width, height)
	if region == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("atlas: failed to allocate in new page")
	}

	imageID = ta.nextImageID
	ta.nextImageID++
	region.ImageID = imageID
	ta.imageRegions[imageID] = region

	// Calculate UV coordinates
	u0 = float32(region.Rect.X) / float32(page.Width)
	v0 = float32(region.Rect.Y) / float32(page.Height)
	u1 = float32(region.Rect.X+region.Rect.Width) / float32(page.Width)
	v1 = float32(region.Rect.Y+region.Rect.Height) / float32(page.Height)

	return imageID, u0, v0, u1, v1, nil
}

// tryAllocateInPage attempts to allocate a region in the given page.
// Returns nil if allocation fails.
func (ta *TextureAtlas) tryAllocateInPage(page *ImagePage, width, height int) *Region {
	// Try to find an existing shelf that fits
	for _, shelf := range page.Shelves {
		if shelf.Height >= height && shelf.X+width <= page.Width {
			region := &Region{
				PageID: page.ID,
				Rect: Rect{
					X:      shelf.X,
					Y:      shelf.Y,
					Width:  width,
					Height: height,
				},
			}
			shelf.X += width
			shelf.Regions = append(shelf.Regions, region)

			// Mark page as dirty
			page.Dirty = true
			if page.DirtyRect.Width == 0 {
				page.DirtyRect = region.Rect
			} else {
				page.DirtyRect = unionRect(page.DirtyRect, region.Rect)
			}

			return region
		}
	}

	// Try to create a new shelf
	var topY int
	if len(page.Shelves) > 0 {
		lastShelf := page.Shelves[len(page.Shelves)-1]
		topY = lastShelf.Y + lastShelf.Height
	}

	if topY+height > page.Height {
		return nil // No space for a new shelf
	}

	shelf := &Shelf{
		Y:      topY,
		Height: height,
		X:      width,
	}

	region := &Region{
		PageID: page.ID,
		Rect: Rect{
			X:      0,
			Y:      topY,
			Width:  width,
			Height: height,
		},
	}

	shelf.Regions = append(shelf.Regions, region)
	page.Shelves = append(page.Shelves, shelf)

	// Mark page as dirty
	page.Dirty = true
	if page.DirtyRect.Width == 0 {
		page.DirtyRect = region.Rect
	} else {
		page.DirtyRect = unionRect(page.DirtyRect, region.Rect)
	}

	return region
}

// allocateNewPage creates a new image atlas page.
func (ta *TextureAtlas) allocateNewPage() (*ImagePage, error) {
	pageID := len(ta.imagePages) + 1 // Page IDs start at 1 (0 is font atlas)

	// Allocate GPU buffer (32-bit RGBA, linear tiling)
	buf, err := ta.allocator.Allocate(
		uint32(ta.imagePageSize),
		uint32(ta.imagePageSize),
		4, // 32-bit RGBA
		render.TilingNone,
	)
	if err != nil {
		return nil, fmt.Errorf("atlas: failed to allocate page buffer: %w", err)
	}

	page := &ImagePage{
		ID:      pageID,
		Buffer:  buf,
		Width:   ta.imagePageSize,
		Height:  ta.imagePageSize,
		Shelves: make([]*Shelf, 0),
	}

	ta.imagePages = append(ta.imagePages, page)
	return page, nil
}

// UploadImageData uploads image data to a previously allocated region.
//
// Parameters:
//   - imageID: The ID returned by AllocateImageRegion
//   - pixels: Raw RGBA pixel data (4 bytes per pixel)
//   - width, height: Dimensions of the image
func (ta *TextureAtlas) UploadImageData(imageID int, pixels []uint8, width, height int) error {
	region, ok := ta.imageRegions[imageID]
	if !ok {
		return fmt.Errorf("atlas: image ID %d not found", imageID)
	}

	if width != region.Rect.Width || height != region.Rect.Height {
		return fmt.Errorf("atlas: image dimensions mismatch")
	}

	// Get the image page
	page := ta.GetImagePage(region.PageID)
	if page == nil {
		return fmt.Errorf("atlas: page %d not found", region.PageID)
	}

	// Upload pixels to GPU buffer via mmap
	data, err := page.Buffer.Mmap()
	if err != nil {
		return fmt.Errorf("atlas: failed to mmap image page buffer: %w", err)
	}
	defer page.Buffer.Munmap(data)

	// Calculate destination offset and copy pixels row by row
	pageStride := page.Width * 4 // 4 bytes per pixel (RGBA)
	srcStride := width * 4

	for y := 0; y < height; y++ {
		dstOffset := (region.Rect.Y+y)*pageStride + region.Rect.X*4
		srcOffset := y * srcStride
		copy(data[dstOffset:dstOffset+srcStride], pixels[srcOffset:srcOffset+srcStride])
	}

	// Munmap automatically syncs changes to GPU

	// Update LRU counter
	region.LRUCounter++

	return nil
}

// evictLRURegions evicts least-recently-used regions to free space.
//
// Attempts to evict enough regions to allocate a region of the given size.
// Returns true if eviction was successful and space may be available.
func (ta *TextureAtlas) evictLRURegions(neededWidth, neededHeight int) bool {
	if len(ta.imageRegions) == 0 {
		return false
	}

	// Get regions sorted by LRU (least recently used first)
	sortedIDs := ta.getRegionsSortedByLRU()

	// Evict 25% of regions as heuristic
	evictCount := max(1, len(sortedIDs)/4)

	for i := 0; i < evictCount && i < len(sortedIDs); i++ {
		ta.FreeImageRegion(sortedIDs[i])
	}

	return true
}

// getRegionsSortedByLRU returns region IDs sorted by LRU counter (ascending).
func (ta *TextureAtlas) getRegionsSortedByLRU() []int {
	type regionWithID struct {
		id      int
		counter uint64
	}

	regions := make([]regionWithID, 0, len(ta.imageRegions))
	for id, region := range ta.imageRegions {
		regions = append(regions, regionWithID{id: id, counter: region.LRUCounter})
	}

	// Simple bubble sort by LRU counter
	for i := 0; i < len(regions); i++ {
		for j := i + 1; j < len(regions); j++ {
			if regions[i].counter > regions[j].counter {
				regions[i], regions[j] = regions[j], regions[i]
			}
		}
	}

	ids := make([]int, len(regions))
	for i, r := range regions {
		ids[i] = r.id
	}
	return ids
}

// FreeImageRegion frees a previously allocated image region.
func (ta *TextureAtlas) FreeImageRegion(imageID int) error {
	region, ok := ta.imageRegions[imageID]
	if !ok {
		return fmt.Errorf("atlas: image ID %d not found", imageID)
	}

	delete(ta.imageRegions, imageID)
	ta.removeRegionFromShelf(region.PageID, imageID)
	return nil
}

// removeRegionFromShelf removes a region from its shelf for potential reuse.
func (ta *TextureAtlas) removeRegionFromShelf(pageID, imageID int) {
	page := ta.GetImagePage(pageID)
	if page == nil {
		return
	}

	for _, shelf := range page.Shelves {
		for i, r := range shelf.Regions {
			if r.ImageID == imageID {
				shelf.Regions = append(shelf.Regions[:i], shelf.Regions[i+1:]...)
				return
			}
		}
	}
}

// GetImagePage returns the image page for a given page ID.
func (ta *TextureAtlas) GetImagePage(pageID int) *ImagePage {
	if pageID < 1 || pageID > len(ta.imagePages) {
		return nil
	}
	return ta.imagePages[pageID-1]
}

// ImagePageCount returns the number of allocated image pages.
func (ta *TextureAtlas) ImagePageCount() int {
	return len(ta.imagePages)
}

// Destroy frees all GPU resources used by the atlas.
func (ta *TextureAtlas) Destroy() error {
	if ta.fontAtlas != nil {
		if err := ta.fontAtlas.Destroy(); err != nil {
			return err
		}
		ta.fontAtlas = nil
	}

	for _, page := range ta.imagePages {
		if err := page.Buffer.Destroy(); err != nil {
			return err
		}
	}
	ta.imagePages = nil
	ta.imageRegions = nil

	return nil
}

// unionRect returns the union of two rectangles.
func unionRect(a, b Rect) Rect {
	x0 := min(a.X, b.X)
	y0 := min(a.Y, b.Y)
	x1 := max(a.X+a.Width, b.X+b.Width)
	y1 := max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
