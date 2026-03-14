// Resource management API for fonts and images.
//
// Phase 9.5: Public API for loading fonts and images with automatic
// GPU atlas management and resource cleanup.

package wain

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"sync"

	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render/atlas"
)

var (
	// ErrNoAtlas is returned when attempting resource operations without an atlas.
	ErrNoAtlas = errors.New("wain: texture atlas not initialized")

	// ErrUnsupportedImageFormat is returned for unsupported image formats.
	ErrUnsupportedImageFormat = errors.New("wain: unsupported image format")

	// ErrInvalidFontData is returned when font data is malformed.
	ErrInvalidFontData = errors.New("wain: invalid font data")
)

// Font represents a loaded font resource.
type Font struct {
	atlas *text.Atlas
	size  float64
	id    int
}

// Size returns the font size in points.
func (f *Font) Size() float64 {
	return f.size
}

// Image represents a loaded image resource.
type Image struct {
	data   image.Image
	width  int
	height int
	id     int
}

// Size returns the image dimensions in pixels.
func (f *Image) Size() (width, height int) {
	return f.width, f.height
}

// ResourceManager manages fonts and images for an application.
type ResourceManager struct {
	mu sync.RWMutex

	// Texture atlas for GPU uploads
	textureAtlas *atlas.TextureAtlas

	// Default embedded font
	defaultFont *Font

	// Loaded resources
	fonts  map[int]*Font
	images map[int]*Image

	// ID generators
	nextFontID  int
	nextImageID int
}

// newResourceManager creates a new resource manager.
func newResourceManager(textureAtlas *atlas.TextureAtlas) *ResourceManager {
	return &ResourceManager{
		textureAtlas: textureAtlas,
		fonts:        make(map[int]*Font),
		images:       make(map[int]*Image),
		nextFontID:   1,
		nextImageID:  1,
	}
}

// initDefaultFont loads the embedded default font.
func (rm *ResourceManager) initDefaultFont() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	atlas, err := text.NewAtlas()
	if err != nil {
		return fmt.Errorf("failed to load default font: %w", err)
	}

	rm.defaultFont = &Font{
		atlas: atlas,
		size:  14.0,
		id:    0,
	}

	return nil
}

// DefaultFont returns the embedded default font.
// DefaultFont returns a font that is automatically initialized when the app starts.
// The font supports printable ASCII characters (0x20-0x7E) with SDF rendering
// for high-quality scaling at any size.
func (rm *ResourceManager) DefaultFont() *Font {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.defaultFont
}

// LoadFont loads a font from the specified path at the given size.
//
// Currently, this function returns the default embedded font with the
// requested size. Custom font loading from TTF files will be implemented
// in a future phase.
//
// Example:
//
//	font := app.LoadFont("path/to/font.ttf", 16.0)
func (rm *ResourceManager) LoadFont(path string, size float64) (*Font, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// For now, return a copy of the default font with the requested size.
	// Future implementation will parse TTF files and generate SDF atlases.
	if rm.defaultFont == nil {
		return nil, ErrInvalidFontData
	}

	font := &Font{
		atlas: rm.defaultFont.atlas,
		size:  size,
		id:    rm.nextFontID,
	}
	rm.nextFontID++
	rm.fonts[font.id] = font

	return font, nil
}

// LoadImage loads an image from the specified path.
//
// Supported formats: PNG, JPEG.
// The image is decoded using Go's image/png and image/jpeg packages
// and can be uploaded to the GPU atlas for rendering.
//
// Example:
//
//	img, err := app.LoadImage("path/to/icon.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (rm *ResourceManager) LoadImage(path string) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer f.Close()

	return rm.LoadImageFromReader(f, path)
}

// LoadImageFromReader loads an image from an io.Reader.
// LoadImageFromReader auto-detects the format based on the magic bytes.
// The filename hint helps with format detection but is optional.
func (rm *ResourceManager) LoadImageFromReader(r io.Reader, filenameHint string) (*Image, error) {
	// Decode the image (auto-detect format)
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Validate format
	if format != "png" && format != "jpeg" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedImageFormat, format)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	bounds := img.Bounds()
	resource := &Image{
		data:   img,
		width:  bounds.Dx(),
		height: bounds.Dy(),
		id:     rm.nextImageID,
	}
	rm.nextImageID++
	rm.images[resource.id] = resource

	return resource, nil
}

// uploadImageToAtlas uploads an image to the GPU texture atlas.
func (rm *ResourceManager) uploadImageToAtlas(img *Image) error {
	if rm.textureAtlas == nil {
		return ErrNoAtlas
	}

	// Convert image to RGBA bytes
	rgba := imageToRGBA(img.data)

	// Upload to atlas
	err := rm.textureAtlas.UploadImageData(img.id, rgba, img.width, img.height)
	if err != nil {
		return fmt.Errorf("failed to upload image to atlas: %w", err)
	}

	return nil
}

// cleanup releases all loaded resources.
func (rm *ResourceManager) cleanup() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Clear font references
	for id := range rm.fonts {
		delete(rm.fonts, id)
	}

	// Clear image references
	for id := range rm.images {
		delete(rm.images, id)
	}

	rm.defaultFont = nil
}

// imageToRGBA converts an image.Image to RGBA byte slice.
func imageToRGBA(img image.Image) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	rgba := make([]byte, width*height*4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			offset := (y*width + x) * 4
			rgba[offset+0] = uint8(r >> 8)
			rgba[offset+1] = uint8(g >> 8)
			rgba[offset+2] = uint8(b >> 8)
			rgba[offset+3] = uint8(a >> 8)
		}
	}

	return rgba
}

// init registers image decoders.
func init() {
	// Ensure PNG and JPEG decoders are registered
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
}

// LoadFont loads a font from the specified path at the given size.
//
// Example:
//
//	font := app.LoadFont("path/to/font.ttf", 16.0)
func (a *App) LoadFont(path string, size float64) (*Font, error) {
	if a.resources == nil {
		return nil, ErrNotRunning
	}
	return a.resources.LoadFont(path, size)
}

// LoadImage loads an image from the specified path.
//
// Supported formats: PNG, JPEG.
//
// Example:
//
//	img, err := app.LoadImage("path/to/icon.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (a *App) LoadImage(path string) (*Image, error) {
	if a.resources == nil {
		return nil, ErrNotRunning
	}
	return a.resources.LoadImage(path)
}

// LoadImageFromReader loads an image from an [io.Reader].
//
// The format is auto-detected from magic bytes; PNG and JPEG are supported.
// filenameHint is optional and used only to improve error messages.
//
// Example:
//
//	resp, err := http.Get("https://example.com/icon.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer resp.Body.Close()
//	img, err := app.LoadImageFromReader(resp.Body, "icon.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (a *App) LoadImageFromReader(r io.Reader, filenameHint string) (*Image, error) {
	if a.resources == nil {
		return nil, ErrNotRunning
	}
	return a.resources.LoadImageFromReader(r, filenameHint)
}

// DefaultFont returns the embedded default font.
// DefaultFont supports printable ASCII characters (0x20-0x7E)
// with SDF rendering for high-quality scaling.
func (a *App) DefaultFont() *Font {
	if a.resources == nil {
		return nil
	}
	return a.resources.DefaultFont()
}
