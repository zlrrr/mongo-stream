package writer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/zlrrr/mongo-stream/internal/config"
	"github.com/zlrrr/mongo-stream/internal/distributor"
	"github.com/zlrrr/mongo-stream/internal/generator"
)

// newTestWriter builds a Writer with a nil mongo client for unit testing.
// Tests that don't call Run (which would actually insert) can use this safely.
func newTestWriter(cfg *config.Config) *Writer {
	dist := distributor.NewUniform(cfg.DBs, cfg.Collections)
	gen := generator.New(42)
	log := zap.NewNop()
	return New(nil, cfg, dist, gen, log)
}

func TestStats_Initial(t *testing.T) {
	cfg := config.Default()
	w := newTestWriter(cfg)
	s := w.Stats()
	if s.DocsInserted != 0 {
		t.Errorf("expected 0 docs, got %d", s.DocsInserted)
	}
	if s.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", s.Errors)
	}
}

func TestStats_AtomicIncrement(t *testing.T) {
	cfg := config.Default()
	w := newTestWriter(cfg)
	atomic.AddInt64(&w.stats.DocsInserted, 500)
	s := w.Stats()
	if s.DocsInserted != 500 {
		t.Errorf("expected 500, got %d", s.DocsInserted)
	}
}

func TestProgressLogger_Runs(t *testing.T) {
	cfg := config.Default()
	cfg.LogInterval = 50 * time.Millisecond
	w := newTestWriter(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		w.runProgressLogger(ctx, stop)
		close(done)
	}()

	// Let it tick a couple of times then stop.
	time.Sleep(150 * time.Millisecond)
	close(stop)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("progress logger did not stop")
	}
}

func TestPrintDistributionTable_NoPanic(t *testing.T) {
	cfg := config.Default()
	w := newTestWriter(cfg)
	// Should not panic.
	w.printDistributionTable()
}
