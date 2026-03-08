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

use super::instruction::{EUOpcode, Register, RegFile};
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
        
        // Logic operations
        (EUOpcode::And, _) => 0x05,
        (EUOpcode::Or, _) => 0x06,
        (EUOpcode::Xor, _) => 0x07,
        (EUOpcode::Not, _) => 0x08,
        
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
}
