// Intel EU Instruction Encoding Tables - Phase 4.3
//
// This module implements the binary encoding of Intel EU instructions.
// Instructions are 128 bits (16 bytes) on Gen9+ hardware.
//
// Encoding format (simplified, Gen9+):
// Bits 0-6: Opcode
// Bits 7-11: Execution size
// Bits 12-15: Destination register number
// ... (full format is complex, see Intel PRM Vol 4)
//
// Reference: Intel PRMs Volume 4, Section "Instruction Format"

use super::instruction::{EUOpcode, Register, RegFile, SendDescriptor};
use super::IntelGen;

/// Encode opcode to binary format
///
/// Opcodes are 7 bits on Gen9+ (bits 0-6 of DWord 0)
/// Different generations may have different opcode values
pub fn encode_opcode(opcode: EUOpcode, gen: IntelGen) -> u8 {
    match (opcode, gen) {
        // ALU operations (common across Gen9/11/12)
        (EUOpcode::Add, _) => 0x40,
        (EUOpcode::Mul, _) => 0x41,
        (EUOpcode::Mad, _) => 0x30,  // Multiply-add
        (EUOpcode::Mov, _) => 0x01,
        (EUOpcode::Sel, _) => 0x02,
        
        // Rounding operations
        (EUOpcode::Rndd, _) => 0x45,  // Round down (floor)
        (EUOpcode::Rndu, _) => 0x46,  // Round up (ceil)
        (EUOpcode::Rnde, _) => 0x44,  // Round to nearest even
        (EUOpcode::Rndz, _) => 0x47,  // Round toward zero (trunc)
        
        // Vector operations
        (EUOpcode::Dp2, _) => 0x54,  // Dot product 2D
        (EUOpcode::Dp3, _) => 0x55,  // Dot product 3D
        (EUOpcode::Dp4, _) => 0x56,  // Dot product 4D
        (EUOpcode::Dph, _) => 0x57,  // Dot product homogeneous
        
        // Logic operations
        (EUOpcode::And, _) => 0x05,
        (EUOpcode::Or, _) => 0x06,
        (EUOpcode::Xor, _) => 0x07,
        (EUOpcode::Not, _) => 0x08,
        (EUOpcode::Shl, _) => 0x09,  // Shift left
        (EUOpcode::Shr, _) => 0x0A,  // Shift right
        (EUOpcode::Asr, _) => 0x0C,  // Arithmetic shift right
        
        // Comparison
        (EUOpcode::Cmp, _) => 0x10,
        
        // Flow control
        (EUOpcode::Jmpi, _) => 0x20,
        (EUOpcode::If, _) => 0x22,
        (EUOpcode::Else, _) => 0x24,
        (EUOpcode::Endif, _) => 0x25,
        (EUOpcode::While, _) => 0x27,
        (EUOpcode::Break, _) => 0x28,
        (EUOpcode::Cont, _) => 0x29,
        
        // SEND instructions
        (EUOpcode::Send, _) => 0x31,
        (EUOpcode::SendC, _) => 0x32,
        
        // Special
        (EUOpcode::Nop, _) => 0x00,
        (EUOpcode::Wait, _) => 0x01,
    }
}

/// Execution size encoding
///
/// Determines how many channels execute in parallel
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExecSize {
    /// 1 channel (scalar)
    Size1 = 0,
    /// 2 channels
    Size2 = 1,
    /// 4 channels
    Size4 = 2,
    /// 8 channels (common for SIMD8)
    Size8 = 3,
    /// 16 channels (common for SIMD16)
    Size16 = 4,
    /// 32 channels (Gen11+)
    Size32 = 5,
}

impl ExecSize {
    /// Alias for Size1 (scalar execution)
    pub const Scalar: ExecSize = ExecSize::Size1;
    /// Encode execution size to 3-bit value (bits 21-23 of DWord 0)
    pub fn encode(&self) -> u8 {
        *self as u8
    }
    
    /// Get number of channels
    pub fn channels(&self) -> u32 {
        match self {
            ExecSize::Size1 => 1,
            ExecSize::Size2 => 2,
            ExecSize::Size4 => 4,
            ExecSize::Size8 => 8,
            ExecSize::Size16 => 16,
            ExecSize::Size32 => 32,
        }
    }
}

/// Data type encoding for register operands
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DataType {
    /// Unsigned 32-bit integer
    UD = 0,
    /// Signed 32-bit integer
    D = 1,
    /// Unsigned 16-bit integer
    UW = 2,
    /// Signed 16-bit integer
    W = 3,
    /// Unsigned 8-bit integer
    UB = 4,
    /// Signed 8-bit integer
    B = 5,
    /// 32-bit float
    F = 7,
    /// 16-bit float (half precision)
    HF = 8,
}

impl DataType {
    /// Encode data type to 4-bit value
    pub fn encode(&self) -> u8 {
        *self as u8
    }
    
    /// Get size in bytes
    pub fn size_bytes(&self) -> u32 {
        match self {
            DataType::UB | DataType::B => 1,
            DataType::UW | DataType::W | DataType::HF => 2,
            DataType::UD | DataType::D | DataType::F => 4,
        }
    }
}

/// Encode register reference to binary format
///
/// Register encoding varies by register file:
/// - GRF: bits for register number + subregister
/// - ARF: architecture register encoding
/// - Immediate: value embedded in instruction
pub fn encode_register(reg: &Register, dtype: DataType) -> u32 {
    match reg.file {
        RegFile::GRF => {
            // GRF encoding: reg_num in bits 0-7, subreg in bits 8-12
            let mut encoded: u32 = reg.num as u32;
            // Subreg is byte offset, needs to be aligned to data type size
            let subreg_aligned = (reg.subreg / dtype.size_bytes() as u8) as u32;
            encoded |= (subreg_aligned << 8) & 0x1F00;
            // Bit 13 = 0 for direct register, 1 for indirect
            // For now, always direct
            encoded
        }
        RegFile::ARF => {
            // ARF encoding: different format for special registers
            // Simplified for now - full implementation needs specific ARF handling
            let mut encoded: u32 = 0x8000;  // Bit 15 = 1 for ARF
            encoded |= (reg.num as u32) << 8;
            encoded |= reg.subreg as u32;
            encoded
        }
        RegFile::Imm => {
            // Immediate values are encoded differently, handle in instruction encoding
            0
        }
    }
}

/// Conditional modifier encoding (affects flag register)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CondMod {
    /// No conditional
    None = 0,
    /// Zero
    Z = 1,
    /// Not zero
    NZ = 2,
    /// Greater than
    G = 3,
    /// Greater or equal
    GE = 4,
    /// Less than
    L = 5,
    /// Less or equal
    LE = 6,
    /// Equal
    E = 7,
    /// Not equal
    NE = 8,
}

impl CondMod {
    /// Encode conditional modifier to 4-bit value
    pub fn encode(&self) -> u8 {
        *self as u8
    }
}

/// Source modifier (negation, absolute value)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SrcMod {
    pub negate: bool,
    pub absolute: bool,
}

impl SrcMod {
    pub const NONE: SrcMod = SrcMod { negate: false, absolute: false };
    pub const NEGATE: SrcMod = SrcMod { negate: true, absolute: false };
    pub const ABSOLUTE: SrcMod = SrcMod { negate: false, absolute: true };
    
    /// Encode source modifier to 2-bit value
    pub fn encode(&self) -> u8 {
        let mut encoded = 0u8;
        if self.absolute {
            encoded |= 0x2;
        }
        if self.negate {
            encoded |= 0x1;
        }
        encoded
    }
}

/// Encode SEND message descriptor
///
/// Message descriptor format (DWord 3 for SEND instructions):
/// Bits 0-3: Response length (in GRF registers, 0-15)
/// Bits 4-8: Message length (in GRF registers, 0-31)
/// Bits 9-13: (reserved/function-specific)
/// Bits 14-17: SFID (Shared Function ID, 0-15)
/// Bits 18-24: Function control (function-specific bits)
/// Bits 25-31: Extended function control
///
/// Reference: Intel PRM Vol 2a, Command Reference, SEND instruction
pub fn encode_send_descriptor(desc: &SendDescriptor) -> u32 {
    let mut encoded: u32 = 0;
    
    // Response length (bits 0-3): how many GRF registers to receive back
    encoded |= (desc.response_length as u32 & 0xF) << 0;
    
    // Message length (bits 4-8): how many GRF registers to send
    encoded |= (desc.message_length as u32 & 0x1F) << 4;
    
    // SFID (bits 14-17): which shared function to target
    encoded |= ((desc.sfid as u32) & 0xF) << 14;
    
    // Function control (bits 18-24): function-specific control bits
    // The exact format depends on the SFID (sampler, URB, etc.)
    encoded |= (desc.function_control & 0x7F) << 18;
    
    encoded
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_opcode_encoding() {
        assert_eq!(encode_opcode(EUOpcode::Add, IntelGen::Gen9), 0x40);
        assert_eq!(encode_opcode(EUOpcode::Mul, IntelGen::Gen9), 0x41);
        assert_eq!(encode_opcode(EUOpcode::Mov, IntelGen::Gen9), 0x01);
        assert_eq!(encode_opcode(EUOpcode::Nop, IntelGen::Gen9), 0x00);
    }

    #[test]
    fn test_exec_size() {
        assert_eq!(ExecSize::Size8.encode(), 3);
        assert_eq!(ExecSize::Size8.channels(), 8);
        assert_eq!(ExecSize::Size16.encode(), 4);
        assert_eq!(ExecSize::Size16.channels(), 16);
    }

    #[test]
    fn test_data_type() {
        assert_eq!(DataType::F.encode(), 7);
        assert_eq!(DataType::F.size_bytes(), 4);
        assert_eq!(DataType::D.encode(), 1);
        assert_eq!(DataType::HF.size_bytes(), 2);
    }

    #[test]
    fn test_register_encoding_grf() {
        let reg = Register {
            file: RegFile::GRF,
            num: 5,
            subreg: 0,
        };
        let encoded = encode_register(&reg, DataType::F);
        assert_eq!(encoded & 0xFF, 5);  // Register number in low byte
    }

    #[test]
    fn test_cond_mod() {
        assert_eq!(CondMod::None.encode(), 0);
        assert_eq!(CondMod::Z.encode(), 1);
        assert_eq!(CondMod::GE.encode(), 4);
    }

    #[test]
    fn test_src_mod() {
        assert_eq!(SrcMod::NONE.encode(), 0);
        
        let neg = SrcMod { negate: true, absolute: false };
        assert_eq!(neg.encode(), 1);
        
        let abs = SrcMod { negate: false, absolute: true };
        assert_eq!(abs.encode(), 2);
        
        let both = SrcMod { negate: true, absolute: true };
        assert_eq!(both.encode(), 3);
    }
    
    #[test]
    fn test_send_descriptor_encoding() {
        use super::super::instruction::SharedFunctionID;
        
        let desc = SendDescriptor {
            sfid: SharedFunctionID::Sampler,
            response_length: 4,   // 4 GRF registers back
            message_length: 2,    // 2 GRF registers sent
            function_control: 0,  // No special control bits
        };
        
        let encoded = encode_send_descriptor(&desc);
        
        // Response length in bits 0-3
        assert_eq!(encoded & 0xF, 4);
        
        // Message length in bits 4-8
        assert_eq!((encoded >> 4) & 0x1F, 2);
        
        // SFID in bits 14-17 (Sampler = 0x2)
        assert_eq!((encoded >> 14) & 0xF, 0x2);
    }
}
