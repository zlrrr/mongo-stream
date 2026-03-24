package distributor

import "sort"

// weighted is the shared base for probability-weighted distributors.
// It holds a pre-built CDF table and is goroutine-safe (read-only after construction).
type weighted struct {
	targets []Target
	cdf     []float64 // len == len(targets); cdf[last] == 1.0
}

// Targets returns the full ordered list of write targets.
func (w *weighted) Targets() []Target { return w.targets }

// pick maps a value p ∈ [0,1) to a target index via binary search on the CDF.
func (w *weighted) pick(p float64) Target {
	i := sort.SearchFloat64s(w.cdf, p)
	if i >= len(w.targets) {
		i = len(w.targets) - 1
	}
	return w.targets[i]
}

// indexToProb converts a document index to a deterministic value in [0,1)
// using the splitmix64 hash so that successive indices are well-spread across
// the probability space (no shared mutable state → goroutine-safe).
func indexToProb(index int64) float64 {
	x := uint64(index)
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x ^= x >> 31
	// Map to [0,1): divide by 2^64.
	return float64(x) / (1 << 32) / (1 << 32)
}

// buildCDF normalises raw weights into a cumulative distribution function.
// weights must be non-negative and have at least one positive value.
func buildCDF(weights []float64) []float64 {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	cdf := make([]float64, len(weights))
	acc := 0.0
	for i, w := range weights {
		acc += w / sum
		cdf[i] = acc
	}
	cdf[len(cdf)-1] = 1.0 // guard against floating-point rounding
	return cdf
}
