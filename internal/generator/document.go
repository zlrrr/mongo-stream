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
// A Generator must not be used concurrently from multiple goroutines.
type Generator struct {
	seed       int64
	rng        *rand.Rand
	payloadBuf []byte // reusable 32-byte buffer for random payload
	hexBuf     []byte // reusable 64-byte buffer for hex encoding
	docsBuf    []interface{}
}

// New creates a Generator. If seed == 0, a time-based seed is used.
func New(seed int64) *Generator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Generator{
		seed:       seed,
		rng:        rand.New(rand.NewSource(seed)), //nolint:gosec
		payloadBuf: make([]byte, 32),
		hexBuf:     make([]byte, 64),
	}
}

// Seed returns the seed that was used to initialise this Generator.
// Writers use it to derive unique seeds for per-worker Generator instances.
func (g *Generator) Seed() int64 { return g.seed }

var tagPool = []string{
	"alpha", "beta", "gamma", "delta", "epsilon",
	"zeta", "eta", "theta", "iota", "kappa",
}

// Next generates the document at the given sequence index.
func (g *Generator) Next(seq int64) Document {
	// Fill pre-allocated buffer with random bytes, then hex-encode in place.
	g.fillRandBytes()
	hex.Encode(g.hexBuf, g.payloadBuf)
	payload := string(g.hexBuf) // copies into immutable string

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

// fillRandBytes fills payloadBuf using Int63 (8 bytes at a time)
// instead of rng.Read which is slower on older Go versions.
func (g *Generator) fillRandBytes() {
	buf := g.payloadBuf
	for i := 0; i < len(buf); i += 8 {
		val := g.rng.Int63()
		remaining := len(buf) - i
		if remaining >= 8 {
			buf[i] = byte(val)
			buf[i+1] = byte(val >> 8)
			buf[i+2] = byte(val >> 16)
			buf[i+3] = byte(val >> 24)
			buf[i+4] = byte(val >> 32)
			buf[i+5] = byte(val >> 40)
			buf[i+6] = byte(val >> 48)
			buf[i+7] = byte(val >> 56)
		} else {
			for j := 0; j < remaining; j++ {
				buf[i+j] = byte(val >> (j * 8))
			}
		}
	}
}

// Batch generates n documents starting at seq.
// The returned slice is reused across calls — callers must consume it
// (e.g. via InsertMany) before calling Batch again.
func (g *Generator) Batch(seq int64, n int) []interface{} {
	if cap(g.docsBuf) < n {
		g.docsBuf = make([]interface{}, n)
	}
	docs := g.docsBuf[:n]
	for i := range docs {
		docs[i] = g.Next(seq + int64(i))
	}
	return docs
}
