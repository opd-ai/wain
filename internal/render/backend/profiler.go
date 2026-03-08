package backend

import (
	"sync"
	"time"
)

// FrameProfiler tracks frame rendering performance metrics.
type FrameProfiler struct {
	mu sync.Mutex

	// Current frame metrics
	frameStartTime time.Time
	cpuStartTime   time.Time
	gpuSubmitTime  time.Time

	// Accumulated statistics
	totalFrames      int64
	totalCPUTimeNs   int64
	totalGPUTimeNs   int64
	totalFrameTimeNs int64
	minFrameTimeNs   int64
	maxFrameTimeNs   int64
	minCPUTimeNs     int64
	maxCPUTimeNs     int64
	minGPUTimeNs     int64
	maxGPUTimeNs     int64

	// Recent history (circular buffer for last 60 frames)
	recentFrameTimes [60]int64
	recentCPUTimes   [60]int64
	recentGPUTimes   [60]int64
	historyIndex     int
}

// NewFrameProfiler creates a new frame profiler.
func NewFrameProfiler() *FrameProfiler {
	return &FrameProfiler{
		minFrameTimeNs: 1<<63 - 1,
		minCPUTimeNs:   1<<63 - 1,
		minGPUTimeNs:   1<<63 - 1,
	}
}

// BeginFrame marks the start of a frame.
func (p *FrameProfiler) BeginFrame() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.frameStartTime = time.Now()
	p.cpuStartTime = p.frameStartTime
}

// MarkGPUSubmit marks when GPU commands are submitted (end of CPU work, start of GPU work).
func (p *FrameProfiler) MarkGPUSubmit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.gpuSubmitTime = time.Now()
}

// EndFrame marks the end of a frame and records timing statistics.
func (p *FrameProfiler) EndFrame() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Calculate times
	frameTimeNs := now.Sub(p.frameStartTime).Nanoseconds()
	cpuTimeNs := p.gpuSubmitTime.Sub(p.cpuStartTime).Nanoseconds()
	gpuTimeNs := now.Sub(p.gpuSubmitTime).Nanoseconds()

	// Update totals
	p.totalFrames++
	p.totalFrameTimeNs += frameTimeNs
	p.totalCPUTimeNs += cpuTimeNs
	p.totalGPUTimeNs += gpuTimeNs

	// Update min/max
	if frameTimeNs < p.minFrameTimeNs {
		p.minFrameTimeNs = frameTimeNs
	}
	if frameTimeNs > p.maxFrameTimeNs {
		p.maxFrameTimeNs = frameTimeNs
	}
	if cpuTimeNs < p.minCPUTimeNs {
		p.minCPUTimeNs = cpuTimeNs
	}
	if cpuTimeNs > p.maxCPUTimeNs {
		p.maxCPUTimeNs = cpuTimeNs
	}
	if gpuTimeNs < p.minGPUTimeNs {
		p.minGPUTimeNs = gpuTimeNs
	}
	if gpuTimeNs > p.maxGPUTimeNs {
		p.maxGPUTimeNs = gpuTimeNs
	}

	// Store in circular buffer
	p.recentFrameTimes[p.historyIndex] = frameTimeNs
	p.recentCPUTimes[p.historyIndex] = cpuTimeNs
	p.recentGPUTimes[p.historyIndex] = gpuTimeNs
	p.historyIndex = (p.historyIndex + 1) % len(p.recentFrameTimes)
}

// GetStats returns current profiling statistics.
func (p *FrameProfiler) GetStats() FrameStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := FrameStats{
		TotalFrames: p.totalFrames,
	}

	if p.totalFrames > 0 {
		stats.AvgFrameTimeMs = float64(p.totalFrameTimeNs) / float64(p.totalFrames) / 1e6
		stats.AvgCPUTimeMs = float64(p.totalCPUTimeNs) / float64(p.totalFrames) / 1e6
		stats.AvgGPUTimeMs = float64(p.totalGPUTimeNs) / float64(p.totalFrames) / 1e6
		stats.MinFrameTimeMs = float64(p.minFrameTimeNs) / 1e6
		stats.MaxFrameTimeMs = float64(p.maxFrameTimeNs) / 1e6
		stats.MinCPUTimeMs = float64(p.minCPUTimeNs) / 1e6
		stats.MaxCPUTimeMs = float64(p.maxCPUTimeNs) / 1e6
		stats.MinGPUTimeMs = float64(p.minGPUTimeNs) / 1e6
		stats.MaxGPUTimeMs = float64(p.maxGPUTimeNs) / 1e6

		// Calculate recent average (last 60 frames or fewer)
		recentCount := int(p.totalFrames)
		if recentCount > len(p.recentFrameTimes) {
			recentCount = len(p.recentFrameTimes)
		}

		var recentFrameSum, recentCPUSum, recentGPUSum int64
		for i := 0; i < recentCount; i++ {
			recentFrameSum += p.recentFrameTimes[i]
			recentCPUSum += p.recentCPUTimes[i]
			recentGPUSum += p.recentGPUTimes[i]
		}

		if recentCount > 0 {
			stats.RecentAvgFrameTimeMs = float64(recentFrameSum) / float64(recentCount) / 1e6
			stats.RecentAvgCPUTimeMs = float64(recentCPUSum) / float64(recentCount) / 1e6
			stats.RecentAvgGPUTimeMs = float64(recentGPUSum) / float64(recentCount) / 1e6
		}
	}

	return stats
}

// Reset clears all profiling statistics.
func (p *FrameProfiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalFrames = 0
	p.totalCPUTimeNs = 0
	p.totalGPUTimeNs = 0
	p.totalFrameTimeNs = 0
	p.minFrameTimeNs = 1<<63 - 1
	p.maxFrameTimeNs = 0
	p.minCPUTimeNs = 1<<63 - 1
	p.maxCPUTimeNs = 0
	p.minGPUTimeNs = 1<<63 - 1
	p.maxGPUTimeNs = 0
	p.recentFrameTimes = [60]int64{}
	p.recentCPUTimes = [60]int64{}
	p.recentGPUTimes = [60]int64{}
	p.historyIndex = 0
}

// FrameStats contains frame profiling statistics.
type FrameStats struct {
	TotalFrames int64

	// Average times over all frames
	AvgFrameTimeMs float64
	AvgCPUTimeMs   float64
	AvgGPUTimeMs   float64

	// Min/Max times
	MinFrameTimeMs float64
	MaxFrameTimeMs float64
	MinCPUTimeMs   float64
	MaxCPUTimeMs   float64
	MinGPUTimeMs   float64
	MaxGPUTimeMs   float64

	// Recent average (last 60 frames)
	RecentAvgFrameTimeMs float64
	RecentAvgCPUTimeMs   float64
	RecentAvgGPUTimeMs   float64
}
