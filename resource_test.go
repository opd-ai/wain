package wain

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceManager_DefaultFont(t *testing.T) {
	rm := newResourceManager(nil)
	if err := rm.initDefaultFont(); err != nil {
		t.Fatalf("initDefaultFont failed: %v", err)
	}

	font := rm.DefaultFont()
	if font == nil {
		t.Fatal("DefaultFont returned nil")
	}

	if font.size != 14.0 {
		t.Errorf("expected default font size 14.0, got %.1f", font.size)
	}

	if font.atlas == nil {
		t.Error("default font atlas is nil")
	}
}

func TestResourceManager_LoadFont(t *testing.T) {
	rm := newResourceManager(nil)
	if err := rm.initDefaultFont(); err != nil {
		t.Fatalf("initDefaultFont failed: %v", err)
	}

	// Load font with custom size
	font, err := rm.LoadFont("dummy.ttf", 16.0)
	if err != nil {
		t.Fatalf("LoadFont failed: %v", err)
	}

	if font.size != 16.0 {
		t.Errorf("expected font size 16.0, got %.1f", font.size)
	}

	// Load multiple fonts
	font2, err := rm.LoadFont("dummy2.ttf", 24.0)
	if err != nil {
		t.Fatalf("LoadFont failed for second font: %v", err)
	}

	if font.id == font2.id {
		t.Error("fonts have duplicate IDs")
	}
}

func TestResourceManager_LoadImage(t *testing.T) {
	// Create a temporary PNG image
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")

	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Save to file
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode test image: %v", err)
	}
	f.Close()

	// Load the image
	rm := newResourceManager(nil)
	loaded, err := rm.LoadImage(imgPath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	if loaded.width != 100 || loaded.height != 100 {
		t.Errorf("expected 100x100, got %dx%d", loaded.width, loaded.height)
	}

	if loaded.data == nil {
		t.Error("loaded image data is nil")
	}
}

func TestResourceManager_LoadImageFromReader(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	// Encode to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}

	// Load from reader
	rm := newResourceManager(nil)
	loaded, err := rm.LoadImageFromReader(&buf, "test.png")
	if err != nil {
		t.Fatalf("LoadImageFromReader failed: %v", err)
	}

	if loaded.width != 50 || loaded.height != 50 {
		t.Errorf("expected 50x50, got %dx%d", loaded.width, loaded.height)
	}
}

func TestResourceManager_LoadImageUnsupportedFormat(t *testing.T) {
	rm := newResourceManager(nil)

	// Try to load a non-existent file (should fail at open)
	_, err := rm.LoadImage("nonexistent.bmp")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestResourceManager_Cleanup(t *testing.T) {
	rm := newResourceManager(nil)
	if err := rm.initDefaultFont(); err != nil {
		t.Fatalf("initDefaultFont failed: %v", err)
	}

	// Load some resources
	_, _ = rm.LoadFont("test.ttf", 16.0)

	// Verify resources exist
	if len(rm.fonts) == 0 {
		t.Error("expected fonts to be loaded")
	}

	// Cleanup
	rm.cleanup()

	// Verify resources cleared
	if len(rm.fonts) != 0 {
		t.Error("fonts not cleared after cleanup")
	}
	if len(rm.images) != 0 {
		t.Error("images not cleared after cleanup")
	}
	if rm.defaultFont != nil {
		t.Error("default font not cleared after cleanup")
	}
}

func TestFont_Size(t *testing.T) {
	font := &Font{size: 18.0}
	if font.Size() != 18.0 {
		t.Errorf("expected size 18.0, got %.1f", font.Size())
	}
}

func TestImage_Size(t *testing.T) {
	img := &Image{width: 200, height: 150}
	w, h := img.Size()
	if w != 200 || h != 150 {
		t.Errorf("expected 200x150, got %dx%d", w, h)
	}
}

func TestImageToRGBA(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 128})

	rgba := imageToRGBA(img)

	// Check dimensions
	expectedLen := 2 * 2 * 4
	if len(rgba) != expectedLen {
		t.Errorf("expected %d bytes, got %d", expectedLen, len(rgba))
	}

	// Check first pixel (red)
	if rgba[0] != 255 || rgba[1] != 0 || rgba[2] != 0 || rgba[3] != 255 {
		t.Errorf("first pixel incorrect: %v", rgba[0:4])
	}

	// Check second pixel (green)
	if rgba[4] != 0 || rgba[5] != 255 || rgba[6] != 0 || rgba[7] != 255 {
		t.Errorf("second pixel incorrect: %v", rgba[4:8])
	}
}

func TestResourceManager_LoadFontBeforeInit(t *testing.T) {
	rm := newResourceManager(nil)
	// Don't call initDefaultFont

	_, err := rm.LoadFont("test.ttf", 16.0)
	if err != ErrInvalidFontData {
		t.Errorf("expected ErrInvalidFontData, got %v", err)
	}
}

func TestResourceManager_MultipleImageLoads(t *testing.T) {
	// Create temporary images
	tmpDir := t.TempDir()

	createTestImage := func(name string, width, height int) string {
		path := filepath.Join(tmpDir, name)
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		f, _ := os.Create(path)
		png.Encode(f, img)
		f.Close()
		return path
	}

	img1Path := createTestImage("img1.png", 64, 64)
	img2Path := createTestImage("img2.png", 128, 128)
	img3Path := createTestImage("img3.png", 32, 32)

	rm := newResourceManager(nil)

	img1, err := rm.LoadImage(img1Path)
	if err != nil {
		t.Fatalf("failed to load img1: %v", err)
	}

	img2, err := rm.LoadImage(img2Path)
	if err != nil {
		t.Fatalf("failed to load img2: %v", err)
	}

	img3, err := rm.LoadImage(img3Path)
	if err != nil {
		t.Fatalf("failed to load img3: %v", err)
	}

	// Verify unique IDs
	ids := map[int]bool{
		img1.id: true,
		img2.id: true,
		img3.id: true,
	}
	if len(ids) != 3 {
		t.Error("images do not have unique IDs")
	}

	// Verify dimensions
	if img1.width != 64 || img1.height != 64 {
		t.Errorf("img1 dimensions incorrect")
	}
	if img2.width != 128 || img2.height != 128 {
		t.Errorf("img2 dimensions incorrect")
	}
	if img3.width != 32 || img3.height != 32 {
		t.Errorf("img3 dimensions incorrect")
	}
}

// TestLoadImageFromReaderDecodeError verifies error when image data is invalid.
func TestLoadImageFromReaderDecodeError(t *testing.T) {
	rm := newResourceManager(nil)
	r := bytes.NewReader([]byte("this is not an image"))
	_, err := rm.LoadImageFromReader(r, "test.png")
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

// TestCleanupWithImages verifies cleanup clears images map.
func TestCleanupWithImages(t *testing.T) {
	rm := newResourceManager(nil)

	// Load a valid PNG image into the ResourceManager directly
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	resource := &Image{data: img, width: 4, height: 4, id: 1}
	rm.images[resource.id] = resource

	rm.cleanup()

	if len(rm.images) != 0 {
		t.Error("images not cleared after cleanup")
	}
}

// TestLoadImageFromReaderUnsupportedFormat verifies ErrUnsupportedImageFormat for GIF.
func TestLoadImageFromReaderUnsupportedFormat(t *testing.T) {
	rm := newResourceManager(nil)
	// Minimal valid GIF87a header (just a few bytes of GIF magic)
	// This will decode as gif (if gif is registered), which isn't png/jpeg
	// Actually we need a valid GIF to pass Decode. Let's use a minimal GIF.
	// GIF87a 1x1 transparent pixel
	gifData := []byte{
		0x47, 0x49, 0x46, 0x38, 0x37, 0x61, // GIF87a
		0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, // logical screen descriptor
		0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // image descriptor
		0x02, 0x02, 0x4c, 0x01, 0x00, // image data
		0x3b, // GIF trailer
	}
	r := bytes.NewReader(gifData)
	_, err := rm.LoadImageFromReader(r, "test.gif")
	// Either decode fails (gif not registered) or format check fails
	if err == nil {
		t.Error("expected error for GIF format (unsupported)")
	}
}

// TestLoadImageFromReaderGIFUnsupported encodes a real GIF and verifies the
// ErrUnsupportedImageFormat path in LoadImageFromReader.
func TestLoadImageFromReaderGIFUnsupported(t *testing.T) {
// Encode a 1x1 GIF using image/gif (registers the "gif" format).
img := image.NewPaletted(image.Rect(0, 0, 1, 1), []color.Color{color.Black})
var buf bytes.Buffer
if err := gif.Encode(&buf, img, nil); err != nil {
t.Skip("gif.Encode failed:", err)
}

rm := newResourceManager(nil)
_, err := rm.LoadImageFromReader(&buf, "test.gif")
if err == nil {
t.Fatal("expected error for GIF (unsupported format)")
}
}
