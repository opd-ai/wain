// Command bench measures software rendering frame times for a standardised UI workload.
//
// This binary renders a fixed scene (500 filled rectangles, 100 text-run placeholders,
// 10 box shadows) through the software backend at 1920×1080 and reports timing
// statistics as JSON.  It is designed to run in CI where no GPU is present.
//
// Usage:
//
//	./bin/bench                # run with defaults, exit 0
//	./bin/bench -frames 120   # more frames for stable measurement
//	./bin/bench -max 16       # fail if mean frame time exceeds 16 ms
//
// Exit codes:
//
//	0   All frames rendered; mean frame time within -max threshold (if set)
//	1   Mean frame time exceeded the -max threshold
//	2   Initialisation error
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
)

const (
	sceneWidth   = 1920
	sceneHeight  = 1080
	sceneRects   = 500
	sceneTexts   = 100
	sceneShadows = 10
)

// BenchResult is the JSON output of a bench run.
type BenchResult struct {
	Frames    int     `json:"frames"`
	MeanMs    float64 `json:"mean_ms"`
	MinMs     float64 `json:"min_ms"`
	MaxMs     float64 `json:"max_ms"`
	StdDevMs  float64 `json:"stddev_ms"`
	ThresholdMs float64 `json:"threshold_ms,omitempty"`
	Pass      bool    `json:"pass"`
}

func main() {
	frames := flag.Int("frames", 60, "number of frames to render")
	maxMs := flag.Float64("max", 0, "fail if mean frame time exceeds this value in ms (0 = no limit)")
	flag.Parse()

	result, err := runBench(*frames, *maxMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench: init error: %v\n", err)
		os.Exit(2)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "bench: JSON encode error: %v\n", err)
		os.Exit(2)
	}

	if !result.Pass {
		fmt.Fprintf(os.Stderr, "bench: FAIL — mean %.3f ms exceeds threshold %.3f ms\n",
			result.MeanMs, result.ThresholdMs)
		os.Exit(1)
	}
}

func runBench(frames int, maxMs float64) (BenchResult, error) {
	b, err := backend.NewSoftwareBackend(backend.SoftwareConfig{
		Width:  sceneWidth,
		Height: sceneHeight,
	})
	if err != nil {
		return BenchResult{}, fmt.Errorf("create software backend: %w", err)
	}
	defer b.Destroy()

	scene := buildScene()

	// Warm-up: discard first two frames to allow Go runtime to settle.
	for range 2 {
		if err := b.Render(scene); err != nil {
			return BenchResult{}, fmt.Errorf("warm-up render: %w", err)
		}
	}

	times := make([]float64, 0, frames)
	for range frames {
		start := time.Now()
		if err := b.Render(scene); err != nil {
			return BenchResult{}, fmt.Errorf("bench render: %w", err)
		}
		times = append(times, float64(time.Since(start).Microseconds())/1000.0)
	}

	result := computeStats(times, maxMs)
	return result, nil
}

// buildScene constructs the standard benchmark scene:
// 500 filled rectangles, 100 text-placeholder rectangles, 10 rounded box-shadows.
func buildScene() *displaylist.DisplayList {
	dl := displaylist.New()

	for i := range sceneRects {
		x := (i * 17) % (sceneWidth - 60)
		y := (i * 13) % (sceneHeight - 40)
		dl.AddFillRect(x, y, 60, 40, primitives.Color{R: uint8(i % 256), G: 128, B: 64, A: 255})
	}

	for i := range sceneTexts {
		x := (i * 103) % (sceneWidth - 120)
		y := (i * 71) % (sceneHeight - 18)
		dl.AddFillRect(x, y, 120, 18, primitives.Color{R: 20, G: 20, B: 20, A: 255})
	}

	for i := range sceneShadows {
		x := (i * 157) % (sceneWidth - 120)
		y := (i * 97) % (sceneHeight - 90)
		dl.AddFillRoundedRect(x, y, 120, 90, 10,
			primitives.Color{R: 180, G: 180, B: 180, A: 120})
	}

	return dl
}

// computeStats summarises a slice of per-frame durations in milliseconds.
func computeStats(times []float64, maxMs float64) BenchResult {
	if len(times) == 0 {
		return BenchResult{Pass: true}
	}

	sum, minT, maxT := sumAndBounds(times)
	mean := sum / float64(len(times))
	stddev := computeStddev(times, mean)

	pass := maxMs <= 0 || mean <= maxMs
	result := BenchResult{
		Frames:   len(times),
		MeanMs:   round3(mean),
		MinMs:    round3(minT),
		MaxMs:    round3(maxT),
		StdDevMs: round3(stddev),
		Pass:     pass,
	}
	if maxMs > 0 {
		result.ThresholdMs = maxMs
	}
	return result
}

// sumAndBounds returns the sum, minimum and maximum of the given values in a single pass.
func sumAndBounds(times []float64) (sum, min, max float64) {
	min = times[0]
	max = times[0]
	for _, t := range times {
		sum += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}
	return sum, min, max
}

// computeStddev returns the population standard deviation of values around the given mean.
func computeStddev(times []float64, mean float64) float64 {
	variance := 0.0
	for _, t := range times {
		d := t - mean
		variance += d * d
	}
	variance /= float64(len(times))
	if variance <= 0 {
		return 0
	}
	return approximateSqrt(variance)
}

// approximateSqrt computes an integer-free square root using Newton's method.
func approximateSqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// round3 rounds a float64 to three decimal places.
func round3(v float64) float64 {
	return float64(int64(v*1000+0.5)) / 1000
}
