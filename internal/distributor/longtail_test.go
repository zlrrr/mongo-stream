package distributor

import (
	"fmt"
	"testing"
)

func TestLongtail_TargetCount(t *testing.T) {
	l := NewLongtail(10, 20)
	if got := len(l.Targets()); got != 200 {
		t.Fatalf("expected 200 targets, got %d", got)
	}
}

func TestLongtail_Next_InRange(t *testing.T) {
	l := NewLongtail(5, 4)
	targets := l.Targets()
	set := make(map[Target]bool, len(targets))
	for _, tgt := range targets {
		set[tgt] = true
	}
	for i := int64(0); i < 10000; i++ {
		got := l.Next(i)
		if !set[got] {
			t.Fatalf("Next(%d) returned unknown target %+v", i, got)
		}
	}
}

// TestLongtail_HeadHeavy verifies that the first target (rank 0) receives
// substantially more writes than the last target (rank n-1).
func TestLongtail_HeadHeavy(t *testing.T) {
	const numTargets = 20
	l := NewLongtail(1, numTargets)
	counts := make([]int64, numTargets)

	const N = 100_000
	for i := int64(0); i < N; i++ {
		tgt := l.Next(i)
		var idx int
		if _, err := parseColIdx(tgt.Collection, &idx); err != nil {
			t.Fatalf("unexpected collection name %q", tgt.Collection)
		}
		counts[idx]++
	}

	head := counts[0]
	tail := counts[numTargets-1]
	if head <= tail {
		t.Errorf("expected head (%d) > tail (%d) for longtail distribution", head, tail)
	}
	// With skew=1.07 and 20 targets, head/tail ratio should be well above 5.
	if tail == 0 {
		tail = 1
	}
	ratio := float64(head) / float64(tail)
	if ratio < 5 {
		t.Errorf("head/tail ratio %.1f < 5 — distribution not skewed enough", ratio)
	}
}

// TestLongtail_HigherSkew_MoreConcentrated verifies that increasing skew
// concentrates more writes on rank-0.
func TestLongtail_HigherSkew_MoreConcentrated(t *testing.T) {
	const numTargets = 50
	countFirst := func(skew float64) int64 {
		l := NewLongtailWithSkew(1, numTargets, skew)
		var c int64
		for i := int64(0); i < 50_000; i++ {
			if l.Next(i).Collection == "col_0" {
				c++
			}
		}
		return c
	}

	low := countFirst(0.5)
	high := countFirst(2.0)
	if high <= low {
		t.Errorf("higher skew (2.0) should give more writes to col_0 than lower skew (0.5): %d vs %d", high, low)
	}
}

// TestLongtail_Deterministic ensures the same index always maps to the same target.
func TestLongtail_Deterministic(t *testing.T) {
	l := NewLongtail(3, 5)
	for i := int64(0); i < 500; i++ {
		a := l.Next(i)
		b := l.Next(i)
		if a != b {
			t.Errorf("Next(%d) non-deterministic: %+v vs %+v", i, a, b)
		}
	}
}

// TestLongtail_Goroutine_Safe calls Next concurrently — the race detector will
// flag any data races if the implementation is not safe.
func TestLongtail_Goroutine_Safe(t *testing.T) {
	l := NewLongtail(4, 5)
	done := make(chan struct{})
	for w := 0; w < 8; w++ {
		go func(offset int64) {
			for i := int64(0); i < 1000; i++ {
				_ = l.Next(i + offset)
			}
			done <- struct{}{}
		}(int64(w) * 1000)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

// parseColIdx is a small helper shared by gaussian and longtail tests.
// It parses "col_N" and writes N into *idx.
func parseColIdx(name string, idx *int) (int, error) {
	n, err := fmt.Sscanf(name, "col_%d", idx)
	return n, err
}
