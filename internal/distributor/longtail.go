package distributor

import "math"

// Longtail distributes writes using a power-law (Zipf) distribution.
// A small number of "hot" targets at the front of the list receive
// disproportionately more writes, while the rest form a long tail.
//
// Weight for rank i (0-based) = 1 / (i+1)^skew
//
// The default skew of 1.07 produces a realistic e-commerce / cache-miss
// workload profile where the top-10 collections see roughly 40 % of traffic.
// Increase skew for a sharper hot-spot; decrease toward 0 for a flatter curve.
//
// Next is goroutine-safe: all state is read-only after construction.
type Longtail struct {
	weighted
	skew float64
}

const defaultSkew = 1.07

// NewLongtail creates a Longtail distributor for numDBs×numCols targets
// using the default skew exponent.
func NewLongtail(numDBs, numCols int) *Longtail {
	return NewLongtailWithSkew(numDBs, numCols, defaultSkew)
}

// NewLongtailWithSkew creates a Longtail distributor with a custom skew exponent.
// skew > 1 concentrates writes on the first few targets;
// skew = 1 is classic Zipf (harmonic series);
// skew < 1 flattens the curve toward uniform.
func NewLongtailWithSkew(numDBs, numCols int, skew float64) *Longtail {
	targets := buildTargets(numDBs, numCols)
	n := len(targets)

	weights := make([]float64, n)
	for i := range weights {
		weights[i] = 1.0 / math.Pow(float64(i+1), skew)
	}

	return &Longtail{
		weighted: weighted{targets: targets, cdf: buildCDF(weights)},
		skew:     skew,
	}
}

// Next returns the target for document at the given index.
func (l *Longtail) Next(index int64) Target {
	return l.pick(indexToProb(index))
}
