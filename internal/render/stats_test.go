package render

import (
	"testing"
)

func TestMemoryStats_RecordAllocation(t *testing.T) {
	stats := NewMemoryStats()

	// Record first allocation
	stats.RecordAllocation(0x1000, 1024)

	snapshot := stats.Snapshot()
	if snapshot.AllocatedBuffers != 1 {
		t.Errorf("expected 1 buffer, got %d", snapshot.AllocatedBuffers)
	}
	if snapshot.AllocatedBytes != 1024 {
		t.Errorf("expected 1024 bytes, got %d", snapshot.AllocatedBytes)
	}
	if snapshot.PeakAllocatedBytes != 1024 {
		t.Errorf("expected peak 1024 bytes, got %d", snapshot.PeakAllocatedBytes)
	}
	if snapshot.TotalAllocations != 1 {
		t.Errorf("expected 1 total allocation, got %d", snapshot.TotalAllocations)
	}

	// Record second allocation
	stats.RecordAllocation(0x2000, 2048)

	snapshot = stats.Snapshot()
	if snapshot.AllocatedBuffers != 2 {
		t.Errorf("expected 2 buffers, got %d", snapshot.AllocatedBuffers)
	}
	if snapshot.AllocatedBytes != 3072 {
		t.Errorf("expected 3072 bytes, got %d", snapshot.AllocatedBytes)
	}
	if snapshot.PeakAllocatedBytes != 3072 {
		t.Errorf("expected peak 3072 bytes, got %d", snapshot.PeakAllocatedBytes)
	}
	if snapshot.TotalAllocations != 2 {
		t.Errorf("expected 2 total allocations, got %d", snapshot.TotalAllocations)
	}
}

func TestMemoryStats_RecordDeallocation(t *testing.T) {
	stats := NewMemoryStats()

	// Allocate and deallocate
	stats.RecordAllocation(0x1000, 1024)
	stats.RecordAllocation(0x2000, 2048)
	stats.RecordDeallocation(0x1000)

	snapshot := stats.Snapshot()
	if snapshot.AllocatedBuffers != 1 {
		t.Errorf("expected 1 buffer, got %d", snapshot.AllocatedBuffers)
	}
	if snapshot.AllocatedBytes != 2048 {
		t.Errorf("expected 2048 bytes, got %d", snapshot.AllocatedBytes)
	}
	if snapshot.PeakAllocatedBytes != 3072 {
		t.Errorf("expected peak 3072 bytes (not affected by deallocation), got %d", snapshot.PeakAllocatedBytes)
	}
	if snapshot.TotalDeallocations != 1 {
		t.Errorf("expected 1 deallocation, got %d", snapshot.TotalDeallocations)
	}
}

func TestMemoryStats_PeakTracking(t *testing.T) {
	stats := NewMemoryStats()

	// Build up to peak
	stats.RecordAllocation(0x1000, 1024)
	stats.RecordAllocation(0x2000, 2048)
	stats.RecordAllocation(0x3000, 4096) // peak = 7168

	// Deallocate
	stats.RecordDeallocation(0x3000)
	stats.RecordDeallocation(0x2000)

	snapshot := stats.Snapshot()
	if snapshot.AllocatedBytes != 1024 {
		t.Errorf("expected 1024 current bytes, got %d", snapshot.AllocatedBytes)
	}
	if snapshot.PeakAllocatedBytes != 7168 {
		t.Errorf("expected peak 7168 bytes, got %d", snapshot.PeakAllocatedBytes)
	}
}

func TestMemoryStats_DeallocateUnknown(t *testing.T) {
	stats := NewMemoryStats()

	// Deallocate unknown handle should not panic
	stats.RecordDeallocation(0x9999)

	snapshot := stats.Snapshot()
	if snapshot.AllocatedBuffers != 0 {
		t.Errorf("expected 0 buffers, got %d", snapshot.AllocatedBuffers)
	}
	if snapshot.TotalDeallocations != 0 {
		t.Errorf("expected 0 deallocations for unknown handle, got %d", snapshot.TotalDeallocations)
	}
}

func TestMemoryStats_Reset(t *testing.T) {
	stats := NewMemoryStats()

	stats.RecordAllocation(0x1000, 1024)
	stats.RecordAllocation(0x2000, 2048)
	stats.RecordDeallocation(0x1000)

	stats.Reset()

	snapshot := stats.Snapshot()
	if snapshot.AllocatedBuffers != 0 {
		t.Errorf("expected 0 buffers after reset, got %d", snapshot.AllocatedBuffers)
	}
	if snapshot.AllocatedBytes != 0 {
		t.Errorf("expected 0 bytes after reset, got %d", snapshot.AllocatedBytes)
	}
	if snapshot.PeakAllocatedBytes != 0 {
		t.Errorf("expected 0 peak bytes after reset, got %d", snapshot.PeakAllocatedBytes)
	}
	if snapshot.TotalAllocations != 0 {
		t.Errorf("expected 0 total allocations after reset, got %d", snapshot.TotalAllocations)
	}
	if snapshot.TotalDeallocations != 0 {
		t.Errorf("expected 0 total deallocations after reset, got %d", snapshot.TotalDeallocations)
	}
}

func TestGlobalMemoryStats(t *testing.T) {
	// Reset global stats before test
	ResetMemoryStats()

	// Simulate some allocations via global API
	globalMemStats.RecordAllocation(0x1000, 512)

	stats := GetMemoryStats()
	if stats.AllocatedBuffers != 1 {
		t.Errorf("expected 1 buffer in global stats, got %d", stats.AllocatedBuffers)
	}
	if stats.AllocatedBytes != 512 {
		t.Errorf("expected 512 bytes in global stats, got %d", stats.AllocatedBytes)
	}

	// Clean up
	globalMemStats.RecordDeallocation(0x1000)
	ResetMemoryStats()
}

func BenchmarkMemoryStats_RecordAllocation(b *testing.B) {
	stats := NewMemoryStats()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stats.RecordAllocation(uintptr(i), 1024)
	}
}

func BenchmarkMemoryStats_RecordDeallocation(b *testing.B) {
	stats := NewMemoryStats()

	// Pre-allocate
	for i := 0; i < b.N; i++ {
		stats.RecordAllocation(uintptr(i), 1024)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.RecordDeallocation(uintptr(i))
	}
}

func BenchmarkMemoryStats_Snapshot(b *testing.B) {
	stats := NewMemoryStats()
	stats.RecordAllocation(0x1000, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stats.Snapshot()
	}
}
