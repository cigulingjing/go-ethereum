package pqcbench

import (
	"math"
	"sort"
)

type Timing struct {
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
}

func Summarize(samples []float64) Timing {
	if len(samples) == 0 {
		return Timing{}
	}
	return Timing{
		MeanMillis:   mean(samples),
		P50Millis:    percentile(samples, 0.50),
		P95Millis:    percentile(samples, 0.95),
		MinMillis:    min(samples),
		MaxMillis:    max(samples),
		SampleMillis: append([]float64(nil), samples...),
	}
}

func Overhead(wasm, cgo float64) float64 {
	if cgo == 0 {
		return math.NaN()
	}
	return (wasm - cgo) / cgo
}

func mean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func min(values []float64) float64 {
	out := values[0]
	for _, v := range values[1:] {
		if v < out {
			out = v
		}
	}
	return out
}

func max(values []float64) float64 {
	out := values[0]
	for _, v := range values[1:] {
		if v > out {
			out = v
		}
	}
	return out
}
