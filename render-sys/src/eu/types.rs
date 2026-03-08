// Intel EU Type System Integration - Phase 4.3
//
// This module provides type analysis and conversion utilities for mapping
// naga IR types to Intel EU register types and instruction data types.
//
// Reference: Intel PRMs Volume 4 (EU ISA), naga::Type documentation

use super::encoding::DataType;
use super::EUCompileError;
use naga::{ScalarKind, Type, TypeInner, VectorSize};

/// Type information for EU code generation
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EUTypeInfo {
    /// Base EU data type
    pub data_type: DataType,
    /// Number of components (1 for scalar, 2-4 for vectors)
    pub components: u8,
    /// Total size in bytes
    pub size_bytes: u32,
}

impl EUTypeInfo {
    /// Create scalar type info
    pub fn scalar(data_type: DataType) -> Self {
        EUTypeInfo {
            data_type,
            components: 1,
            size_bytes: data_type.size_bytes(),
        }
    }

    /// Create vector type info
    pub fn vector(data_type: DataType, components: u8) -> Self {
        EUTypeInfo {
            data_type,
            components,
            size_bytes: data_type.size_bytes() * components as u32,
        }
    }

    /// Check if this is a scalar type
    pub fn is_scalar(&self) -> bool {
        self.components == 1
    }

    /// Check if this is a vector type
    pub fn is_vector(&self) -> bool {
        self.components > 1
    }

    /// Get the number of GRF registers needed to hold this type
    pub fn num_registers(&self) -> u32 {
        // Each GRF is 32 bytes on Gen9+
        // Round up to next register boundary
        (self.size_bytes + 31) / 32
    }
}

/// Analyze a naga type and convert to EU type information
pub fn analyze_type(
    ty: &Type,
    types: &naga::UniqueArena<Type>,
) -> Result<EUTypeInfo, EUCompileError> {
    match &ty.inner {
        TypeInner::Scalar { kind, width } => {
            let data_type = scalar_to_data_type(*kind, *width)?;
            Ok(EUTypeInfo::scalar(data_type))
        }
        TypeInner::Vector { size, kind, width } => {
            let data_type = scalar_to_data_type(*kind, *width)?;
            let components = vector_size_to_components(*size);
            Ok(EUTypeInfo::vector(data_type, components))
        }
        TypeInner::Matrix {
            columns,
            rows,
            width,
        } => {
            // Matrix is treated as array of column vectors
            // For now, we only support float matrices
            let data_type = scalar_to_data_type(ScalarKind::Float, *width)?;
            let col_size = vector_size_to_components(*rows);
            let num_cols = vector_size_to_components(*columns);
            // Total components = columns * rows
            let total_components = col_size * num_cols;
            
            Ok(EUTypeInfo {
                data_type,
                components: total_components,
                size_bytes: data_type.size_bytes() * total_components as u32,
            })
        }
        TypeInner::Pointer { base, .. } => {
            // Recursively analyze the pointed-to type
            let base_ty = &types[*base];
            analyze_type(base_ty, types)
        }
        TypeInner::Array { base, size, .. } => {
            // For arrays, we need to know the size
            let base_ty = &types[*base];
            let base_info = analyze_type(base_ty, types)?;
            
            match size {
                naga::ArraySize::Constant(handle) => {
                    // For constant-sized arrays, we would need access to the constants arena
                    // For now, just return base type info
                    // Full implementation would multiply by array size
                    Ok(base_info)
                }
                naga::ArraySize::Dynamic => {
                    // Dynamic arrays are handled differently (runtime size)
                    Ok(base_info)
                }
            }
        }
        _ => {
            Err(EUCompileError::from(format!(
                "Unsupported type for EU compilation: {:?}",
                ty.inner
            )))
        }
    }
}

/// Convert naga scalar kind and width to EU data type
fn scalar_to_data_type(kind: ScalarKind, width: u8) -> Result<DataType, EUCompileError> {
    match (kind, width) {
        // Float types
        (ScalarKind::Float, 2) => Ok(DataType::HF),  // 16-bit half float
        (ScalarKind::Float, 4) => Ok(DataType::F),   // 32-bit float
        
        // Signed integer types
        (ScalarKind::Sint, 1) => Ok(DataType::B),    // 8-bit signed
        (ScalarKind::Sint, 2) => Ok(DataType::W),    // 16-bit signed
        (ScalarKind::Sint, 4) => Ok(DataType::D),    // 32-bit signed
        
        // Unsigned integer types
        (ScalarKind::Uint, 1) => Ok(DataType::UB),   // 8-bit unsigned
        (ScalarKind::Uint, 2) => Ok(DataType::UW),   // 16-bit unsigned
        (ScalarKind::Uint, 4) => Ok(DataType::UD),   // 32-bit unsigned
        
        // Bool is represented as 32-bit unsigned in EU
        (ScalarKind::Bool, _) => Ok(DataType::UD),
        
        _ => Err(EUCompileError::from(format!(
            "Unsupported scalar type: {:?} with width {}",
            kind, width
        ))),
    }
}

/// Convert naga vector size to number of components
fn vector_size_to_components(size: VectorSize) -> u8 {
    match size {
        VectorSize::Bi => 2,
        VectorSize::Tri => 3,
        VectorSize::Quad => 4,
    }
}

/// Type conversion operation kind
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TypeConversionKind {
    /// Float to signed integer (truncate)
    FloatToSint,
    /// Float to unsigned integer (truncate)
    FloatToUint,
    /// Signed integer to float
    SintToFloat,
    /// Unsigned integer to float
    UintToFloat,
    /// Signed integer widening (e.g., i16 -> i32)
    SintWiden,
    /// Unsigned integer widening (e.g., u16 -> u32)
    UintWiden,
    /// Signed integer narrowing with saturation
    SintNarrow,
    /// Unsigned integer narrowing with saturation
    UintNarrow,
    /// Bitcast (reinterpret bits without conversion)
    Bitcast,
}

impl TypeConversionKind {
    /// Determine conversion kind from source and destination types
    pub fn from_types(
        src_kind: ScalarKind,
        src_width: u8,
        dst_kind: ScalarKind,
        dst_width: u8,
    ) -> Result<Self, EUCompileError> {
        match (src_kind, dst_kind, src_width.cmp(&dst_width)) {
            // Float <-> Integer conversions
            (ScalarKind::Float, ScalarKind::Sint, _) => Ok(TypeConversionKind::FloatToSint),
            (ScalarKind::Float, ScalarKind::Uint, _) => Ok(TypeConversionKind::FloatToUint),
            (ScalarKind::Sint, ScalarKind::Float, _) => Ok(TypeConversionKind::SintToFloat),
            (ScalarKind::Uint, ScalarKind::Float, _) => Ok(TypeConversionKind::UintToFloat),
            
            // Integer widening
            (ScalarKind::Sint, ScalarKind::Sint, std::cmp::Ordering::Less) => {
                Ok(TypeConversionKind::SintWiden)
            }
            (ScalarKind::Uint, ScalarKind::Uint, std::cmp::Ordering::Less) => {
                Ok(TypeConversionKind::UintWiden)
            }
            
            // Integer narrowing
            (ScalarKind::Sint, ScalarKind::Sint, std::cmp::Ordering::Greater) => {
                Ok(TypeConversionKind::SintNarrow)
            }
            (ScalarKind::Uint, ScalarKind::Uint, std::cmp::Ordering::Greater) => {
                Ok(TypeConversionKind::UintNarrow)
            }
            
            // Same type and width - bitcast
            (_, _, std::cmp::Ordering::Equal) if src_kind == dst_kind => {
                Ok(TypeConversionKind::Bitcast)
            }
            
            _ => Err(EUCompileError::from(format!(
                "Unsupported type conversion: {:?}({}) -> {:?}({})",
                src_kind, src_width, dst_kind, dst_width
            ))),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use naga::{Type, TypeInner, VectorSize};

    #[test]
    fn test_scalar_type_analysis() {
        let types = naga::UniqueArena::new();
        
        // Test float32 scalar
        let f32_ty = Type {
            name: None,
            inner: TypeInner::Scalar {
                kind: ScalarKind::Float,
                width: 4,
            },
        };
        let info = analyze_type(&f32_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::F);
        assert_eq!(info.components, 1);
        assert_eq!(info.size_bytes, 4);
        assert!(info.is_scalar());
        assert!(!info.is_vector());
        
        // Test i32 scalar
        let i32_ty = Type {
            name: None,
            inner: TypeInner::Scalar {
                kind: ScalarKind::Sint,
                width: 4,
            },
        };
        let info = analyze_type(&i32_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::D);
        assert_eq!(info.components, 1);
        assert_eq!(info.size_bytes, 4);
        
        // Test u32 scalar
        let u32_ty = Type {
            name: None,
            inner: TypeInner::Scalar {
                kind: ScalarKind::Uint,
                width: 4,
            },
        };
        let info = analyze_type(&u32_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::UD);
        assert_eq!(info.components, 1);
        assert_eq!(info.size_bytes, 4);
    }

    #[test]
    fn test_vector_type_analysis() {
        let types = naga::UniqueArena::new();
        
        // Test vec2<f32>
        let vec2_ty = Type {
            name: None,
            inner: TypeInner::Vector {
                size: VectorSize::Bi,
                kind: ScalarKind::Float,
                width: 4,
            },
        };
        let info = analyze_type(&vec2_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::F);
        assert_eq!(info.components, 2);
        assert_eq!(info.size_bytes, 8);
        assert!(!info.is_scalar());
        assert!(info.is_vector());
        
        // Test vec4<f32>
        let vec4_ty = Type {
            name: None,
            inner: TypeInner::Vector {
                size: VectorSize::Quad,
                kind: ScalarKind::Float,
                width: 4,
            },
        };
        let info = analyze_type(&vec4_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::F);
        assert_eq!(info.components, 4);
        assert_eq!(info.size_bytes, 16);
    }

    #[test]
    fn test_type_conversion_kind() {
        // Float to int
        let kind = TypeConversionKind::from_types(
            ScalarKind::Float, 4,
            ScalarKind::Sint, 4,
        ).unwrap();
        assert_eq!(kind, TypeConversionKind::FloatToSint);
        
        // Int to float
        let kind = TypeConversionKind::from_types(
            ScalarKind::Uint, 4,
            ScalarKind::Float, 4,
        ).unwrap();
        assert_eq!(kind, TypeConversionKind::UintToFloat);
        
        // Integer widening
        let kind = TypeConversionKind::from_types(
            ScalarKind::Sint, 2,
            ScalarKind::Sint, 4,
        ).unwrap();
        assert_eq!(kind, TypeConversionKind::SintWiden);
        
        // Integer narrowing
        let kind = TypeConversionKind::from_types(
            ScalarKind::Uint, 4,
            ScalarKind::Uint, 2,
        ).unwrap();
        assert_eq!(kind, TypeConversionKind::UintNarrow);
    }

    #[test]
    fn test_half_float_support() {
        let types = naga::UniqueArena::new();
        
        // Test f16 (half float)
        let f16_ty = Type {
            name: None,
            inner: TypeInner::Scalar {
                kind: ScalarKind::Float,
                width: 2,
            },
        };
        let info = analyze_type(&f16_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::HF);
        assert_eq!(info.size_bytes, 2);
    }

    #[test]
    fn test_matrix_type_analysis() {
        let types = naga::UniqueArena::new();
        
        // Test mat4x4<f32>
        let mat4_ty = Type {
            name: None,
            inner: TypeInner::Matrix {
                columns: VectorSize::Quad,
                rows: VectorSize::Quad,
                width: 4,
            },
        };
        let info = analyze_type(&mat4_ty, &types).unwrap();
        assert_eq!(info.data_type, DataType::F);
        assert_eq!(info.components, 16); // 4x4 = 16
        assert_eq!(info.size_bytes, 64);  // 16 * 4 bytes
    }
}
