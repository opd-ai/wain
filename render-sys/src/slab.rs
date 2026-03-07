/// Slab allocator for sub-allocating from large GPU buffers.
///
/// This module implements a simple slab allocator that manages regions
/// within a large GPU buffer, reducing allocation overhead and fragmentation.

use std::io;
use crate::allocator::{Buffer, BufferAllocator, TilingFormat};

/// A region within a slab.
#[derive(Debug, Clone, Copy)]
struct Region {
    offset: u64,
    size: u64,
    used: bool,
}

/// Sub-allocated buffer region from a slab.
#[derive(Debug)]
pub struct SlabAllocation {
    pub offset: u64,
    pub size: u64,
    pub handle: u32,
}

/// Slab allocator that manages sub-allocations from a large GPU buffer.
pub struct SlabAllocator {
    buffer: Buffer,
    regions: Vec<Region>,
    total_size: u64,
}

impl SlabAllocator {
    /// Create a new slab allocator with a large backing buffer.
    ///
    /// The slab size should be large enough to accommodate multiple allocations
    /// (e.g., 64 MB or 256 MB) to amortize allocation overhead.
    pub fn new(
        allocator: &BufferAllocator,
        slab_size: u64,
        width: u32,
        height: u32,
        bpp: u32,
        tiling: TilingFormat,
    ) -> io::Result<Self> {
        let buffer = allocator.allocate(width, height, bpp, tiling)?;

        // Initialize with one large free region
        let regions = vec![Region {
            offset: 0,
            size: slab_size,
            used: false,
        }];

        Ok(Self {
            buffer,
            regions,
            total_size: slab_size,
        })
    }

    /// Allocate a region from the slab.
    ///
    /// Returns a SlabAllocation containing the offset and size within the
    /// backing buffer, or an error if no suitable free region is found.
    pub fn alloc(&mut self, size: u64) -> io::Result<SlabAllocation> {
        // Align size to 4KB boundaries for better GPU performance
        let aligned_size = (size + 4095) & !4095;

        // Find first free region large enough (first-fit strategy)
        for i in 0..self.regions.len() {
            if !self.regions[i].used && self.regions[i].size >= aligned_size {
                let offset = self.regions[i].offset;
                let region_size = self.regions[i].size;

                if region_size == aligned_size {
                    // Exact fit - mark region as used
                    self.regions[i].used = true;
                } else {
                    // Split region: mark first part as used, second part as free
                    self.regions[i] = Region {
                        offset,
                        size: aligned_size,
                        used: true,
                    };
                    self.regions.insert(i + 1, Region {
                        offset: offset + aligned_size,
                        size: region_size - aligned_size,
                        used: false,
                    });
                }

                return Ok(SlabAllocation {
                    offset,
                    size: aligned_size,
                    handle: self.buffer.handle,
                });
            }
        }

        Err(io::Error::new(
            io::ErrorKind::OutOfMemory,
            "no free region large enough in slab",
        ))
    }

    /// Free a previously allocated region back to the slab.
    ///
    /// Coalesces adjacent free regions to reduce fragmentation.
    pub fn free(&mut self, allocation: SlabAllocation) -> io::Result<()> {
        // Find the region matching this allocation
        let mut found_index = None;
        for (i, region) in self.regions.iter().enumerate() {
            if region.offset == allocation.offset && region.size == allocation.size && region.used {
                found_index = Some(i);
                break;
            }
        }

        let index = found_index.ok_or_else(|| {
            io::Error::new(io::ErrorKind::InvalidInput, "allocation not found in slab")
        })?;

        // Mark region as free
        self.regions[index].used = false;

        // Coalesce with adjacent free regions
        self.coalesce(index);

        Ok(())
    }

    /// Coalesce free regions to reduce fragmentation.
    fn coalesce(&mut self, start_index: usize) {
        let i = start_index;

        // Coalesce with next region if free
        while i + 1 < self.regions.len() {
            if !self.regions[i].used && !self.regions[i + 1].used {
                let next_size = self.regions[i + 1].size;
                self.regions[i].size += next_size;
                self.regions.remove(i + 1);
            } else {
                break;
            }
        }

        // Coalesce with previous region if free
        if i > 0 && !self.regions[i - 1].used && !self.regions[i].used {
            let current_size = self.regions[i].size;
            self.regions[i - 1].size += current_size;
            self.regions.remove(i);
        }
    }

    /// Get the total size of the slab.
    pub fn total_size(&self) -> u64 {
        self.total_size
    }

    /// Get the amount of free space in the slab.
    pub fn free_size(&self) -> u64 {
        self.regions
            .iter()
            .filter(|r| !r.used)
            .map(|r| r.size)
            .sum()
    }

    /// Get the amount of used space in the slab.
    pub fn used_size(&self) -> u64 {
        self.total_size - self.free_size()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn region_initialization() {
        let region = Region {
            offset: 0,
            size: 1024,
            used: false,
        };
        assert_eq!(region.offset, 0);
        assert_eq!(region.size, 1024);
        assert!(!region.used);
    }

    #[test]
    fn slab_allocation_structure() {
        let alloc = SlabAllocation {
            offset: 4096,
            size: 8192,
            handle: 42,
        };
        assert_eq!(alloc.offset, 4096);
        assert_eq!(alloc.size, 8192);
        assert_eq!(alloc.handle, 42);
    }

    #[test]
    fn size_alignment() {
        let size = 1000u64;
        let aligned = (size + 4095) & !4095;
        assert_eq!(aligned, 4096);

        let size = 4096u64;
        let aligned = (size + 4095) & !4095;
        assert_eq!(aligned, 4096);

        let size = 5000u64;
        let aligned = (size + 4095) & !4095;
        assert_eq!(aligned, 8192);
    }
}
