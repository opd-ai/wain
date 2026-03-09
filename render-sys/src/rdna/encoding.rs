// AMD RDNA Instruction Binary Encoding
//
// This module encodes RDNA instructions into binary machine code.
// RDNA uses a variable-length encoding:
// - Most instructions are 32-bit (4 bytes)
// - Extended instructions (VOP3, MIMG, EXP) are 64-bit (8 bytes)
//
// Reference: AMD RDNA ISA Architecture Manual, Encoding Formats

use super::instruction::*;

/// RDNA instruction opcode values
///
/// These constants define the opcode field for various RDNA instruction formats.
/// VOP1, VOP2, SOP1, and SOP2 instructions have variable-length opcode fields.
///
/// Reference: AMD RDNA ISA Architecture Manual, Instruction Set Reference
pub mod rdna_opcodes {
    // VOP1 opcodes (8-bit)
    pub const VOP1_MOV_B32: u8 = 0x01;
    pub const VOP1_CVT_F32_I32: u8 = 0x0B;
    pub const VOP1_ABS_F32: u8 = 0x0C;
    pub const VOP1_CVT_I32_F32: u8 = 0x0D;
    pub const VOP1_NEG_F32: u8 = 0x0E;
    pub const VOP1_RCP_F32: u8 = 0x2A;
    pub const VOP1_RSQ_F32: u8 = 0x2E;
    pub const VOP1_SQRT_F32: u8 = 0x33;
    
    // VOP2 opcodes (6-bit)
    pub const VOP2_ADD_F32: u8 = 0x03;
    pub const VOP2_SUB_F32: u8 = 0x04;
    pub const VOP2_MUL_F32: u8 = 0x08;
    pub const VOP2_MIN_F32: u8 = 0x0A;
    pub const VOP2_MAX_F32: u8 = 0x0B;
    pub const VOP2_AND_B32: u8 = 0x15;
    pub const VOP2_OR_B32: u8 = 0x16;
    pub const VOP2_XOR_B32: u8 = 0x17;
    pub const VOP2_ADD_U32: u8 = 0x19;
    pub const VOP2_SUB_U32: u8 = 0x1A;
    pub const VOP2_MUL_LO_U32: u8 = 0x1C;
    
    // VOP3 opcodes (10-bit)
    pub const VOP3_CNDMASK_B32: u16 = 0x101;
    pub const VOP3_FMA_F32: u16 = 0x1C3;
    
    // SOP1 opcodes (8-bit)
    pub const SOP1_MOV_B32: u8 = 0x03;
    pub const SOP1_NOT_B32: u8 = 0x07;
    
    // SOP2 opcodes (7-bit)
    pub const SOP2_ADD_U32: u8 = 0x00;
    pub const SOP2_SUB_U32: u8 = 0x01;
    pub const SOP2_AND_B32: u8 = 0x0E;
    pub const SOP2_OR_B32: u8 = 0x0F;
}

/// Bitfield constants and encoding prefixes for RDNA instruction formats
pub mod bitfields {
    // Instruction encoding prefixes (high bits)
    pub const VOP1_PREFIX: u32 = 0x7E;
    pub const VOP3_PREFIX: u32 = 0xD1000000;
    pub const SOP1_PREFIX: u32 = 0xBE800000;
    pub const SOP2_PREFIX: u32 = 0x80000000;
    pub const MIMG_PREFIX: u32 = 0xF0000000;
    pub const EXP_PREFIX: u32 = 0xC4000000;
    
    // VOP3 opcode mask
    pub const VOP3_OPCODE_MASK: u32 = 0x3FF;
    
    // ImageDim encoding values
    pub const DIM_1D: u8 = 0x08;
    pub const DIM_2D: u8 = 0x09;
    pub const DIM_3D: u8 = 0x0A;
    pub const DIM_CUBE: u8 = 0x0C;
    
    // Export target encoding
    pub const EXPORT_MRT_MASK: u8 = 0x7;
    pub const EXPORT_POSITION: u8 = 0x0C;
    pub const EXPORT_PARAM_BASE: u8 = 0x20;
    pub const EXPORT_PARAM_MASK: u8 = 0x1F;
    
    // MIMG DMASK (RGBA enable mask)
    pub const RGBA_ENABLE_MASK: u32 = 0xF;
}

/// Encode a RDNA instruction to binary machine code
pub fn encode_instruction(inst: &RDNAInstruction) -> Vec<u8> {
    match inst {
        RDNAInstruction::VOP1(vop1) => encode_vop1(vop1),
        RDNAInstruction::VOP2(vop2) => encode_vop2(vop2),
        RDNAInstruction::VOP3(vop3) => encode_vop3(vop3),
        RDNAInstruction::SOP1(sop1) => encode_sop1(sop1),
        RDNAInstruction::SOP2(sop2) => encode_sop2(sop2),
        RDNAInstruction::ImageSample(img) => encode_image_sample(img),
        RDNAInstruction::Export(exp) => encode_export(exp),
    }
}

/// Encode VOP1 instruction (32-bit format)
/// Format: [opcode:8][dst:8][src:9][encoding:7]
fn encode_vop1(inst: &VOP1) -> Vec<u8> {
    use rdna_opcodes::*;
    
    let (opcode, dst, src) = match inst {
        VOP1::MovB32 { dst, src } => (VOP1_MOV_B32, dst.index(), src.index()),
        VOP1::CvtF32I32 { dst, src } => (VOP1_CVT_F32_I32, dst.index(), src.index()),
        VOP1::CvtI32F32 { dst, src } => (VOP1_CVT_I32_F32, dst.index(), src.index()),
        VOP1::RcpF32 { dst, src } => (VOP1_RCP_F32, dst.index(), src.index()),
        VOP1::RsqF32 { dst, src } => (VOP1_RSQ_F32, dst.index(), src.index()),
        VOP1::SqrtF32 { dst, src } => (VOP1_SQRT_F32, dst.index(), src.index()),
        VOP1::AbsF32 { dst, src } => (VOP1_ABS_F32, dst.index(), src.index()),
        VOP1::NegF32 { dst, src } => (VOP1_NEG_F32, dst.index(), src.index()),
    };

    let encoding: u32 = bitfields::VOP1_PREFIX | ((opcode as u32) << 9) | ((dst as u32) << 17) | ((src as u32) << 0);
    encoding.to_le_bytes().to_vec()
}

/// Encode VOP2 instruction (32-bit format)
/// Format: [opcode:6][dst:8][src0:9][src1:9]
fn encode_vop2(inst: &VOP2) -> Vec<u8> {
    use rdna_opcodes::*;
    
    let (opcode, dst, src0, src1) = match inst {
        VOP2::AddF32 { dst, src0, src1 } => (VOP2_ADD_F32, dst.index(), src0.index(), src1.index()),
        VOP2::SubF32 { dst, src0, src1 } => (VOP2_SUB_F32, dst.index(), src0.index(), src1.index()),
        VOP2::MulF32 { dst, src0, src1 } => (VOP2_MUL_F32, dst.index(), src0.index(), src1.index()),
        VOP2::MinF32 { dst, src0, src1 } => (VOP2_MIN_F32, dst.index(), src0.index(), src1.index()),
        VOP2::MaxF32 { dst, src0, src1 } => (VOP2_MAX_F32, dst.index(), src0.index(), src1.index()),
        VOP2::AddU32 { dst, src0, src1 } => (VOP2_ADD_U32, dst.index(), src0.index(), src1.index()),
        VOP2::SubU32 { dst, src0, src1 } => (VOP2_SUB_U32, dst.index(), src0.index(), src1.index()),
        VOP2::MulLoU32 { dst, src0, src1 } => (VOP2_MUL_LO_U32, dst.index(), src0.index(), src1.index()),
        VOP2::AndB32 { dst, src0, src1 } => (VOP2_AND_B32, dst.index(), src0.index(), src1.index()),
        VOP2::OrB32 { dst, src0, src1 } => (VOP2_OR_B32, dst.index(), src0.index(), src1.index()),
        VOP2::XorB32 { dst, src0, src1 } => (VOP2_XOR_B32, dst.index(), src0.index(), src1.index()),
    };

    let encoding: u32 = ((opcode as u32) << 25) | ((src1 as u32) << 9) | ((dst as u32) << 17) | (src0 as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode VOP3 instruction (64-bit format)
/// Format: [opcode:10][dst:8][abs:3][src0:9][src1:9][src2:9][omod:2][neg:3]
fn encode_vop3(inst: &VOP3) -> Vec<u8> {
    use rdna_opcodes::*;
    
    let (opcode, dst, src0, src1, src2) = match inst {
        VOP3::FmaF32 { dst, src0, src1, src2 } => {
            (VOP3_FMA_F32, dst.index(), src0.index(), src1.index(), src2.index())
        }
        VOP3::CndmaskB32 { dst, src0, src1, src2 } => {
            (VOP3_CNDMASK_B32, dst.index(), src0.index(), src1.index(), src2.index())
        }
    };

    let word0: u32 = bitfields::VOP3_PREFIX | ((opcode as u32 & bitfields::VOP3_OPCODE_MASK) << 16) | (dst as u32);
    let word1: u32 = ((src2 as u32) << 18) | ((src1 as u32) << 9) | (src0 as u32);
    
    let mut bytes = Vec::with_capacity(8);
    bytes.extend_from_slice(&word0.to_le_bytes());
    bytes.extend_from_slice(&word1.to_le_bytes());
    bytes
}

/// Encode SOP1 instruction (32-bit format)
fn encode_sop1(inst: &SOP1) -> Vec<u8> {
    use rdna_opcodes::*;
    
    let (opcode, dst, src) = match inst {
        SOP1::MovB32 { dst, src } => (SOP1_MOV_B32, dst.index(), src.index()),
        SOP1::NotB32 { dst, src } => (SOP1_NOT_B32, dst.index(), src.index()),
    };

    let encoding: u32 = bitfields::SOP1_PREFIX | ((opcode as u32) << 8) | ((dst as u32) << 16) | (src as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode SOP2 instruction (32-bit format)
fn encode_sop2(inst: &SOP2) -> Vec<u8> {
    use rdna_opcodes::*;
    
    let (opcode, dst, src0, src1) = match inst {
        SOP2::AddU32 { dst, src0, src1 } => (SOP2_ADD_U32, dst.index(), src0.index(), src1.index()),
        SOP2::SubU32 { dst, src0, src1 } => (SOP2_SUB_U32, dst.index(), src0.index(), src1.index()),
        SOP2::AndB32 { dst, src0, src1 } => (SOP2_AND_B32, dst.index(), src0.index(), src1.index()),
        SOP2::OrB32 { dst, src0, src1 } => (SOP2_OR_B32, dst.index(), src0.index(), src1.index()),
    };

    let encoding: u32 = bitfields::SOP2_PREFIX | ((opcode as u32) << 23) | ((dst as u32) << 16) | ((src1 as u32) << 8) | (src0 as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode MIMG (image sample) instruction (64-bit format)
fn encode_image_sample(inst: &ImageSample) -> Vec<u8> {
    use bitfields::*;
    
    let dmask = RGBA_ENABLE_MASK;
    let dim_bits = match inst.dim {
        ImageDim::Dim1D => DIM_1D,
        ImageDim::Dim2D => DIM_2D,
        ImageDim::Dim3D => DIM_3D,
        ImageDim::Cube => DIM_CUBE,
    };

    let word0: u32 = MIMG_PREFIX | (dmask << 8) | ((dim_bits as u32) << 0);
    let word1: u32 = ((inst.texture.index() as u32) << 16) 
                   | ((inst.sampler.index() as u32) << 8)
                   | ((inst.dst.index() as u32) << 0);

    let mut bytes = Vec::with_capacity(8);
    bytes.extend_from_slice(&word0.to_le_bytes());
    bytes.extend_from_slice(&word1.to_le_bytes());
    bytes
}

/// Encode EXP (export) instruction (64-bit format)
fn encode_export(inst: &Export) -> Vec<u8> {
    use bitfields::*;
    
    let target_bits = match inst.target {
        ExportTarget::MRT(n) => n & EXPORT_MRT_MASK,
        ExportTarget::Position => EXPORT_POSITION,
        ExportTarget::Parameter(n) => EXPORT_PARAM_BASE + (n & EXPORT_PARAM_MASK),
    };

    let done_bit = if inst.done { 1u32 } else { 0 };
    let compressed_bit = if inst.compressed { 1u32 } else { 0 };

    let word0: u32 = EXP_PREFIX
                   | ((done_bit) << 11)
                   | ((compressed_bit) << 10)
                   | ((inst.enable_mask as u32) << 12)
                   | ((target_bits as u32) << 4);
    
    let word1: u32 = ((inst.src[3].index() as u32) << 24)
                   | ((inst.src[2].index() as u32) << 16)
                   | ((inst.src[1].index() as u32) << 8)
                   | ((inst.src[0].index() as u32) << 0);

    let mut bytes = Vec::with_capacity(8);
    bytes.extend_from_slice(&word0.to_le_bytes());
    bytes.extend_from_slice(&word1.to_le_bytes());
    bytes
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::super::types::*;

    #[test]
    fn test_encode_vop1() {
        let inst = VOP1::MovB32 {
            dst: VGPR::new(0),
            src: VGPR::new(1),
        };
        let bytes = encode_vop1(&inst);
        assert_eq!(bytes.len(), 4);
    }

    #[test]
    fn test_encode_vop2() {
        let inst = VOP2::AddF32 {
            dst: VGPR::new(0),
            src0: VGPR::new(1),
            src1: VGPR::new(2),
        };
        let bytes = encode_vop2(&inst);
        assert_eq!(bytes.len(), 4);
    }

    #[test]
    fn test_encode_vop3() {
        let inst = VOP3::FmaF32 {
            dst: VGPR::new(0),
            src0: VGPR::new(1),
            src1: VGPR::new(2),
            src2: VGPR::new(3),
        };
        let bytes = encode_vop3(&inst);
        assert_eq!(bytes.len(), 8);
    }

    #[test]
    fn test_encode_image_sample() {
        let inst = ImageSample {
            dst: VGPR::new(0),
            addr: VGPR::new(4),
            sampler: SGPR::new(0),
            texture: SGPR::new(4),
            dim: ImageDim::Dim2D,
        };
        let bytes = encode_image_sample(&inst);
        assert_eq!(bytes.len(), 8);
    }

    #[test]
    fn test_encode_export() {
        let inst = Export {
            target: ExportTarget::MRT(0),
            enable_mask: 0xF,
            src: [VGPR::new(0), VGPR::new(1), VGPR::new(2), VGPR::new(3)],
            compressed: false,
            done: true,
        };
        let bytes = encode_export(&inst);
        assert_eq!(bytes.len(), 8);
    }

    #[test]
    fn test_full_instruction_encoding() {
        let inst = RDNAInstruction::VOP2(VOP2::MulF32 {
            dst: VGPR::new(5),
            src0: VGPR::new(1),
            src1: VGPR::new(2),
        });
        let bytes = encode_instruction(&inst);
        assert_eq!(bytes.len(), 4);
    }
}
