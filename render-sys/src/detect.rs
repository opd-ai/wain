/// GPU hardware detection module.
///
/// This module provides functions to detect GPU generation from i915/Xe/AMDGPU
/// kernel parameters via I915_GETPARAM, DRM_IOCTL_XE_DEVICE_QUERY, and AMDGPU_INFO.

use std::io;
use crate::drm::DrmDevice;
use crate::i915::{GetParam, I915_PARAM_CHIPSET_ID};
use crate::xe::IntelDriver;
use crate::amd::{DeviceInfo, GpuDevInfo, AMDGPU_INFO_DEV_INFO};

/// GPU generation enumeration.
///
/// Represents different GPU generations supported by this driver.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GpuGeneration {
    // Intel generations
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
    
    // AMD generations
    /// RDNA1: Navi 10/12/14 (RX 5000 series, 2019)
    /// Family ID: 143 (AMDGPU_FAMILY_NV)
    AmdRdna1,
    
    /// RDNA2: Navi 21/22/23/24 (RX 6000 series, Steam Deck, 2020-2022)
    /// Family IDs: 144 (Van Gogh/Steam Deck), 146 (Yellow Carp)
    AmdRdna2,
    
    /// RDNA3: Navi 31/32/33 (RX 7000 series, Phoenix APU, 2022+)
    /// Family IDs: 148 (GC 11.0.0), 149 (GC 11.0.1/Phoenix)
    AmdRdna3,
    
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
            GpuGeneration::AmdRdna1 => "AMD RDNA1 (RX 5000 series)",
            GpuGeneration::AmdRdna2 => "AMD RDNA2 (RX 6000 series, Steam Deck)",
            GpuGeneration::AmdRdna3 => "AMD RDNA3 (RX 7000 series, Phoenix)",
            GpuGeneration::Unknown => "Unknown",
        }
    }

    /// Check if this generation supports GPU command submission.
    pub fn supports_command_submission(&self) -> bool {
        !matches!(self, GpuGeneration::Unknown)
    }
    
    /// Check if this is an AMD GPU.
    pub fn is_amd(&self) -> bool {
        matches!(self, GpuGeneration::AmdRdna1 | GpuGeneration::AmdRdna2 | GpuGeneration::AmdRdna3)
    }
    
    /// Check if this is an Intel GPU.
    pub fn is_intel(&self) -> bool {
        matches!(self, GpuGeneration::Gen9 | GpuGeneration::Gen11 | GpuGeneration::Gen12 | GpuGeneration::Xe)
    }
}

impl DrmDevice {
    /// Detect GPU generation from i915/Xe/AMDGPU kernel parameters.
    ///
    /// This function queries the chipset ID via I915_GETPARAM (for i915 driver),
    /// DRM_IOCTL_XE_DEVICE_QUERY (for Xe driver), or AMDGPU_INFO (for AMDGPU driver)
    /// and maps it to a GpuGeneration.
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
        // Try Intel driver detection first
        let driver = self.detect_intel_driver()?;

        match driver {
            IntelDriver::I915 => self.detect_i915_generation(),
            IntelDriver::Xe => Ok(GpuGeneration::Xe),
            IntelDriver::Unknown => {
                // Try AMD driver detection
                self.detect_amd_generation()
            }
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
    
    /// Detect GPU generation from AMDGPU driver via AMDGPU_INFO.
    fn detect_amd_generation(&self) -> io::Result<GpuGeneration> {
        use crate::amd::{AMDGPU_FAMILY_NV, AMDGPU_FAMILY_VGH, AMDGPU_FAMILY_YC,
                         AMDGPU_FAMILY_GC_11_0_0, AMDGPU_FAMILY_GC_11_0_1};
        
        let mut dev_info = GpuDevInfo {
            device_id: 0,
            chip_rev: 0,
            external_rev: 0,
            pci_rev: 0,
            family: 0,
            num_shader_engines: 0,
            num_shader_arrays_per_engine: 0,
            gpu_counter_freq: 0,
            max_engine_clk: 0,
            max_memory_clk: 0,
            cu_active_number: 0,
            cu_ao_mask: 0,
            cu_bitmap: [[0; 4]; 4],
            enabled_rb_pipes_mask: 0,
            num_rb_pipes: 0,
            num_hw_gfx_contexts: 0,
            padding: 0,
            ids_flags: 0,
            virtual_address_offset: 0,
            virtual_address_max: 0,
            virtual_address_alignment: 0,
            pte_fragment_size: 0,
            gart_page_size: 0,
            ce_ram_size: 0,
            vram_type: 0,
            vram_bit_width: 0,
            vce_harvest_config: 0,
            gc_double_offchip_lds_buf: 0,
            prim_buf_gpu_addr: 0,
            pos_buf_gpu_addr: 0,
            cntl_sb_buf_gpu_addr: 0,
            param_buf_gpu_addr: 0,
            prim_buf_size: 0,
            pos_buf_size: 0,
            cntl_sb_buf_size: 0,
            param_buf_size: 0,
            wave_front_size: 0,
            num_shader_visible_vgprs: 0,
            num_cu_per_sh: 0,
            num_tcc_blocks: 0,
            gs_vgt_table_depth: 0,
            gs_prim_buffer_depth: 0,
            max_gs_waves_per_vgt: 0,
            padding2: 0,
            cu_ao_bitmap: [[0; 4]; 4],
            high_va_offset: 0,
            high_va_max: 0,
            pa_sc_tile_steering_override: 0,
            tcc_disabled_mask: 0,
        };

        let mut req = DeviceInfo::new(
            AMDGPU_INFO_DEV_INFO,
            &mut dev_info as *mut GpuDevInfo as u64,
            std::mem::size_of::<GpuDevInfo>() as u32
        );

        // Try AMDGPU_INFO query - if it fails, this is not an AMD GPU
        if self.amdgpu_info(&mut req).is_err() {
            return Ok(GpuGeneration::Unknown);
        }

        // Map family ID to GPU generation
        // Reference: Mesa src/amd/common/amd_family.h
        let generation = match dev_info.family {
            AMDGPU_FAMILY_NV => GpuGeneration::AmdRdna1,
            AMDGPU_FAMILY_VGH | AMDGPU_FAMILY_YC => GpuGeneration::AmdRdna2,
            AMDGPU_FAMILY_GC_11_0_0 | AMDGPU_FAMILY_GC_11_0_1 => GpuGeneration::AmdRdna3,
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
        assert_eq!(GpuGeneration::AmdRdna1.name(), "AMD RDNA1 (RX 5000 series)");
        assert_eq!(GpuGeneration::AmdRdna2.name(), "AMD RDNA2 (RX 6000 series, Steam Deck)");
        assert_eq!(GpuGeneration::AmdRdna3.name(), "AMD RDNA3 (RX 7000 series, Phoenix)");
        assert_eq!(GpuGeneration::Unknown.name(), "Unknown");
    }

    #[test]
    fn generation_supports_command_submission() {
        assert!(GpuGeneration::Gen9.supports_command_submission());
        assert!(GpuGeneration::Gen11.supports_command_submission());
        assert!(GpuGeneration::Gen12.supports_command_submission());
        assert!(GpuGeneration::Xe.supports_command_submission());
        assert!(GpuGeneration::AmdRdna1.supports_command_submission());
        assert!(GpuGeneration::AmdRdna2.supports_command_submission());
        assert!(GpuGeneration::AmdRdna3.supports_command_submission());
        assert!(!GpuGeneration::Unknown.supports_command_submission());
    }
    
    #[test]
    fn generation_is_amd() {
        assert!(!GpuGeneration::Gen9.is_amd());
        assert!(!GpuGeneration::Gen11.is_amd());
        assert!(!GpuGeneration::Gen12.is_amd());
        assert!(!GpuGeneration::Xe.is_amd());
        assert!(GpuGeneration::AmdRdna1.is_amd());
        assert!(GpuGeneration::AmdRdna2.is_amd());
        assert!(GpuGeneration::AmdRdna3.is_amd());
        assert!(!GpuGeneration::Unknown.is_amd());
    }
    
    #[test]
    fn generation_is_intel() {
        assert!(GpuGeneration::Gen9.is_intel());
        assert!(GpuGeneration::Gen11.is_intel());
        assert!(GpuGeneration::Gen12.is_intel());
        assert!(GpuGeneration::Xe.is_intel());
        assert!(!GpuGeneration::AmdRdna1.is_intel());
        assert!(!GpuGeneration::AmdRdna2.is_intel());
        assert!(!GpuGeneration::AmdRdna3.is_intel());
        assert!(!GpuGeneration::Unknown.is_intel());
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
