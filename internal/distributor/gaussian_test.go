package distributor

import (
	"math"
	"testing"
)

func TestGaussian_TargetCount(t *testing.T) {
	g := NewGaussian(10, 20)
	if got := len(g.Targets()); got != 200 {
		t.Fatalf("expected 200 targets, got %d", got)
	}
}

func TestGaussian_Next_InRange(t *testing.T) {
	g := NewGaussian(5, 4)
	targets := g.Targets()
	set := make(map[Target]bool, len(targets))
	for _, tgt := range targets {
		set[tgt] = true
	}
	for i := int64(0); i < 10000; i++ {
		got := g.Next(i)
		if !set[got] {
			t.Fatalf("Next(%d) returned unknown target %+v", i, got)
		}
	}
}

// TestGaussian_CentreHeavy verifies that the middle bucket receives more writes
// than the outermost bucket (bell-curve shape).
func TestGaussian_CentreHeavy(t *testing.T) {
	const numDBs, numCols = 1, 20 // 20 targets in a single DB
	g := NewGaussian(numDBs, numCols)
	counts := make([]int64, numCols)

	const N = 100_000
	for i := int64(0); i < N; i++ {
		tgt := g.Next(i)
		// Extract collection index from "col_X"
		var idx int
		if _, err := parseColIdx(tgt.Collection, &idx); err != nil {
			t.Fatalf("unexpected collection name %q", tgt.Collection)
		}
		counts[idx]++
	}

	centre := counts[numCols/2]
	edge := counts[0]
	if centre <= edge {
		t.Errorf("expected centre (%d) > edge (%d) for Gaussian distribution", centre, edge)
	}
	// Centre should attract substantially more writes (at least 5×).
	if float64(centre) < 5*float64(edge) {
		t.Errorf("centre/edge ratio %.1f < 5 — distribution not bell-shaped enough",
			float64(centre)/math.Max(float64(edge), 1))
	}
}

// TestGaussian_Deterministic ensures the same index always maps to the same target.
func TestGaussian_Deterministic(t *testing.T) {
	g := NewGaussian(3, 5)
	for i := int64(0); i < 500; i++ {
		a := g.Next(i)
		b := g.Next(i)
		if a != b {
			t.Errorf("Next(%d) non-deterministic: %+v vs %+v", i, a, b)
		}
	}
}

// TestGaussian_Goroutine_Safe calls Next concurrently — the race detector will
// flag any data races if the implementation is not safe.
func TestGaussian_Goroutine_Safe(t *testing.T) {
	g := NewGaussian(4, 5)
	done := make(chan struct{})
	for w := 0; w < 8; w++ {
		go func(offset int64) {
			for i := int64(0); i < 1000; i++ {
				_ = g.Next(i + offset)
			}
			done <- struct{}{}
		}(int64(w) * 1000)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}
