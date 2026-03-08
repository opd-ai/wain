/// 3D State Commands
///
/// These commands configure specific stages of the 3D rendering pipeline:
/// - Clip state
/// - Rasterization (SF - Strip/Fan)
/// - Fragment shader (WM - Windower/Masker)
/// - Pixel shader (PS)

use super::{GpuCommand, CommandType};

/// 3DSTATE_CLIP - Clipping configuration
///
/// Configures the clipping stage of the 3D pipeline, including
/// viewport clipping, user clip planes, and clip modes.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1: Clip enable flags and modes
/// - DWord 2-3: Additional clipping parameters
#[derive(Debug, Clone)]
pub struct State3DClip {
    /// Enable clipping
    pub clip_enable: bool,
    /// API mode (0=OGL, 1=DX)
    pub api_mode: u32,
    /// Viewport XY clip test enable
    pub viewport_xy_clip_test_enable: bool,
}

impl State3DClip {
    /// Create a new 3DSTATE_CLIP with default settings.
    pub fn new() -> Self {
        Self {
            clip_enable: true,
            api_mode: 0, // OpenGL mode
            viewport_xy_clip_test_enable: true,
        }
    }
}

impl Default for State3DClip {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for State3DClip {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7812; // 3DSTATE_CLIP opcode
        let length = 3; // 4 DWords total
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let mut dw1 = 0u32;
        if self.clip_enable {
            dw1 |= 1 << 31; // Clip enable
        }
        dw1 |= (self.api_mode & 1) << 30; // API mode
        if self.viewport_xy_clip_test_enable {
            dw1 |= 1 << 28; // Viewport XY clip test enable
        }
        
        vec![
            dw0,
            dw1,
            0, // DWord 2 (reserved/additional parameters)
            0, // DWord 3 (reserved)
        ]
    }
}

/// 3DSTATE_SF - Rasterization setup (Strip/Fan)
///
/// Configures the rasterization stage, including:
/// - Cull mode
/// - Fill mode (wireframe/solid)
/// - Front face winding
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1-3: Rasterization parameters
#[derive(Debug, Clone)]
pub struct State3DSF {
    /// Cull mode: 0=none, 1=front, 2=back, 3=both
    pub cull_mode: u32,
    /// Front face winding: 0=CW, 1=CCW
    pub front_winding: u32,
}

impl State3DSF {
    /// Create a new 3DSTATE_SF with default settings (no culling, CCW front).
    pub fn new() -> Self {
        Self {
            cull_mode: 0, // No culling
            front_winding: 1, // CCW
        }
    }
}

impl Default for State3DSF {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for State3DSF {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7813; // 3DSTATE_SF opcode
        let length = 3; // 4 DWords total
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let dw1 = ((self.cull_mode & 3) << 29) | ((self.front_winding & 1) << 0);
        
        vec![
            dw0,
            dw1,
            0, // DWord 2 (point width, line width, etc.)
            0, // DWord 3 (additional parameters)
        ]
    }
}

/// 3DSTATE_WM - Windower/Masker configuration
///
/// Configures the fragment shader stage dispatch and early depth/stencil
/// testing modes.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1-2: Fragment shader dispatch and testing configuration
#[derive(Debug, Clone)]
pub struct State3DWM {
    /// Enable early depth/stencil test
    pub early_depth_stencil_control: bool,
    /// Pixel shader dispatch enable
    pub pixel_shader_kill_enable: bool,
}

impl State3DWM {
    /// Create a new 3DSTATE_WM with default settings.
    pub fn new() -> Self {
        Self {
            early_depth_stencil_control: false,
            pixel_shader_kill_enable: true,
        }
    }
}

impl Default for State3DWM {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for State3DWM {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7814; // 3DSTATE_WM opcode
        let length = 1; // 2 DWords total
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let mut dw1 = 0u32;
        if self.early_depth_stencil_control {
            dw1 |= 1 << 31; // Early depth/stencil control
        }
        if self.pixel_shader_kill_enable {
            dw1 |= 1 << 25; // Pixel shader kill enable
        }
        
        vec![dw0, dw1]
    }
}

/// 3DSTATE_PS - Pixel Shader configuration
///
/// Configures the pixel shader (fragment shader) program and dispatch.
/// Points to the compiled shader kernel and sets execution parameters.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1-2: Kernel start pointer (64-bit address)
/// - DWord 3: Thread dispatch settings
/// - DWord 4-11: Additional shader parameters
#[derive(Debug, Clone)]
pub struct State3DPS {
    /// Kernel start address (GPU virtual address of shader binary)
    pub kernel_start_pointer: u64,
    /// 8-pixel dispatch enable
    pub dispatch_8: bool,
    /// 16-pixel dispatch enable
    pub dispatch_16: bool,
    /// 32-pixel dispatch enable
    pub dispatch_32: bool,
}

impl State3DPS {
    /// Create a new 3DSTATE_PS with default settings.
    pub fn new(kernel_addr: u64) -> Self {
        Self {
            kernel_start_pointer: kernel_addr,
            dispatch_8: true,
            dispatch_16: false,
            dispatch_32: false,
        }
    }
}

impl GpuCommand for State3DPS {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7820; // 3DSTATE_PS opcode
        let length = 11; // 12 DWords total
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let dw1 = (self.kernel_start_pointer & 0xFFFFFFFF) as u32;
        let dw2 = (self.kernel_start_pointer >> 32) as u32;
        
        let mut dw3 = 0u32;
        if self.dispatch_8 {
            dw3 |= 1 << 0; // 8-pixel dispatch
        }
        if self.dispatch_16 {
            dw3 |= 1 << 1; // 16-pixel dispatch
        }
        if self.dispatch_32 {
            dw3 |= 1 << 2; // 32-pixel dispatch
        }
        
        vec![
            dw0, dw1, dw2, dw3,
            0, 0, 0, 0, // DWords 4-7 (shader parameters)
            0, 0, 0, 0, // DWords 8-11 (additional parameters)
        ]
    }
}

/// 3DSTATE_BLEND_STATE - Blending and color write configuration
///
/// Configures alpha blending, logic operations, and color write masks
/// for up to 8 render targets.
///
/// Gen9-Gen12 format:
/// - Pointer to BLEND_STATE structure in dynamic state heap
/// - Each render target uses 2 DWords in the state structure
#[derive(Debug, Clone)]
pub struct State3DBlendState {
    /// Pointer to BLEND_STATE structure (64-byte aligned)
    pub blend_state_pointer: u64,
}

impl State3DBlendState {
    /// Create a new 3DSTATE_BLEND_STATE command.
    ///
    /// # Arguments
    ///
    /// * `pointer` - GPU address of BLEND_STATE structure (must be 64-byte aligned)
    pub fn new(pointer: u64) -> Self {
        assert_eq!(pointer & 0x3F, 0, "BLEND_STATE pointer must be 64-byte aligned");
        Self {
            blend_state_pointer: pointer,
        }
    }
}

impl GpuCommand for State3DBlendState {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x781D; // 3DSTATE_BLEND_STATE_POINTERS opcode
        let length = 1; // 2 DWords total

        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        let dw1 = (self.blend_state_pointer & !0x3F) as u32; // Low 32 bits, enforce alignment

        vec![dw0, dw1]
    }
}

/// BLEND_STATE structure for render target blending configuration
///
/// This structure lives in the dynamic state heap and is referenced
/// by 3DSTATE_BLEND_STATE_POINTERS.
#[derive(Debug, Clone)]
pub struct BlendState {
    /// Per-render-target blend configurations (up to 8)
    pub render_targets: Vec<RenderTargetBlend>,
}

/// Blend configuration for a single render target
#[derive(Debug, Clone)]
pub struct RenderTargetBlend {
    /// Enable alpha blending
    pub blend_enable: bool,
    /// Source blend factor for color
    pub src_color_blend: BlendFactor,
    /// Destination blend factor for color
    pub dst_color_blend: BlendFactor,
    /// Color blend operation
    pub color_blend_op: BlendOp,
    /// Source blend factor for alpha
    pub src_alpha_blend: BlendFactor,
    /// Destination blend factor for alpha
    pub dst_alpha_blend: BlendFactor,
    /// Alpha blend operation
    pub alpha_blend_op: BlendOp,
    /// Color write mask (RGBA bits)
    pub write_mask: u8,
}

/// Blend factor enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u32)]
pub enum BlendFactor {
    Zero = 0,
    One = 1,
    SrcColor = 2,
    OneMinusSrcColor = 3,
    DstColor = 4,
    OneMinusDstColor = 5,
    SrcAlpha = 6,
    OneMinusSrcAlpha = 7,
    DstAlpha = 8,
    OneMinusDstAlpha = 9,
    ConstantColor = 10,
    OneMinusConstantColor = 11,
    ConstantAlpha = 12,
    OneMinusConstantAlpha = 13,
    SrcAlphaSaturate = 14,
}

/// Blend operation enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u32)]
pub enum BlendOp {
    Add = 0,
    Subtract = 1,
    ReverseSubtract = 2,
    Min = 3,
    Max = 4,
}

impl RenderTargetBlend {
    /// Create default blend configuration (no blending, all writes enabled)
    pub fn opaque() -> Self {
        Self {
            blend_enable: false,
            src_color_blend: BlendFactor::One,
            dst_color_blend: BlendFactor::Zero,
            color_blend_op: BlendOp::Add,
            src_alpha_blend: BlendFactor::One,
            dst_alpha_blend: BlendFactor::Zero,
            alpha_blend_op: BlendOp::Add,
            write_mask: 0xF, // RGBA all enabled
        }
    }

    /// Create alpha blending configuration (standard Porter-Duff SrcOver)
    pub fn alpha_blend() -> Self {
        Self {
            blend_enable: true,
            src_color_blend: BlendFactor::SrcAlpha,
            dst_color_blend: BlendFactor::OneMinusSrcAlpha,
            color_blend_op: BlendOp::Add,
            src_alpha_blend: BlendFactor::One,
            dst_alpha_blend: BlendFactor::OneMinusSrcAlpha,
            alpha_blend_op: BlendOp::Add,
            write_mask: 0xF,
        }
    }

    /// Serialize to 2 DWords per Intel PRM format
    pub fn serialize(&self) -> [u32; 2] {
        let dw0 = if self.blend_enable { 1 << 31 } else { 0 }
            | ((self.src_color_blend as u32) << 26)
            | ((self.dst_color_blend as u32) << 21)
            | ((self.color_blend_op as u32) << 18)
            | ((self.src_alpha_blend as u32) << 13)
            | ((self.dst_alpha_blend as u32) << 8)
            | ((self.alpha_blend_op as u32) << 5);

        let dw1 = (self.write_mask as u32) << 27;

        [dw0, dw1]
    }
}

impl BlendState {
    /// Create blend state with a single render target (opaque)
    pub fn opaque() -> Self {
        Self {
            render_targets: vec![RenderTargetBlend::opaque()],
        }
    }

    /// Create blend state with a single render target (alpha blending)
    pub fn alpha_blend() -> Self {
        Self {
            render_targets: vec![RenderTargetBlend::alpha_blend()],
        }
    }

    /// Serialize to DWords for upload to dynamic state heap
    pub fn serialize(&self) -> Vec<u32> {
        let mut result = Vec::new();
        for rt in &self.render_targets {
            let [dw0, dw1] = rt.serialize();
            result.push(dw0);
            result.push(dw1);
        }
        result
    }
}

/// 3DSTATE_VERTEX_BUFFERS - Vertex buffer configuration
///
/// Defines the layout and location of vertex buffers in GPU memory.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWords 1-4: Per-buffer state (repeated for each buffer)
#[derive(Debug, Clone)]
pub struct State3DVertexBuffers {
    /// Vertex buffer descriptors
    pub buffers: Vec<VertexBufferDescriptor>,
}

/// Vertex buffer descriptor
#[derive(Debug, Clone)]
pub struct VertexBufferDescriptor {
    /// Buffer index (0-31)
    pub index: u32,
    /// Buffer start address
    pub address: u64,
    /// Buffer size in bytes
    pub size: u32,
    /// Stride between vertices in bytes
    pub stride: u32,
}

impl State3DVertexBuffers {
    /// Create an empty vertex buffer state.
    pub fn new() -> Self {
        Self {
            buffers: Vec::new(),
        }
    }
    
    /// Add a vertex buffer.
    pub fn add_buffer(mut self, index: u32, address: u64, size: u32, stride: u32) -> Self {
        self.buffers.push(VertexBufferDescriptor {
            index,
            address,
            size,
            stride,
        });
        self
    }
}

impl Default for State3DVertexBuffers {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for State3DVertexBuffers {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7808; // 3DSTATE_VERTEX_BUFFERS
        let length = (self.buffers.len() * 4) - 1; // 4 DWords per buffer, minus 1 for length encoding
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | (length as u32);
        
        let mut result = vec![dw0];
        
        for buf in &self.buffers {
            // DWord 0: Buffer index and stride
            result.push((buf.index << 26) | (buf.stride & 0x7FF));
            // DWord 1-2: Buffer address
            result.push((buf.address & 0xFFFFFFFF) as u32);
            result.push((buf.address >> 32) as u32);
            // DWord 3: Buffer size
            result.push(buf.size);
        }
        
        result
    }
}

/// 3DSTATE_VERTEX_ELEMENTS - Vertex element format
///
/// Defines how vertex attributes are extracted from vertex buffers.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWords 1-2: Per-element state (repeated for each element)
#[derive(Debug, Clone)]
pub struct State3DVertexElements {
    /// Vertex element descriptors
    pub elements: Vec<VertexElementDescriptor>,
}

/// Vertex element descriptor
#[derive(Debug, Clone)]
pub struct VertexElementDescriptor {
    /// Buffer index to read from
    pub buffer_index: u32,
    /// Offset within buffer (bytes)
    pub offset: u32,
    /// Format (R32G32B32A32_FLOAT = 0x123, etc.)
    pub format: u32,
}

impl State3DVertexElements {
    /// Create an empty vertex element state.
    pub fn new() -> Self {
        Self {
            elements: Vec::new(),
        }
    }
    
    /// Add a vertex element.
    pub fn add_element(mut self, buffer_index: u32, offset: u32, format: u32) -> Self {
        self.elements.push(VertexElementDescriptor {
            buffer_index,
            offset,
            format,
        });
        self
    }
}

impl Default for State3DVertexElements {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for State3DVertexElements {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7809; // 3DSTATE_VERTEX_ELEMENTS
        let length = (self.elements.len() * 2) - 1; // 2 DWords per element
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | (length as u32);
        
        let mut result = vec![dw0];
        
        for elem in &self.elements {
            // DWord 0: Element state
            result.push((elem.buffer_index << 26) | (elem.offset & 0x7FF));
            // DWord 1: Format
            result.push(elem.format);
        }
        
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn state_3d_clip_default() {
        let cmd = State3DClip::new();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 4);
        assert_ne!(dwords[1] & (1 << 31), 0); // Clip enable
        assert_ne!(dwords[1] & (1 << 28), 0); // Viewport XY clip
    }

    #[test]
    fn state_3d_sf_default() {
        let cmd = State3DSF::new();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 4);
        assert_eq!(dwords[1] & 1, 1); // CCW winding
    }

    #[test]
    fn state_3d_wm_default() {
        let cmd = State3DWM::new();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 2);
        assert_ne!(dwords[1] & (1 << 25), 0); // Pixel shader kill enable
    }

    #[test]
    fn state_3d_ps_kernel_addr() {
        let cmd = State3DPS::new(0xDEADBEEF_12345678);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 12);
        assert_eq!(dwords[1], 0x12345678); // Low 32 bits
        assert_eq!(dwords[2], 0xDEADBEEF); // High 32 bits
        assert_ne!(dwords[3] & 1, 0); // 8-pixel dispatch enabled
    }

    #[test]
    fn state_3d_vertex_buffers_single() {
        let cmd = State3DVertexBuffers::new()
            .add_buffer(0, 0x1000, 1024, 32);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 5); // Header + 4 DWords per buffer
        assert_eq!(dwords[1] & 0x7FF, 32); // Stride
    }

    #[test]
    fn state_3d_vertex_elements_single() {
        let cmd = State3DVertexElements::new()
            .add_element(0, 0, 0x123);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 3); // Header + 2 DWords per element
        assert_eq!(dwords[2], 0x123); // Format
    }

    #[test]
    fn state_3d_blend_state_pointer() {
        let cmd = State3DBlendState::new(0x1000);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 2);
        assert_eq!(dwords[1], 0x1000); // Pointer
    }

    #[test]
    #[should_panic(expected = "64-byte aligned")]
    fn state_3d_blend_state_alignment() {
        State3DBlendState::new(0x1001); // Not 64-byte aligned
    }

    #[test]
    fn blend_state_opaque() {
        let state = BlendState::opaque();
        let dwords = state.serialize();
        
        assert_eq!(dwords.len(), 2); // 1 RT × 2 DWords
        assert_eq!(dwords[0] & (1 << 31), 0); // Blend disabled
        assert_eq!(dwords[1] >> 27, 0xF); // All color channels writable
    }

    #[test]
    fn blend_state_alpha() {
        let state = BlendState::alpha_blend();
        let dwords = state.serialize();
        
        assert_eq!(dwords.len(), 2);
        assert_ne!(dwords[0] & (1 << 31), 0); // Blend enabled
        
        // Extract blend factors from DWord 0
        let src_color = (dwords[0] >> 26) & 0x1F;
        let dst_color = (dwords[0] >> 21) & 0x1F;
        
        assert_eq!(src_color, BlendFactor::SrcAlpha as u32);
        assert_eq!(dst_color, BlendFactor::OneMinusSrcAlpha as u32);
    }

    #[test]
    fn render_target_blend_serialize() {
        let rt = RenderTargetBlend::alpha_blend();
        let [dw0, dw1] = rt.serialize();
        
        assert_ne!(dw0 & (1 << 31), 0); // Blend enabled
        assert_eq!(dw1 >> 27, 0xF); // RGBA write mask
    }
}
