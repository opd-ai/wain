/// 3D Pipeline State Commands
///
/// These commands configure the 3D rendering pipeline, including:
/// - Pipeline mode selection (3D vs GPGPU)
/// - Base addresses for state buffers
/// - Viewport configuration
/// - Clipping and rasterization state

use super::{GpuCommand, CommandType};

/// PIPELINE_SELECT - Select between 3D and GPGPU pipeline modes
///
/// This command must be issued before any 3D rendering commands to
/// configure the GPU for 3D graphics (vs compute/GPGPU mode).
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header and pipeline selection
#[derive(Debug, Clone, Copy)]
pub struct PipelineSelect {
    /// Pipeline mode: false = 3D, true = GPGPU
    pub gpgpu_mode: bool,
}

impl PipelineSelect {
    /// Create a PIPELINE_SELECT command for 3D mode.
    pub fn new_3d() -> Self {
        Self { gpgpu_mode: false }
    }
    
    /// Create a PIPELINE_SELECT command for GPGPU mode.
    pub fn new_gpgpu() -> Self {
        Self { gpgpu_mode: true }
    }
}

impl GpuCommand for PipelineSelect {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x6904; // PIPELINE_SELECT opcode
        let mut dw0 = (1 << 29) | (opcode << 16);
        
        if self.gpgpu_mode {
            dw0 |= 1 << 0; // Pipeline selection bit
        }
        
        vec![dw0]
    }
}

/// STATE_BASE_ADDRESS - Set base addresses for indirect state
///
/// This command configures the base addresses for various state buffers
/// used by the 3D pipeline (surface state, dynamic state, etc.).
/// Critical for proper pipeline operation.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1-2: General state base address
/// - DWord 3-4: Surface state base address
/// - DWord 5-6: Dynamic state base address
/// - DWord 7-8: Indirect object base address
/// - DWord 9-10: Instruction base address
/// - DWords 11+: Bounds and modifiers
#[derive(Debug, Clone)]
pub struct StateBaseAddress {
    /// General state base address (64-bit)
    pub general_state_base: u64,
    /// Surface state base address (64-bit)
    pub surface_state_base: u64,
    /// Dynamic state base address (64-bit)
    pub dynamic_state_base: u64,
    /// Indirect object base address (64-bit)
    pub indirect_object_base: u64,
    /// Instruction base address (64-bit)
    pub instruction_base: u64,
}

impl StateBaseAddress {
    /// Create a new STATE_BASE_ADDRESS command.
    pub fn new() -> Self {
        Self {
            general_state_base: 0,
            surface_state_base: 0,
            dynamic_state_base: 0,
            indirect_object_base: 0,
            instruction_base: 0,
        }
    }
}

impl Default for StateBaseAddress {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for StateBaseAddress {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7801; // STATE_BASE_ADDRESS 3D opcode
        let length = 15; // 16 DWords total (index 0-15)
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        vec![
            dw0,
            // General state base address (enabled)
            (self.general_state_base & 0xFFFFFFFF) as u32 | 1,
            (self.general_state_base >> 32) as u32,
            // Surface state base address (enabled)
            (self.surface_state_base & 0xFFFFFFFF) as u32 | 1,
            (self.surface_state_base >> 32) as u32,
            // Dynamic state base address (enabled)
            (self.dynamic_state_base & 0xFFFFFFFF) as u32 | 1,
            (self.dynamic_state_base >> 32) as u32,
            // Indirect object base address (enabled)
            (self.indirect_object_base & 0xFFFFFFFF) as u32 | 1,
            (self.indirect_object_base >> 32) as u32,
            // Instruction base address (enabled)
            (self.instruction_base & 0xFFFFFFFF) as u32 | 1,
            (self.instruction_base >> 32) as u32,
            // Upper bounds (set to maximum)
            0xFFFFF000, // General state buffer size
            0xFFFFF000, // Dynamic state buffer size
            0xFFFFF000, // Indirect object buffer size
            0xFFFFF000, // Instruction buffer size
            0, // Padding
        ]
    }
}

/// 3DSTATE_VIEWPORT_STATE_POINTERS_CC - Viewport state pointer
///
/// Points to the CC (Color Calc) viewport state data in dynamic state.
#[derive(Debug, Clone)]
pub struct State3DViewportStatePointersCC {
    /// Offset into dynamic state base where viewport data resides
    pub viewport_offset: u32,
}

impl GpuCommand for State3DViewportStatePointersCC {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7823; // 3DSTATE_VIEWPORT_STATE_POINTERS_CC
        let length = 0; // 2 DWords total
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        vec![
            dw0,
            self.viewport_offset,
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pipeline_select_3d() {
        let cmd = PipelineSelect::new_3d();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 1);
        assert_eq!(dwords[0] & 1, 0); // 3D mode (bit 0 = 0)
    }

    #[test]
    fn pipeline_select_gpgpu() {
        let cmd = PipelineSelect::new_gpgpu();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 1);
        assert_eq!(dwords[0] & 1, 1); // GPGPU mode (bit 0 = 1)
    }

    #[test]
    fn state_base_address_serialization() {
        let mut cmd = StateBaseAddress::new();
        cmd.surface_state_base = 0x1000_0000;
        cmd.dynamic_state_base = 0x2000_0000;
        
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 16);
        // Check surface state base address
        assert_eq!(dwords[3], 0x1000_0001); // Low 32 bits + enable bit
        assert_eq!(dwords[4], 0); // High 32 bits
        // Check dynamic state base address
        assert_eq!(dwords[5], 0x2000_0001); // Low 32 bits + enable bit
        assert_eq!(dwords[6], 0); // High 32 bits
    }

    #[test]
    fn state_base_address_header() {
        let cmd = StateBaseAddress::new();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords[0] >> 29, 3); // Command type = 3D
    }

    #[test]
    fn viewport_state_pointers_serialization() {
        let cmd = State3DViewportStatePointersCC {
            viewport_offset: 0x100,
        };
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 2);
        assert_eq!(dwords[1], 0x100);
    }
}
