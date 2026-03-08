// AMD RDNA Register Types and File Definitions
//
// RDNA architecture uses two main register files:
// - VGPR (Vector General Purpose Registers): Per-lane vector operations (256 registers)
// - SGPR (Scalar General Purpose Registers): Wave-wide scalar operations (106 registers)
//
// Reference: AMD RDNA ISA documentation

/// Vector General Purpose Register (VGPR)
/// Each VGPR is 32 bits × wave size (32 or 64 lanes)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct VGPR(pub u8);

impl VGPR {
    pub fn new(index: u8) -> Self {
        // u8 already enforces 0-255 range, no need to check
        VGPR(index)
    }

    pub fn index(&self) -> u8 {
        self.0
    }
}

/// Scalar General Purpose Register (SGPR)
/// Each SGPR is 32 bits, shared across the entire wave
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct SGPR(pub u8);

impl SGPR {
    pub fn new(index: u8) -> Self {
        assert!(index < 106, "SGPR index out of range");
        SGPR(index)
    }

    pub fn index(&self) -> u8 {
        self.0
    }
}

/// RDNA register file - tracks allocated registers
#[derive(Debug)]
pub struct RegisterFile {
    /// Next available VGPR index
    next_vgpr: u8,
    /// Next available SGPR index
    next_sgpr: u8,
}

impl RegisterFile {
    pub fn new() -> Self {
        RegisterFile {
            next_vgpr: 0,
            next_sgpr: 0,
        }
    }

    /// Allocate a new VGPR
    ///
    /// # Panics
    ///
    /// Panics if all 255 VGPRs have already been allocated (overflow guard).
    pub fn alloc_vgpr(&mut self) -> VGPR {
        assert!(self.next_vgpr < 255, "VGPR exhausted: cannot allocate register index 255 or higher");
        let vgpr = VGPR::new(self.next_vgpr);
        self.next_vgpr += 1;
        vgpr
    }

    /// Allocate a new SGPR
    ///
    /// # Panics
    ///
    /// Panics if all 255 SGPRs have already been allocated (overflow guard).
    pub fn alloc_sgpr(&mut self) -> SGPR {
        assert!(self.next_sgpr < 255, "SGPR exhausted: cannot allocate register index 255 or higher");
        let sgpr = SGPR::new(self.next_sgpr);
        self.next_sgpr += 1;
        sgpr
    }

    /// Get count of allocated VGPRs
    pub fn vgpr_count(&self) -> u8 {
        self.next_vgpr
    }

    /// Get count of allocated SGPRs
    pub fn sgpr_count(&self) -> u8 {
        self.next_sgpr
    }
}

impl Default for RegisterFile {
    fn default() -> Self {
        Self::new()
    }
}

/// Data type size for operations
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DataType {
    F32,
    I32,
    U32,
    F16,
    I16,
    U16,
}

impl DataType {
    pub fn size_bytes(&self) -> usize {
        match self {
            DataType::F32 | DataType::I32 | DataType::U32 => 4,
            DataType::F16 | DataType::I16 | DataType::U16 => 2,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_vgpr_creation() {
        let vgpr = VGPR::new(5);
        assert_eq!(vgpr.index(), 5);
        
        // Test max value
        let vgpr_max = VGPR::new(255);
        assert_eq!(vgpr_max.index(), 255);
    }

    #[test]
    fn test_sgpr_creation() {
        let sgpr = SGPR::new(10);
        assert_eq!(sgpr.index(), 10);
    }

    #[test]
    fn test_register_file_allocation() {
        let mut rf = RegisterFile::new();
        
        let v0 = rf.alloc_vgpr();
        let v1 = rf.alloc_vgpr();
        assert_eq!(v0.index(), 0);
        assert_eq!(v1.index(), 1);
        assert_eq!(rf.vgpr_count(), 2);

        let s0 = rf.alloc_sgpr();
        let s1 = rf.alloc_sgpr();
        assert_eq!(s0.index(), 0);
        assert_eq!(s1.index(), 1);
        assert_eq!(rf.sgpr_count(), 2);
    }

    #[test]
    fn test_data_type_sizes() {
        assert_eq!(DataType::F32.size_bytes(), 4);
        assert_eq!(DataType::I32.size_bytes(), 4);
        assert_eq!(DataType::F16.size_bytes(), 2);
    }
}
