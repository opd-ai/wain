/// PM4 (Packet Manager version 4) command packet generation for AMD GPUs.
///
/// This module implements PM4 packet encoding for RDNA2/RDNA3 GPUs.
/// PM4 is AMD's command submission format - the equivalent of Intel's 3DSTATE commands.
///
/// Reference:
/// - AMD public register databases
/// - Mesa src/amd/common/sid.h and src/amd/vulkan/ (RADV)
/// - PM4 Type 3 packets are the primary command format

// PM4 command streams are arrays of u32 values interpreted byte-by-byte by
// the AMD GPU, which is always little-endian. Reinterpreting those u32s as
// raw bytes (via slice::from_raw_parts in PM4Builder::as_bytes) only produces
// the correct byte order on a little-endian host.
#[cfg(not(target_endian = "little"))]
compile_error!("PM4 command streams require little-endian architecture");

use std::io;

/// PM4 packet types.
///
/// Type 3 packets are used for most GPU commands (drawing, state changes, etc.)
/// Type 0/1/2 are for legacy/special purposes.
#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PacketType {
    Type0 = 0,  // Register write
    Type1 = 1,  // Reserved
    Type2 = 2,  // Filler packet
    Type3 = 3,  // Primary command packet
}

/// PM4 Type 3 opcodes for GFX pipeline commands.
///
/// These are the subset needed for basic UI rendering.
/// Full opcode list varies by GPU generation - these are common across RDNA2/3.
#[repr(u16)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PM4Opcode {
    /// NOP - No operation
    Nop = 0x10,
    
    /// SET_CONTEXT_REG - Set context registers (0xA000-0xAFFF range)
    SetContextReg = 0x69,
    
    /// SET_SH_REG - Set shader registers (0x2C00-0x2FFF range)
    SetShReg = 0x76,
    
    /// SET_UCONFIG_REG - Set user config registers (0xC000-0xCFFF range)
    SetUconfigReg = 0x79,
    
    /// DRAW_INDEX_AUTO - Draw using auto-incrementing vertex index
    DrawIndexAuto = 0x2D,
    
    /// DRAW_INDEX_2 - Draw indexed primitives
    DrawIndex2 = 0x2E,
    
    /// EVENT_WRITE - Write event for synchronization
    EventWrite = 0x46,
    
    /// SURFACE_SYNC - Synchronize surface access
    SurfaceSync = 0x43,
    
    /// ACQUIRE_MEM - Memory barrier and cache flush
    AcquireMem = 0x58,
    
    /// RELEASE_MEM - Release memory and signal fence
    ReleaseMem = 0x59,
    
    /// SET_BASE - Set base address for indirect buffers
    SetBase = 0x75,
    
    /// CLEAR_STATE - Clear GPU state to defaults
    ClearState = 0x12,
}

/// Event types for EVENT_WRITE packet.
#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EventType {
    /// Cache flush before read
    CacheFlushAndInvEvent = 0x16,
    
    /// End of pipe timestamp
    BottomOfPipeTs = 0x28,
    
    /// VS partial flush
    VsPartialFlush = 0x0F,
    
    /// PS partial flush  
    PsPartialFlush = 0x10,
    
    /// CS partial flush
    CsPartialFlush = 0x07,
}

/// PM4 packet builder - constructs PM4 command buffers.
///
/// Similar to Intel CommandBuilder, this accumulates PM4 packets
/// into a byte buffer for GPU submission.
pub struct PM4Builder {
    data: Vec<u32>,  // PM4 packets are aligned to DWORD boundaries
}

impl PM4Builder {
    /// Create a new PM4 packet builder.
    pub fn new() -> Self {
        Self {
            data: Vec::with_capacity(1024),
        }
    }
    
    /// Get the accumulated packet data as bytes.
    pub fn as_bytes(&self) -> &[u8] {
        // SAFETY: Reinterpret Vec<u32> as &[u8]:
        // - Vec<u32> guarantees valid, aligned, initialized memory
        // - Byte length = u32 count * 4 (no overflow: Vec::len() * 4 < isize::MAX)
        // - Lifetime tied to self (slice cannot outlive Vec)
        // - u8 has no alignment requirements (u32's alignment is sufficient)
        unsafe {
            std::slice::from_raw_parts(
                self.data.as_ptr() as *const u8,
                self.data.len() * 4
            )
        }
    }
    
    /// Get the number of DWords in the buffer.
    pub fn len_dwords(&self) -> usize {
        self.data.len()
    }
    
    /// Get the size in bytes.
    pub fn len_bytes(&self) -> usize {
        self.data.len() * 4
    }
    
    /// Emit a raw DWORD to the packet stream.
    fn emit_dword(&mut self, dword: u32) {
        self.data.push(dword);
    }
    
    /// Encode a Type 3 packet header.
    ///
    /// Format (bits):
    /// [31:30] = Type (3)
    /// [29:28] = Shader type (0 for most commands)
    /// [27:16] = Opcode
    /// [15:14] = Reserved
    /// [13:0]  = Count (number of DWORDs - 1, excluding header)
    fn type3_header(opcode: PM4Opcode, count: u16) -> u32 {
        let pkt_type = (PacketType::Type3 as u32) << 30;
        let op = (opcode as u32) << 16;
        let cnt = (count as u32) & 0x3FFF;
        pkt_type | op | cnt
    }
    
    /// Emit a NOP packet with optional payload.
    ///
    /// NOP packets can carry data or be used for padding.
    pub fn nop(&mut self, payload_dwords: u16) {
        let header = Self::type3_header(PM4Opcode::Nop, payload_dwords);
        self.emit_dword(header);
        for _ in 0..payload_dwords {
            self.emit_dword(0);
        }
    }
    
    /// Emit SET_CONTEXT_REG packet.
    ///
    /// Sets context registers in the 0xA000-0xAFFF range.
    /// `reg_offset` is the offset from 0xA000 (in DWORDs).
    /// `values` are the register values to write consecutively.
    ///
    /// Example: set_context_reg(0x123, &[0xAABBCCDD]) sets register 0xA123
    pub fn set_context_reg(&mut self, reg_offset: u16, values: &[u32]) {
        if values.is_empty() {
            return;
        }
        
        let count = (1 + values.len()) as u16 - 1; // reg_offset + values - header
        let header = Self::type3_header(PM4Opcode::SetContextReg, count);
        
        self.emit_dword(header);
        self.emit_dword(reg_offset as u32);
        
        for &value in values {
            self.emit_dword(value);
        }
    }
    
    /// Emit SET_SH_REG packet.
    ///
    /// Sets shader registers in the 0x2C00-0x2FFF range.
    /// `reg_offset` is the offset from 0x2C00 (in DWORDs).
    pub fn set_sh_reg(&mut self, reg_offset: u16, values: &[u32]) {
        if values.is_empty() {
            return;
        }
        
        let count = (1 + values.len()) as u16 - 1;
        let header = Self::type3_header(PM4Opcode::SetShReg, count);
        
        self.emit_dword(header);
        self.emit_dword(reg_offset as u32);
        
        for &value in values {
            self.emit_dword(value);
        }
    }
    
    /// Emit SET_UCONFIG_REG packet.
    ///
    /// Sets user config registers in the 0xC000-0xCFFF range.
    /// `reg_offset` is the offset from 0xC000 (in DWORDs).
    pub fn set_uconfig_reg(&mut self, reg_offset: u16, values: &[u32]) {
        if values.is_empty() {
            return;
        }
        
        let count = (1 + values.len()) as u16 - 1;
        let header = Self::type3_header(PM4Opcode::SetUconfigReg, count);
        
        self.emit_dword(header);
        self.emit_dword(reg_offset as u32);
        
        for &value in values {
            self.emit_dword(value);
        }
    }
    
    /// Emit DRAW_INDEX_AUTO packet.
    ///
    /// Draws primitives using auto-incrementing vertex indices.
    /// This is the equivalent of Intel's 3DPRIMITIVE command.
    ///
    /// # Arguments
    ///
    /// * `vertex_count` - Number of vertices to draw
    /// * `prim_type` - Primitive type (see PrimitiveType)
    pub fn draw_index_auto(&mut self, vertex_count: u32, prim_type: PrimitiveType) {
        let header = Self::type3_header(PM4Opcode::DrawIndexAuto, 3);
        
        self.emit_dword(header);
        self.emit_dword(vertex_count);
        
        // DW2: Draw initiator
        // [5:0] = source select (DI_SRC_SEL_AUTO_INDEX = 2)
        // [10:6] = primitive type
        let draw_initiator = 2 | ((prim_type as u32) << 6);
        self.emit_dword(draw_initiator);
        
        // DW3: Instance count (1 for non-instanced)
        self.emit_dword(1);
    }
    
    /// Emit EVENT_WRITE packet.
    ///
    /// Writes an event for synchronization or profiling.
    pub fn event_write(&mut self, event_type: EventType) {
        let header = Self::type3_header(PM4Opcode::EventWrite, 1);
        
        self.emit_dword(header);
        
        // DW1: Event index and type
        // [3:0] = event type
        // [11:8] = event index (0 for most events)
        let event_dw = (event_type as u32) & 0x3F;
        self.emit_dword(event_dw);
    }
    
    /// Emit SURFACE_SYNC packet (cache flush/invalidate).
    ///
    /// Ensures coherency between different pipeline stages.
    /// Deprecated on newer GPUs in favor of ACQUIRE_MEM/RELEASE_MEM.
    pub fn surface_sync(&mut self, coher_cntl: u32, coher_size: u32, coher_base: u64) {
        let header = Self::type3_header(PM4Opcode::SurfaceSync, 4);
        
        self.emit_dword(header);
        self.emit_dword(coher_cntl);  // Coherency control flags
        self.emit_dword(coher_size);  // Size in bytes
        
        // Base address (48-bit GPU address in DW3:DW4)
        self.emit_dword((coher_base & 0xFFFFFFFF) as u32);
        self.emit_dword((coher_base >> 32) as u32);
    }
    
    /// Emit ACQUIRE_MEM packet (memory barrier and cache flush).
    ///
    /// RDNA2+ replacement for SURFACE_SYNC.
    /// Used to ensure memory operations complete before proceeding.
    pub fn acquire_mem(&mut self, coher_cntl: u32, coher_size: u32, gcr_cntl: u32) {
        let header = Self::type3_header(PM4Opcode::AcquireMem, 6);
        
        self.emit_dword(header);
        self.emit_dword(coher_cntl);  // CP_COHER_CNTL flags
        self.emit_dword(coher_size);  // Size
        
        // Base address (low/high) - 0 for global sync
        self.emit_dword(0);
        self.emit_dword(0);
        
        // Poll interval
        self.emit_dword(0x00000A00);  // Default poll interval
        
        // GCR (Graphics Cache Rinse) control
        self.emit_dword(gcr_cntl);
    }
    
    /// Emit RELEASE_MEM packet (memory release and fence signal).
    ///
    /// Complements ACQUIRE_MEM - signals completion and optionally writes fence.
    pub fn release_mem(&mut self, event_type: EventType, fence_va: u64, fence_value: u64) {
        let header = Self::type3_header(PM4Opcode::ReleaseMem, 6);
        
        self.emit_dword(header);
        
        // DW1: Event type and data selection
        // [5:0] = event type
        // [10:8] = data_sel (0=discard, 1=send 32b, 2=send 64b, 3=send GPU clock)
        let event_flags = (event_type as u32) | (2 << 8); // Send 64-bit data
        self.emit_dword(event_flags);
        
        // DW2: Reserved/control flags
        self.emit_dword(0);
        
        // DW3-4: Fence GPU virtual address (48-bit)
        self.emit_dword((fence_va & 0xFFFFFFFF) as u32);
        self.emit_dword((fence_va >> 32) as u32);
        
        // DW5-6: Fence value (64-bit)
        self.emit_dword((fence_value & 0xFFFFFFFF) as u32);
        self.emit_dword((fence_value >> 32) as u32);
    }
    
    /// Emit CLEAR_STATE packet.
    ///
    /// Resets GPU pipeline state to known defaults.
    /// Useful at the start of a command buffer.
    pub fn clear_state(&mut self) {
        let header = Self::type3_header(PM4Opcode::ClearState, 0);
        self.emit_dword(header);
    }
}

impl Default for PM4Builder {
    fn default() -> Self {
        Self::new()
    }
}

/// Primitive topology types for AMD GPUs.
#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PrimitiveType {
    PointList = 0,
    LineList = 1,
    LineStrip = 2,
    TriangleList = 3,
    TriangleFan = 4,
    TriangleStrip = 5,
    RectList = 8,  // RDNA-specific rectangle primitive
}

/// Coherency control flags for SURFACE_SYNC / ACQUIRE_MEM.
///
/// These flags control which caches to flush/invalidate.
pub mod coher_cntl {
    /// Invalidate texture L1 cache
    pub const TC_ACTION_ENA: u32 = 1 << 23;
    
    /// Invalidate texture cache metadata
    pub const TC_WB_ACTION_ENA: u32 = 1 << 15;
    
    /// Invalidate shader L0 cache
    pub const SH_ICACHE_ACTION_ENA: u32 = 1 << 29;
    
    /// Invalidate shader L1 cache
    pub const SH_KCACHE_ACTION_ENA: u32 = 1 << 27;
    
    /// Full pipeline flush
    pub const FULL_FLUSH: u32 = TC_ACTION_ENA | TC_WB_ACTION_ENA | 
                                 SH_ICACHE_ACTION_ENA | SH_KCACHE_ACTION_ENA;
}

/// GCR (Graphics Cache Rinse) control flags for ACQUIRE_MEM on RDNA2+.
pub mod gcr_cntl {
    /// Invalidate graphics L0 cache
    pub const GL0_INV: u32 = 1 << 0;
    
    /// Invalidate graphics L1 cache
    pub const GL1_INV: u32 = 1 << 1;
    
    /// Writeback and invalidate graphics L2 cache
    pub const GL2_WB: u32 = 1 << 2;
    pub const GL2_INV: u32 = 1 << 3;
    
    /// Full graphics cache flush
    pub const FULL_FLUSH: u32 = GL0_INV | GL1_INV | GL2_WB | GL2_INV;
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_type3_header() {
        let header = PM4Builder::type3_header(PM4Opcode::Nop, 0);
        
        // Check packet type (bits 31:30 = 3)
        assert_eq!((header >> 30) & 0x3, 3);
        
        // Check opcode (bits 27:16 = 0x10 for NOP)
        assert_eq!((header >> 16) & 0xFFF, 0x10);
        
        // Check count (bits 13:0 = 0)
        assert_eq!(header & 0x3FFF, 0);
    }
    
    #[test]
    fn test_nop() {
        let mut pm4 = PM4Builder::new();
        pm4.nop(0);
        
        assert_eq!(pm4.len_dwords(), 1);
        assert_eq!(pm4.len_bytes(), 4);
        
        let header = pm4.data[0];
        assert_eq!((header >> 30) & 0x3, 3); // Type 3
        assert_eq!((header >> 16) & 0xFFF, 0x10); // NOP opcode
    }
    
    #[test]
    fn test_nop_with_payload() {
        let mut pm4 = PM4Builder::new();
        pm4.nop(4);
        
        assert_eq!(pm4.len_dwords(), 5); // header + 4 payload
    }
    
    #[test]
    fn test_set_context_reg() {
        let mut pm4 = PM4Builder::new();
        pm4.set_context_reg(0x100, &[0xDEADBEEF]);
        
        assert_eq!(pm4.len_dwords(), 3); // header + offset + value
        
        let header = pm4.data[0];
        assert_eq!((header >> 16) & 0xFFF, PM4Opcode::SetContextReg as u32);
        assert_eq!(header & 0x3FFF, 1); // count = 1 (offset + 1 value - header)
        
        assert_eq!(pm4.data[1], 0x100); // offset
        assert_eq!(pm4.data[2], 0xDEADBEEF); // value
    }
    
    #[test]
    fn test_set_context_reg_multiple() {
        let mut pm4 = PM4Builder::new();
        pm4.set_context_reg(0x200, &[0x11111111, 0x22222222, 0x33333333]);
        
        assert_eq!(pm4.len_dwords(), 5); // header + offset + 3 values
        assert_eq!(pm4.data[1], 0x200);
        assert_eq!(pm4.data[2], 0x11111111);
        assert_eq!(pm4.data[3], 0x22222222);
        assert_eq!(pm4.data[4], 0x33333333);
    }
    
    #[test]
    fn test_set_sh_reg() {
        let mut pm4 = PM4Builder::new();
        pm4.set_sh_reg(0x50, &[0xABCD1234]);
        
        assert_eq!(pm4.len_dwords(), 3);
        
        let header = pm4.data[0];
        assert_eq!((header >> 16) & 0xFFF, PM4Opcode::SetShReg as u32);
    }
    
    #[test]
    fn test_draw_index_auto() {
        let mut pm4 = PM4Builder::new();
        pm4.draw_index_auto(3, PrimitiveType::TriangleList);
        
        assert_eq!(pm4.len_dwords(), 4); // header + vertex_count + initiator + instance_count
        
        let vertex_count = pm4.data[1];
        assert_eq!(vertex_count, 3);
        
        let draw_initiator = pm4.data[2];
        assert_eq!(draw_initiator & 0x3F, 2); // DI_SRC_SEL_AUTO_INDEX
        assert_eq!((draw_initiator >> 6) & 0x1F, PrimitiveType::TriangleList as u32);
        
        let instance_count = pm4.data[3];
        assert_eq!(instance_count, 1);
    }
    
    #[test]
    fn test_event_write() {
        let mut pm4 = PM4Builder::new();
        pm4.event_write(EventType::VsPartialFlush);
        
        assert_eq!(pm4.len_dwords(), 2); // header + event
        
        let event = pm4.data[1];
        assert_eq!(event & 0x3F, EventType::VsPartialFlush as u32);
    }
    
    #[test]
    fn test_surface_sync() {
        let mut pm4 = PM4Builder::new();
        let base_addr = 0x1234_5678_9ABC_0000u64;
        pm4.surface_sync(coher_cntl::FULL_FLUSH, 4096, base_addr);
        
        assert_eq!(pm4.len_dwords(), 5); // header + 4 params
        
        assert_eq!(pm4.data[1], coher_cntl::FULL_FLUSH);
        assert_eq!(pm4.data[2], 4096);
        
        // Check 64-bit address encoding
        let addr_low = pm4.data[3] as u64;
        let addr_high = pm4.data[4] as u64;
        let reconstructed = addr_low | (addr_high << 32);
        assert_eq!(reconstructed, base_addr);
    }
    
    #[test]
    fn test_acquire_mem() {
        let mut pm4 = PM4Builder::new();
        pm4.acquire_mem(
            coher_cntl::FULL_FLUSH,
            0,
            gcr_cntl::FULL_FLUSH
        );
        
        assert_eq!(pm4.len_dwords(), 7); // header + 6 params
        assert_eq!(pm4.data[1], coher_cntl::FULL_FLUSH);
        assert_eq!(pm4.data[6], gcr_cntl::FULL_FLUSH);
    }
    
    #[test]
    fn test_release_mem() {
        let mut pm4 = PM4Builder::new();
        let fence_addr = 0xFEED_FACE_0000u64;
        let fence_val = 0x1234_5678_9ABC_DEF0u64;
        
        pm4.release_mem(EventType::BottomOfPipeTs, fence_addr, fence_val);
        
        assert_eq!(pm4.len_dwords(), 7); // header + 6 params
        
        // Check event flags
        let event_flags = pm4.data[1];
        assert_eq!(event_flags & 0x3F, EventType::BottomOfPipeTs as u32);
        assert_eq!((event_flags >> 8) & 0x7, 2); // data_sel = 2 (64-bit)
        
        // Check fence address
        let addr_low = pm4.data[3] as u64;
        let addr_high = pm4.data[4] as u64;
        assert_eq!(addr_low | (addr_high << 32), fence_addr);
        
        // Check fence value
        let val_low = pm4.data[5] as u64;
        let val_high = pm4.data[6] as u64;
        assert_eq!(val_low | (val_high << 32), fence_val);
    }
    
    #[test]
    fn test_clear_state() {
        let mut pm4 = PM4Builder::new();
        pm4.clear_state();
        
        assert_eq!(pm4.len_dwords(), 1);
        
        let header = pm4.data[0];
        assert_eq!((header >> 16) & 0xFFF, PM4Opcode::ClearState as u32);
        assert_eq!(header & 0x3FFF, 0); // count = 0
    }
    
    #[test]
    fn test_as_bytes() {
        let mut pm4 = PM4Builder::new();
        pm4.nop(0);
        pm4.nop(0);
        
        let bytes = pm4.as_bytes();
        assert_eq!(bytes.len(), 8); // 2 DWORDs = 8 bytes
    }
    
    #[test]
    fn test_builder_chain() {
        let mut pm4 = PM4Builder::new();
        
        pm4.clear_state();
        pm4.set_context_reg(0x100, &[0xAAAA]);
        pm4.draw_index_auto(3, PrimitiveType::TriangleList);
        pm4.event_write(EventType::PsPartialFlush);
        
        assert_eq!(pm4.len_dwords(), 1 + 3 + 4 + 2); // Sum of all packets
    }
}
