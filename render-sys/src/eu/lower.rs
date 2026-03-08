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
use super::encoding::{ExecSize, DataType, CondMod};
use super::{EUCompileError, IntelGen};
use naga::{BinaryOperator, Expression, Function, MathFunction, Module, UnaryOperator};
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

    /// Lower a math function
    ///
    /// Generates EU instructions for: Abs, Min, Max, Floor, Ceil, Round, Fract, Sqrt, Mix
    pub fn lower_math(
        &mut self,
        fun: MathFunction,
        arg: naga::Handle<Expression>,
        arg1: Option<naga::Handle<Expression>>,
        arg2: Option<naga::Handle<Expression>>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        match fun {
            MathFunction::Abs => {
                // Absolute value: MOV with source absolute modifier
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;

                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                let src0 = Register { file: RegFile::GRF, num: src_phys.grf_num, subreg: 0 };

                let mut inst = EUInstruction::new(EUOpcode::Mov);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_src0_absolute(true);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::Min | MathFunction::Max => {
                // Min/Max: SEL with conditional modifier
                let arg1 = arg1.ok_or_else(|| EUCompileError::from("Min/Max requires two arguments"))?;
                
                let src0_reg = self.get_or_alloc_reg(arg)?;
                let src1_reg = self.get_or_alloc_reg(arg1)?;
                let dst_reg = self.alloc_reg(result)?;

                let src0_phys = self.reg_alloc.get_physical(src0_reg)
                    .ok_or_else(|| EUCompileError::from("Source 0 not allocated"))?;
                let src1_phys = self.reg_alloc.get_physical(src1_reg)
                    .ok_or_else(|| EUCompileError::from("Source 1 not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                let src0 = Register { file: RegFile::GRF, num: src0_phys.grf_num, subreg: 0 };
                let src1 = Register { file: RegFile::GRF, num: src1_phys.grf_num, subreg: 0 };

                let mut inst = EUInstruction::new(EUOpcode::Sel);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_src1(src1);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                inst.set_src1_type(DataType::F);
                
                // Set conditional based on min/max
                if fun == MathFunction::Min {
                    inst.set_cond_mod(CondMod::L); // Less than (select src0 if src0 < src1)
                } else {
                    inst.set_cond_mod(CondMod::G); // Greater than
                }
                
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::Floor | MathFunction::Ceil | MathFunction::Round | MathFunction::Fract => {
                // These require specialized EU instructions or multi-instruction sequences
                // For now, return error - will implement in next iteration
                Err(EUCompileError::from(format!(
                    "Math function {:?} not yet implemented", fun
                )))
            }
            MathFunction::Sqrt => {
                // Sqrt: Use EU math instruction (implementation varies by generation)
                // For now, return error - requires SEND instruction to math function unit
                Err(EUCompileError::from("Sqrt not yet implemented"))
            }
            MathFunction::Mix => {
                // Mix (lerp): result = x * (1 - a) + y * a
                // This is a multi-instruction sequence
                let arg1 = arg1.ok_or_else(|| EUCompileError::from("Mix requires arg1 (y)"))?;
                let arg2 = arg2.ok_or_else(|| EUCompileError::from("Mix requires arg2 (a)"))?;
                
                // For now, return error - will implement multi-instruction lowering
                Err(EUCompileError::from("Mix not yet implemented (requires multi-instruction lowering)"))
            }
            _ => {
                Err(EUCompileError::from(format!(
                    "Unsupported math function: {:?}", fun
                )))
            }
        }
    }

    /// Lower a comparison/relational operation
    ///
    /// Generates EU CMP instructions for: Equal, NotEqual, Less, LessEqual, Greater, GreaterEqual
    pub fn lower_comparison(
        &mut self,
        op: BinaryOperator,
        arg: naga::Handle<Expression>,
        arg1: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        let src0_reg = self.get_or_alloc_reg(arg)?;
        let src1_reg = self.get_or_alloc_reg(arg1)?;
        let dst_reg = self.alloc_reg(result)?;

        let src0_phys = self.reg_alloc.get_physical(src0_reg)
            .ok_or_else(|| EUCompileError::from("Source 0 not allocated"))?;
        let src1_phys = self.reg_alloc.get_physical(src1_reg)
            .ok_or_else(|| EUCompileError::from("Source 1 not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
        let src0 = Register { file: RegFile::GRF, num: src0_phys.grf_num, subreg: 0 };
        let src1 = Register { file: RegFile::GRF, num: src1_phys.grf_num, subreg: 0 };

        let cond_mod = match op {
            BinaryOperator::Equal => CondMod::E,
            BinaryOperator::NotEqual => CondMod::NE,
            BinaryOperator::Less => CondMod::L,
            BinaryOperator::LessEqual => CondMod::LE,
            BinaryOperator::Greater => CondMod::G,
            BinaryOperator::GreaterEqual => CondMod::GE,
            _ => {
                return Err(EUCompileError::from(format!(
                    "Unsupported comparison operator: {:?}", op
                )));
            }
        };

        let mut inst = EUInstruction::new(EUOpcode::Cmp);
        inst.set_dst(dst);
        inst.set_src0(src0);
        inst.set_src1(src1);
        inst.set_cond_mod(cond_mod);
        inst.set_exec_size(ExecSize::Scalar);
        inst.set_dst_type(DataType::F);
        inst.set_src0_type(DataType::F);
        inst.set_src1_type(DataType::F);
        
        self.instructions.push(inst);
        Ok(())
    }

    /// Lower a select operation (ternary conditional)
    ///
    /// Generates EU SEL instruction for: condition ? accept : reject
    pub fn lower_select(
        &mut self,
        condition: naga::Handle<Expression>,
        accept: naga::Handle<Expression>,
        reject: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        // Select is implemented as predicated MOV instructions
        // This is a simplified version - full implementation requires predication support
        
        let cond_reg = self.get_or_alloc_reg(condition)?;
        let accept_reg = self.get_or_alloc_reg(accept)?;
        let reject_reg = self.get_or_alloc_reg(reject)?;
        let dst_reg = self.alloc_reg(result)?;

        let cond_phys = self.reg_alloc.get_physical(cond_reg)
            .ok_or_else(|| EUCompileError::from("Condition not allocated"))?;
        let accept_phys = self.reg_alloc.get_physical(accept_reg)
            .ok_or_else(|| EUCompileError::from("Accept not allocated"))?;
        let reject_phys = self.reg_alloc.get_physical(reject_reg)
            .ok_or_else(|| EUCompileError::from("Reject not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
        let accept_src = Register { file: RegFile::GRF, num: accept_phys.grf_num, subreg: 0 };
        let reject_src = Register { file: RegFile::GRF, num: reject_phys.grf_num, subreg: 0 };

        // For now, use SEL with condition as comparison result
        // Full implementation would use predication
        let mut inst = EUInstruction::new(EUOpcode::Sel);
        inst.set_dst(dst);
        inst.set_src0(accept_src);
        inst.set_src1(reject_src);
        inst.set_exec_size(ExecSize::Scalar);
        inst.set_dst_type(DataType::F);
        inst.set_src0_type(DataType::F);
        inst.set_src1_type(DataType::F);
        // Note: Proper predication would check condition register
        
        self.instructions.push(inst);
        Ok(())
    }

    /// Lower floating-point division
    ///
    /// Division on EU requires a multi-instruction sequence using IEEE divide algorithm
    pub fn lower_divide(
        &mut self,
        left: naga::Handle<Expression>,
        right: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        // EU floating-point division algorithm:
        // 1. Compute reciprocal approximation: rcp = 1/right (using MATH.INV)
        // 2. Refine with Newton-Raphson: rcp = rcp * (2 - right * rcp)
        // 3. Multiply: result = left * rcp
        //
        // This requires SEND instruction to math function unit
        // For now, return error - will implement when SEND lowering is ready
        
        Err(EUCompileError::from(
            "Division requires SEND instruction support (deferred to next iteration)"
        ))
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

    #[test]
    fn test_lower_math_abs() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(-5.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Abs,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Abs, arg, None, None, result);
        assert!(res.is_ok(), "Abs lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_math_min() {
        let mut module = naga::Module::default();
        
        let arg0 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let arg1 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Min,
            arg: arg0,
            arg1: Some(arg1),
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Min, arg0, Some(arg1), None, result);
        assert!(res.is_ok(), "Min lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_math_max() {
        let mut module = naga::Module::default();
        
        let arg0 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let arg1 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Max,
            arg: arg0,
            arg1: Some(arg1),
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Max, arg0, Some(arg1), None, result);
        assert!(res.is_ok(), "Max lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_comparison_equal() {
        let mut module = naga::Module::default();
        
        let arg0 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let arg1 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Equal,
            left: arg0,
            right: arg1,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_comparison(BinaryOperator::Equal, arg0, arg1, result);
        assert!(res.is_ok(), "Equal lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_comparison_less() {
        let mut module = naga::Module::default();
        
        let arg0 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let arg1 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(2.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Less,
            left: arg0,
            right: arg1,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_comparison(BinaryOperator::Less, arg0, arg1, result);
        assert!(res.is_ok(), "Less lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_comparison_greater_equal() {
        let mut module = naga::Module::default();
        
        let arg0 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let arg1 = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::GreaterEqual,
            left: arg0,
            right: arg1,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_comparison(BinaryOperator::GreaterEqual, arg0, arg1, result);
        assert!(res.is_ok(), "GreaterEqual lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_select() {
        let mut module = naga::Module::default();
        
        let condition = module.const_expressions.append(Expression::Literal(
            naga::Literal::Bool(true)
        ), Default::default());
        let accept = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let reject = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(0.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Select {
            condition,
            accept,
            reject,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_select(condition, accept, reject, result);
        assert!(res.is_ok(), "Select lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_divide_deferred() {
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

        // Test via lower_divide directly
        let res = ctx.lower_divide(left, right, result);
        assert!(res.is_err(), "Divide should return error (deferred)");
        assert!(res.unwrap_err().to_string().contains("SEND instruction"));
    }
}
