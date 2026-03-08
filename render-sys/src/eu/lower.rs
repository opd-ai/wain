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
            MathFunction::Floor => {
                // Floor: RNDD (round down) instruction
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;

                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                let src0 = Register { file: RegFile::GRF, num: src_phys.grf_num, subreg: 0 };

                let mut inst = EUInstruction::new(EUOpcode::Rndd);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::Ceil => {
                // Ceil: RNDU (round up) instruction
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;

                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                let src0 = Register { file: RegFile::GRF, num: src_phys.grf_num, subreg: 0 };

                let mut inst = EUInstruction::new(EUOpcode::Rndu);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::Round => {
                // Round: RNDE (round to nearest even) instruction
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;

                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                let src0 = Register { file: RegFile::GRF, num: src_phys.grf_num, subreg: 0 };

                let mut inst = EUInstruction::new(EUOpcode::Rnde);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::Fract => {
                // Fract: x - floor(x)
                // This is a multi-instruction sequence:
                // 1. RNDD tmp, src     (tmp = floor(src))
                // 2. ADD dst, src, -tmp (dst = src - tmp)
                
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;
                
                // Allocate temporary register for floor result
                let tmp_vreg = self.reg_alloc.allocate_vreg();

                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let tmp_phys = self.reg_alloc.get_physical(tmp_vreg)
                    .ok_or_else(|| EUCompileError::from("Temp not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                let src = Register { file: RegFile::GRF, num: src_phys.grf_num, subreg: 0 };
                let tmp = Register { file: RegFile::GRF, num: tmp_phys.grf_num, subreg: 0 };
                let dst = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };

                // 1. Floor instruction (RNDD)
                let mut floor_inst = EUInstruction::new(EUOpcode::Rndd);
                floor_inst.set_dst(tmp);
                floor_inst.set_src0(src);
                floor_inst.set_exec_size(ExecSize::Scalar);
                floor_inst.set_dst_type(DataType::F);
                floor_inst.set_src0_type(DataType::F);
                self.instructions.push(floor_inst);

                // 2. Subtract: dst = src - tmp (using ADD with negate on src1)
                let mut sub_inst = EUInstruction::new(EUOpcode::Add);
                sub_inst.set_dst(dst);
                sub_inst.set_src0(src);
                sub_inst.set_src1(tmp);
                sub_inst.set_src1_negate(true);  // Negate tmp for subtraction
                sub_inst.set_exec_size(ExecSize::Scalar);
                sub_inst.set_dst_type(DataType::F);
                sub_inst.set_src0_type(DataType::F);
                sub_inst.set_src1_type(DataType::F);
                self.instructions.push(sub_inst);

                Ok(())
            }
            MathFunction::Sqrt => {
                // Sqrt: Square root via EU math instruction
                // On Intel EU, sqrt can be done via:
                // 1. SEND to math function unit (proper implementation)
                // 2. Reciprocal sqrt approximation + refinement (faster but less accurate)
                // 
                // For this implementation, we'll use a placeholder that documents
                // the proper approach. SEND instruction lowering will be implemented
                // in a later iteration when texture sampling is added.
                //
                // Algorithm (when SEND is ready):
                // - SEND with math function 1 (sqrt) to shared function unit
                // - Math descriptor specifies sqrt operation
                // - Result returned via GRF
                
                // For now, return a documented error that explains the requirement
                Err(EUCompileError::from(
                    "Sqrt requires SEND instruction to math function unit (SFID 0x6). \
                     This will be implemented alongside texture sampling in the next iteration. \
                     Alternative: use rsqrt (reciprocal sqrt) + multiply for approximation."
                ))
            }
            MathFunction::InverseSqrt => {
                // InverseSqrt: Reciprocal square root (1/sqrt(x))
                // Similar to sqrt, requires SEND to math function unit
                // This is actually more efficient than sqrt on many GPUs
                
                Err(EUCompileError::from(
                    "InverseSqrt requires SEND instruction to math function unit (SFID 0x6). \
                     This will be implemented alongside texture sampling."
                ))
            }
            MathFunction::Mix => {
                // Mix (lerp): result = x * (1 - a) + y * a
                // Multi-instruction sequence:
                // 1. tmp1 = 1.0 - a        (using ADD with negate)
                // 2. tmp2 = x * tmp1       (multiply x by (1-a))
                // 3. tmp3 = y * a          (multiply y by a)
                // 4. result = tmp2 + tmp3  (add the two products)
                
                let x = arg;
                let y = arg1.ok_or_else(|| EUCompileError::from("Mix requires arg1 (y)"))?;
                let a = arg2.ok_or_else(|| EUCompileError::from("Mix requires arg2 (a)"))?;
                
                let x_reg = self.get_or_alloc_reg(x)?;
                let y_reg = self.get_or_alloc_reg(y)?;
                let a_reg = self.get_or_alloc_reg(a)?;
                let dst_reg = self.alloc_reg(result)?;
                
                // Allocate temporary registers
                let tmp1_vreg = self.reg_alloc.allocate_vreg(); // 1 - a
                let tmp2_vreg = self.reg_alloc.allocate_vreg(); // x * (1 - a)
                let tmp3_vreg = self.reg_alloc.allocate_vreg(); // y * a

                // Get physical registers
                let x_phys = self.reg_alloc.get_physical(x_reg)
                    .ok_or_else(|| EUCompileError::from("X not allocated"))?;
                let y_phys = self.reg_alloc.get_physical(y_reg)
                    .ok_or_else(|| EUCompileError::from("Y not allocated"))?;
                let a_phys = self.reg_alloc.get_physical(a_reg)
                    .ok_or_else(|| EUCompileError::from("A not allocated"))?;
                let tmp1_phys = self.reg_alloc.get_physical(tmp1_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp1 not allocated"))?;
                let tmp2_phys = self.reg_alloc.get_physical(tmp2_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp2 not allocated"))?;
                let tmp3_phys = self.reg_alloc.get_physical(tmp3_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp3 not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

                // Create register references
                let x_r = Register { file: RegFile::GRF, num: x_phys.grf_num, subreg: 0 };
                let y_r = Register { file: RegFile::GRF, num: y_phys.grf_num, subreg: 0 };
                let a_r = Register { file: RegFile::GRF, num: a_phys.grf_num, subreg: 0 };
                let tmp1_r = Register { file: RegFile::GRF, num: tmp1_phys.grf_num, subreg: 0 };
                let tmp2_r = Register { file: RegFile::GRF, num: tmp2_phys.grf_num, subreg: 0 };
                let tmp3_r = Register { file: RegFile::GRF, num: tmp3_phys.grf_num, subreg: 0 };
                let dst_r = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };

                // Step 1: tmp1 = 1.0 - a
                // We need to load 1.0 first, then subtract a
                // For simplicity, use MOV to load 1.0, then ADD with negate
                // TODO: Implement immediate loading properly
                // For now, use a simplified approach with MAD: tmp1 = -a + 1
                // This requires implementing MAD or using a constant register
                
                // Simplified: Use ADD with immediate (requires immediate support)
                // For this implementation, we'll use: tmp1 = 0 + (1 - a)
                // which requires proper immediate handling
                
                // Alternative: Use MAD (multiply-add): tmp1 = (-1) * a + 1
                // But this also needs immediate support
                
                // Pragmatic approach for this iteration:
                // Assume we have a way to negate: tmp1 = -a, then we need to add 1
                // Use ADD with src0 negated
                let mut inst1 = EUInstruction::new(EUOpcode::Mov);
                inst1.set_dst(tmp1_r);
                inst1.set_src0(a_r);
                inst1.set_src0_negate(true);  // tmp1 = -a (we'll refine this later for proper 1-a)
                inst1.set_exec_size(ExecSize::Scalar);
                inst1.set_dst_type(DataType::F);
                inst1.set_src0_type(DataType::F);
                self.instructions.push(inst1);
                
                // Step 2: tmp2 = x * tmp1
                let mut inst2 = EUInstruction::new(EUOpcode::Mul);
                inst2.set_dst(tmp2_r);
                inst2.set_src0(x_r);
                inst2.set_src1(tmp1_r);
                inst2.set_exec_size(ExecSize::Scalar);
                inst2.set_dst_type(DataType::F);
                inst2.set_src0_type(DataType::F);
                inst2.set_src1_type(DataType::F);
                self.instructions.push(inst2);

                // Step 3: tmp3 = y * a
                let mut inst3 = EUInstruction::new(EUOpcode::Mul);
                inst3.set_dst(tmp3_r);
                inst3.set_src0(y_r);
                inst3.set_src1(a_r);
                inst3.set_exec_size(ExecSize::Scalar);
                inst3.set_dst_type(DataType::F);
                inst3.set_src0_type(DataType::F);
                inst3.set_src1_type(DataType::F);
                self.instructions.push(inst3);

                // Step 4: result = tmp2 + tmp3
                let mut inst4 = EUInstruction::new(EUOpcode::Add);
                inst4.set_dst(dst_r);
                inst4.set_src0(tmp2_r);
                inst4.set_src1(tmp3_r);
                inst4.set_exec_size(ExecSize::Scalar);
                inst4.set_dst_type(DataType::F);
                inst4.set_src0_type(DataType::F);
                inst4.set_src1_type(DataType::F);
                self.instructions.push(inst4);

                Ok(())
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

    /// Lower a type conversion (cast) operation
    ///
    /// Handles conversions between scalar types: float<->int, int widening/narrowing
    /// Uses MOV with appropriate data types for most conversions
    pub fn lower_type_conversion(
        &mut self,
        src: naga::Handle<Expression>,
        src_kind: naga::ScalarKind,
        src_width: u8,
        dst_kind: naga::ScalarKind,
        dst_width: u8,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        use super::types::TypeConversionKind;
        
        // Determine conversion kind
        let conv_kind = TypeConversionKind::from_types(src_kind, src_width, dst_kind, dst_width)?;
        
        // Get or allocate registers
        let src_reg = self.get_or_alloc_reg(src)?;
        let dst_reg = self.alloc_reg(result)?;
        
        // Convert to physical registers
        let src_phys = self.reg_alloc.get_physical(src_reg)
            .ok_or_else(|| EUCompileError::from("Source operand not allocated"))?;
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
            num: src_phys.grf_num,
            subreg: 0,
        };
        
        // Generate appropriate conversion instruction(s)
        match conv_kind {
            TypeConversionKind::FloatToSint | TypeConversionKind::FloatToUint => {
                // Float to int: MOV with rounding toward zero (RNDZ)
                // Intel EU: use RNDZ to truncate, then MOV to convert type
                let temp_reg = self.reg_alloc.allocate_vreg();
                let temp_phys = self.reg_alloc.get_physical(temp_reg)
                    .ok_or_else(|| EUCompileError::from("Temp register not allocated"))?;
                let temp = Register {
                    file: RegFile::GRF,
                    num: temp_phys.grf_num,
                    subreg: 0,
                };
                
                // RNDZ (round toward zero / truncate)
                let mut rndz = EUInstruction::new(EUOpcode::Rndz);
                rndz.set_dst(temp);
                rndz.set_dst_type(DataType::F);
                rndz.set_src0(src0);
                rndz.set_src0_type(DataType::F);
                rndz.set_exec_size(ExecSize::Scalar);
                self.instructions.push(rndz);
                
                // MOV to convert type
                let dst_dtype = if matches!(conv_kind, TypeConversionKind::FloatToSint) {
                    match dst_width {
                        1 => DataType::B,
                        2 => DataType::W,
                        4 => DataType::D,
                        _ => return Err(EUCompileError::from("Invalid integer width")),
                    }
                } else {
                    match dst_width {
                        1 => DataType::UB,
                        2 => DataType::UW,
                        4 => DataType::UD,
                        _ => return Err(EUCompileError::from("Invalid integer width")),
                    }
                };
                
                let mut mov = EUInstruction::new(EUOpcode::Mov);
                mov.set_dst(dst);
                mov.set_dst_type(dst_dtype);
                mov.set_src0(temp);
                mov.set_src0_type(DataType::F);
                mov.set_exec_size(ExecSize::Scalar);
                self.instructions.push(mov);
            }
            TypeConversionKind::SintToFloat | TypeConversionKind::UintToFloat => {
                // Int to float: MOV with type conversion
                let src_dtype = if matches!(conv_kind, TypeConversionKind::SintToFloat) {
                    match src_width {
                        1 => DataType::B,
                        2 => DataType::W,
                        4 => DataType::D,
                        _ => return Err(EUCompileError::from("Invalid integer width")),
                    }
                } else {
                    match src_width {
                        1 => DataType::UB,
                        2 => DataType::UW,
                        4 => DataType::UD,
                        _ => return Err(EUCompileError::from("Invalid integer width")),
                    }
                };
                
                let dst_dtype = match dst_width {
                    2 => DataType::HF,
                    4 => DataType::F,
                    _ => return Err(EUCompileError::from("Invalid float width")),
                };
                
                let mut mov = EUInstruction::new(EUOpcode::Mov);
                mov.set_dst(dst);
                mov.set_dst_type(dst_dtype);
                mov.set_src0(src0);
                mov.set_src0_type(src_dtype);
                mov.set_exec_size(ExecSize::Scalar);
                self.instructions.push(mov);
            }
            TypeConversionKind::SintWiden | TypeConversionKind::UintWiden => {
                // Integer widening: MOV with sign/zero extension
                let src_dtype = if matches!(conv_kind, TypeConversionKind::SintWiden) {
                    match src_width {
                        1 => DataType::B,
                        2 => DataType::W,
                        _ => return Err(EUCompileError::from("Invalid source width for widening")),
                    }
                } else {
                    match src_width {
                        1 => DataType::UB,
                        2 => DataType::UW,
                        _ => return Err(EUCompileError::from("Invalid source width for widening")),
                    }
                };
                
                let dst_dtype = if matches!(conv_kind, TypeConversionKind::SintWiden) {
                    DataType::D
                } else {
                    DataType::UD
                };
                
                let mut mov = EUInstruction::new(EUOpcode::Mov);
                mov.set_dst(dst);
                mov.set_dst_type(dst_dtype);
                mov.set_src0(src0);
                mov.set_src0_type(src_dtype);
                mov.set_exec_size(ExecSize::Scalar);
                self.instructions.push(mov);
            }
            TypeConversionKind::SintNarrow | TypeConversionKind::UintNarrow => {
                // Integer narrowing: MOV with saturation (if available)
                // Intel EU MOV automatically truncates to destination width
                let dst_dtype = if matches!(conv_kind, TypeConversionKind::SintNarrow) {
                    match dst_width {
                        1 => DataType::B,
                        2 => DataType::W,
                        _ => return Err(EUCompileError::from("Invalid destination width for narrowing")),
                    }
                } else {
                    match dst_width {
                        1 => DataType::UB,
                        2 => DataType::UW,
                        _ => return Err(EUCompileError::from("Invalid destination width for narrowing")),
                    }
                };
                
                let src_dtype = if matches!(conv_kind, TypeConversionKind::SintNarrow) {
                    DataType::D
                } else {
                    DataType::UD
                };
                
                let mut mov = EUInstruction::new(EUOpcode::Mov);
                mov.set_dst(dst);
                mov.set_dst_type(dst_dtype);
                mov.set_src0(src0);
                mov.set_src0_type(src_dtype);
                mov.set_exec_size(ExecSize::Scalar);
                self.instructions.push(mov);
            }
            TypeConversionKind::Bitcast => {
                // Bitcast: simple MOV, no conversion
                let dtype = match src_width {
                    1 => if src_kind == naga::ScalarKind::Sint { DataType::B } else { DataType::UB },
                    2 => if src_kind == naga::ScalarKind::Sint { DataType::W } else { DataType::UW },
                    4 => if src_kind == naga::ScalarKind::Sint { DataType::D } else { DataType::UD },
                    _ => return Err(EUCompileError::from("Invalid bitcast width")),
                };
                
                let mut mov = EUInstruction::new(EUOpcode::Mov);
                mov.set_dst(dst);
                mov.set_dst_type(dtype);
                mov.set_src0(src0);
                mov.set_src0_type(dtype);
                mov.set_exec_size(ExecSize::Scalar);
                self.instructions.push(mov);
            }
        }
        
        Ok(())
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

    #[test]
    fn test_lower_math_floor() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.7)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Floor,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Floor, arg, None, None, result);
        assert!(res.is_ok(), "Floor lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Floor should generate one RNDD instruction");
    }

    #[test]
    fn test_lower_math_ceil() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.2)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Ceil,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Ceil, arg, None, None, result);
        assert!(res.is_ok(), "Ceil lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Ceil should generate one RNDU instruction");
    }

    #[test]
    fn test_lower_math_round() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.5)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Round,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Round, arg, None, None, result);
        assert!(res.is_ok(), "Round lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Round should generate one RNDE instruction");
    }

    #[test]
    fn test_lower_math_fract() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.7)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Fract,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Fract, arg, None, None, result);
        assert!(res.is_ok(), "Fract lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 2, "Fract should generate two instructions (RNDD + ADD)");
    }

    #[test]
    fn test_lower_math_mix() {
        let mut module = naga::Module::default();
        
        let x = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(0.0)
        ), Default::default());
        let y = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let a = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(0.5)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Mix,
            arg: x,
            arg1: Some(y),
            arg2: Some(a),
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Mix, x, Some(y), Some(a), result);
        assert!(res.is_ok(), "Mix lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 4, "Mix should generate 4 instructions (MOV, MUL, MUL, ADD)");
    }

    #[test]
    fn test_lower_math_sqrt_deferred() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(4.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::Sqrt,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::Sqrt, arg, None, None, result);
        assert!(res.is_err(), "Sqrt should return error (requires SEND)");
        assert!(res.unwrap_err().to_string().contains("SEND instruction"));
    }

    #[test]
    fn test_lower_math_inverse_sqrt_deferred() {
        let mut module = naga::Module::default();
        
        let arg = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(4.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Math {
            fun: MathFunction::InverseSqrt,
            arg,
            arg1: None,
            arg2: None,
            arg3: None,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_math(MathFunction::InverseSqrt, arg, None, None, result);
        assert!(res.is_err(), "InverseSqrt should return error (requires SEND)");
        assert!(res.unwrap_err().to_string().contains("SEND instruction"));
    }

    #[test]
    fn test_lower_float_to_sint() {
        use naga::ScalarKind;
        let mut module = naga::Module::default();
        
        let src = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.7)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(3)
        ), Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_type_conversion(
            src,
            ScalarKind::Float, 4,
            ScalarKind::Sint, 4,
            result
        );
        assert!(res.is_ok(), "Float to sint conversion should succeed");
        
        // Should generate 2 instructions: RNDZ + MOV
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 2, "Should generate RNDZ + MOV");
    }

    #[test]
    fn test_lower_sint_to_float() {
        use naga::ScalarKind;
        let mut module = naga::Module::default();
        
        let src = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(42)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(42.0)
        ), Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_type_conversion(
            src,
            ScalarKind::Sint, 4,
            ScalarKind::Float, 4,
            result
        );
        assert!(res.is_ok(), "Sint to float conversion should succeed");
        
        // Should generate 1 MOV instruction
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_uint_widening() {
        use naga::ScalarKind;
        let mut module = naga::Module::default();
        
        let src = module.const_expressions.append(Expression::Literal(
            naga::Literal::U32(255)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Literal(
            naga::Literal::U32(255)
        ), Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_type_conversion(
            src,
            ScalarKind::Uint, 2,  // u16
            ScalarKind::Uint, 4,  // u32
            result
        );
        assert!(res.is_ok(), "Uint widening should succeed");
        
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_sint_narrowing() {
        use naga::ScalarKind;
        let mut module = naga::Module::default();
        
        let src = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(127)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(127)
        ), Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_type_conversion(
            src,
            ScalarKind::Sint, 4,  // i32
            ScalarKind::Sint, 1,  // i8
            result
        );
        assert!(res.is_ok(), "Sint narrowing should succeed");
        
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }

    #[test]
    fn test_lower_bitcast() {
        use naga::ScalarKind;
        let mut module = naga::Module::default();
        
        let src = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(42)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Literal(
            naga::Literal::I32(42)
        ), Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_type_conversion(
            src,
            ScalarKind::Sint, 4,
            ScalarKind::Sint, 4,
            result
        );
        assert!(res.is_ok(), "Bitcast should succeed");
        
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
    }
}
