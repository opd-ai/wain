/// 3D Primitive Rendering Commands
///
/// These commands trigger actual rendering of geometry:
/// - Primitive topology selection
/// - Indexed/non-indexed draws
/// - Instanced rendering

use super::{GpuCommand, CommandType};

/// 3DPRIMITIVE - Draw vertices
///
/// This is the primary command to trigger rendering of geometry.
/// It specifies the primitive topology (triangles, lines, points),
/// vertex count, and instancing parameters.
///
/// Gen9-Gen12 format:
/// - DWord 0: Command header
/// - DWord 1: Topology and flags
/// - DWord 2: Vertex count
/// - DWord 3: Start vertex location
/// - DWord 4: Instance count
/// - DWord 5: Start instance location
/// - DWord 6: Base vertex location (for indexed draws)
#[derive(Debug, Clone)]
pub struct Primitive3D {
    /// Primitive topology type
    pub topology: PrimitiveTopology,
    /// Number of vertices to draw
    pub vertex_count: u32,
    /// Starting vertex location
    pub start_vertex: u32,
    /// Number of instances (1 for non-instanced)
    pub instance_count: u32,
    /// Starting instance location
    pub start_instance: u32,
    /// Base vertex location (added to vertex indices)
    pub base_vertex: u32,
    /// Enable indexed rendering
    pub indexed: bool,
}

/// Primitive topology types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PrimitiveTopology {
    /// Point list
    PointList = 0x01,
    /// Line list
    LineList = 0x02,
    /// Line strip
    LineStrip = 0x03,
    /// Triangle list
    TriangleList = 0x04,
    /// Triangle strip
    TriangleStrip = 0x05,
    /// Triangle fan
    TriangleFan = 0x06,
}

impl Primitive3D {
    /// Create a new 3DPRIMITIVE command for a triangle list.
    pub fn new_triangle_list(vertex_count: u32) -> Self {
        Self {
            topology: PrimitiveTopology::TriangleList,
            vertex_count,
            start_vertex: 0,
            instance_count: 1,
            start_instance: 0,
            base_vertex: 0,
            indexed: false,
        }
    }
    
    /// Create a new 3DPRIMITIVE command for a line list.
    pub fn new_line_list(vertex_count: u32) -> Self {
        Self {
            topology: PrimitiveTopology::LineList,
            vertex_count,
            start_vertex: 0,
            instance_count: 1,
            start_instance: 0,
            base_vertex: 0,
            indexed: false,
        }
    }
    
    /// Create a new 3DPRIMITIVE command for a point list.
    pub fn new_point_list(vertex_count: u32) -> Self {
        Self {
            topology: PrimitiveTopology::PointList,
            vertex_count,
            start_vertex: 0,
            instance_count: 1,
            start_instance: 0,
            base_vertex: 0,
            indexed: false,
        }
    }
    
    /// Enable indexed rendering.
    pub fn indexed(mut self, indexed: bool) -> Self {
        self.indexed = indexed;
        self
    }
    
    /// Set the start vertex location.
    pub fn start_vertex(mut self, start_vertex: u32) -> Self {
        self.start_vertex = start_vertex;
        self
    }
    
    /// Set the instance count.
    pub fn instance_count(mut self, instance_count: u32) -> Self {
        self.instance_count = instance_count;
        self
    }
}

impl GpuCommand for Primitive3D {
    fn serialize(&self) -> Vec<u32> {
        let opcode = 0x7A00; // 3DPRIMITIVE opcode (3D)
        let length = 6; // 7 DWords total
        
        let dw0 = (CommandType::Primitive3D.opcode_type() << 29) | (opcode << 16) | length;
        
        let mut dw1 = (self.topology as u32) & 0x3F; // Topology in bits 5:0
        if self.indexed {
            dw1 |= 1 << 8; // Indexed draw enable
        }
        
        vec![
            dw0,
            dw1,
            self.vertex_count,
            self.start_vertex,
            self.instance_count,
            self.start_instance,
            self.base_vertex,
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn primitive_topology_values() {
        assert_eq!(PrimitiveTopology::PointList as u32, 0x01);
        assert_eq!(PrimitiveTopology::LineList as u32, 0x02);
        assert_eq!(PrimitiveTopology::TriangleList as u32, 0x04);
    }

    #[test]
    fn primitive_3d_triangle_list() {
        let cmd = Primitive3D::new_triangle_list(3);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 7);
        assert_eq!(dwords[1] & 0x3F, PrimitiveTopology::TriangleList as u32);
        assert_eq!(dwords[2], 3); // Vertex count
        assert_eq!(dwords[4], 1); // Instance count
    }

    #[test]
    fn primitive_3d_indexed() {
        let cmd = Primitive3D::new_triangle_list(6).indexed(true);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 7);
        assert_ne!(dwords[1] & (1 << 8), 0); // Indexed bit set
    }

    #[test]
    fn primitive_3d_line_list() {
        let cmd = Primitive3D::new_line_list(4);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords.len(), 7);
        assert_eq!(dwords[1] & 0x3F, PrimitiveTopology::LineList as u32);
        assert_eq!(dwords[2], 4); // Vertex count
    }

    #[test]
    fn primitive_3d_with_start_vertex() {
        let cmd = Primitive3D::new_triangle_list(3).start_vertex(10);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords[3], 10); // Start vertex
    }

    #[test]
    fn primitive_3d_instanced() {
        let cmd = Primitive3D::new_triangle_list(3).instance_count(5);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords[4], 5); // Instance count
    }

    #[test]
    fn primitive_3d_header() {
        let cmd = Primitive3D::new_triangle_list(3);
        let dwords = cmd.serialize();
        
        assert_eq!(dwords[0] >> 29, 3); // Command type = 3D
    }
}
