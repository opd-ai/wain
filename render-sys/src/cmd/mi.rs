/// MI (Machine Interface) Commands
///
/// Machine interface commands control GPU execution flow, batch buffer
/// chaining, and synchronization. These commands are used across all
/// GPU generations with minimal variation.

use super::{GpuCommand, CommandType};

/// MI_BATCH_BUFFER_START - Chain to another batch buffer
///
/// This command causes the GPU to start executing commands from a new
/// batch buffer. Used for indirect command submission and batch chaining.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1: Batch buffer address (low 32 bits)
/// - DWord 2: Batch buffer address (high 32 bits, Gen8+)
#[derive(Debug, Clone)]
pub struct MiBatchBufferStart {
    /// GPU virtual address of the batch buffer to execute
    pub address: u64,
    /// Second level batch buffer (if true)
    pub second_level: bool,
}

impl MiBatchBufferStart {
    /// Create a new MI_BATCH_BUFFER_START command.
    pub fn new(address: u64) -> Self {
        Self {
            address,
            second_level: false,
        }
    }
    
    /// Set this as a second-level batch buffer.
    pub fn second_level(mut self, second_level: bool) -> Self {
        self.second_level = second_level;
        self
    }
}

impl GpuCommand for MiBatchBufferStart {
    fn serialize(&self) -> Vec<u32> {
        let mi_opcode = 0x31; // MI_BATCH_BUFFER_START opcode
        let length = 1; // Additional DWords beyond the header (2 DWords for address)
        
        let mut dw0 = (CommandType::MI.opcode_type() << 29) | (mi_opcode << 23) | length;
        if self.second_level {
            dw0 |= 1 << 8; // Second Level Batch Buffer bit
        }
        
        vec![
            dw0,
            (self.address & 0xFFFFFFFF) as u32, // Low 32 bits
            (self.address >> 32) as u32,        // High 32 bits
        ]
    }
}

/// MI_NOOP - No operation
///
/// Used for padding and alignment in batch buffers.
#[derive(Debug, Clone, Copy)]
pub struct MiNoop;

impl GpuCommand for MiNoop {
    fn serialize(&self) -> Vec<u32> {
        vec![0x0] // MI_NOOP is all zeros
    }
}

/// PIPE_CONTROL - Pipeline synchronization and cache flushing
///
/// This command provides fine-grained control over pipeline stalls,
/// cache flushes, and synchronization points. Critical for ensuring
/// rendering commands complete before subsequent operations.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1: Control flags
/// - DWord 2-3: Write address (if post-sync write enabled)
/// - DWord 4-5: Write data (if immediate write)
#[derive(Debug, Clone)]
pub struct PipeControl {
    /// Stall at pixel scoreboard (wait for all prior rendering)
    pub stall_at_scoreboard: bool,
    /// Flush render target cache
    pub render_target_cache_flush: bool,
    /// Flush depth cache
    pub depth_cache_flush: bool,
    /// Invalidate texture cache
    pub texture_cache_invalidate: bool,
    /// Command streamer stall enable
    pub cs_stall: bool,
    /// Address to write completion status (optional)
    pub post_sync_address: Option<u64>,
    /// Immediate data to write on completion (optional)
    pub post_sync_data: Option<u64>,
}

impl PipeControl {
    /// Create a new PIPE_CONTROL with default settings (no-op).
    pub fn new() -> Self {
        Self {
            stall_at_scoreboard: false,
            render_target_cache_flush: false,
            depth_cache_flush: false,
            texture_cache_invalidate: false,
            cs_stall: false,
            post_sync_address: None,
            post_sync_data: None,
        }
    }
    
    /// Flush all rendering and caches.
    pub fn full_flush() -> Self {
        Self {
            stall_at_scoreboard: true,
            render_target_cache_flush: true,
            depth_cache_flush: true,
            texture_cache_invalidate: true,
            cs_stall: true,
            post_sync_address: None,
            post_sync_data: None,
        }
    }
}

impl Default for PipeControl {
    fn default() -> Self {
        Self::new()
    }
}

impl GpuCommand for PipeControl {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7A00; // PIPE_CONTROL 3D opcode
        let mut length = 4; // Base length (5 DWords total)
        
        let has_post_sync = self.post_sync_address.is_some();
        if !has_post_sync {
            length = 4; // 5 DWords when no post-sync
        }
        
        let dw0 = (CommandType::State3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let mut dw1 = 0u32;
        if self.stall_at_scoreboard {
            dw1 |= 1 << 1; // Stall at Pixel Scoreboard
        }
        if self.render_target_cache_flush {
            dw1 |= 1 << 12; // Render Target Cache Flush Enable
        }
        if self.depth_cache_flush {
            dw1 |= 1 << 0; // Depth Cache Flush Enable
        }
        if self.texture_cache_invalidate {
            dw1 |= 1 << 10; // Texture Cache Invalidation Enable
        }
        if self.cs_stall {
            dw1 |= 1 << 20; // CS Stall
        }
        if has_post_sync {
            dw1 |= 1 << 14; // Post-Sync Operation: Write Immediate Data
        }
        
        let mut result = vec![dw0, dw1];
        
        if let Some(addr) = self.post_sync_address {
            result.push((addr & 0xFFFFFFFF) as u32);
            result.push((addr >> 32) as u32);
        } else {
            result.push(0); // Address low
            result.push(0); // Address high
        }
        
        if let Some(data) = self.post_sync_data {
            result.push((data & 0xFFFFFFFF) as u32);
            result.push((data >> 32) as u32);
        } else {
            result.push(0); // Data low
            result.push(0); // Data high  (padding even when not used)
        }
        
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mi_batch_buffer_start_serialization() {
        let cmd = MiBatchBufferStart::new(0x1234_5678_9ABC_DEF0);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 3);
        assert_eq!(dwords[1], 0x9ABC_DEF0); // Low 32 bits
        assert_eq!(dwords[2], 0x1234_5678); // High 32 bits
    }

    #[test]
    fn mi_batch_buffer_start_second_level() {
        let cmd = MiBatchBufferStart::new(0x1000).second_level(true);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 3);
        assert_ne!(dwords[0] & (1 << 8), 0); // Second level bit set
    }

    #[test]
    fn mi_noop_serialization() {
        let cmd = MiNoop;
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 1);
        assert_eq!(dwords[0], 0x0);
    }

    #[test]
    fn pipe_control_noop() {
        let cmd = PipeControl::new();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 6); // Header + flags + addr + data
        // DWord 0: command header
        assert_eq!(dwords[0] >> 29, 3); // Command type = 3D
    }

    #[test]
    fn pipe_control_full_flush() {
        let cmd = PipeControl::full_flush();
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 6);
        // Check flush bits are set in DWord 1
        assert_ne!(dwords[1] & (1 << 1), 0); // Stall at scoreboard
        assert_ne!(dwords[1] & (1 << 12), 0); // RT cache flush
        assert_ne!(dwords[1] & (1 << 0), 0); // Depth cache flush
        assert_ne!(dwords[1] & (1 << 10), 0); // Texture cache invalidate
        assert_ne!(dwords[1] & (1 << 20), 0); // CS stall
    }

    #[test]
    fn pipe_control_with_post_sync() {
        let mut cmd = PipeControl::new();
        cmd.post_sync_address = Some(0xDEADBEEF_CAFEBABE);
        cmd.post_sync_data = Some(0x42);
        
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 6);
        assert_ne!(dwords[1] & (1 << 14), 0); // Post-sync bit set
        assert_eq!(dwords[2], 0xCAFE_BABE); // Address low
        assert_eq!(dwords[3], 0xDEAD_BEEF); // Address high
        assert_eq!(dwords[4], 0x42); // Data low
    }
}
