package generator

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNext_Fields(t *testing.T) {
	g := New(42)
	doc := g.Next(0)

	keys := make(map[string]bool)
	for _, e := range doc {
		keys[e.Key] = true
	}
	for _, required := range []string{"seq", "ts", "payload", "tags", "value"} {
		if !keys[required] {
			t.Errorf("missing field %q", required)
		}
	}
}

func TestNext_SeqField(t *testing.T) {
	g := New(1)
	for i := int64(0); i < 5; i++ {
		doc := g.Next(i)
		for _, e := range doc {
			if e.Key == "seq" {
				if e.Value.(int64) != i {
					t.Errorf("seq mismatch: got %v, want %d", e.Value, i)
				}
			}
		}
	}
}

func TestBatch_Length(t *testing.T) {
	g := New(99)
	batch := g.Batch(0, 50)
	if len(batch) != 50 {
		t.Fatalf("expected 50 docs, got %d", len(batch))
	}
}

func TestBatch_Types(t *testing.T) {
	g := New(7)
	batch := g.Batch(0, 3)
	for _, item := range batch {
		if _, ok := item.(bson.D); !ok {
			t.Errorf("expected bson.D, got %T", item)
		}
	}
}

func TestDeterministic(t *testing.T) {
	// Same seed → same first payload.
	g1 := New(1234)
	g2 := New(1234)
	d1 := g1.Next(0)
	d2 := g2.Next(0)

	getPayload := func(d Document) string {
		for _, e := range d {
			if e.Key == "payload" {
				return e.Value.(string)
			}
		}
		return ""
	}
	if getPayload(d1) != getPayload(d2) {
		t.Error("same seed should produce same payload")
	}
}
