package render

import (
	"sync"
	"sync/atomic"
)

// MemoryStats tracks GPU memory allocation statistics.
type MemoryStats struct {
	// AllocatedBuffers is the current number of allocated GPU buffers.
	AllocatedBuffers atomic.Int64

	// AllocatedBytes is the total GPU memory currently allocated in bytes.
	AllocatedBytes atomic.Int64

	// PeakAllocatedBytes is the peak GPU memory usage in bytes.
	PeakAllocatedBytes atomic.Int64

	// TotalAllocations is the cumulative number of allocations.
	TotalAllocations atomic.Int64

	// TotalDeallocations is the cumulative number of deallocations.
	TotalDeallocations atomic.Int64

	mu              sync.Mutex
	allocationSizes map[uintptr]uint64 // tracks size of each allocation by handle address
}

// NewMemoryStats creates a new memory statistics tracker.
func NewMemoryStats() *MemoryStats {
	return &MemoryStats{
		allocationSizes: make(map[uintptr]uint64),
	}
}

// RecordAllocation records a new buffer allocation.
func (s *MemoryStats) RecordAllocation(handle uintptr, sizeBytes uint64) {
	s.mu.Lock()
	s.allocationSizes[handle] = sizeBytes
	s.mu.Unlock()

	s.AllocatedBuffers.Add(1)
	s.TotalAllocations.Add(1)
	current := s.AllocatedBytes.Add(int64(sizeBytes))

	// Update peak if necessary
	for {
		peak := s.PeakAllocatedBytes.Load()
		if current <= peak {
			break
		}
		if s.PeakAllocatedBytes.CompareAndSwap(peak, current) {
			break
		}
	}
}

// RecordDeallocation records a buffer deallocation.
func (s *MemoryStats) RecordDeallocation(handle uintptr) {
	s.mu.Lock()
	sizeBytes, ok := s.allocationSizes[handle]
	if ok {
		delete(s.allocationSizes, handle)
	}
	s.mu.Unlock()

	if ok {
		s.AllocatedBuffers.Add(-1)
		s.TotalDeallocations.Add(1)
		s.AllocatedBytes.Add(-int64(sizeBytes))
	}
}

// Snapshot returns current memory statistics.
func (s *MemoryStats) Snapshot() MemorySnapshot {
	return MemorySnapshot{
		AllocatedBuffers:   s.AllocatedBuffers.Load(),
		AllocatedBytes:     s.AllocatedBytes.Load(),
		PeakAllocatedBytes: s.PeakAllocatedBytes.Load(),
		TotalAllocations:   s.TotalAllocations.Load(),
		TotalDeallocations: s.TotalDeallocations.Load(),
	}
}

// Reset clears all statistics (primarily for testing).
func (s *MemoryStats) Reset() {
	s.mu.Lock()
	s.allocationSizes = make(map[uintptr]uint64)
	s.mu.Unlock()

	s.AllocatedBuffers.Store(0)
	s.AllocatedBytes.Store(0)
	s.PeakAllocatedBytes.Store(0)
	s.TotalAllocations.Store(0)
	s.TotalDeallocations.Store(0)
}

// MemorySnapshot is an immutable snapshot of memory statistics.
type MemorySnapshot struct {
	AllocatedBuffers   int64
	AllocatedBytes     int64
	PeakAllocatedBytes int64
	TotalAllocations   int64
	TotalDeallocations int64
}

// Global memory statistics instance
var globalMemStats = NewMemoryStats()

// GetMemoryStats returns the global memory statistics.
//
// GetMemoryStats provides a snapshot of current GPU buffer allocation state
// for monitoring and profiling. It is safe to call from any goroutine.
//
// Example usage:
//
//	snap := render.GetMemoryStats()
//	fmt.Printf("Allocated: %d buffers / %d bytes (peak %d bytes)\n",
//	    snap.AllocatedBuffers, snap.AllocatedBytes, snap.PeakAllocatedBytes)
func GetMemoryStats() MemorySnapshot {
	return globalMemStats.Snapshot()
}

// ResetMemoryStats resets the global memory statistics (for testing).
func ResetMemoryStats() {
	globalMemStats.Reset()
}
