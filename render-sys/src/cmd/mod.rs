/// Intel GPU Command Encoding Module
///
/// This module provides Rust structs for Intel 3D pipeline commands,
/// targeting Gen9-Gen12 GPU generations. Each struct serializes to the
/// binary format specified in Intel PRM Volume 2 (Command Reference).
///
/// References:
/// - Intel PRMs Volume 2 (Command Reference)
/// - Mesa genxml files (src/intel/genxml/*.xml)

pub mod mi;
pub mod pipeline;
pub mod state;
pub mod primitive;

pub use mi::*;
pub use pipeline::*;
pub use state::*;
pub use primitive::*;

use crate::detect::GpuGeneration;

/// Trait for GPU commands that can be serialized to binary.
pub trait GpuCommand {
    /// Serialize the command to a vector of 32-bit dwords.
    fn serialize(&self) -> Vec<u32>;
    
    /// Get the command length in dwords (including header).
    fn len_dwords(&self) -> usize {
        self.serialize().len()
    }
}

/// Command opcode types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CommandType {
    /// MI (Machine Interface) commands - GPU control
    MI,
    /// 3D pipeline state commands
    State3D,
    /// 3D primitive rendering commands
    Primitive3D,
}

impl CommandType {
    /// Get the command type field value
    pub fn opcode_type(&self) -> u32 {
        match self {
            CommandType::MI => 0,
            CommandType::State3D => 3,
            CommandType::Primitive3D => 3,
        }
    }
}

/// Generation-specific command variant selector
#[derive(Debug, Clone, Copy)]
pub struct CommandEncoder {
    generation: GpuGeneration,
}

impl CommandEncoder {
    /// Create a new command encoder for the given GPU generation.
    pub fn new(generation: GpuGeneration) -> Self {
        Self { generation }
    }
    
    /// Check if the generation supports a specific command.
    pub fn supports_command(&self, _command_name: &str) -> bool {
        // For Gen9-Gen12, most 3D commands are compatible
        matches!(self.generation, 
            GpuGeneration::Gen9 | 
            GpuGeneration::Gen11 | 
            GpuGeneration::Gen12
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn command_type_opcodes() {
        assert_eq!(CommandType::MI.opcode_type(), 0);
        assert_eq!(CommandType::State3D.opcode_type(), 3);
        assert_eq!(CommandType::Primitive3D.opcode_type(), 3);
    }

    #[test]
    fn encoder_supports_gen9() {
        let encoder = CommandEncoder::new(GpuGeneration::Gen9);
        assert!(encoder.supports_command("3DSTATE_VIEWPORT"));
    }

    #[test]
    fn encoder_supports_gen12() {
        let encoder = CommandEncoder::new(GpuGeneration::Gen12);
        assert!(encoder.supports_command("3DSTATE_PS"));
    }
}
