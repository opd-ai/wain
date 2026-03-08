// Intel EU Register Allocator - Phase 4.3
//
// This module implements register allocation for the Intel EU ISA.
// It maps naga IR SSA values to physical GRF (General Register File) registers.
//
// Strategy: Simple linear-scan allocator for the initial implementation.
// Intel EU has 128 GRF registers on Gen9+ (r0-r127), each 32 bytes (8 DWORDs).
//
// Reference: Intel PRMs Volume 4, Section on Register File

use std::collections::HashMap;
use naga::Handle;

/// Virtual register representing a naga IR value
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct VirtualReg(pub u32);

/// Physical EU GRF register
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PhysicalReg {
    pub grf_num: u8,  // GRF number (0-127)
    pub subreg: u8,   // Sub-register offset (0-31 bytes)
}

/// Register allocation context
pub struct RegAllocator {
    /// Map from virtual to physical registers
    allocation: HashMap<VirtualReg, PhysicalReg>,
    /// Next available GRF register
    next_grf: u8,
    /// Next virtual register ID
    next_vreg: u32,
}

impl RegAllocator {
    /// Create a new register allocator
    pub fn new() -> Self {
        RegAllocator {
            allocation: HashMap::new(),
            // r0 and r1 are reserved for special purposes on Intel EU
            next_grf: 2,
            next_vreg: 0,
        }
    }
    
    /// Allocate a new virtual register
    pub fn allocate_vreg(&mut self) -> VirtualReg {
        let vreg = VirtualReg(self.next_vreg);
        self.next_vreg += 1;
        
        // Immediately allocate a physical register for it
        self.allocate(vreg);
        vreg
    }
    
    /// Get the physical register for a virtual register (alias for get)
    pub fn get_physical(&self, vreg: VirtualReg) -> Option<PhysicalReg> {
        self.get(vreg)
    }

    /// Allocate a physical register for a virtual register
    ///
    /// Returns the allocated physical register or None if out of registers
    pub fn allocate(&mut self, vreg: VirtualReg) -> Option<PhysicalReg> {
        // Check if already allocated
        if let Some(preg) = self.allocation.get(&vreg) {
            return Some(*preg);
        }

        // Allocate new register
        if self.next_grf >= 128 {
            return None;  // Out of registers
        }

        let preg = PhysicalReg {
            grf_num: self.next_grf,
            subreg: 0,
        };

        self.next_grf += 1;
        self.allocation.insert(vreg, preg);
        Some(preg)
    }

    /// Get the physical register for a virtual register
    pub fn get(&self, vreg: VirtualReg) -> Option<PhysicalReg> {
        self.allocation.get(&vreg).copied()
    }

    /// Reset allocator state
    pub fn reset(&mut self) {
        self.allocation.clear();
        self.next_grf = 2;
        self.next_vreg = 0;
    }
}

impl Default for RegAllocator {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_regalloc_basic() {
        let mut allocator = RegAllocator::new();
        
        let v0 = VirtualReg(0);
        let v1 = VirtualReg(1);
        
        let p0 = allocator.allocate(v0).unwrap();
        let p1 = allocator.allocate(v1).unwrap();
        
        assert_eq!(p0.grf_num, 2);  // r0, r1 reserved
        assert_eq!(p1.grf_num, 3);
        
        // Requesting same vreg returns same preg
        let p0_again = allocator.allocate(v0).unwrap();
        assert_eq!(p0_again.grf_num, p0.grf_num);
    }

    #[test]
    fn test_regalloc_reset() {
        let mut allocator = RegAllocator::new();
        
        let v0 = VirtualReg(0);
        allocator.allocate(v0).unwrap();
        
        allocator.reset();
        
        assert!(allocator.get(v0).is_none());
        
        let p0 = allocator.allocate(v0).unwrap();
        assert_eq!(p0.grf_num, 2);
    }

    #[test]
    fn test_regalloc_get() {
        let mut allocator = RegAllocator::new();
        
        let v0 = VirtualReg(0);
        assert!(allocator.get(v0).is_none());
        
        allocator.allocate(v0).unwrap();
        assert!(allocator.get(v0).is_some());
    }
}
