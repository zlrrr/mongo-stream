package distributor

import "math"

// Gaussian distributes writes across targets following a bell-curve (normal)
// distribution centred on the middle target.  The outermost targets receive
// exponentially fewer writes than the centre, modelling workloads where a
// "hot" mid-range of collections attracts most traffic.
//
// sigma is set to total/6 so that ±3σ spans the full target list (≈99.7 %
// of the Gaussian mass falls inside the range).
//
// Next is goroutine-safe: all state is read-only after construction.
type Gaussian struct {
	weighted
}

// NewGaussian creates a Gaussian distributor for numDBs×numCols targets.
func NewGaussian(numDBs, numCols int) *Gaussian {
	targets := buildTargets(numDBs, numCols)
	n := len(targets)
	mu := float64(n-1) / 2.0
	sigma := float64(n) / 6.0
	if sigma == 0 {
		sigma = 1
	}

	weights := make([]float64, n)
	for i := range weights {
		z := (float64(i) - mu) / sigma
		weights[i] = math.Exp(-0.5 * z * z)
	}

	return &Gaussian{weighted: weighted{targets: targets, cdf: buildCDF(weights)}}
}

// Next returns the target for document at the given index.
func (g *Gaussian) Next(index int64) Target {
	return g.pick(indexToProb(index))
}
