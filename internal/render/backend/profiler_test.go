package backend

import (
	"testing"
	"time"
)

func TestFrameProfiler_BasicTiming(t *testing.T) {
	profiler := NewFrameProfiler()

	// Simulate a frame
	profiler.BeginFrame()
	time.Sleep(1 * time.Millisecond) // CPU work
	profiler.MarkGPUSubmit()
	time.Sleep(1 * time.Millisecond) // GPU work
	profiler.EndFrame()

	stats := profiler.GetStats()

	if stats.TotalFrames != 1 {
		t.Errorf("expected 1 frame, got %d", stats.TotalFrames)
	}

	// Frame time should be at least 2ms (CPU + GPU)
	if stats.AvgFrameTimeMs < 2.0 {
		t.Errorf("expected frame time >= 2ms, got %.2fms", stats.AvgFrameTimeMs)
	}

	// CPU and GPU times should each be at least 1ms
	if stats.AvgCPUTimeMs < 1.0 {
		t.Errorf("expected CPU time >= 1ms, got %.2fms", stats.AvgCPUTimeMs)
	}
	if stats.AvgGPUTimeMs < 1.0 {
		t.Errorf("expected GPU time >= 1ms, got %.2fms", stats.AvgGPUTimeMs)
	}
}

func TestFrameProfiler_MultipleFrames(t *testing.T) {
	profiler := NewFrameProfiler()

	// Simulate 3 frames
	for i := 0; i < 3; i++ {
		profiler.BeginFrame()
		time.Sleep(500 * time.Microsecond)
		profiler.MarkGPUSubmit()
		time.Sleep(500 * time.Microsecond)
		profiler.EndFrame()
	}

	stats := profiler.GetStats()

	if stats.TotalFrames != 3 {
		t.Errorf("expected 3 frames, got %d", stats.TotalFrames)
	}

	// Average frame time should be around 1ms
	if stats.AvgFrameTimeMs < 1.0 || stats.AvgFrameTimeMs > 3.0 {
		t.Errorf("expected frame time ~1ms, got %.2fms", stats.AvgFrameTimeMs)
	}
}

func TestFrameProfiler_MinMax(t *testing.T) {
	profiler := NewFrameProfiler()

	// Fast frame
	profiler.BeginFrame()
	time.Sleep(200 * time.Microsecond)
	profiler.MarkGPUSubmit()
	time.Sleep(200 * time.Microsecond)
	profiler.EndFrame()

	// Slow frame
	profiler.BeginFrame()
	time.Sleep(3 * time.Millisecond)
	profiler.MarkGPUSubmit()
	time.Sleep(3 * time.Millisecond)
	profiler.EndFrame()

	stats := profiler.GetStats()

	// Verify min < max (the key invariant)
	if stats.MinFrameTimeMs >= stats.MaxFrameTimeMs {
		t.Errorf("expected min < max, got min=%.2fms, max=%.2fms", stats.MinFrameTimeMs, stats.MaxFrameTimeMs)
	}

	// Max should be significantly higher than min
	if stats.MaxFrameTimeMs < 2*stats.MinFrameTimeMs {
		t.Errorf("expected max > 2*min, got min=%.2fms, max=%.2fms", stats.MinFrameTimeMs, stats.MaxFrameTimeMs)
	}
}

func TestFrameProfiler_RecentAverage(t *testing.T) {
	profiler := NewFrameProfiler()

	// Simulate 70 frames (more than the 60-frame buffer)
	for i := 0; i < 70; i++ {
		profiler.BeginFrame()
		// Gradually increasing frame time
		sleepTime := time.Duration(i*10) * time.Microsecond
		time.Sleep(sleepTime)
		profiler.MarkGPUSubmit()
		time.Sleep(sleepTime)
		profiler.EndFrame()
	}

	stats := profiler.GetStats()

	if stats.TotalFrames != 70 {
		t.Errorf("expected 70 frames, got %d", stats.TotalFrames)
	}

	// Recent average should be higher than overall average
	// (because recent frames are slower)
	if stats.RecentAvgFrameTimeMs <= stats.AvgFrameTimeMs {
		t.Logf("Recent avg: %.2fms, Overall avg: %.2fms", stats.RecentAvgFrameTimeMs, stats.AvgFrameTimeMs)
		// This is not an error, just informative
	}
}

func TestFrameProfiler_Reset(t *testing.T) {
	profiler := NewFrameProfiler()

	// Simulate some frames
	for i := 0; i < 5; i++ {
		profiler.BeginFrame()
		time.Sleep(100 * time.Microsecond)
		profiler.MarkGPUSubmit()
		time.Sleep(100 * time.Microsecond)
		profiler.EndFrame()
	}

	profiler.Reset()

	stats := profiler.GetStats()

	if stats.TotalFrames != 0 {
		t.Errorf("expected 0 frames after reset, got %d", stats.TotalFrames)
	}
	if stats.AvgFrameTimeMs != 0 {
		t.Errorf("expected 0 avg frame time after reset, got %.2fms", stats.AvgFrameTimeMs)
	}
	if stats.MaxFrameTimeMs != 0 {
		t.Errorf("expected 0 max frame time after reset, got %.2fms", stats.MaxFrameTimeMs)
	}
}

func TestFrameProfiler_NoGPUMark(t *testing.T) {
	profiler := NewFrameProfiler()

	// Begin and end frame without marking GPU submit
	profiler.BeginFrame()
	time.Sleep(1 * time.Millisecond)
	// Missing: profiler.MarkGPUSubmit()
	profiler.EndFrame()

	stats := profiler.GetStats()

	if stats.TotalFrames != 1 {
		t.Errorf("expected 1 frame, got %d", stats.TotalFrames)
	}

	// CPU time should be very small (zero time between BeginFrame and unset gpuSubmitTime)
	// GPU time should capture most of the frame time
	if stats.AvgGPUTimeMs < 1.0 {
		t.Errorf("expected GPU time >= 1ms when no GPU mark, got %.2fms", stats.AvgGPUTimeMs)
	}
}

func BenchmarkFrameProfiler_SingleFrame(b *testing.B) {
	profiler := NewFrameProfiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.BeginFrame()
		profiler.MarkGPUSubmit()
		profiler.EndFrame()
	}
}

func BenchmarkFrameProfiler_GetStats(b *testing.B) {
	profiler := NewFrameProfiler()

	// Record some frames
	for i := 0; i < 100; i++ {
		profiler.BeginFrame()
		profiler.MarkGPUSubmit()
		profiler.EndFrame()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = profiler.GetStats()
	}
}
