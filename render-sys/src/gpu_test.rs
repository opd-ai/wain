// GPU shader testing infrastructure for Phase 4.5
//
// This module provides utilities for testing shaders on actual GPU hardware:
// - Allocate small test render targets
// - Submit batch buffers with compiled shaders
// - Read back rendered pixels via CPU mmap
// - Compare against software rasterizer reference output
//
// Tests are marked #[ignore] by default as they require Intel GPU hardware.
// Run with: cargo test -- --ignored

use crate::drm::DrmDevice;
use crate::allocator::{BufferAllocator, TilingFormat};
use crate::detect::GpuGeneration;

/// Test render target dimensions (small for fast tests)
pub const TEST_RT_WIDTH: u32 = 64;
pub const TEST_RT_HEIGHT: u32 = 64;
pub const TEST_RT_BPP: u32 = 4; // ARGB8888

/// Maximum per-channel difference for pixel comparison (0-255 range)
/// This accounts for minor precision differences between CPU and GPU
pub const DEFAULT_TOLERANCE: u8 = 2;

/// RGBA pixel
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Pixel {
    pub r: u8,
    pub g: u8,
    pub b: u8,
    pub a: u8,
}

impl Pixel {
    pub fn new(r: u8, g: u8, b: u8, a: u8) -> Self {
        Self { r, g, b, a }
    }

    /// Compare pixels with tolerance
    pub fn approx_eq(&self, other: &Pixel, tolerance: u8) -> bool {
        let dr = (self.r as i16 - other.r as i16).abs();
        let dg = (self.g as i16 - other.g as i16).abs();
        let db = (self.b as i16 - other.b as i16).abs();
        let da = (self.a as i16 - other.a as i16).abs();
        
        dr <= tolerance as i16 && 
        dg <= tolerance as i16 && 
        db <= tolerance as i16 && 
        da <= tolerance as i16
    }
}

/// Image buffer for test comparisons
pub struct ImageBuffer {
    pub width: u32,
    pub height: u32,
    pub pixels: Vec<Pixel>,
}

impl ImageBuffer {
    /// Create a new image buffer filled with a single color
    pub fn new_solid(width: u32, height: u32, color: Pixel) -> Self {
        let pixel_count = (width * height) as usize;
        Self {
            width,
            height,
            pixels: vec![color; pixel_count],
        }
    }

    /// Create from raw ARGB8888 bytes
    pub fn from_argb8888(width: u32, height: u32, data: &[u8]) -> Result<Self, String> {
        let expected_len = (width * height * TEST_RT_BPP) as usize;
        if data.len() != expected_len {
            return Err(format!(
                "Invalid data length: expected {} bytes, got {}",
                expected_len,
                data.len()
            ));
        }

        let mut pixels = Vec::with_capacity((width * height) as usize);
        for chunk in data.chunks_exact(4) {
            // ARGB8888 format: B, G, R, A (little-endian)
            pixels.push(Pixel::new(chunk[2], chunk[1], chunk[0], chunk[3]));
        }

        Ok(Self { width, height, pixels })
    }

    /// Get pixel at (x, y)
    pub fn get(&self, x: u32, y: u32) -> Option<&Pixel> {
        if x >= self.width || y >= self.height {
            return None;
        }
        let idx = (y * self.width + x) as usize;
        self.pixels.get(idx)
    }

    /// Compare two images with tolerance
    pub fn compare(&self, other: &ImageBuffer, tolerance: u8) -> Result<(), String> {
        if self.width != other.width || self.height != other.height {
            return Err(format!(
                "Image dimensions mismatch: {}x{} vs {}x{}",
                self.width, self.height, other.width, other.height
            ));
        }

        let mut mismatches = Vec::new();
        for y in 0..self.height {
            for x in 0..self.width {
                let idx = (y * self.width + x) as usize;
                let p1 = &self.pixels[idx];
                let p2 = &other.pixels[idx];
                
                if !p1.approx_eq(p2, tolerance) {
                    mismatches.push((x, y, *p1, *p2));
                    if mismatches.len() >= 10 {
                        // Limit error output
                        break;
                    }
                }
            }
        }

        if mismatches.is_empty() {
            Ok(())
        } else {
            let mut msg = format!(
                "Images differ: {} mismatched pixels (tolerance={})\n",
                mismatches.len(),
                tolerance
            );
            for (x, y, p1, p2) in mismatches.iter().take(5) {
                msg.push_str(&format!(
                    "  ({}, {}): ({},{},{},{}) vs ({},{},{},{})\n",
                    x, y, p1.r, p1.g, p1.b, p1.a, p2.r, p2.g, p2.b, p2.a
                ));
            }
            Err(msg)
        }
    }
}

/// GPU test context
pub struct GpuTestContext {
    allocator: BufferAllocator,
    generation: GpuGeneration,
}

impl GpuTestContext {
    /// Create a new GPU test context
    /// Returns None if no Intel GPU is available
    pub fn new() -> Option<Self> {
        const DRM_PATH: &str = "/dev/dri/renderD128";
        
        let device = DrmDevice::open(DRM_PATH).ok()?;
        let generation = device.detect_gpu_generation().ok()?;
        
        // Only support Gen9+ for now
        match generation {
            GpuGeneration::Gen9 | GpuGeneration::Gen11 | 
            GpuGeneration::Gen12 | GpuGeneration::Xe => {}
            _ => return None,
        }

        // Determine driver type based on generation
        let driver = match generation {
            GpuGeneration::Xe => crate::allocator::DriverType::Xe,
            _ => crate::allocator::DriverType::I915,
        };

        let allocator = BufferAllocator::new(device, driver);

        Some(Self {
            allocator,
            generation,
        })
    }

    /// Allocate a test render target
    pub fn allocate_render_target(&mut self) -> Result<crate::allocator::Buffer, String> {
        self.allocator
            .allocate(TEST_RT_WIDTH, TEST_RT_HEIGHT, TEST_RT_BPP, TilingFormat::None)
            .map_err(|e| format!("Failed to allocate render target: {:?}", e))
    }

    /// Read back pixels from a buffer (placeholder - needs mmap implementation)
    pub fn readback_pixels(&self, _buffer: &crate::allocator::Buffer) -> Result<ImageBuffer, String> {
        // TODO: Implement CPU mmap and readback
        // For now, return a placeholder solid color image
        Ok(ImageBuffer::new_solid(
            TEST_RT_WIDTH,
            TEST_RT_HEIGHT,
            Pixel::new(0, 0, 255, 255), // Blue
        ))
    }

    /// GPU generation
    pub fn generation(&self) -> GpuGeneration {
        self.generation
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pixel_comparison() {
        let p1 = Pixel::new(100, 150, 200, 255);
        let p2 = Pixel::new(101, 151, 199, 254);
        
        assert!(p1.approx_eq(&p2, 2));
        assert!(!p1.approx_eq(&p2, 0));
    }

    #[test]
    fn test_image_buffer_creation() {
        let img = ImageBuffer::new_solid(64, 64, Pixel::new(255, 0, 0, 255));
        
        assert_eq!(img.width, 64);
        assert_eq!(img.height, 64);
        assert_eq!(img.pixels.len(), 64 * 64);
        assert_eq!(img.get(0, 0).unwrap(), &Pixel::new(255, 0, 0, 255));
    }

    #[test]
    fn test_image_comparison_identical() {
        let img1 = ImageBuffer::new_solid(64, 64, Pixel::new(255, 0, 0, 255));
        let img2 = ImageBuffer::new_solid(64, 64, Pixel::new(255, 0, 0, 255));
        
        assert!(img1.compare(&img2, 0).is_ok());
    }

    #[test]
    fn test_image_comparison_different() {
        let img1 = ImageBuffer::new_solid(64, 64, Pixel::new(255, 0, 0, 255));
        let img2 = ImageBuffer::new_solid(64, 64, Pixel::new(0, 0, 255, 255));
        
        assert!(img1.compare(&img2, 2).is_err());
    }

    #[test]
    fn test_image_from_argb8888() {
        // Create 2x2 test image: ARGB8888 format (B, G, R, A in memory)
        let data = vec![
            // Row 0
            255, 0, 0, 255,   // Blue pixel (0,0)
            0, 255, 0, 255,   // Green pixel (1,0)
            // Row 1
            0, 0, 255, 255,   // Red pixel (0,1)
            255, 255, 255, 255, // White pixel (1,1)
        ];
        
        let img = ImageBuffer::from_argb8888(2, 2, &data).unwrap();
        
        assert_eq!(img.get(0, 0).unwrap(), &Pixel::new(0, 0, 255, 255)); // Blue
        assert_eq!(img.get(1, 0).unwrap(), &Pixel::new(0, 255, 0, 255)); // Green
        assert_eq!(img.get(0, 1).unwrap(), &Pixel::new(255, 0, 0, 255)); // Red
        assert_eq!(img.get(1, 1).unwrap(), &Pixel::new(255, 255, 255, 255)); // White
    }

    #[test]
    #[ignore] // Requires Intel GPU hardware
    fn test_gpu_context_creation() {
        if let Some(ctx) = GpuTestContext::new() {
            println!("GPU test context created: {:?}", ctx.generation());
        } else {
            println!("No Intel GPU available, test skipped");
        }
    }
}
