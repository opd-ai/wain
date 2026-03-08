// AMD RDNA Register Allocator
//
// Simple linear-scan register allocator for RDNA VGPR and SGPR files.
// Maps naga IR SSA values to physical VGPR/SGPR registers.
//
// Reference: Similar to EU backend's regalloc.rs approach

use super::types::{VGPR, SGPR, RegisterFile};
use naga::Handle;
use std::collections::HashMap;

/// Register allocation context
pub struct RegisterAllocator {
    /// Physical register file
    reg_file: RegisterFile,
    /// Map from naga expression handles to allocated VGPRs
    vgpr_map: HashMap<u32, VGPR>,
    /// Map from naga expression handles to allocated SGPRs (for uniforms/constants)
    sgpr_map: HashMap<u32, SGPR>,
}

impl RegisterAllocator {
    pub fn new() -> Self {
        RegisterAllocator {
            reg_file: RegisterFile::new(),
            vgpr_map: HashMap::new(),
            sgpr_map: HashMap::new(),
        }
    }

    /// Allocate a VGPR for a naga expression
    pub fn alloc_vgpr_for_expr(&mut self, handle: Handle<naga::Expression>) -> VGPR {
        let key = handle.index() as u32;
        if let Some(&vgpr) = self.vgpr_map.get(&key) {
            return vgpr;
        }
        
        let vgpr = self.reg_file.alloc_vgpr();
        self.vgpr_map.insert(key, vgpr);
        vgpr
    }

    /// Get the VGPR allocated for an expression (must have been allocated)
    pub fn get_vgpr(&self, handle: Handle<naga::Expression>) -> Option<VGPR> {
        let key = handle.index() as u32;
        self.vgpr_map.get(&key).copied()
    }

    /// Allocate a SGPR for a uniform/constant
    pub fn alloc_sgpr(&mut self) -> SGPR {
        self.reg_file.alloc_sgpr()
    }

    /// Get total VGPR count (for shader resource calculation)
    pub fn vgpr_count(&self) -> u8 {
        self.reg_file.vgpr_count()
    }

    /// Get total SGPR count (for shader resource calculation)
    pub fn sgpr_count(&self) -> u8 {
        self.reg_file.sgpr_count()
    }

    /// Reserve specific VGPRs for inputs (vertex attributes, fragment interpolants)
    pub fn reserve_input_vgprs(&mut self, count: u8) -> Vec<VGPR> {
        (0..count).map(|_| self.reg_file.alloc_vgpr()).collect()
    }

    /// Reserve specific VGPRs for outputs (typically for export instruction)
    pub fn reserve_output_vgprs(&mut self, count: u8) -> Vec<VGPR> {
        (0..count).map(|_| self.reg_file.alloc_vgpr()).collect()
    }
}

impl Default for RegisterAllocator {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use naga::{Arena, Expression, Literal};

    #[test]
    fn test_basic_allocation() {
        let mut alloc = RegisterAllocator::new();
        
        // Create a dummy expression arena
        let mut arena = Arena::new();
        let handle1 = arena.append(Expression::Literal(Literal::F32(1.0)), Default::default());
        let handle2 = arena.append(Expression::Literal(Literal::F32(2.0)), Default::default());

        let v0 = alloc.alloc_vgpr_for_expr(handle1);
        let v1 = alloc.alloc_vgpr_for_expr(handle2);

        assert_eq!(v0.index(), 0);
        assert_eq!(v1.index(), 1);
        assert_eq!(alloc.vgpr_count(), 2);
    }

    #[test]
    fn test_expression_reuse() {
        let mut alloc = RegisterAllocator::new();
        
        let mut arena = Arena::new();
        let handle = arena.append(Expression::Literal(Literal::F32(1.0)), Default::default());

        let v0 = alloc.alloc_vgpr_for_expr(handle);
        let v1 = alloc.alloc_vgpr_for_expr(handle); // Should return same register

        assert_eq!(v0.index(), v1.index());
        assert_eq!(alloc.vgpr_count(), 1);
    }

    #[test]
    fn test_sgpr_allocation() {
        let mut alloc = RegisterAllocator::new();
        
        let s0 = alloc.alloc_sgpr();
        let s1 = alloc.alloc_sgpr();

        assert_eq!(s0.index(), 0);
        assert_eq!(s1.index(), 1);
        assert_eq!(alloc.sgpr_count(), 2);
    }

    #[test]
    fn test_reserve_input_vgprs() {
        let mut alloc = RegisterAllocator::new();
        
        let inputs = alloc.reserve_input_vgprs(4);
        assert_eq!(inputs.len(), 4);
        assert_eq!(inputs[0].index(), 0);
        assert_eq!(inputs[3].index(), 3);
        assert_eq!(alloc.vgpr_count(), 4);
    }

    #[test]
    fn test_reserve_output_vgprs() {
        let mut alloc = RegisterAllocator::new();
        
        let outputs = alloc.reserve_output_vgprs(4);
        assert_eq!(outputs.len(), 4);
        assert_eq!(alloc.vgpr_count(), 4);
    }
}
