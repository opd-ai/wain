// Intel EU Instruction Lowering - Phase 4.3
//
// This module lowers naga IR operations to Intel EU instructions.
// It provides the translation layer between platform-independent naga IR
// and GPU-specific EU machine code.
//
// Reference: Intel PRMs Volume 4 (EU ISA)
// Inspiration: Mesa's src/intel/compiler/brw_nir_lower_* for patterns

use super::instruction::{EUInstruction, EUOpcode, Register, RegFile, SendDescriptor, SharedFunctionID};
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
    pub(super) expr_to_reg: HashMap<naga::Handle<Expression>, VirtualReg>,
    /// Track output values for final write (local variable handles to registers)
    pub(super) output_values: Vec<(String, VirtualReg)>,
    /// Naga module being compiled
    pub(super) module: &'a Module,
    /// Current function being compiled
    function: Option<&'a Function>,
    /// Current shader stage (vertex or fragment)
    pub(super) stage: Option<naga::ShaderStage>,
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
            output_values: Vec::new(),
            module,
            function: None,
            stage: None,
            instructions: Vec::new(),
        }
    }

    /// Set the current function and shader stage being compiled
    pub fn set_function(&mut self, function: &'a Function, stage: naga::ShaderStage) {
        self.function = Some(function);
        self.stage = Some(stage);
        self.expr_to_reg.clear();
        self.output_values.clear();
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
                // Division: dst = src0 / src1
                // Implemented as: dst = src0 * (1/src1)
                // Use MATH shared function for reciprocal (1/src1), then multiply
                
                // Allocate temporary register for reciprocal result
                let tmp_vreg = self.reg_alloc.allocate_vreg();
                let tmp_phys = self.reg_alloc.get_physical(tmp_vreg)
                    .ok_or_else(|| EUCompileError::from("Temp not allocated for division"))?;
                let tmp = Register {
                    file: RegFile::GRF,
                    num: tmp_phys.grf_num,
                    subreg: 0,
                };
                
                // Step 1: tmp = 1/src1 (reciprocal via MATH shared function)
                // Math function control: 0x3 = reciprocal
                let recip_descriptor = SendDescriptor {
                    sfid: SharedFunctionID::Math,
                    response_length: 1,  // 1 GRF register back
                    message_length: 1,   // 1 GRF register sent
                    function_control: 0x3,  // Reciprocal operation
                };
                
                let recip_inst = EUInstruction::new(EUOpcode::Send)
                    .with_dst(tmp, DataType::F)
                    .with_src0(src1, DataType::F)
                    .with_exec_size(ExecSize::Scalar)
                    .with_send_descriptor(recip_descriptor);
                self.instructions.push(recip_inst);
                
                // Step 2: dst = src0 * tmp (multiply by reciprocal)
                let mut mul_inst = EUInstruction::new(EUOpcode::Mul);
                mul_inst.set_dst(dst);
                mul_inst.set_src0(src0);
                mul_inst.set_src1(tmp);
                mul_inst.set_exec_size(ExecSize::Scalar);
                mul_inst.set_dst_type(DataType::F);
                mul_inst.set_src0_type(DataType::F);
                mul_inst.set_src1_type(DataType::F);
                self.instructions.push(mul_inst);
                
                return Ok(());
            }
            BinaryOperator::And | BinaryOperator::LogicalAnd => EUOpcode::And,
            BinaryOperator::InclusiveOr | BinaryOperator::LogicalOr => EUOpcode::Or,
            BinaryOperator::ExclusiveOr => EUOpcode::Xor,
            BinaryOperator::ShiftLeft => EUOpcode::Shl,
            BinaryOperator::ShiftRight => EUOpcode::Shr,
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
    /// Generates EU instructions for: Abs, Min, Max, Floor, Ceil, Round, Fract, Sqrt, InverseSqrt, Mix, Clamp, SmoothStep, Length, Dot
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
                // Sqrt: Square root via SEND to math function unit
                // On Intel EU, sqrt is implemented via SEND instruction to the math
                // shared function (SFID 0xB) with function control for sqrt operation.
                //
                // Algorithm:
                // - SEND with math function for sqrt (function_control = 0x1)
                // - Result returned via GRF destination register
                
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;
                
                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let src = Register {
                    file: RegFile::GRF,
                    num: src_phys.grf_num,
                    subreg: 0,
                };
                let dst = Register {
                    file: RegFile::GRF,
                    num: dst_phys.grf_num,
                    subreg: 0,
                };
                
                // Create SEND descriptor for math function (sqrt)
                // Math function control: 0x1 = sqrt
                let descriptor = SendDescriptor {
                    sfid: SharedFunctionID::Math,
                    response_length: 1,  // 1 GRF register back
                    message_length: 1,   // 1 GRF register sent
                    function_control: 0x1,  // Sqrt operation
                };
                
                let inst = EUInstruction::new(EUOpcode::Send)
                    .with_dst(dst, DataType::F)
                    .with_src0(src, DataType::F)
                    .with_exec_size(ExecSize::Scalar)
                    .with_send_descriptor(descriptor);
                
                self.instructions.push(inst);
                Ok(())
            }
            MathFunction::InverseSqrt => {
                // InverseSqrt: Reciprocal square root (1/sqrt(x))
                // Implemented via SEND to math function unit with rsqrt operation
                // This is typically faster than sqrt on Intel GPUs
                
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;
                
                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let src = Register {
                    file: RegFile::GRF,
                    num: src_phys.grf_num,
                    subreg: 0,
                };
                let dst = Register {
                    file: RegFile::GRF,
                    num: dst_phys.grf_num,
                    subreg: 0,
                };
                
                // Create SEND descriptor for math function (rsqrt)
                // Math function control: 0x2 = rsqrt (reciprocal sqrt)
                let descriptor = SendDescriptor {
                    sfid: SharedFunctionID::Math,
                    response_length: 1,  // 1 GRF register back
                    message_length: 1,   // 1 GRF register sent
                    function_control: 0x2,  // Rsqrt operation
                };
                
                let inst = EUInstruction::new(EUOpcode::Send)
                    .with_dst(dst, DataType::F)
                    .with_src0(src, DataType::F)
                    .with_exec_size(ExecSize::Scalar)
                    .with_send_descriptor(descriptor);
                
                self.instructions.push(inst);
                Ok(())
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
            MathFunction::Clamp => {
                // Clamp: clamp(x, min_val, max_val) = max(min_val, min(x, max_val))
                // Multi-instruction sequence:
                // 1. tmp = min(x, max_val)   (use SEL with conditional)
                // 2. result = max(min_val, tmp)  (use SEL with conditional)
                
                let x = arg;
                let min_val = arg1.ok_or_else(|| EUCompileError::from("Clamp requires arg1 (min_val)"))?;
                let max_val = arg2.ok_or_else(|| EUCompileError::from("Clamp requires arg2 (max_val)"))?;
                
                let x_reg = self.get_or_alloc_reg(x)?;
                let min_reg = self.get_or_alloc_reg(min_val)?;
                let max_reg = self.get_or_alloc_reg(max_val)?;
                let dst_reg = self.alloc_reg(result)?;
                
                // Allocate temporary register for intermediate min result
                let tmp_vreg = self.reg_alloc.allocate_vreg();
                
                // Get physical registers
                let x_phys = self.reg_alloc.get_physical(x_reg)
                    .ok_or_else(|| EUCompileError::from("X not allocated"))?;
                let min_phys = self.reg_alloc.get_physical(min_reg)
                    .ok_or_else(|| EUCompileError::from("Min not allocated"))?;
                let max_phys = self.reg_alloc.get_physical(max_reg)
                    .ok_or_else(|| EUCompileError::from("Max not allocated"))?;
                let tmp_phys = self.reg_alloc.get_physical(tmp_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let x_r = Register { file: RegFile::GRF, num: x_phys.grf_num, subreg: 0 };
                let min_r = Register { file: RegFile::GRF, num: min_phys.grf_num, subreg: 0 };
                let max_r = Register { file: RegFile::GRF, num: max_phys.grf_num, subreg: 0 };
                let tmp_r = Register { file: RegFile::GRF, num: tmp_phys.grf_num, subreg: 0 };
                let dst_r = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                
                // Step 1: tmp = min(x, max_val)  (select x if x < max_val, else max_val)
                let mut inst1 = EUInstruction::new(EUOpcode::Sel);
                inst1.set_dst(tmp_r);
                inst1.set_src0(x_r);
                inst1.set_src1(max_r);
                inst1.set_cond_mod(CondMod::L); // Less than - select src0 if src0 < src1
                inst1.set_exec_size(ExecSize::Scalar);
                inst1.set_dst_type(DataType::F);
                inst1.set_src0_type(DataType::F);
                inst1.set_src1_type(DataType::F);
                self.instructions.push(inst1);
                
                // Step 2: result = max(min_val, tmp)  (select tmp if tmp > min_val, else min_val)
                let mut inst2 = EUInstruction::new(EUOpcode::Sel);
                inst2.set_dst(dst_r);
                inst2.set_src0(tmp_r);
                inst2.set_src1(min_r);
                inst2.set_cond_mod(CondMod::G); // Greater than - select src0 if src0 > src1
                inst2.set_exec_size(ExecSize::Scalar);
                inst2.set_dst_type(DataType::F);
                inst2.set_src0_type(DataType::F);
                inst2.set_src1_type(DataType::F);
                self.instructions.push(inst2);
                
                Ok(())
            }
            MathFunction::SmoothStep => {
                // SmoothStep: smoothstep(edge0, edge1, x)
                // Formula: t^2 * (3 - 2*t) where t = clamp((x - edge0) / (edge1 - edge0), 0, 1)
                // 
                // Multi-instruction sequence:
                // 1. tmp1 = edge1 - edge0         (denominator)
                // 2. tmp2 = x - edge0             (numerator)
                // 3. tmp3 = tmp2 / tmp1           (t unclamped)
                // 4. tmp4 = clamp(tmp3, 0, 1)     (t clamped)
                // 5. tmp5 = 2 * tmp4              (2*t)
                // 6. tmp6 = 3 - tmp5              (3 - 2*t)
                // 7. tmp7 = tmp4 * tmp4           (t^2)
                // 8. result = tmp7 * tmp6         (t^2 * (3 - 2*t))
                
                let edge0 = arg;
                let edge1 = arg1.ok_or_else(|| EUCompileError::from("SmoothStep requires arg1 (edge1)"))?;
                let x = arg2.ok_or_else(|| EUCompileError::from("SmoothStep requires arg2 (x)"))?;
                
                let edge0_reg = self.get_or_alloc_reg(edge0)?;
                let edge1_reg = self.get_or_alloc_reg(edge1)?;
                let x_reg = self.get_or_alloc_reg(x)?;
                let dst_reg = self.alloc_reg(result)?;
                
                // Allocate temporary registers
                let tmp1_vreg = self.reg_alloc.allocate_vreg(); // edge1 - edge0
                let tmp2_vreg = self.reg_alloc.allocate_vreg(); // x - edge0
                let tmp3_vreg = self.reg_alloc.allocate_vreg(); // t unclamped
                let tmp4_vreg = self.reg_alloc.allocate_vreg(); // t clamped
                let tmp5_vreg = self.reg_alloc.allocate_vreg(); // 2*t
                let tmp6_vreg = self.reg_alloc.allocate_vreg(); // 3 - 2*t
                let tmp7_vreg = self.reg_alloc.allocate_vreg(); // t^2
                
                // Get physical registers
                let edge0_phys = self.reg_alloc.get_physical(edge0_reg)
                    .ok_or_else(|| EUCompileError::from("Edge0 not allocated"))?;
                let edge1_phys = self.reg_alloc.get_physical(edge1_reg)
                    .ok_or_else(|| EUCompileError::from("Edge1 not allocated"))?;
                let x_phys = self.reg_alloc.get_physical(x_reg)
                    .ok_or_else(|| EUCompileError::from("X not allocated"))?;
                let tmp1_phys = self.reg_alloc.get_physical(tmp1_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp1 not allocated"))?;
                let tmp2_phys = self.reg_alloc.get_physical(tmp2_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp2 not allocated"))?;
                let tmp3_phys = self.reg_alloc.get_physical(tmp3_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp3 not allocated"))?;
                let tmp4_phys = self.reg_alloc.get_physical(tmp4_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp4 not allocated"))?;
                let tmp5_phys = self.reg_alloc.get_physical(tmp5_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp5 not allocated"))?;
                let tmp6_phys = self.reg_alloc.get_physical(tmp6_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp6 not allocated"))?;
                let tmp7_phys = self.reg_alloc.get_physical(tmp7_vreg)
                    .ok_or_else(|| EUCompileError::from("Tmp7 not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let edge0_r = Register { file: RegFile::GRF, num: edge0_phys.grf_num, subreg: 0 };
                let edge1_r = Register { file: RegFile::GRF, num: edge1_phys.grf_num, subreg: 0 };
                let x_r = Register { file: RegFile::GRF, num: x_phys.grf_num, subreg: 0 };
                let tmp1_r = Register { file: RegFile::GRF, num: tmp1_phys.grf_num, subreg: 0 };
                let tmp2_r = Register { file: RegFile::GRF, num: tmp2_phys.grf_num, subreg: 0 };
                let tmp3_r = Register { file: RegFile::GRF, num: tmp3_phys.grf_num, subreg: 0 };
                let tmp4_r = Register { file: RegFile::GRF, num: tmp4_phys.grf_num, subreg: 0 };
                let tmp5_r = Register { file: RegFile::GRF, num: tmp5_phys.grf_num, subreg: 0 };
                let tmp6_r = Register { file: RegFile::GRF, num: tmp6_phys.grf_num, subreg: 0 };
                let tmp7_r = Register { file: RegFile::GRF, num: tmp7_phys.grf_num, subreg: 0 };
                let dst_r = Register { file: RegFile::GRF, num: dst_phys.grf_num, subreg: 0 };
                
                // Step 1: tmp1 = edge1 - edge0 (using ADD with negate)
                let mut inst1 = EUInstruction::new(EUOpcode::Add);
                inst1.set_dst(tmp1_r);
                inst1.set_src0(edge1_r);
                inst1.set_src1(edge0_r);
                inst1.set_src1_negate(true);
                inst1.set_exec_size(ExecSize::Scalar);
                inst1.set_dst_type(DataType::F);
                inst1.set_src0_type(DataType::F);
                inst1.set_src1_type(DataType::F);
                self.instructions.push(inst1);
                
                // Step 2: tmp2 = x - edge0
                let mut inst2 = EUInstruction::new(EUOpcode::Add);
                inst2.set_dst(tmp2_r);
                inst2.set_src0(x_r);
                inst2.set_src1(edge0_r);
                inst2.set_src1_negate(true);
                inst2.set_exec_size(ExecSize::Scalar);
                inst2.set_dst_type(DataType::F);
                inst2.set_src0_type(DataType::F);
                inst2.set_src1_type(DataType::F);
                self.instructions.push(inst2);
                
                // Step 3: tmp3 = tmp2 / tmp1 (using MATH instruction for divide)
                // For now, use a simplified DIV approximation via multiply by reciprocal
                // Real implementation would use MATH shared function
                let mut inst3 = EUInstruction::new(EUOpcode::Mul);
                inst3.set_dst(tmp3_r);
                inst3.set_src0(tmp2_r);
                inst3.set_src1(tmp1_r); // This should be 1/tmp1, simplified for now
                inst3.set_exec_size(ExecSize::Scalar);
                inst3.set_dst_type(DataType::F);
                inst3.set_src0_type(DataType::F);
                inst3.set_src1_type(DataType::F);
                self.instructions.push(inst3);
                
                // Step 4: tmp4 = clamp(tmp3, 0, 1)
                // Implement inline clamp: max(0, min(tmp3, 1))
                // For simplification, we'll use SEL instructions
                // First: select min(tmp3, 1) - this requires a constant 1
                // Second: select max(result, 0) - this requires a constant 0
                // Simplified: Just use tmp3 for now (proper implementation needs constant support)
                let mut inst4 = EUInstruction::new(EUOpcode::Mov);
                inst4.set_dst(tmp4_r);
                inst4.set_src0(tmp3_r); // Simplified - proper clamp needs constants
                inst4.set_exec_size(ExecSize::Scalar);
                inst4.set_dst_type(DataType::F);
                inst4.set_src0_type(DataType::F);
                self.instructions.push(inst4);
                
                // Step 5: tmp5 = 2 * tmp4
                // Needs constant 2 - simplified
                let mut inst5 = EUInstruction::new(EUOpcode::Mul);
                inst5.set_dst(tmp5_r);
                inst5.set_src0(tmp4_r);
                inst5.set_src1(tmp4_r); // Should be constant 2, using tmp4 as placeholder
                inst5.set_exec_size(ExecSize::Scalar);
                inst5.set_dst_type(DataType::F);
                inst5.set_src0_type(DataType::F);
                inst5.set_src1_type(DataType::F);
                self.instructions.push(inst5);
                
                // Step 6: tmp6 = 3 - tmp5
                // Needs constant 3 - simplified
                let mut inst6 = EUInstruction::new(EUOpcode::Add);
                inst6.set_dst(tmp6_r);
                inst6.set_src0(tmp5_r); // Should be 3, using tmp5 as placeholder
                inst6.set_src1(tmp5_r);
                inst6.set_src1_negate(true);
                inst6.set_exec_size(ExecSize::Scalar);
                inst6.set_dst_type(DataType::F);
                inst6.set_src0_type(DataType::F);
                inst6.set_src1_type(DataType::F);
                self.instructions.push(inst6);
                
                // Step 7: tmp7 = tmp4 * tmp4 (t^2)
                let mut inst7 = EUInstruction::new(EUOpcode::Mul);
                inst7.set_dst(tmp7_r);
                inst7.set_src0(tmp4_r);
                inst7.set_src1(tmp4_r);
                inst7.set_exec_size(ExecSize::Scalar);
                inst7.set_dst_type(DataType::F);
                inst7.set_src0_type(DataType::F);
                inst7.set_src1_type(DataType::F);
                self.instructions.push(inst7);
                
                // Step 8: result = tmp7 * tmp6 (t^2 * (3 - 2*t))
                let mut inst8 = EUInstruction::new(EUOpcode::Mul);
                inst8.set_dst(dst_r);
                inst8.set_src0(tmp7_r);
                inst8.set_src1(tmp6_r);
                inst8.set_exec_size(ExecSize::Scalar);
                inst8.set_dst_type(DataType::F);
                inst8.set_src0_type(DataType::F);
                inst8.set_src1_type(DataType::F);
                self.instructions.push(inst8);
                
                Ok(())
            }
            MathFunction::Length => {
                // Length: length(v) = sqrt(dot(v, v))
                // For a vector, compute sqrt of sum of squared components
                // For scalar, just return absolute value
                //
                // Multi-instruction sequence (for vec2):
                // 1. tmp1 = v.x * v.x
                // 2. tmp2 = v.y * v.y
                // 3. tmp3 = tmp1 + tmp2
                // 4. result = sqrt(tmp3)
                //
                // For now, simplified scalar implementation: result = abs(arg)
                
                let src_reg = self.get_or_alloc_reg(arg)?;
                let dst_reg = self.alloc_reg(result)?;
                
                let src_phys = self.reg_alloc.get_physical(src_reg)
                    .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let src = Register {
                    file: RegFile::GRF,
                    num: src_phys.grf_num,
                    subreg: 0,
                };
                let dst = Register {
                    file: RegFile::GRF,
                    num: dst_phys.grf_num,
                    subreg: 0,
                };
                
                // Simplified scalar length: just absolute value
                // Real vector implementation would compute sqrt(x^2 + y^2 + z^2 + w^2)
                let mut inst = EUInstruction::new(EUOpcode::Mov);
                inst.set_dst(dst);
                inst.set_src0(src);
                inst.set_src0_absolute(true);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                self.instructions.push(inst);
                
                Ok(())
            }
            MathFunction::Dot => {
                // Dot product: dot(v1, v2)
                // For vec2: v1.x * v2.x + v1.y * v2.y
                // For vec3: v1.x * v2.x + v1.y * v2.y + v1.z * v2.z
                // For vec4: v1.x * v2.x + v1.y * v2.y + v1.z * v2.z + v1.w * v2.w
                //
                // Simplified scalar implementation: v1 * v2
                
                let arg1 = arg1.ok_or_else(|| EUCompileError::from("Dot requires two arguments"))?;
                
                let src0_reg = self.get_or_alloc_reg(arg)?;
                let src1_reg = self.get_or_alloc_reg(arg1)?;
                let dst_reg = self.alloc_reg(result)?;
                
                let src0_phys = self.reg_alloc.get_physical(src0_reg)
                    .ok_or_else(|| EUCompileError::from("Source 0 not allocated"))?;
                let src1_phys = self.reg_alloc.get_physical(src1_reg)
                    .ok_or_else(|| EUCompileError::from("Source 1 not allocated"))?;
                let dst_phys = self.reg_alloc.get_physical(dst_reg)
                    .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
                
                let src0 = Register {
                    file: RegFile::GRF,
                    num: src0_phys.grf_num,
                    subreg: 0,
                };
                let src1 = Register {
                    file: RegFile::GRF,
                    num: src1_phys.grf_num,
                    subreg: 0,
                };
                let dst = Register {
                    file: RegFile::GRF,
                    num: dst_phys.grf_num,
                    subreg: 0,
                };
                
                // Simplified scalar dot product: just multiply
                // Real vector implementation would use DP2/DP3/DP4 instructions or MAD sequence
                let mut inst = EUInstruction::new(EUOpcode::Mul);
                inst.set_dst(dst);
                inst.set_src0(src0);
                inst.set_src1(src1);
                inst.set_exec_size(ExecSize::Scalar);
                inst.set_dst_type(DataType::F);
                inst.set_src0_type(DataType::F);
                inst.set_src1_type(DataType::F);
                self.instructions.push(inst);
                
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

    /// Lower image/texture sampling operation
    ///
    /// Generates SEND instruction to sampler shared function for texture reads
    /// This is the core operation for reading from textures in shaders.
    ///
    /// Arguments:
    /// - image: Handle to the image/texture resource
    /// - sampler: Handle to the sampler state
    /// - coordinate: Handle to texture coordinates (vec2/vec3/vec4)
    /// - result: Handle to store the sampled color value
    pub fn lower_image_sample(
        &mut self,
        _image: naga::Handle<Expression>,
        _sampler: naga::Handle<Expression>,
        coordinate: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        // Texture sampling via SEND to sampler shared function (SFID 0x2)
        //
        // Algorithm:
        // 1. Pack texture coordinates into GRF message payload
        // 2. SEND with sampler message descriptor
        // 3. Sampler returns RGBA color in destination GRF
        //
        // Message descriptor format for sampler:
        // - Function control bits specify: texture type (2D/3D/cube),
        //   filtering mode (bilinear/trilinear), and message type (sample)
        //
        // Reference: Intel PRM Vol 4, Part 1, Section "Sampler Messages"
        
        let coord_reg = self.get_or_alloc_reg(coordinate)?;
        let dst_reg = self.alloc_reg(result)?;
        
        let coord_phys = self.reg_alloc.get_physical(coord_reg)
            .ok_or_else(|| EUCompileError::from("Coordinate not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;
        
        let coord = Register {
            file: RegFile::GRF,
            num: coord_phys.grf_num,
            subreg: 0,
        };
        let dst = Register {
            file: RegFile::GRF,
            num: dst_phys.grf_num,
            subreg: 0,
        };
        
        // Create SEND descriptor for sampler
        // Function control for 2D texture sample with bilinear filtering:
        // Bits 0-4: Message type (0x0 = sample)
        // Bits 5-8: Sampler index (0)
        // Bits 9-12: Binding table index (0 for now)
        // Bit 13: SIMD mode (0 = SIMD8, 1 = SIMD16)
        let descriptor = SendDescriptor {
            sfid: SharedFunctionID::Sampler,
            response_length: 4,  // 4 GRF registers back (RGBA, 32-bit float each)
            message_length: 2,   // 2 GRF registers sent (U, V coordinates)
            function_control: 0x0000,  // Sample operation, sampler 0, binding 0
        };
        
        let inst = EUInstruction::new(EUOpcode::Send)
            .with_dst(dst, DataType::F)
            .with_src0(coord, DataType::F)
            .with_exec_size(ExecSize::Size8)  // SIMD8 for texture sampling
            .with_send_descriptor(descriptor);
        
        self.instructions.push(inst);
        Ok(())
    }

    /// Emit render target write operation (fragment shader output)
    ///
    /// Generates SEND instruction to write color to render target.
    /// Called at shader end with the final output color value.
    pub fn emit_render_target_write(
        &mut self,
        value_reg: VirtualReg,
    ) -> Result<(), EUCompileError> {
        // Get physical register containing the color value
        let src_phys = self.reg_alloc.get_physical(value_reg)
            .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
        let src = Register {
            file: RegFile::GRF,
            num: src_phys.grf_num,
            subreg: 0,
        };
        
        // Render target write message descriptor:
        // - SFID: RenderTargetCache (0x5)
        // - Response length: 0 (write, no response)
        // - Message length: 4 (RGBA, 4 GRF registers)
        // - Function control: RT write message type (0xc = SIMD8 RT write)
        let descriptor = SendDescriptor {
            sfid: SharedFunctionID::RenderTargetCache,
            response_length: 0,  // No response for write
            message_length: 4,   // Send 4 GRF registers (RGBA)
            function_control: 0xc,  // SIMD8 RT write, RT index 0
        };
        
        // Null destination for write-only operation
        let dst = Register {
            file: RegFile::ARF,
            num: 0,  // Null register
            subreg: 0,
        };
        
        let inst = EUInstruction::new(EUOpcode::Send)
            .with_dst(dst, DataType::F)
            .with_src0(src, DataType::F)
            .with_exec_size(ExecSize::Size8)
            .with_send_descriptor(descriptor);
        
        self.instructions.push(inst);
        Ok(())
    }

    /// Emit URB write operation (vertex shader output)
    ///
    /// Generates SEND instruction to write vertex outputs to URB.
    /// Called at shader end with final output values.
    pub fn emit_urb_write(
        &mut self,
        location: u32,
        value_reg: VirtualReg,
    ) -> Result<(), EUCompileError> {
        // Get physical register containing the value
        let src_phys = self.reg_alloc.get_physical(value_reg)
            .ok_or_else(|| EUCompileError::from("Source not allocated"))?;
        let src = Register {
            file: RegFile::GRF,
            num: src_phys.grf_num,
            subreg: 0,
        };
        
        // URB write message descriptor:
        // - Response length: 0 (write, no response)
        // - Message length: 4 (vec4, 4 GRF registers)
        // - Function control: URB write message, offset based on location
        let descriptor = SendDescriptor {
            sfid: SharedFunctionID::URB,
            response_length: 0,  // No response for write
            message_length: 4,   // Send 4 GRF registers (vec4)
            function_control: (location << 4) | 0x7,  // URB write opcode
        };
        
        // Destination is null for write-only
        let dst = Register {
            file: RegFile::ARF,
            num: 0,  // Null register
            subreg: 0,
        };
        
        let inst = EUInstruction::new(EUOpcode::Send)
            .with_dst(dst, DataType::F)
            .with_src0(src, DataType::F)
            .with_exec_size(ExecSize::Size8)
            .with_send_descriptor(descriptor);
        
        self.instructions.push(inst);
        Ok(())
    }

    /// Type conversion (cast) operation
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

    /// Lower vector splat operation (broadcast scalar to all components)
    ///
    /// Generates MOV instruction with scalar source and vector destination
    pub fn lower_splat(
        &mut self,
        scalar: naga::Handle<Expression>,
        size: naga::VectorSize,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        use naga::VectorSize;

        let scalar_reg = self.get_or_alloc_reg(scalar)?;
        let dst_reg = self.alloc_reg(result)?;

        let scalar_phys = self.reg_alloc.get_physical(scalar_reg)
            .ok_or_else(|| EUCompileError::from("Scalar operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        let src = Register {
            file: RegFile::GRF,
            num: scalar_phys.grf_num,
            subreg: 0,
        };
        let dst = Register {
            file: RegFile::GRF,
            num: dst_phys.grf_num,
            subreg: 0,
        };

        let exec_size = match size {
            VectorSize::Bi => ExecSize::Size2,
            VectorSize::Tri | VectorSize::Quad => ExecSize::Size4,
        };

        let mut mov = EUInstruction::new(EUOpcode::Mov);
        mov.set_dst(dst);
        mov.set_dst_type(DataType::F);
        mov.set_src0(src);
        mov.set_src0_type(DataType::F);
        mov.set_exec_size(exec_size);
        self.instructions.push(mov);

        Ok(())
    }

    /// Lower vector swizzle operation
    ///
    /// Extracts and rearranges vector components using MOV with region addressing
    pub fn lower_swizzle(
        &mut self,
        vector: naga::Handle<Expression>,
        pattern: &[naga::SwizzleComponent; 4],
        size: naga::VectorSize,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        use naga::{SwizzleComponent, VectorSize};

        let vec_reg = self.get_or_alloc_reg(vector)?;
        let dst_reg = self.alloc_reg(result)?;

        let vec_phys = self.reg_alloc.get_physical(vec_reg)
            .ok_or_else(|| EUCompileError::from("Vector operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        let num_components = match size {
            VectorSize::Bi => 2,
            VectorSize::Tri => 3,
            VectorSize::Quad => 4,
        };

        // For simple swizzles, we can use a single MOV with swizzle control
        // For complex swizzles, we need multiple MOV instructions
        // TODO: Implement swizzle control bits in instruction encoding
        // For now, implement component-wise with multiple MOVs

        for i in 0..num_components {
            let component_idx = match pattern[i] {
                SwizzleComponent::X => 0,
                SwizzleComponent::Y => 1,
                SwizzleComponent::Z => 2,
                SwizzleComponent::W => 3,
            };

            let src = Register {
                file: RegFile::GRF,
                num: vec_phys.grf_num,
                subreg: (component_idx * 4) as u8, // Each float component is 4 bytes
            };
            let dst = Register {
                file: RegFile::GRF,
                num: dst_phys.grf_num,
                subreg: (i * 4) as u8,
            };

            let mut mov = EUInstruction::new(EUOpcode::Mov);
            mov.set_dst(dst);
            mov.set_dst_type(DataType::F);
            mov.set_src0(src);
            mov.set_src0_type(DataType::F);
            mov.set_exec_size(ExecSize::Scalar);
            self.instructions.push(mov);
        }

        Ok(())
    }

    /// Lower dot product operation
    ///
    /// Uses DP2, DP3, or DP4 instructions based on vector size
    pub fn lower_dot(
        &mut self,
        left: naga::Handle<Expression>,
        right: naga::Handle<Expression>,
        size: naga::VectorSize,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        use naga::VectorSize;

        let left_reg = self.get_or_alloc_reg(left)?;
        let right_reg = self.get_or_alloc_reg(right)?;
        let dst_reg = self.alloc_reg(result)?;

        let left_phys = self.reg_alloc.get_physical(left_reg)
            .ok_or_else(|| EUCompileError::from("Left operand not allocated"))?;
        let right_phys = self.reg_alloc.get_physical(right_reg)
            .ok_or_else(|| EUCompileError::from("Right operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

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

        // Use DP2, DP3, or DP4 instruction based on vector size
        let opcode = match size {
            VectorSize::Bi => EUOpcode::Dp2,
            VectorSize::Tri => EUOpcode::Dp3,
            VectorSize::Quad => EUOpcode::Dp4,
        };

        let mut dp = EUInstruction::new(opcode);
        dp.set_dst(dst);
        dp.set_dst_type(DataType::F);
        dp.set_src0(src0);
        dp.set_src0_type(DataType::F);
        dp.set_src1(src1);
        dp.set_src1_type(DataType::F);
        dp.set_exec_size(ExecSize::Scalar); // Dot product result is scalar
        self.instructions.push(dp);

        Ok(())
    }

    /// Lower cross product operation (vec3 only)
    ///
    /// Cross product: (a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x)
    pub fn lower_cross(
        &mut self,
        left: naga::Handle<Expression>,
        right: naga::Handle<Expression>,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        let left_reg = self.get_or_alloc_reg(left)?;
        let right_reg = self.get_or_alloc_reg(right)?;
        let dst_reg = self.alloc_reg(result)?;

        let left_phys = self.reg_alloc.get_physical(left_reg)
            .ok_or_else(|| EUCompileError::from("Left operand not allocated"))?;
        let right_phys = self.reg_alloc.get_physical(right_reg)
            .ok_or_else(|| EUCompileError::from("Right operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

        // Allocate temporary registers for intermediate results
        let temp1_vreg = self.reg_alloc.allocate_vreg();
        let temp2_vreg = self.reg_alloc.allocate_vreg();
        let temp1_phys = self.reg_alloc.get_physical(temp1_vreg)
            .ok_or_else(|| EUCompileError::from("Temp1 not allocated"))?;
        let temp2_phys = self.reg_alloc.get_physical(temp2_vreg)
            .ok_or_else(|| EUCompileError::from("Temp2 not allocated"))?;

        // Compute each component of the cross product
        // x = a.y * b.z - a.z * b.y
        // y = a.z * b.x - a.x * b.z
        // z = a.x * b.y - a.y * b.x

        for i in 0..3 {
            let (idx1, idx2) = match i {
                0 => (1, 2), // x: a.y, a.z
                1 => (2, 0), // y: a.z, a.x
                2 => (0, 1), // z: a.x, a.y
                _ => unreachable!(),
            };

            // temp1 = a[idx1] * b[idx2]
            let mut mul1 = EUInstruction::new(EUOpcode::Mul);
            mul1.set_dst(Register {
                file: RegFile::GRF,
                num: temp1_phys.grf_num,
                subreg: 0,
            });
            mul1.set_dst_type(DataType::F);
            mul1.set_src0(Register {
                file: RegFile::GRF,
                num: left_phys.grf_num,
                subreg: (idx1 * 4) as u8,
            });
            mul1.set_src0_type(DataType::F);
            mul1.set_src1(Register {
                file: RegFile::GRF,
                num: right_phys.grf_num,
                subreg: (idx2 * 4) as u8,
            });
            mul1.set_src1_type(DataType::F);
            mul1.set_exec_size(ExecSize::Scalar);
            self.instructions.push(mul1);

            // temp2 = a[idx2] * b[idx1]
            let mut mul2 = EUInstruction::new(EUOpcode::Mul);
            mul2.set_dst(Register {
                file: RegFile::GRF,
                num: temp2_phys.grf_num,
                subreg: 0,
            });
            mul2.set_dst_type(DataType::F);
            mul2.set_src0(Register {
                file: RegFile::GRF,
                num: left_phys.grf_num,
                subreg: (idx2 * 4) as u8,
            });
            mul2.set_src0_type(DataType::F);
            mul2.set_src1(Register {
                file: RegFile::GRF,
                num: right_phys.grf_num,
                subreg: (idx1 * 4) as u8,
            });
            mul2.set_src1_type(DataType::F);
            mul2.set_exec_size(ExecSize::Scalar);
            self.instructions.push(mul2);

            // dst[i] = temp1 - temp2
            let mut add = EUInstruction::new(EUOpcode::Add);
            add.set_dst(Register {
                file: RegFile::GRF,
                num: dst_phys.grf_num,
                subreg: (i * 4) as u8,
            });
            add.set_dst_type(DataType::F);
            add.set_src0(Register {
                file: RegFile::GRF,
                num: temp1_phys.grf_num,
                subreg: 0,
            });
            add.set_src0_type(DataType::F);
            add.set_src1(Register {
                file: RegFile::GRF,
                num: temp2_phys.grf_num,
                subreg: 0,
            });
            add.set_src1_type(DataType::F);
            add.set_src1_negate(true); // Negate src1 to perform subtraction
            add.set_exec_size(ExecSize::Scalar);
            self.instructions.push(add);
        }

        Ok(())
    }

    /// Lower vector arithmetic with proper execution size
    ///
    /// Similar to lower_binary_arith but handles vector execution sizes
    pub fn lower_vector_arith(
        &mut self,
        op: BinaryOperator,
        left: naga::Handle<Expression>,
        right: naga::Handle<Expression>,
        size: naga::VectorSize,
        result: naga::Handle<Expression>,
    ) -> Result<(), EUCompileError> {
        use naga::VectorSize;

        let left_reg = self.get_or_alloc_reg(left)?;
        let right_reg = self.get_or_alloc_reg(right)?;
        let dst_reg = self.alloc_reg(result)?;

        let left_phys = self.reg_alloc.get_physical(left_reg)
            .ok_or_else(|| EUCompileError::from("Left operand not allocated"))?;
        let right_phys = self.reg_alloc.get_physical(right_reg)
            .ok_or_else(|| EUCompileError::from("Right operand not allocated"))?;
        let dst_phys = self.reg_alloc.get_physical(dst_reg)
            .ok_or_else(|| EUCompileError::from("Destination not allocated"))?;

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

        let exec_size = match size {
            VectorSize::Bi => ExecSize::Size2,
            VectorSize::Tri | VectorSize::Quad => ExecSize::Size4,
        };

        let opcode = match op {
            BinaryOperator::Add => EUOpcode::Add,
            BinaryOperator::Multiply => EUOpcode::Mul,
            BinaryOperator::Subtract => {
                // Subtract implemented as Add with negated source
                let mut add = EUInstruction::new(EUOpcode::Add);
                add.set_dst(dst);
                add.set_dst_type(DataType::F);
                add.set_src0(src0);
                add.set_src0_type(DataType::F);
                add.set_src1(src1);
                add.set_src1_type(DataType::F);
                add.set_src1_negate(true);
                add.set_exec_size(exec_size);
                self.instructions.push(add);
                return Ok(());
            }
            BinaryOperator::Divide => {
                return Err(EUCompileError::from(
                    "Vector division requires SEND instruction (deferred)"
                ));
            }
            _ => {
                return Err(EUCompileError::from(format!(
                    "Unsupported vector binary operation: {:?}",
                    op
                )));
            }
        };

        let mut inst = EUInstruction::new(opcode);
        inst.set_dst(dst);
        inst.set_dst_type(DataType::F);
        inst.set_src0(src0);
        inst.set_src0_type(DataType::F);
        inst.set_src1(src1);
        inst.set_src1_type(DataType::F);
        inst.set_exec_size(exec_size);
        self.instructions.push(inst);

        Ok(())
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
    fn test_lower_divide_implemented() {
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
        assert!(res.is_ok(), "Division lowering should succeed");
        
        // Division is implemented as reciprocal + multiply, so we should have 2 instructions
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 2, "Division should generate 2 instructions (SEND for reciprocal + MUL)");
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
    fn test_lower_vector_splat() {
        let mut module = naga::Module::default();
        
        let scalar = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(5.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Splat {
            size: naga::VectorSize::Quad,
            value: scalar,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_splat(scalar, naga::VectorSize::Quad, result);
        assert!(res.is_ok(), "Splat lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        // Verify it's a MOV instruction with vec4 execution size
        assert_eq!(instructions[0].opcode(), EUOpcode::Mov);
    }

    #[test]
    fn test_lower_dot_product_vec2() {
        let mut module = naga::Module::default();
        
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(2.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Add,  // Placeholder, actual dot product in naga is MathFunction
            left,
            right,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_dot(left, right, naga::VectorSize::Bi, result);
        assert!(res.is_ok(), "Dot product lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        assert_eq!(instructions[0].opcode(), EUOpcode::Dp2);
    }

    #[test]
    fn test_lower_dot_product_vec3() {
        let mut module = naga::Module::default();
        
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

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_dot(left, right, naga::VectorSize::Tri, result);
        assert!(res.is_ok(), "Dot product vec3 lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        assert_eq!(instructions[0].opcode(), EUOpcode::Dp3);
    }

    #[test]
    fn test_lower_dot_product_vec4() {
        let mut module = naga::Module::default();
        
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

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_dot(left, right, naga::VectorSize::Quad, result);
        assert!(res.is_ok(), "Dot product vec4 lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        assert_eq!(instructions[0].opcode(), EUOpcode::Dp4);
    }

    #[test]
    fn test_lower_cross_product() {
        let mut module = naga::Module::default();
        
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

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_cross(left, right, result);
        assert!(res.is_ok(), "Cross product lowering should succeed");

        let instructions = ctx.instructions();
        // Cross product generates 9 instructions: 3 * (mul, mul, add)
        assert_eq!(instructions.len(), 9);
    }

    #[test]
    fn test_lower_vector_add() {
        let mut module = naga::Module::default();
        
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

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_vector_arith(
            BinaryOperator::Add,
            left,
            right,
            naga::VectorSize::Quad,
            result
        );
        assert!(res.is_ok(), "Vector add should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        assert_eq!(instructions[0].opcode(), EUOpcode::Add);
    }

    #[test]
    fn test_lower_vector_multiply() {
        let mut module = naga::Module::default();
        
        let left = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(3.0)
        ), Default::default());
        let right = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(4.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Binary {
            op: BinaryOperator::Multiply,
            left,
            right,
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let res = ctx.lower_vector_arith(
            BinaryOperator::Multiply,
            left,
            right,
            naga::VectorSize::Tri,
            result
        );
        assert!(res.is_ok(), "Vector multiply should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1);
        assert_eq!(instructions[0].opcode(), EUOpcode::Mul);
    }

    #[test]
    fn test_lower_swizzle() {
        use naga::SwizzleComponent;
        
        let mut module = naga::Module::default();
        
        let vector = module.const_expressions.append(Expression::Literal(
            naga::Literal::F32(1.0)
        ), Default::default());
        let result = module.const_expressions.append(Expression::Swizzle {
            size: naga::VectorSize::Quad,
            vector,
            pattern: [
                SwizzleComponent::X,
                SwizzleComponent::Y,
                SwizzleComponent::Z,
                SwizzleComponent::W,
            ],
        }, Default::default());

        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);

        let pattern = [
            SwizzleComponent::Z,
            SwizzleComponent::Y,
            SwizzleComponent::X,
            SwizzleComponent::W,
        ];
        let res = ctx.lower_swizzle(vector, &pattern, naga::VectorSize::Quad, result);
        assert!(res.is_ok(), "Swizzle lowering should succeed");

        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 4); // 4 MOV instructions for vec4 swizzle
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
    fn test_lower_math_sqrt() {
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
        assert!(res.is_ok(), "Sqrt should succeed via SEND instruction");
        
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Sqrt should generate one SEND instruction");
    }

    #[test]
    fn test_lower_math_inverse_sqrt() {
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
        assert!(res.is_ok(), "InverseSqrt should succeed via SEND instruction");
        
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "InverseSqrt should generate one SEND instruction");
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

    #[test]
    fn test_emit_render_target_write() {
        // Test that we can emit a render target write instruction
        let module = naga::Module::default();
        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);
        
        // Allocate a virtual register
        let vreg = ctx.reg_alloc.allocate_vreg();
        
        // Emit RT write
        let result = ctx.emit_render_target_write(vreg);
        assert!(result.is_ok(), "Failed to emit render target write: {:?}", result.err());
        
        // Check that a SEND instruction was generated
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Should generate exactly one instruction");
        
        // Verify it's a Send instruction (format check)
        let inst_debug = format!("{:?}", instructions[0]);
        assert!(inst_debug.contains("Send"), "Expected Send instruction, got: {}", inst_debug);
    }

    #[test]
    fn test_emit_urb_write() {
        // Test that we can emit a URB write instruction for vertex shader outputs
        let module = naga::Module::default();
        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);
        
        // Allocate a virtual register
        let vreg = ctx.reg_alloc.allocate_vreg();
        
        // Emit URB write for location 0 (position)
        let result = ctx.emit_urb_write(0, vreg);
        assert!(result.is_ok(), "Failed to emit URB write: {:?}", result.err());
        
        // Check that a SEND instruction was generated
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 1, "Should generate exactly one instruction");
        
        // Verify it's a Send instruction
        let inst_debug = format!("{:?}", instructions[0]);
        assert!(inst_debug.contains("Send"), "Expected Send instruction, got: {}", inst_debug);
    }

    #[test]
    fn test_emit_multiple_urb_writes() {
        // Test emitting multiple URB writes (position + user outputs)
        let module = naga::Module::default();
        let mut ctx = LoweringContext::new(IntelGen::Gen9, &module);
        
        // Allocate registers for position and color output
        let position_vreg = ctx.reg_alloc.allocate_vreg();
        let color_vreg = ctx.reg_alloc.allocate_vreg();
        
        // Emit URB writes
        ctx.emit_urb_write(0, position_vreg).unwrap();  // Position at location 0
        ctx.emit_urb_write(1, color_vreg).unwrap();     // Color at location 1
        
        // Should have two SEND instructions
        let instructions = ctx.instructions();
        assert_eq!(instructions.len(), 2, "Should generate two URB write instructions");
    }
}
