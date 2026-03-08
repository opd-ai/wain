// Intel EU Instruction Encoding - Phase 4.3
//
// This module defines the binary instruction format for Intel EU ISA.
// Each instruction is encoded as 128 bits (16 bytes) on Gen9+ hardware.
//
// Reference: Intel PRMs Volume 4 (Execution Unit ISA), Section on Instruction Format

use super::IntelGen;
use super::encoding::{encode_opcode, encode_register, encode_send_descriptor, ExecSize, DataType, CondMod, SrcMod};

/// EU instruction opcode categories
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum EUOpcode {
    // ALU operations
    Add,
    Mul,
    Mad,  // Multiply-add
    Mov,
    Sel,  // Select (conditional move)
    
    // Rounding operations
    Rndd,  // Round down (floor)
    Rndu,  // Round up (ceil)
    Rnde,  // Round to nearest even
    Rndz,  // Round toward zero (trunc)
    
    // Logic operations
    And,
    Or,
    Xor,
    Not,
    Shl,  // Shift left
    Shr,  // Shift right (arithmetic)
    Asr,  // Arithmetic shift right
    
    // Comparison
    Cmp,
    
    // Flow control
    Jmpi,  // Jump indirect
    If,
    Else,
    Endif,
    While,
    Break,
    Cont,
    
    // SEND instructions (memory/texture/URB access)
    Send,
    SendC,  // Conditional send
    
    // Special
    Nop,
    Wait,
}

/// Register file type
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum RegFile {
    /// General Register File (GRF) - main working registers
    GRF,
    /// Architecture Register File (ARF) - special purpose registers
    ARF,
    /// Immediate value
    Imm,
}

/// Register reference
#[derive(Debug, Clone, Copy)]
#[allow(dead_code)]
pub struct Register {
    pub file: RegFile,
    pub num: u8,
    pub subreg: u8,  // Sub-register offset
}

/// Shared Function ID for SEND instructions
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum SharedFunctionID {
    /// Null function (no operation)
    Null = 0x0,
    /// Sampler (texture sampling)
    Sampler = 0x2,
    /// Gateway (barrier synchronization)
    Gateway = 0x3,
    /// Data Cache (load/store operations)
    DataCache = 0x4,
    /// Render Target Cache (pixel write)
    RenderTargetCache = 0x5,
    /// URB (Unified Return Buffer - vertex/geometry data)
    URB = 0x6,
    /// Thread Spawner
    ThreadSpawner = 0x7,
    /// Math function unit (sqrt, sin, cos, etc.)
    Math = 0xb,
}

/// Message descriptor for SEND instructions
#[derive(Debug, Clone, Copy)]
#[allow(dead_code)]
pub struct SendDescriptor {
    /// Shared function ID (SFID) - which fixed function to send to
    pub sfid: SharedFunctionID,
    /// Response length in GRF registers (0-15)
    pub response_length: u8,
    /// Message length in GRF registers (0-15)
    pub message_length: u8,
    /// Function-specific control bits
    pub function_control: u32,
}

/// EU instruction (128-bit / 16-byte format for Gen9+)
#[derive(Debug)]
pub struct EUInstruction {
    opcode: EUOpcode,
    dst: Option<Register>,
    src0: Option<Register>,
    src1: Option<Register>,
    src2: Option<Register>,
    
    // Instruction modifiers
    exec_size: ExecSize,
    dst_type: DataType,
    src0_type: DataType,
    src1_type: DataType,
    cond_mod: CondMod,
    src0_mod: SrcMod,
    src1_mod: SrcMod,
    
    // Predication
    predicate_enable: bool,
    predicate_inverse: bool,
    
    // SEND-specific fields (only used when opcode is Send or SendC)
    send_descriptor: Option<SendDescriptor>,
}

impl EUInstruction {
    /// Create a new instruction with default modifiers
    pub fn new(opcode: EUOpcode) -> Self {
        EUInstruction {
            opcode,
            dst: None,
            src0: None,
            src1: None,
            src2: None,
            exec_size: ExecSize::Size8,
            dst_type: DataType::F,
            src0_type: DataType::F,
            src1_type: DataType::F,
            cond_mod: CondMod::None,
            src0_mod: SrcMod::NONE,
            src1_mod: SrcMod::NONE,
            predicate_enable: false,
            predicate_inverse: false,
            send_descriptor: None,
        }
    }
    
    /// Set destination register
    pub fn with_dst(mut self, reg: Register, dtype: DataType) -> Self {
        self.dst = Some(reg);
        self.dst_type = dtype;
        self
    }
    
    /// Set source 0 register
    pub fn with_src0(mut self, reg: Register, dtype: DataType) -> Self {
        self.src0 = Some(reg);
        self.src0_type = dtype;
        self
    }
    
    /// Set source 1 register
    pub fn with_src1(mut self, reg: Register, dtype: DataType) -> Self {
        self.src1 = Some(reg);
        self.src1_type = dtype;
        self
    }
    
    /// Set execution size
    pub fn with_exec_size(mut self, exec_size: ExecSize) -> Self {
        self.exec_size = exec_size;
        self
    }
    
    /// Set conditional modifier
    pub fn with_cond_mod(mut self, cond_mod: CondMod) -> Self {
        self.cond_mod = cond_mod;
        self
    }
    
    /// Set source 0 modifier
    pub fn with_src0_mod(mut self, src_mod: SrcMod) -> Self {
        self.src0_mod = src_mod;
        self
    }
    
    /// Set SEND descriptor (for SEND instructions only)
    pub fn with_send_descriptor(mut self, descriptor: SendDescriptor) -> Self {
        self.send_descriptor = Some(descriptor);
        self
    }
    
    // Mutable setter methods for imperative instruction building
    
    /// Set destination register (mutable)
    pub fn set_dst(&mut self, reg: Register) {
        self.dst = Some(reg);
    }
    
    /// Set source 0 register (mutable)
    pub fn set_src0(&mut self, reg: Register) {
        self.src0 = Some(reg);
    }
    
    /// Set source 1 register (mutable)
    pub fn set_src1(&mut self, reg: Register) {
        self.src1 = Some(reg);
    }
    
    /// Set execution size (mutable)
    pub fn set_exec_size(&mut self, exec_size: ExecSize) {
        self.exec_size = exec_size;
    }
    
    /// Set destination data type (mutable)
    pub fn set_dst_type(&mut self, dtype: DataType) {
        self.dst_type = dtype;
    }
    
    /// Set source 0 data type (mutable)
    pub fn set_src0_type(&mut self, dtype: DataType) {
        self.src0_type = dtype;
    }
    
    /// Set source 1 data type (mutable)
    pub fn set_src1_type(&mut self, dtype: DataType) {
        self.src1_type = dtype;
    }
    
    /// Set source 0 negate modifier (mutable)
    pub fn set_src0_negate(&mut self, negate: bool) {
        self.src0_mod = if negate { SrcMod::NEGATE } else { SrcMod::NONE };
    }
    
    /// Set source 1 negate modifier (mutable)
    pub fn set_src1_negate(&mut self, negate: bool) {
        self.src1_mod = if negate { SrcMod::NEGATE } else { SrcMod::NONE };
    }
    
    /// Set source 0 absolute modifier (mutable)
    pub fn set_src0_absolute(&mut self, absolute: bool) {
        self.src0_mod = if absolute { SrcMod::ABSOLUTE } else { SrcMod::NONE };
    }
    
    /// Set source 1 absolute modifier (mutable)
    pub fn set_src1_absolute(&mut self, absolute: bool) {
        self.src1_mod = if absolute { SrcMod::ABSOLUTE } else { SrcMod::NONE };
    }
    
    /// Set conditional modifier (mutable)
    pub fn set_cond_mod(&mut self, cond_mod: CondMod) {
        self.cond_mod = cond_mod;
    }
    
    /// Encode instruction to binary format
    ///
    /// Returns 16 bytes (128 bits) on Gen9+
    /// 
    /// Binary format (Gen9+):
    /// DWord 0 (bits 0-31):
    ///   - Bits 0-6: Opcode
    ///   - Bits 7-11: Execution size
    ///   - Bits 12-15: Conditional modifier
    ///   - Bits 16-20: (various control fields)
    ///   - Bits 21-23: Execution size (extended)
    ///   - Bits 24-27: Destination data type
    ///   - Bits 28-31: Source 0 data type
    /// DWord 1: Destination register encoding
    /// DWord 2: Source 0 register encoding
    /// DWord 3: Source 1 register encoding OR message descriptor (for SEND)
    pub fn encode(&self, gen: IntelGen) -> [u8; 16] {
        let mut binary = [0u8; 16];
        
        // DWord 0: Opcode and control fields
        let opcode_bits = encode_opcode(self.opcode, gen) as u32;
        let exec_size_bits = (self.exec_size.encode() as u32) << 21;
        let cond_mod_bits = (self.cond_mod.encode() as u32) << 12;
        let dst_type_bits = (self.dst_type.encode() as u32) << 24;
        let src0_type_bits = (self.src0_type.encode() as u32) << 28;
        
        let dword0 = opcode_bits | exec_size_bits | cond_mod_bits | dst_type_bits | src0_type_bits;
        binary[0..4].copy_from_slice(&dword0.to_le_bytes());
        
        // DWord 1: Destination register
        if let Some(ref dst) = self.dst {
            let dst_bits = encode_register(dst, self.dst_type);
            binary[4..8].copy_from_slice(&dst_bits.to_le_bytes());
        }
        
        // DWord 2: Source 0 register
        if let Some(ref src0) = self.src0 {
            let mut src0_bits = encode_register(src0, self.src0_type);
            // Add source modifiers (bits 16-17)
            src0_bits |= (self.src0_mod.encode() as u32) << 16;
            binary[8..12].copy_from_slice(&src0_bits.to_le_bytes());
        }
        
        // DWord 3: Source 1 register OR message descriptor (for SEND)
        if self.opcode == EUOpcode::Send || self.opcode == EUOpcode::SendC {
            // For SEND instructions, DWord 3 contains the message descriptor
            if let Some(ref desc) = self.send_descriptor {
                let desc_bits = encode_send_descriptor(desc);
                binary[12..16].copy_from_slice(&desc_bits.to_le_bytes());
            }
        } else {
            // For normal instructions, DWord 3 contains Source 1 register
            if let Some(ref src1) = self.src1 {
                let mut src1_bits = encode_register(src1, self.src1_type);
                // Add source modifiers (bits 16-17)
                src1_bits |= (self.src1_mod.encode() as u32) << 16;
                binary[12..16].copy_from_slice(&src1_bits.to_le_bytes());
            }
        }
        
        binary
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::super::encoding::{ExecSize, DataType, CondMod, SrcMod};

    #[test]
    fn test_instruction_creation() {
        let inst = EUInstruction::new(EUOpcode::Add);
        assert_eq!(inst.opcode, EUOpcode::Add);
    }

    #[test]
    fn test_instruction_encoding_nop() {
        let inst = EUInstruction::new(EUOpcode::Nop);
        let binary = inst.encode(IntelGen::Gen9);
        assert_eq!(binary.len(), 16);
        
        // Check opcode is encoded in low bits of DWord 0
        assert_eq!(binary[0] & 0x7F, 0x00);  // NOP opcode
    }
    
    #[test]
    fn test_instruction_encoding_add() {
        let dst = Register {
            file: RegFile::GRF,
            num: 10,
            subreg: 0,
        };
        let src0 = Register {
            file: RegFile::GRF,
            num: 20,
            subreg: 0,
        };
        let src1 = Register {
            file: RegFile::GRF,
            num: 30,
            subreg: 0,
        };
        
        let inst = EUInstruction::new(EUOpcode::Add)
            .with_dst(dst, DataType::F)
            .with_src0(src0, DataType::F)
            .with_src1(src1, DataType::F)
            .with_exec_size(ExecSize::Size8);
        
        let binary = inst.encode(IntelGen::Gen9);
        assert_eq!(binary.len(), 16);
        
        // Check opcode (bits 0-6 of DWord 0)
        assert_eq!(binary[0] & 0x7F, 0x40);  // ADD opcode
        
        // Check destination register in DWord 1
        assert_eq!(binary[4], 10);  // dst reg num
    }
    
    #[test]
    fn test_instruction_with_modifiers() {
        let dst = Register {
            file: RegFile::GRF,
            num: 5,
            subreg: 0,
        };
        let src0 = Register {
            file: RegFile::GRF,
            num: 6,
            subreg: 0,
        };
        
        let inst = EUInstruction::new(EUOpcode::Mov)
            .with_dst(dst, DataType::F)
            .with_src0(src0, DataType::F)
            .with_exec_size(ExecSize::Size16)
            .with_cond_mod(CondMod::Z)
            .with_src0_mod(SrcMod { negate: true, absolute: false });
        
        let binary = inst.encode(IntelGen::Gen9);
        
        // Verify encoding completed (basic smoke test)
        assert_eq!(binary.len(), 16);
        assert_eq!(binary[0] & 0x7F, 0x01);  // MOV opcode
    }
    
    #[test]
    fn test_builder_pattern() {
        let reg = Register {
            file: RegFile::GRF,
            num: 1,
            subreg: 0,
        };
        
        // Test builder pattern chaining
        let inst = EUInstruction::new(EUOpcode::Mul)
            .with_dst(reg, DataType::F)
            .with_src0(reg, DataType::F)
            .with_src1(reg, DataType::F)
            .with_exec_size(ExecSize::Size4);
        
        assert_eq!(inst.opcode, EUOpcode::Mul);
        assert!(inst.dst.is_some());
        assert!(inst.src0.is_some());
        assert!(inst.src1.is_some());
    }
}
