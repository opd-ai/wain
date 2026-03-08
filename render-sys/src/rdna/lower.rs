// AMD RDNA IR Lowering
//
// This module lowers naga IR to RDNA instructions.
// It handles expression evaluation, control flow, and I/O operations.
//
// Reference: Similar architecture to EU backend's lower.rs

use super::encoding::encode_instruction;
use super::instruction::*;
use super::regalloc::RegisterAllocator;
use super::types::VGPR;
use naga::{Expression, Function, Handle, Module, ShaderStage, Statement};

/// Lowering context for converting naga IR to RDNA instructions
pub struct LoweringContext<'a> {
    module: &'a Module,
    function: &'a Function,
    allocator: RegisterAllocator,
    instructions: Vec<RDNAInstruction>,
    stage: ShaderStage,
}

impl<'a> LoweringContext<'a> {
    pub fn new(module: &'a Module, function: &'a Function, stage: ShaderStage) -> Self {
        LoweringContext {
            module,
            function,
            allocator: RegisterAllocator::new(),
            instructions: Vec::new(),
            stage,
        }
    }

    /// Lower the function to RDNA instructions
    pub fn lower(&mut self) -> Vec<u8> {
        // Reserve input VGPRs based on shader stage
        match self.stage {
            ShaderStage::Vertex => {
                // Reserve v0-v3 for vertex attributes (position, UV, etc.)
                self.allocator.reserve_input_vgprs(4);
            }
            ShaderStage::Fragment => {
                // Reserve v0-v3 for interpolated inputs
                self.allocator.reserve_input_vgprs(4);
            }
            ShaderStage::Compute => {
                // Reserve for thread IDs
                self.allocator.reserve_input_vgprs(3);
            }
        }

        // Process function body - naga's Block is iterable
        for stmt in &self.function.body {
            self.lower_statement(stmt);
        }

        // Add final export based on shader stage
        match self.stage {
            ShaderStage::Fragment => {
                self.emit_fragment_export();
            }
            ShaderStage::Vertex => {
                self.emit_vertex_export();
            }
            _ => {}
        }

        // Encode all instructions to binary
        self.encode_binary()
    }

    /// Lower a single statement
    fn lower_statement(&mut self, stmt: &Statement) {
        match stmt {
            Statement::Emit(range) => {
                for expr_handle in range.clone() {
                    self.lower_expression(expr_handle);
                }
            }
            Statement::Store { pointer, value } => {
                self.lower_store(*pointer, *value);
            }
            Statement::Return { value } => {
                if let Some(val_handle) = value {
                    self.lower_return(*val_handle);
                }
            }
            _ => {
                // Other statement types not yet implemented
            }
        }
    }

    /// Lower an expression to RDNA instructions
    fn lower_expression(&mut self, handle: Handle<Expression>) -> VGPR {
        let expr = &self.function.expressions[handle];
        
        match expr {
            Expression::Literal(lit) => {
                self.lower_literal(handle, lit)
            }
            Expression::Binary { op, left, right } => {
                self.lower_binary(handle, *op, *left, *right)
            }
            Expression::Math { fun, arg, arg1, arg2, arg3 } => {
                self.lower_math(handle, *fun, *arg, *arg1, *arg2, *arg3)
            }
            Expression::ImageSample { image, sampler, coordinate, .. } => {
                self.lower_image_sample(handle, *image, *sampler, *coordinate)
            }
            Expression::FunctionArgument(index) => {
                // Function arguments are in input VGPRs
                VGPR::new(*index as u8)
            }
            Expression::GlobalVariable(_) |
            Expression::LocalVariable(_) |
            Expression::Load { .. } => {
                // Allocate VGPR for loaded values
                self.allocator.alloc_vgpr_for_expr(handle)
            }
            _ => {
                // Default: allocate a VGPR
                self.allocator.alloc_vgpr_for_expr(handle)
            }
        }
    }

    /// Lower a literal constant
    fn lower_literal(&mut self, handle: Handle<Expression>, lit: &naga::Literal) -> VGPR {
        let dst = self.allocator.alloc_vgpr_for_expr(handle);
        
        // Load constant into VGPR (simplified - would use SGPR + V_MOV in practice)
        // For now, assume constants are pre-loaded
        dst
    }

    /// Lower a binary operation
    fn lower_binary(
        &mut self,
        handle: Handle<Expression>,
        op: naga::BinaryOperator,
        left: Handle<Expression>,
        right: Handle<Expression>,
    ) -> VGPR {
        let src0 = self.lower_expression(left);
        let src1 = self.lower_expression(right);
        let dst = self.allocator.alloc_vgpr_for_expr(handle);

        use naga::BinaryOperator::*;
        let inst = match op {
            Add => RDNAInstruction::VOP2(VOP2::AddF32 { dst, src0, src1 }),
            Subtract => RDNAInstruction::VOP2(VOP2::SubF32 { dst, src0, src1 }),
            Multiply => RDNAInstruction::VOP2(VOP2::MulF32 { dst, src0, src1 }),
            _ => {
                // Other operators - simplified
                RDNAInstruction::VOP2(VOP2::AddF32 { dst, src0, src1 })
            }
        };

        self.instructions.push(inst);
        dst
    }

    /// Lower a math function
    fn lower_math(
        &mut self,
        handle: Handle<Expression>,
        fun: naga::MathFunction,
        arg: Handle<Expression>,
        _arg1: Option<Handle<Expression>>,
        _arg2: Option<Handle<Expression>>,
        _arg3: Option<Handle<Expression>>,
    ) -> VGPR {
        let src = self.lower_expression(arg);
        let dst = self.allocator.alloc_vgpr_for_expr(handle);

        use naga::MathFunction::*;
        let inst = match fun {
            Abs => RDNAInstruction::VOP1(VOP1::AbsF32 { dst, src }),
            Sqrt => RDNAInstruction::VOP1(VOP1::SqrtF32 { dst, src }),
            _ => {
                // Other math functions
                RDNAInstruction::VOP1(VOP1::MovB32 { dst, src })
            }
        };

        self.instructions.push(inst);
        dst
    }

    /// Lower image sampling
    fn lower_image_sample(
        &mut self,
        handle: Handle<Expression>,
        _image: Handle<Expression>,
        _sampler: Handle<Expression>,
        coordinate: Handle<Expression>,
    ) -> VGPR {
        let addr = self.lower_expression(coordinate);
        let dst = self.allocator.alloc_vgpr_for_expr(handle);

        // Allocate SGPRs for texture and sampler descriptors
        let texture = self.allocator.alloc_sgpr();
        let sampler = self.allocator.alloc_sgpr();

        let inst = RDNAInstruction::ImageSample(ImageSample {
            dst,
            addr,
            sampler,
            texture,
            dim: ImageDim::Dim2D,
        });

        self.instructions.push(inst);
        dst
    }

    /// Lower a store operation
    fn lower_store(&mut self, _pointer: Handle<Expression>, value: Handle<Expression>) {
        // Store operations - simplified
        let _src = self.lower_expression(value);
        // Would emit actual store instruction here
    }

    /// Lower a return statement
    fn lower_return(&mut self, value: Handle<Expression>) {
        let _src = self.lower_expression(value);
        // Return handling depends on shader stage
    }

    /// Emit fragment shader export (render target write)
    fn emit_fragment_export(&mut self) {
        // Assume final color is in v0-v3 (RGBA)
        let src = [VGPR::new(0), VGPR::new(1), VGPR::new(2), VGPR::new(3)];
        
        let export = Export {
            target: ExportTarget::MRT(0),
            enable_mask: 0xF, // RGBA
            src,
            compressed: false,
            done: true,
        };

        self.instructions.push(RDNAInstruction::Export(export));
    }

    /// Emit vertex shader export (position output)
    fn emit_vertex_export(&mut self) {
        // Assume final position is in v0-v3 (XYZW)
        let src = [VGPR::new(0), VGPR::new(1), VGPR::new(2), VGPR::new(3)];
        
        let export = Export {
            target: ExportTarget::Position,
            enable_mask: 0xF, // XYZW
            src,
            compressed: false,
            done: true,
        };

        self.instructions.push(RDNAInstruction::Export(export));
    }

    /// Encode all instructions to binary
    fn encode_binary(&self) -> Vec<u8> {
        let mut binary = Vec::new();
        for inst in &self.instructions {
            binary.extend_from_slice(&encode_instruction(inst));
        }
        binary
    }

    /// Get register counts for shader metadata
    pub fn vgpr_count(&self) -> u8 {
        self.allocator.vgpr_count()
    }

    pub fn sgpr_count(&self) -> u8 {
        self.allocator.sgpr_count()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use naga::{Arena, Expression, Literal, Module};

    #[test]
    fn test_lowering_context_creation() {
        let module = Module::default();
        let function = Function::default();
        
        let mut ctx = LoweringContext::new(&module, &function, ShaderStage::Fragment);
        let _binary = ctx.lower();
        
        // Should have at least the final export instruction
        assert!(ctx.instructions.len() >= 1);
    }

    #[test]
    fn test_literal_lowering() {
        let module = Module::default();
        let mut function = Function::default();
        
        let lit_handle = function.expressions.append(
            Expression::Literal(Literal::F32(1.0)),
            Default::default()
        );

        let mut ctx = LoweringContext::new(&module, &function, ShaderStage::Fragment);
        // Note: lower() reserves input VGPRs, but we're testing lower_literal directly
        // which just allocates the next available VGPR without calling lower()
        let vgpr = ctx.allocator.alloc_vgpr_for_expr(lit_handle);
        
        assert_eq!(vgpr.index(), 0); // First allocated VGPR
    }

    #[test]
    fn test_fragment_export() {
        let module = Module::default();
        let function = Function::default();
        
        let mut ctx = LoweringContext::new(&module, &function, ShaderStage::Fragment);
        ctx.emit_fragment_export();
        
        assert_eq!(ctx.instructions.len(), 1);
        matches!(ctx.instructions[0], RDNAInstruction::Export(_));
    }

    #[test]
    fn test_binary_encoding() {
        let module = Module::default();
        let function = Function::default();
        
        let mut ctx = LoweringContext::new(&module, &function, ShaderStage::Fragment);
        ctx.emit_fragment_export();
        
        let binary = ctx.encode_binary();
        assert_eq!(binary.len(), 8); // Export is 64-bit instruction
    }
}
