// AMD RDNA Instruction Binary Encoding
//
// This module encodes RDNA instructions into binary machine code.
// RDNA uses a variable-length encoding:
// - Most instructions are 32-bit (4 bytes)
// - Extended instructions (VOP3, MIMG, EXP) are 64-bit (8 bytes)
//
// Reference: AMD RDNA ISA Architecture Manual, Encoding Formats

use super::instruction::*;

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
    let (opcode, dst, src) = match inst {
        VOP1::MovB32 { dst, src } => (0x01, dst.index(), src.index()),
        VOP1::CvtF32I32 { dst, src } => (0x0B, dst.index(), src.index()),
        VOP1::CvtI32F32 { dst, src } => (0x0D, dst.index(), src.index()),
        VOP1::RcpF32 { dst, src } => (0x2A, dst.index(), src.index()),
        VOP1::RsqF32 { dst, src } => (0x2E, dst.index(), src.index()),
        VOP1::SqrtF32 { dst, src } => (0x33, dst.index(), src.index()),
        VOP1::AbsF32 { dst, src } => (0x0C, dst.index(), src.index()),
        VOP1::NegF32 { dst, src } => (0x0E, dst.index(), src.index()),
    };

    // VOP1 encoding: 0x7E prefix
    let encoding: u32 = 0x7E | ((opcode as u32) << 9) | ((dst as u32) << 17) | ((src as u32) << 0);
    encoding.to_le_bytes().to_vec()
}

/// Encode VOP2 instruction (32-bit format)
/// Format: [opcode:6][dst:8][src0:9][src1:9]
fn encode_vop2(inst: &VOP2) -> Vec<u8> {
    let (opcode, dst, src0, src1) = match inst {
        VOP2::AddF32 { dst, src0, src1 } => (0x03, dst.index(), src0.index(), src1.index()),
        VOP2::SubF32 { dst, src0, src1 } => (0x04, dst.index(), src0.index(), src1.index()),
        VOP2::MulF32 { dst, src0, src1 } => (0x08, dst.index(), src0.index(), src1.index()),
        VOP2::MinF32 { dst, src0, src1 } => (0x0A, dst.index(), src0.index(), src1.index()),
        VOP2::MaxF32 { dst, src0, src1 } => (0x0B, dst.index(), src0.index(), src1.index()),
        VOP2::AddU32 { dst, src0, src1 } => (0x19, dst.index(), src0.index(), src1.index()),
        VOP2::SubU32 { dst, src0, src1 } => (0x1A, dst.index(), src0.index(), src1.index()),
        VOP2::MulLoU32 { dst, src0, src1 } => (0x1C, dst.index(), src0.index(), src1.index()),
        VOP2::AndB32 { dst, src0, src1 } => (0x15, dst.index(), src0.index(), src1.index()),
        VOP2::OrB32 { dst, src0, src1 } => (0x16, dst.index(), src0.index(), src1.index()),
        VOP2::XorB32 { dst, src0, src1 } => (0x17, dst.index(), src0.index(), src1.index()),
    };

    // VOP2 encoding format
    let encoding: u32 = ((opcode as u32) << 25) | ((src1 as u32) << 9) | ((dst as u32) << 17) | (src0 as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode VOP3 instruction (64-bit format)
/// Format: [opcode:10][dst:8][abs:3][src0:9][src1:9][src2:9][omod:2][neg:3]
fn encode_vop3(inst: &VOP3) -> Vec<u8> {
    let (opcode, dst, src0, src1, src2) = match inst {
        VOP3::FmaF32 { dst, src0, src1, src2 } => {
            (0x1C3, dst.index(), src0.index(), src1.index(), src2.index())
        }
        VOP3::CndmaskB32 { dst, src0, src1, src2 } => {
            (0x101, dst.index(), src0.index(), src1.index(), src2.index())
        }
    };

    // VOP3 encoding: 0xD1 prefix (64-bit)
    let word0: u32 = 0xD1000000 | ((opcode as u32 & 0x3FF) << 16) | (dst as u32);
    let word1: u32 = ((src2 as u32) << 18) | ((src1 as u32) << 9) | (src0 as u32);
    
    let mut bytes = Vec::with_capacity(8);
    bytes.extend_from_slice(&word0.to_le_bytes());
    bytes.extend_from_slice(&word1.to_le_bytes());
    bytes
}

/// Encode SOP1 instruction (32-bit format)
fn encode_sop1(inst: &SOP1) -> Vec<u8> {
    let (opcode, dst, src) = match inst {
        SOP1::MovB32 { dst, src } => (0x03, dst.index(), src.index()),
        SOP1::NotB32 { dst, src } => (0x07, dst.index(), src.index()),
    };

    // SOP1 encoding: 0xBE prefix
    let encoding: u32 = 0xBE800000 | ((opcode as u32) << 8) | ((dst as u32) << 16) | (src as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode SOP2 instruction (32-bit format)
fn encode_sop2(inst: &SOP2) -> Vec<u8> {
    let (opcode, dst, src0, src1) = match inst {
        SOP2::AddU32 { dst, src0, src1 } => (0x00, dst.index(), src0.index(), src1.index()),
        SOP2::SubU32 { dst, src0, src1 } => (0x01, dst.index(), src0.index(), src1.index()),
        SOP2::AndB32 { dst, src0, src1 } => (0x0E, dst.index(), src0.index(), src1.index()),
        SOP2::OrB32 { dst, src0, src1 } => (0x0F, dst.index(), src0.index(), src1.index()),
    };

    // SOP2 encoding: 0x80 prefix
    let encoding: u32 = 0x80000000 | ((opcode as u32) << 23) | ((dst as u32) << 16) | ((src1 as u32) << 8) | (src0 as u32);
    encoding.to_le_bytes().to_vec()
}

/// Encode MIMG (image sample) instruction (64-bit format)
fn encode_image_sample(inst: &ImageSample) -> Vec<u8> {
    let dmask = 0xF; // RGBA enable mask
    let dim_bits = match inst.dim {
        ImageDim::Dim1D => 0x08,
        ImageDim::Dim2D => 0x09,
        ImageDim::Dim3D => 0x0A,
        ImageDim::Cube => 0x0C,
    };

    // MIMG encoding: 0xF0 prefix, opcode for sample instruction
    let word0: u32 = 0xF0000000 | (dmask << 8) | (dim_bits << 0);
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
    let target_bits = match inst.target {
        ExportTarget::MRT(n) => n & 0x7,
        ExportTarget::Position => 0x0C,
        ExportTarget::Parameter(n) => 0x20 + (n & 0x1F),
    };

    let done_bit = if inst.done { 1u32 } else { 0 };
    let compressed_bit = if inst.compressed { 1u32 } else { 0 };

    // EXP encoding: 0xC4 prefix
    let word0: u32 = 0xC4000000 
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
