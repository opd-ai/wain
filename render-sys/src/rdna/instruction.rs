// AMD RDNA Instruction Definitions
//
// RDNA instructions are encoded in 32-bit or 64-bit words depending on instruction type.
// Main instruction categories:
// - VOP: Vector ALU operations (VOP1, VOP2, VOP3)
// - SOP: Scalar ALU operations (SOP1, SOP2, SOPC, SOPK, SOPP)
// - MIMG: Image/texture sampling instructions
// - EXP: Export instructions (for render target writes)
//
// Reference: AMD RDNA ISA Architecture Manual

use super::types::{VGPR, SGPR};

/// VOP1 - Single-operand vector ALU instruction
#[derive(Debug, Clone)]
pub enum VOP1 {
    /// dst = src (move)
    MovB32 { dst: VGPR, src: VGPR },
    /// dst = float(src) (convert int to float)
    CvtF32I32 { dst: VGPR, src: VGPR },
    /// dst = int(src) (convert float to int)
    CvtI32F32 { dst: VGPR, src: VGPR },
    /// dst = 1.0 / src
    RcpF32 { dst: VGPR, src: VGPR },
    /// dst = 1.0 / sqrt(src)
    RsqF32 { dst: VGPR, src: VGPR },
    /// dst = sqrt(src)
    SqrtF32 { dst: VGPR, src: VGPR },
    /// dst = abs(src)
    AbsF32 { dst: VGPR, src: VGPR },
    /// dst = -src
    NegF32 { dst: VGPR, src: VGPR },
}

/// VOP2 - Two-operand vector ALU instruction
#[derive(Debug, Clone)]
pub enum VOP2 {
    /// dst = src0 + src1
    AddF32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 - src1
    SubF32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 * src1
    MulF32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = min(src0, src1)
    MinF32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = max(src0, src1)
    MaxF32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 + src1 (integer)
    AddU32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 - src1 (integer)
    SubU32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 * src1 (integer, low 32 bits)
    MulLoU32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 & src1
    AndB32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 | src1
    OrB32 { dst: VGPR, src0: VGPR, src1: VGPR },
    /// dst = src0 ^ src1
    XorB32 { dst: VGPR, src0: VGPR, src1: VGPR },
}

/// VOP3 - Three-operand vector ALU instruction (extended format)
#[derive(Debug, Clone)]
pub enum VOP3 {
    /// dst = src0 * src1 + src2 (fused multiply-add)
    FmaF32 { dst: VGPR, src0: VGPR, src1: VGPR, src2: VGPR },
    /// dst = (src0 < src1) ? src2 : src3 (select on condition)
    CndmaskB32 { dst: VGPR, src0: VGPR, src1: VGPR, src2: VGPR },
}

/// SOP1 - Single-operand scalar ALU instruction
#[derive(Debug, Clone)]
pub enum SOP1 {
    /// dst = src (move scalar)
    MovB32 { dst: SGPR, src: SGPR },
    /// dst = ~src (bitwise NOT)
    NotB32 { dst: SGPR, src: SGPR },
}

/// SOP2 - Two-operand scalar ALU instruction
#[derive(Debug, Clone)]
pub enum SOP2 {
    /// dst = src0 + src1
    AddU32 { dst: SGPR, src0: SGPR, src1: SGPR },
    /// dst = src0 - src1
    SubU32 { dst: SGPR, src0: SGPR, src1: SGPR },
    /// dst = src0 & src1
    AndB32 { dst: SGPR, src0: SGPR, src1: SGPR },
    /// dst = src0 | src1
    OrB32 { dst: SGPR, src0: SGPR, src1: SGPR },
}

/// MIMG - Image/texture sampling instruction
#[derive(Debug, Clone)]
pub struct ImageSample {
    /// Destination VGPR for sampled color (RGBA)
    pub dst: VGPR,
    /// Address VGPR (texture coordinates)
    pub addr: VGPR,
    /// Sampler resource descriptor (SGPR base)
    pub sampler: SGPR,
    /// Texture resource descriptor (SGPR base)
    pub texture: SGPR,
    /// Dimension (1D, 2D, 3D, Cube)
    pub dim: ImageDim,
}

/// Image dimension for sampling
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ImageDim {
    Dim1D,
    Dim2D,
    Dim3D,
    Cube,
}

/// EXP - Export instruction (for render target writes)
#[derive(Debug, Clone)]
pub struct Export {
    /// Export target (MRT 0-7, or position/parameter for vertex shader)
    pub target: ExportTarget,
    /// Enable mask for RGBA channels
    pub enable_mask: u8,
    /// Source VGPRs for RGBA channels
    pub src: [VGPR; 4],
    /// Compressed export (combine channels)
    pub compressed: bool,
    /// Done flag (marks final export in shader)
    pub done: bool,
}

/// Export target for EXP instruction
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExportTarget {
    /// Multiple Render Target (0-7)
    MRT(u8),
    /// Vertex shader position output
    Position,
    /// Vertex shader parameter output
    Parameter(u8),
}

/// Complete RDNA instruction set
#[derive(Debug, Clone)]
pub enum RDNAInstruction {
    VOP1(VOP1),
    VOP2(VOP2),
    VOP3(VOP3),
    SOP1(SOP1),
    SOP2(SOP2),
    ImageSample(ImageSample),
    Export(Export),
}

impl RDNAInstruction {
    /// Get the size of this instruction in bytes
    pub fn size_bytes(&self) -> usize {
        match self {
            RDNAInstruction::VOP1(_) |
            RDNAInstruction::VOP2(_) |
            RDNAInstruction::SOP1(_) |
            RDNAInstruction::SOP2(_) => 4, // 32-bit encoding
            
            RDNAInstruction::VOP3(_) |
            RDNAInstruction::ImageSample(_) |
            RDNAInstruction::Export(_) => 8, // 64-bit encoding
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_vop1_creation() {
        let inst = VOP1::MovB32 {
            dst: VGPR::new(0),
            src: VGPR::new(1),
        };
        matches!(inst, VOP1::MovB32 { .. });
    }

    #[test]
    fn test_vop2_creation() {
        let inst = VOP2::AddF32 {
            dst: VGPR::new(0),
            src0: VGPR::new(1),
            src1: VGPR::new(2),
        };
        matches!(inst, VOP2::AddF32 { .. });
    }

    #[test]
    fn test_image_sample() {
        let inst = ImageSample {
            dst: VGPR::new(0),
            addr: VGPR::new(4),
            sampler: SGPR::new(0),
            texture: SGPR::new(4),
            dim: ImageDim::Dim2D,
        };
        assert_eq!(inst.dim, ImageDim::Dim2D);
    }

    #[test]
    fn test_export() {
        let inst = Export {
            target: ExportTarget::MRT(0),
            enable_mask: 0xF,
            src: [VGPR::new(0), VGPR::new(1), VGPR::new(2), VGPR::new(3)],
            compressed: false,
            done: true,
        };
        assert_eq!(inst.enable_mask, 0xF);
        assert!(inst.done);
    }

    #[test]
    fn test_instruction_sizes() {
        let vop1 = RDNAInstruction::VOP1(VOP1::MovB32 { dst: VGPR::new(0), src: VGPR::new(1) });
        assert_eq!(vop1.size_bytes(), 4);

        let export = RDNAInstruction::Export(Export {
            target: ExportTarget::MRT(0),
            enable_mask: 0xF,
            src: [VGPR::new(0), VGPR::new(1), VGPR::new(2), VGPR::new(3)],
            compressed: false,
            done: true,
        });
        assert_eq!(export.size_bytes(), 8);
    }
}
