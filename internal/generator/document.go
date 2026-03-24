package generator

import (
	"encoding/hex"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Document represents a generated MongoDB document.
type Document = bson.D

// Generator produces random BSON documents.
type Generator struct {
	rng *rand.Rand
}

// New creates a Generator. If seed == 0, a time-based seed is used.
func New(seed int64) *Generator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Generator{rng: rand.New(rand.NewSource(seed))} //nolint:gosec
}

var tagPool = []string{
	"alpha", "beta", "gamma", "delta", "epsilon",
	"zeta", "eta", "theta", "iota", "kappa",
}

// Next generates the document at the given sequence index.
func (g *Generator) Next(seq int64) Document {
	payloadBytes := make([]byte, 32)
	_, _ = g.rng.Read(payloadBytes)
	payload := hex.EncodeToString(payloadBytes)

	tagCount := g.rng.Intn(3) + 1
	tags := make([]string, tagCount)
	for i := range tags {
		tags[i] = tagPool[g.rng.Intn(len(tagPool))]
	}

	return bson.D{
		{Key: "seq", Value: seq},
		{Key: "ts", Value: time.Now().UTC()},
		{Key: "payload", Value: payload},
		{Key: "tags", Value: tags},
		{Key: "value", Value: g.rng.Float64() * 1000},
	}
}

// Batch generates n documents starting at seq.
func (g *Generator) Batch(seq int64, n int) []interface{} {
	docs := make([]interface{}, n)
	for i := range docs {
		docs[i] = g.Next(seq + int64(i))
	}
	return docs
}
