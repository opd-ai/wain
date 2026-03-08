/// Surface State & Sampler State Management
///
/// This module provides encoding for Intel GPU surface and sampler state entries.
/// Surface states describe render targets and texture sources, while sampler states
/// configure texture filtering modes.
///
/// References:
/// - Intel PRMs Volume 5 (Memory Views)
/// - Intel PRMs Volume 7 (3D Media GPGPU)
/// - Mesa ISL (Intel Surface Layout) library

use crate::detect::GpuGeneration;
use std::io;

/// Surface format enumeration (subset of Intel formats).
///
/// These correspond to SURFACE_FORMAT in Intel PRM Vol. 5.
/// Only the formats needed for UI rendering are included.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SurfaceFormat {
    /// 8-bit red, green, blue, alpha (32 bpp)
    R8G8B8A8_UNORM = 0x044,
    /// 8-bit blue, green, red, alpha (32 bpp) - common format
    B8G8R8A8_UNORM = 0x0C0,
    /// 8-bit single channel (grayscale, for SDF atlas)
    R8_UNORM = 0x002,
    /// 16-bit floating point RGBA (64 bpp)
    R16G16B16A16_FLOAT = 0x0C2,
}

impl SurfaceFormat {
    /// Get the bits per pixel for this format.
    pub fn bpp(&self) -> u32 {
        match self {
            SurfaceFormat::R8_UNORM => 8,
            SurfaceFormat::R8G8B8A8_UNORM | SurfaceFormat::B8G8R8A8_UNORM => 32,
            SurfaceFormat::R16G16B16A16_FLOAT => 64,
        }
    }
}

/// Surface type enumeration.
///
/// Specifies how the GPU should interpret the surface memory layout.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SurfaceType {
    /// 1D surface (line)
    Surface1D = 0,
    /// 2D surface (image, render target)
    Surface2D = 1,
    /// 3D surface (volume)
    Surface3D = 2,
    /// Cube map
    SurfaceCube = 3,
    /// Buffer (unstructured linear data)
    Buffer = 4,
}

/// Tiling mode for surface memory layout.
///
/// Matches the tiling modes used in buffer allocation.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TilingMode {
    /// Linear (no tiling) - for CPU-accessible buffers
    Linear = 0,
    /// X-tiling - for general render targets
    TileX = 1,
    /// Y-tiling - for better cache utilization
    TileY = 2,
    /// Tile-Yf (Gen9+) - finer-grained Y-tiling
    TileYf = 3,
}

/// RENDER_SURFACE_STATE structure.
///
/// Describes a surface (render target or texture source) to the GPU.
/// The structure size and layout varies by generation; this implementation
/// targets Gen9-Gen12.
///
/// Gen9-Gen12 format: 16 DWords (64 bytes)
#[derive(Debug, Clone)]
pub struct RenderSurfaceState {
    /// Surface type (1D/2D/3D/Cube/Buffer)
    pub surface_type: SurfaceType,
    /// Pixel format
    pub surface_format: SurfaceFormat,
    /// Tiling mode
    pub tiling_mode: TilingMode,
    /// Surface width in pixels (minus 1)
    pub width: u32,
    /// Surface height in pixels (minus 1)
    pub height: u32,
    /// Surface depth (for 3D) or array length (minus 1)
    pub depth: u32,
    /// Surface pitch in bytes (minus 1) - for linear surfaces
    pub surface_pitch: u32,
    /// Minimum LOD (for mipmapping, usually 0)
    pub min_lod: u32,
    /// Mip count (minus 1, usually 0 for non-mipmapped)
    pub mip_count: u32,
    /// Base address of surface memory (GPU virtual address)
    /// This will be patched by relocation
    pub base_address: u64,
    /// X offset in pixels (for texture arrays)
    pub x_offset: u32,
    /// Y offset in pixels (for texture arrays)
    pub y_offset: u32,
    /// Enable shader writes (for UAVs/render targets)
    pub shader_channel_select_red: u32,
    pub shader_channel_select_green: u32,
    pub shader_channel_select_blue: u32,
    pub shader_channel_select_alpha: u32,
}

impl RenderSurfaceState {
    /// Create a new render surface state with default settings.
    pub fn new() -> Self {
        Self {
            surface_type: SurfaceType::Surface2D,
            surface_format: SurfaceFormat::B8G8R8A8_UNORM,
            tiling_mode: TilingMode::Linear,
            width: 0,
            height: 0,
            depth: 0,
            surface_pitch: 0,
            min_lod: 0,
            mip_count: 0,
            base_address: 0,
            x_offset: 0,
            y_offset: 0,
            // Identity swizzle (R->R, G->G, B->B, A->A)
            shader_channel_select_red: 4,    // RED
            shader_channel_select_green: 5,  // GREEN
            shader_channel_select_blue: 6,   // BLUE
            shader_channel_select_alpha: 7,  // ALPHA
        }
    }

    /// Create a 2D render target surface state.
    ///
    /// # Arguments
    /// * `width` - Surface width in pixels
    /// * `height` - Surface height in pixels
    /// * `pitch` - Surface pitch in bytes
    /// * `format` - Pixel format
    /// * `tiling` - Tiling mode
    pub fn render_target(
        width: u32,
        height: u32,
        pitch: u32,
        format: SurfaceFormat,
        tiling: TilingMode,
    ) -> Self {
        let mut state = Self::new();
        state.surface_type = SurfaceType::Surface2D;
        state.surface_format = format;
        state.tiling_mode = tiling;
        state.width = width.saturating_sub(1);
        state.height = height.saturating_sub(1);
        state.depth = 0;
        state.surface_pitch = pitch.saturating_sub(1);
        state.mip_count = 0;
        state
    }

    /// Create a 2D texture source surface state.
    ///
    /// # Arguments
    /// * `width` - Texture width in pixels
    /// * `height` - Texture height in pixels
    /// * `pitch` - Texture pitch in bytes
    /// * `format` - Pixel format
    pub fn texture_2d(
        width: u32,
        height: u32,
        pitch: u32,
        format: SurfaceFormat,
    ) -> Self {
        let mut state = Self::new();
        state.surface_type = SurfaceType::Surface2D;
        state.surface_format = format;
        state.tiling_mode = TilingMode::Linear;
        state.width = width.saturating_sub(1);
        state.height = height.saturating_sub(1);
        state.depth = 0;
        state.surface_pitch = pitch.saturating_sub(1);
        state.mip_count = 0;
        state
    }

    /// Serialize the surface state to binary (16 DWords for Gen9-Gen12).
    ///
    /// The base_address field (DWords 8-9) will typically need a relocation entry.
    pub fn serialize(&self, generation: GpuGeneration) -> Vec<u32> {
        let mut dwords = vec![0u32; 16];

        // DWord 0: Surface type, format
        dwords[0] = ((self.surface_type as u32) << 29)
            | ((self.surface_format as u32) << 18);

        // DWord 1: Surface array (not used for single surfaces)
        dwords[1] = 0;

        // DWord 2: Width, height
        dwords[2] = ((self.height & 0x3FFF) << 16) | (self.width & 0x3FFF);

        // DWord 3: Depth, pitch
        dwords[3] = ((self.surface_pitch & 0x1FFFF) << 3) | (self.depth & 0x7FF);

        // DWord 4: MinLOD, MipCount, tiling mode
        dwords[4] = ((self.tiling_mode as u32) << 12)
            | ((self.mip_count & 0xF) << 4)
            | (self.min_lod & 0xF);

        // DWord 5: X offset, Y offset
        dwords[5] = ((self.y_offset & 0x3FFF) << 16) | ((self.x_offset >> 2) & 0x1FFF);

        // DWord 6-7: Auxiliary surfaces (not used for basic rendering)
        dwords[6] = 0;
        dwords[7] = 0;

        // DWord 8-9: Base address (will be patched by relocation)
        dwords[8] = (self.base_address & 0xFFFFFFFF) as u32;
        dwords[9] = ((self.base_address >> 32) & 0xFFFFFFFF) as u32;

        // DWord 10: Base address high (Gen12+)
        dwords[10] = 0;

        // DWord 11: Shader channel select
        dwords[11] = ((self.shader_channel_select_alpha & 0x7) << 16)
            | ((self.shader_channel_select_blue & 0x7) << 12)
            | ((self.shader_channel_select_green & 0x7) << 8)
            | ((self.shader_channel_select_red & 0x7) << 4);

        // DWord 12-15: Resource min LOD, clear color, etc. (default 0)
        dwords[12] = 0;
        dwords[13] = 0;
        dwords[14] = 0;
        dwords[15] = 0;

        dwords
    }
}

impl Default for RenderSurfaceState {
    fn default() -> Self {
        Self::new()
    }
}

/// Sampler filter mode enumeration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SamplerFilter {
    /// Nearest neighbor (point sampling)
    Nearest = 0,
    /// Bilinear interpolation
    Linear = 1,
}

/// Sampler address mode (wrapping behavior at texture edges).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SamplerAddressMode {
    /// Repeat (wrap)
    Repeat = 0,
    /// Clamp to edge
    ClampToEdge = 1,
    /// Clamp to border color
    ClampToBorder = 2,
    /// Mirror repeat
    Mirror = 3,
}

/// SAMPLER_STATE structure.
///
/// Configures texture sampling behavior (filtering, wrapping).
///
/// Gen9-Gen12 format: 4 DWords (16 bytes)
#[derive(Debug, Clone)]
pub struct SamplerState {
    /// Magnification filter (when texture is enlarged)
    pub mag_filter: SamplerFilter,
    /// Minification filter (when texture is shrunk)
    pub min_filter: SamplerFilter,
    /// Mipmap filter
    pub mip_filter: SamplerFilter,
    /// Address mode for U coordinate (horizontal)
    pub address_mode_u: SamplerAddressMode,
    /// Address mode for V coordinate (vertical)
    pub address_mode_v: SamplerAddressMode,
    /// Address mode for W coordinate (depth)
    pub address_mode_w: SamplerAddressMode,
    /// LOD bias (signed, 8.8 fixed point)
    pub lod_bias: i16,
    /// Minimum LOD
    pub min_lod: u32,
    /// Maximum LOD
    pub max_lod: u32,
    /// Border color (for ClampToBorder mode)
    pub border_color_red: f32,
    pub border_color_green: f32,
    pub border_color_blue: f32,
    pub border_color_alpha: f32,
}

impl SamplerState {
    /// Create a new sampler state with default settings (bilinear, clamp to edge).
    pub fn new() -> Self {
        Self {
            mag_filter: SamplerFilter::Linear,
            min_filter: SamplerFilter::Linear,
            mip_filter: SamplerFilter::Nearest,
            address_mode_u: SamplerAddressMode::ClampToEdge,
            address_mode_v: SamplerAddressMode::ClampToEdge,
            address_mode_w: SamplerAddressMode::ClampToEdge,
            lod_bias: 0,
            min_lod: 0,
            max_lod: 0,
            border_color_red: 0.0,
            border_color_green: 0.0,
            border_color_blue: 0.0,
            border_color_alpha: 0.0,
        }
    }

    /// Create a bilinear sampler with clamp to edge (common for UI textures).
    pub fn bilinear() -> Self {
        Self::new()
    }

    /// Create a nearest neighbor sampler with clamp to edge.
    pub fn nearest() -> Self {
        let mut state = Self::new();
        state.mag_filter = SamplerFilter::Nearest;
        state.min_filter = SamplerFilter::Nearest;
        state.mip_filter = SamplerFilter::Nearest;
        state
    }

    /// Serialize the sampler state to binary (4 DWords for Gen9-Gen12).
    pub fn serialize(&self, generation: GpuGeneration) -> Vec<u32> {
        let mut dwords = vec![0u32; 4];

        // DWord 0: Filter modes and address modes
        dwords[0] = ((self.address_mode_w as u32) << 6)
            | ((self.address_mode_v as u32) << 3)
            | (self.address_mode_u as u32)
            | ((self.mag_filter as u32) << 17)
            | ((self.min_filter as u32) << 14)
            | ((self.mip_filter as u32) << 20);

        // DWord 1: LOD bias and clamps
        let lod_bias_fixed = (self.lod_bias as u32) & 0x1FFF;
        dwords[1] = (lod_bias_fixed << 1)
            | ((self.min_lod & 0xFFF) << 20)
            | ((self.max_lod & 0xFFF) << 8);

        // DWord 2-3: Border color (simplified, use index 0 for default)
        dwords[2] = 0; // Border color pointer (use pre-defined color)
        dwords[3] = 0; // Reserved

        dwords
    }
}

impl Default for SamplerState {
    fn default() -> Self {
        Self::new()
    }
}

/// Binding table manager.
///
/// Manages a binding table that maps shader binding indices to surface state entries.
/// The binding table is itself a surface (array of offsets) in the surface state heap.
pub struct BindingTable {
    /// Binding table entries (offsets into surface state heap, in bytes)
    entries: Vec<u32>,
}

impl BindingTable {
    /// Create a new empty binding table.
    pub fn new() -> Self {
        Self {
            entries: Vec::new(),
        }
    }

    /// Create a binding table with a pre-allocated capacity.
    pub fn with_capacity(capacity: usize) -> Self {
        Self {
            entries: Vec::with_capacity(capacity),
        }
    }

    /// Add a surface binding.
    ///
    /// # Arguments
    /// * `surface_state_offset` - Offset in bytes from the start of the surface state heap
    ///
    /// # Returns
    /// The binding index (0-based) that shaders use to reference this surface.
    pub fn add_binding(&mut self, surface_state_offset: u32) -> u32 {
        let index = self.entries.len() as u32;
        self.entries.push(surface_state_offset);
        index
    }

    /// Get the number of bindings.
    pub fn len(&self) -> usize {
        self.entries.len()
    }

    /// Check if the binding table is empty.
    pub fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }

    /// Serialize the binding table to binary.
    ///
    /// The binding table is an array of surface state offsets (one DWord per binding).
    pub fn serialize(&self) -> Vec<u32> {
        self.entries.clone()
    }

    /// Validate that all binding indices are aligned.
    ///
    /// Surface state offsets must be 64-byte aligned.
    pub fn validate(&self) -> Result<(), io::Error> {
        for (i, offset) in self.entries.iter().enumerate() {
            if offset % 64 != 0 {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidInput,
                    format!("Binding {} has misaligned offset: {} (must be 64-byte aligned)", i, offset),
                ));
            }
        }
        Ok(())
    }
}

impl Default for BindingTable {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn surface_format_bpp() {
        assert_eq!(SurfaceFormat::R8_UNORM.bpp(), 8);
        assert_eq!(SurfaceFormat::R8G8B8A8_UNORM.bpp(), 32);
        assert_eq!(SurfaceFormat::B8G8R8A8_UNORM.bpp(), 32);
        assert_eq!(SurfaceFormat::R16G16B16A16_FLOAT.bpp(), 64);
    }

    #[test]
    fn render_target_creation() {
        let rt = RenderSurfaceState::render_target(
            1920,
            1080,
            1920 * 4,
            SurfaceFormat::B8G8R8A8_UNORM,
            TilingMode::TileX,
        );

        assert_eq!(rt.surface_type, SurfaceType::Surface2D);
        assert_eq!(rt.surface_format, SurfaceFormat::B8G8R8A8_UNORM);
        assert_eq!(rt.tiling_mode, TilingMode::TileX);
        assert_eq!(rt.width, 1919); // width - 1
        assert_eq!(rt.height, 1079); // height - 1
        assert_eq!(rt.surface_pitch, 1920 * 4 - 1); // pitch - 1
    }

    #[test]
    fn texture_2d_creation() {
        let tex = RenderSurfaceState::texture_2d(
            256,
            256,
            256,
            SurfaceFormat::R8_UNORM,
        );

        assert_eq!(tex.surface_type, SurfaceType::Surface2D);
        assert_eq!(tex.surface_format, SurfaceFormat::R8_UNORM);
        assert_eq!(tex.tiling_mode, TilingMode::Linear);
        assert_eq!(tex.width, 255); // width - 1
        assert_eq!(tex.height, 255); // height - 1
    }

    #[test]
    fn surface_state_serialization() {
        let rt = RenderSurfaceState::render_target(
            1024,
            768,
            1024 * 4,
            SurfaceFormat::B8G8R8A8_UNORM,
            TilingMode::Linear,
        );

        let dwords = rt.serialize(GpuGeneration::Gen12);
        
        // Should produce 16 DWords (64 bytes)
        assert_eq!(dwords.len(), 16);
        
        // DWord 0: Surface type and format
        let dw0 = dwords[0];
        assert_eq!((dw0 >> 29) & 0x7, SurfaceType::Surface2D as u32);
        assert_eq!((dw0 >> 18) & 0x1FF, SurfaceFormat::B8G8R8A8_UNORM as u32);
        
        // DWord 2: Width and height (minus 1)
        let dw2 = dwords[2];
        assert_eq!(dw2 & 0x3FFF, 1023); // width - 1
        assert_eq!((dw2 >> 16) & 0x3FFF, 767); // height - 1
    }

    #[test]
    fn sampler_bilinear() {
        let sampler = SamplerState::bilinear();
        
        assert_eq!(sampler.mag_filter, SamplerFilter::Linear);
        assert_eq!(sampler.min_filter, SamplerFilter::Linear);
        assert_eq!(sampler.address_mode_u, SamplerAddressMode::ClampToEdge);
        assert_eq!(sampler.address_mode_v, SamplerAddressMode::ClampToEdge);
    }

    #[test]
    fn sampler_nearest() {
        let sampler = SamplerState::nearest();
        
        assert_eq!(sampler.mag_filter, SamplerFilter::Nearest);
        assert_eq!(sampler.min_filter, SamplerFilter::Nearest);
        assert_eq!(sampler.mip_filter, SamplerFilter::Nearest);
    }

    #[test]
    fn sampler_state_serialization() {
        let sampler = SamplerState::bilinear();
        let dwords = sampler.serialize(GpuGeneration::Gen12);
        
        // Should produce 4 DWords (16 bytes)
        assert_eq!(dwords.len(), 4);
        
        // DWord 0 contains filter modes
        let dw0 = dwords[0];
        assert_eq!((dw0 >> 17) & 0x7, SamplerFilter::Linear as u32);
        assert_eq!((dw0 >> 14) & 0x7, SamplerFilter::Linear as u32);
    }

    #[test]
    fn binding_table_basic() {
        let mut bt = BindingTable::new();
        assert_eq!(bt.len(), 0);
        assert!(bt.is_empty());
        
        let idx0 = bt.add_binding(0);
        let idx1 = bt.add_binding(64);
        let idx2 = bt.add_binding(128);
        
        assert_eq!(idx0, 0);
        assert_eq!(idx1, 1);
        assert_eq!(idx2, 2);
        assert_eq!(bt.len(), 3);
        assert!(!bt.is_empty());
    }

    #[test]
    fn binding_table_serialization() {
        let mut bt = BindingTable::new();
        bt.add_binding(0);
        bt.add_binding(64);
        bt.add_binding(128);
        
        let dwords = bt.serialize();
        assert_eq!(dwords.len(), 3);
        assert_eq!(dwords[0], 0);
        assert_eq!(dwords[1], 64);
        assert_eq!(dwords[2], 128);
    }

    #[test]
    fn binding_table_validation_aligned() {
        let mut bt = BindingTable::new();
        bt.add_binding(0);
        bt.add_binding(64);
        bt.add_binding(128);
        
        assert!(bt.validate().is_ok());
    }

    #[test]
    fn binding_table_validation_misaligned() {
        let mut bt = BindingTable::new();
        bt.add_binding(0);
        bt.add_binding(63); // Not 64-byte aligned
        
        assert!(bt.validate().is_err());
    }

    #[test]
    fn binding_table_with_capacity() {
        let bt = BindingTable::with_capacity(10);
        assert_eq!(bt.len(), 0);
        assert!(bt.is_empty());
    }
}
