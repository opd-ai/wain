// Intel EU Instruction Lowering - Phase 4.3
//
// This module lowers naga IR operations to Intel EU instructions.
// It provides the translation layer between platform-independent naga IR
// and GPU-specific EU machine code.
//
// Reference: Intel PRMs Volume 4 (EU ISA)
// Inspiration: Mesa's src/intel/compiler/brw_nir_lower_* for patterns

use super::instruction::{EUInstruction, EUOpcode, Register, RegFile};
use super::regalloc::{RegAllocator, VirtualReg};
use super::encoding::{ExecSize, DataType};
use super::{EUCompileError, IntelGen};
use naga::{BinaryOperator, Expression, Function, Module, UnaryOperator};
use std::collections::HashMap;

/// Instruction lowering context
pub struct LoweringContext<'a> {
    /// Target GPU generation
    gen: IntelGen,
    /// Register allocator
    reg_alloc: RegAllocator,
    /// Map from naga expression handles to virtual registers
    expr_to_reg: HashMap<naga::Handle<Expression>, VirtualReg>,
    /// Naga module being compiled
    module: &'a Module,
    /// Current function being compiled
    function: Option<&'a Function>,
    /// Generated EU instructions
    instructions: Vec<EUInstruction>,
}

impl<'a> LoweringContext<'a> {
    /// Create a new lowering context
    pub fn new(gen: IntelGen, module: &'a Module) -> Self {
        LoweringContext {
            gen,
            reg_alloc: RegAllocator::new(),
            expr_to_reg: HashMap::new(),
            module,
            function: None,
            instructions: Vec::new(),
        }
    }

    /// Set the current function being compiled
    pub fn set_function(&mut self, function: &'a Function) {
        self.function = Some(function);
        self.expr_to_reg.clear();
        self.instructions.clear();
    }

    /// Get the generated instructions
    pub fn instructions(&self) -> &[EUInstruction] {
        &self.instructions
    }

    /// Take the generated instructions, consuming them
    pub fn take_instructions(&mut self) -> Vec<EUInstruction> {
        std::mem::take(&mut self.instructions)
    }

    /// Lower a binary arithmetic operation
    ///
    /// Generates EU instructions for: Add, Subtract, Multiply, Divide
    pub fn lower_binary_arith(
        &mut self,
        op: BinaryOperator,
        left: naga::Handle<Expression>,
        right: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        // Get or allocate registers for operands
        let left_reg = self.get_or_alloc_reg(left)?;
        let right_reg = self.get_or_alloc_reg(right)?;
        let dst_reg = self.alloc_reg(result)?;

        // Convert virtual registers to physical registers
        let left_phys = self.reg_alloc.get_physical(left_reg)
            .ok_or_else(|| EUCompileError::from("Left operand not allocated"))?;
        let right_phys = self.reg_alloc.get_physical(right_reg)
            .ok_or_else(|| EUCompileError::from("Right operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        // Create register references
        let dst = Register {
            file: RegFile::GRF,
            num: dst_phys.grf_num,
            subreg: 0,
        };
        let src0 = Register {
            file: RegFile::GRF,
            num: left_phys.grf_num,
            subreg: 0,
        };
        let src1 = Register {
            file: RegFile::GRF,
            num: right_phys.grf_num,
            subreg: 0,
        };

        // Select EU opcode based on naga operation
        let eu_opcode = match op {
            BinaryOperator::Add => EUOpcode::Add,
            BinaryOperator::Subtract => {
                // Subtract is implemented as dst = src0 + (-src1)
                // We'll use Add with source modifier on src1
                EUOpcode::Add
            }
            BinaryOperator::Multiply => EUOpcode::Mul,
            BinaryOperator::Divide => {
                // Division requires multi-instruction sequence on EU
                // For now, return error - will implement later
                return Err(EUCompileError::from(
                    "Division lowering not yet implemented"
                ));
            }
            _ => {
                return Err(EUCompileError::from(format!(
                    "Unsupported binary operator: {:?}",
                    op
                )));
            }
        };

        // Create the instruction
        let mut inst = EUInstruction::new(eu_opcode);
        inst.set_dst(dst);
        inst.set_src0(src0);
        inst.set_src1(src1);
        inst.set_exec_size(ExecSize::Scalar); // Start with scalar, vectorize later
        inst.set_dst_type(DataType::F); // Default to float, refine based on type later
        inst.set_src0_type(DataType::F);
        inst.set_src1_type(DataType::F);

        // Handle subtract with source negation
        if matches!(op, BinaryOperator::Subtract) {
            inst.set_src1_negate(true);
        }

        self.instructions.push(inst);
        Ok(())
    }

    /// Lower a unary arithmetic operation
    ///
    /// Generates EU instructions for: Negate
    pub fn lower_unary_arith(
        &mut self,
        op: UnaryOperator,
        expr: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        let src_reg = self.get_or_alloc_reg(expr)?;
        let dst_reg = self.alloc_reg(result)?;

        let src_phys = self.reg_alloc.get_physical(src_reg)
            .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        let dst = Register {
            file: RegFile::GRF,
            num: dst_phys.grf_num,
            subreg: 0,
        };
        let src0 = Register {
            file: RegFile::GRF,
            num: src_phys.grf_num,
            subreg: 0,
        };

        match op {
            UnaryOperator::Negate => {
                // Negate is implemented as MOV with source negation
                let mut inst = EUInstruction::new(EUOpcode::Mov);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_src0_negate(true);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                self.instructions.push(inst);
                Ok(())
            }
            UnaryOperator::LogicalNot | UnaryOperator::BitwiseNot => {
                // NOT instruction
                let mut inst = EUInstruction::new(EUOpcode::Not);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::UD); // Integer type for bitwise
                inst.set_src0_type(DataType::UD);
                self.instructions.push(inst);
                Ok(())
            }
        }
    }

    /// Get or allocate a virtual register for an expression
    fn get_or_alloc_reg(
        &mut self,
        expr: naga::Handle<Expression>,
    ) -> Result<VirtualReg, EUCompileError> {
        if let Some(&reg) = self.expr_to_reg.get(&expr) {
            Ok(reg)
        } else {
            self.alloc_reg(expr)
        }
    }

    /// Allocate a new virtual register for an expression
    fn alloc_reg(
        &mut self,
        expr: naga::Handle<Expression>,
    ) -> Result<VirtualReg, EUCompileError> {
        let vreg = self.reg_alloc.allocate_vreg();
        self.expr_to_reg.insert(expr, vreg);
        Ok(vreg)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use naga::ShaderStage;
    use crate::shader::ShaderModule;

    #[test]
    fn test_lowering_context_creation() {
        let module = naga::Module::default();
        let ctx = LoweringContext::new(IntelGen::Gen9, &module);
        assert_eq!(ctx.gen, IntelGen::Gen9);
        assert_eq!(ctx.instructions().len(), 0);
    }

    #[test]
    fn test_lower_add_instruction() {
        let mut module = naga::Module::default();
        
        // Create dummy expression handles (must be done before creating context)
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(2.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Add,
            left,
            right,
        }, Default::default());

        // Now create context with immutable reference
        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        // Lower the add operation
        let res = ctx.lower_binary_arith(BinaryOperator::Add, left, right, result);
        assert!(res.is_ok(), "Add lowering should succeed");

        // Verify an instruction was generated
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Should generate one instruction");
    }

    #[test]
    fn test_lower_subtract_with_negation() {
        let mut module = naga::Module::default();
        
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Subtract,
            left,
            right,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_binary_arith(BinaryOperator::Subtract, left, right, result);
        assert!(res.is_ok(), "Subtract lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_multiply() {
        let mut module = naga::Module::default();
        
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(4.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Multiply,
            left,
            right,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_binary_arith(BinaryOperator::Multiply, left, right, result);
        assert!(res.is_ok(), "Multiply lowering should succeed");
    }

    #[test]
    fn test_lower_divide_not_implemented() {
        let mut module = naga::Module::default();
        
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(10.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(2.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Divide,
            left,
            right,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_binary_arith(BinaryOperator::Divide, left, right, result);
        assert!(res.is_err(), "Division should return error (not implemented)");
        assert!(res.unwrap_err().to_string().contains("not yet implemented"));
    }

    #[test]
    fn test_lower_negate() {
        let mut module = naga::Module::default();
        
        let expr = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Unary {
            op: UnaryOperator::Negate,
            expr,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_unary_arith(UnaryOperator::Negate, expr, result);
        assert!(res.is_ok(), "Negate lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_not() {
        let mut module = naga::Module::default();
        
        let expr = module.const_expressions.append(Expression::Literal(
            naga::Literal::U32(0xFF)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Unary {
            op: UnaryOperator::BitwiseNot,
            expr,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_unary_arith(UnaryOperator::BitwiseNot, expr, result);
        assert!(res.is_ok(), "BitwiseNot lowering should succeed");
    }

    #[test]
    fn test_multiple_instructions() {
        let mut module = naga::Module::default();
        
        // Create a chain: result = (a + b) * c
        let a = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let b = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(2.0)
        ), Default::default());
        let c = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        
        let sum = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Add,
            left: a,
            right: b,
        }, Default::default());
        
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Multiply,
            left: sum,
            right: c,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        // Lower both operations
        ctx.lower_binary_arith(BinaryOperator::Add, a, b, sum).unwrap();
        ctx.lower_binary_arith(BinaryOperator::Multiply, sum, c, result).unwrap();

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 2, "Should generate two instructions");
    }
}
