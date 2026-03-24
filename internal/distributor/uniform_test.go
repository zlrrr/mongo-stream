package distributor

import (
	"fmt"
	"testing"
)

func TestNewUniform_TargetCount(t *testing.T) {
	u := NewUniform(10, 20)
	if len(u.Targets()) != 200 {
		t.Fatalf("expected 200 targets, got %d", len(u.Targets()))
	}
}

func TestNewUniform_TargetNames(t *testing.T) {
	u := NewUniform(2, 3)
	targets := u.Targets()
	expected := []Target{
		{DB: "db_0", Collection: "col_0"},
		{DB: "db_0", Collection: "col_1"},
		{DB: "db_0", Collection: "col_2"},
		{DB: "db_1", Collection: "col_0"},
		{DB: "db_1", Collection: "col_1"},
		{DB: "db_1", Collection: "col_2"},
	}
	if len(targets) != len(expected) {
		t.Fatalf("expected %d targets, got %d", len(expected), len(targets))
	}
	for i, want := range expected {
		if targets[i] != want {
			t.Errorf("target[%d]: got %+v, want %+v", i, targets[i], want)
		}
	}
}

func TestUniform_Next_RoundRobin(t *testing.T) {
	u := NewUniform(2, 2)
	// 4 targets: (db_0,col_0), (db_0,col_1), (db_1,col_0), (db_1,col_1)
	cases := []struct {
		index int64
		want  Target
	}{
		{0, Target{"db_0", "col_0"}},
		{1, Target{"db_0", "col_1"}},
		{2, Target{"db_1", "col_0"}},
		{3, Target{"db_1", "col_1"}},
		{4, Target{"db_0", "col_0"}}, // wraps around
		{8, Target{"db_0", "col_0"}},
	}
	for _, tc := range cases {
		got := u.Next(tc.index)
		if got != tc.want {
			t.Errorf("Next(%d) = %+v; want %+v", tc.index, got, tc.want)
		}
	}
}

func TestUniform_EvenDistribution(t *testing.T) {
	numDBs, numCols := 3, 4
	u := NewUniform(numDBs, numCols)
	total := int64(numDBs * numCols)
	counts := make(map[string]int64)
	N := int64(1200)
	for i := int64(0); i < N; i++ {
		tgt := u.Next(i)
		key := fmt.Sprintf("%s.%s", tgt.DB, tgt.Collection)
		counts[key]++
	}
	expected := N / total
	for key, cnt := range counts {
		if cnt != expected {
			t.Errorf("target %s: got %d, want %d", key, cnt, expected)
		}
	}
}
