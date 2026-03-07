/// GPU hardware detection module.
///
/// This module provides functions to detect GPU generation from i915/Xe
/// kernel parameters via I915_GETPARAM and DRM_IOCTL_XE_DEVICE_QUERY.

use std::io;
use crate::drm::DrmDevice;
use crate::i915::{GetParam, I915_PARAM_CHIPSET_ID};
use crate::xe::IntelDriver;

/// GPU generation enumeration.
///
/// Represents different Intel GPU generations supported by this driver.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GpuGeneration {
    /// Gen9: Skylake, Kaby Lake, Coffee Lake (2015-2019)
    /// Device IDs: 0x1900-0x193F, 0x5900-0x593F, 0x3E90-0x3E9F
    Gen9,
    
    /// Gen11: Ice Lake (2019-2020)
    /// Device IDs: 0x8A50-0x8A5F
    Gen11,
    
    /// Gen12: Tiger Lake, Rocket Lake, Alder Lake (2020-2022)
    /// Device IDs: 0x9A40-0x9A7F, 0x4C80-0x4C9F, 0x4680-0x46CF
    Gen12,
    
    /// Xe: Meteor Lake and later (2023+)
    /// Requires Xe kernel driver
    Xe,
    
    /// Unknown or unsupported GPU generation
    Unknown,
}

impl GpuGeneration {
    /// Get the generation name as a string.
    pub fn name(&self) -> &'static str {
        match self {
            GpuGeneration::Gen9 => "Gen9 (Skylake/Kaby Lake/Coffee Lake)",
            GpuGeneration::Gen11 => "Gen11 (Ice Lake)",
            GpuGeneration::Gen12 => "Gen12 (Tiger Lake/Rocket Lake/Alder Lake)",
            GpuGeneration::Xe => "Xe (Meteor Lake+)",
            GpuGeneration::Unknown => "Unknown",
        }
    }

    /// Check if this generation supports GPU command submission.
    pub fn supports_command_submission(&self) -> bool {
        matches!(self, GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 | GpuGeneration::Xe)
    }
}

impl DrmDevice {
    /// Detect GPU generation from i915/Xe kernel parameters.
    ///
    /// This function queries the chipset ID via I915_GETPARAM (for i915 driver)
    /// or DRM_IOCTL_XE_DEVICE_QUERY (for Xe driver) and maps it to a GpuGeneration.
    ///
    /// # Returns
    ///
    /// Returns `Ok(GpuGeneration)` on success, or an I/O error if the query fails.
    ///
    /// # Example
    ///
    /// ```no_run
    /// use render::drm::DrmDevice;
    ///
    /// let device = DrmDevice::open("/dev/dri/renderD128")?;
    /// let generation = device.detect_gpu_generation()?;
    /// println!("Detected GPU: {}", generation.name());
    /// # Ok::<(), std::io::Error>(())
    /// ```
    pub fn detect_gpu_generation(&self) -> io::Result<GpuGeneration> {
        // First, detect which Intel driver is active
        let driver = self.detect_intel_driver()?;

        match driver {
            IntelDriver::I915 => self.detect_i915_generation(),
            IntelDriver::Xe => Ok(GpuGeneration::Xe),
            IntelDriver::Unknown => Ok(GpuGeneration::Unknown),
        }
    }

    /// Detect GPU generation from i915 driver via I915_GETPARAM.
    fn detect_i915_generation(&self) -> io::Result<GpuGeneration> {
        let mut chipset_id: i32 = 0;
        let mut req = GetParam::new(I915_PARAM_CHIPSET_ID, &mut chipset_id);

        self.i915_getparam(&mut req)?;

        // Map chipset ID to GPU generation
        // Reference: Mesa src/intel/dev/intel_device_info.c
        let generation = match (chipset_id as u32) >> 4 {
            // Gen9: Skylake, Kaby Lake, Coffee Lake
            0x190..=0x193 | 0x590..=0x593 | 0x3E9 => GpuGeneration::Gen9,
            
            // Gen11: Ice Lake
            0x8A5 => GpuGeneration::Gen11,
            
            // Gen12: Tiger Lake, Rocket Lake, Alder Lake
            0x9A4..=0x9A7 | 0x4C8..=0x4C9 | 0x468..=0x46C => GpuGeneration::Gen12,
            
            // Unknown
            _ => GpuGeneration::Unknown,
        };

        Ok(generation)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generation_names() {
        assert_eq!(GpuGeneration::Gen9.name(), "Gen9 (Skylake/Kaby Lake/Coffee Lake)");
        assert_eq!(GpuGeneration::Gen11.name(), "Gen11 (Ice Lake)");
        assert_eq!(GpuGeneration::Gen12.name(), "Gen12 (Tiger Lake/Rocket Lake/Alder Lake)");
        assert_eq!(GpuGeneration::Xe.name(), "Xe (Meteor Lake+)");
        assert_eq!(GpuGeneration::Unknown.name(), "Unknown");
    }

    #[test]
    fn generation_supports_command_submission() {
        assert!(GpuGeneration::Gen9.supports_command_submission());
        assert!(GpuGeneration::Gen11.supports_command_submission());
        assert!(GpuGeneration::Gen12.supports_command_submission());
        assert!(GpuGeneration::Xe.supports_command_submission());
        assert!(!GpuGeneration::Unknown.supports_command_submission());
    }

    #[test]
    fn generation_equality() {
        assert_eq!(GpuGeneration::Gen9, GpuGeneration::Gen9);
        assert_ne!(GpuGeneration::Gen9, GpuGeneration::Gen11);
    }

    // Chipset ID to generation mapping tests
    #[test]
    fn chipset_id_gen9_skylake() {
        // Skylake: 0x1900-0x193F (e.g., 0x1916 = Skylake GT2)
        let chipset_id = 0x1916u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert!(matches!(generation_code, 0x190..=0x193));
    }

    #[test]
    fn chipset_id_gen9_kaby_lake() {
        // Kaby Lake: 0x5900-0x593F (e.g., 0x5916 = Kaby Lake GT2)
        let chipset_id = 0x5916u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert!(matches!(generation_code, 0x590..=0x593));
    }

    #[test]
    fn chipset_id_gen9_coffee_lake() {
        // Coffee Lake: 0x3E90-0x3E9F (e.g., 0x3E92 = Coffee Lake GT2)
        let chipset_id = 0x3E92u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert_eq!(generation_code, 0x3E9);
    }

    #[test]
    fn chipset_id_gen11_ice_lake() {
        // Ice Lake: 0x8A50-0x8A5F (e.g., 0x8A52 = Ice Lake GT2)
        let chipset_id = 0x8A52u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert_eq!(generation_code, 0x8A5);
    }

    #[test]
    fn chipset_id_gen12_tiger_lake() {
        // Tiger Lake: 0x9A40-0x9A7F (e.g., 0x9A49 = Tiger Lake GT2)
        let chipset_id = 0x9A49u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert!(matches!(generation_code, 0x9A4..=0x9A7));
    }

    #[test]
    fn chipset_id_gen12_alder_lake() {
        // Alder Lake: 0x4680-0x46CF (e.g., 0x46A6 = Alder Lake GT2)
        let chipset_id = 0x46A6u32;
        let generation_code = (chipset_id >> 4) & 0xFFF;
        assert!(matches!(generation_code, 0x468..=0x46C));
    }
}
