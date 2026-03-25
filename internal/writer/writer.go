package writer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"github.com/zlrrr/mongo-stream/internal/config"
	"github.com/zlrrr/mongo-stream/internal/distributor"
	"github.com/zlrrr/mongo-stream/internal/generator"
)

// Stats holds running counters for the writer.
type Stats struct {
	DocsInserted int64
	Errors       int64
	BytesWritten int64
}

// Writer orchestrates parallel document writes to MongoDB.
type Writer struct {
	client *mongo.Client
	cfg    *config.Config
	dist   distributor.Distributor
	// baseSeed is used to derive per-worker seeds; each worker gets its own
	// Generator so that rand.Rand is never shared across goroutines.
	baseSeed int64
	log      *zap.Logger
	stats    Stats
	// collCache maps each target to its pre-built *mongo.Collection handle,
	// avoiding repeated Database().Collection() calls on the hot path.
	collCache map[distributor.Target]*mongo.Collection
	// insertOpts is the pre-built InsertMany options (ordered=false).
	insertOpts *options.InsertManyOptionsBuilder
}

// New creates a Writer.
func New(
	client *mongo.Client,
	cfg *config.Config,
	dist distributor.Distributor,
	gen *generator.Generator,
	log *zap.Logger,
) *Writer {
	w := &Writer{
		client:     client,
		cfg:        cfg,
		dist:       dist,
		baseSeed:   gen.Seed(),
		log:        log,
		insertOpts: options.InsertMany().SetOrdered(false),
	}
	w.buildCollCache()
	return w
}

// buildCollCache pre-creates *mongo.Collection handles for every target.
// Collection objects in the Go driver are lightweight and goroutine-safe.
func (w *Writer) buildCollCache() {
	if w.client == nil {
		return // nil client in unit tests
	}
	targets := w.dist.Targets()
	w.collCache = make(map[distributor.Target]*mongo.Collection, len(targets))
	for _, t := range targets {
		w.collCache[t] = w.client.Database(t.DB).Collection(t.Collection)
	}
}

// Stats returns a snapshot of current counters.
func (w *Writer) Stats() Stats {
	return Stats{
		DocsInserted: atomic.LoadInt64(&w.stats.DocsInserted),
		Errors:       atomic.LoadInt64(&w.stats.Errors),
	}
}

// Run starts the write loop and blocks until ctx is cancelled or total docs written.
func (w *Writer) Run(ctx context.Context) error {
	// Print distribution table before starting.
	w.printDistributionTable()

	// Start progress logger.
	stopProgress := make(chan struct{})
	var wgProgress sync.WaitGroup
	wgProgress.Add(1)
	go func() {
		defer wgProgress.Done()
		w.runProgressLogger(ctx, stopProgress)
	}()

	// Work queue: each item is the starting seq for a batch.
	jobs := make(chan int64, w.cfg.Concurrency*4)

	var wgWorkers sync.WaitGroup
	for i := 0; i < w.cfg.Concurrency; i++ {
		// Each worker gets its own Generator so that rand.Rand is never shared
		// across goroutines (math/rand.(*Rand) is not goroutine-safe).
		workerGen := generator.New(w.baseSeed + int64(i+1))
		wgWorkers.Add(1)
		go func(g *generator.Generator) {
			defer wgWorkers.Done()
			w.workerLoop(ctx, jobs, g)
		}(workerGen)
	}

	// Feed jobs.
	var seq int64
	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wgWorkers.Wait()
			close(stopProgress)
			wgProgress.Wait()
			w.logFinalStats()
			return ctx.Err()
		default:
		}

		if w.cfg.Total > 0 {
			inserted := atomic.LoadInt64(&w.stats.DocsInserted)
			if inserted >= w.cfg.Total {
				close(jobs)
				wgWorkers.Wait()
				close(stopProgress)
				wgProgress.Wait()
				w.logFinalStats()
				return nil
			}
		}

		select {
		case <-ctx.Done():
			close(jobs)
			wgWorkers.Wait()
			close(stopProgress)
			wgProgress.Wait()
			w.logFinalStats()
			return ctx.Err()
		case jobs <- seq:
			seq += int64(w.cfg.BatchSize)
		}
	}
}

// workerLoop consumes jobs and inserts batches.
// gen is the per-worker Generator; it must not be shared with other goroutines.
func (w *Writer) workerLoop(ctx context.Context, jobs <-chan int64, gen *generator.Generator) {
	for seq := range jobs {
		if ctx.Err() != nil {
			return
		}

		batchSize := w.cfg.BatchSize
		if w.cfg.Total > 0 {
			remaining := w.cfg.Total - atomic.LoadInt64(&w.stats.DocsInserted)
			if remaining <= 0 {
				return
			}
			if int64(batchSize) > remaining {
				batchSize = int(remaining)
			}
		}

		docs := gen.Batch(seq, batchSize)
		target := w.dist.Next(seq / int64(w.cfg.BatchSize))

		coll := w.collCache[target]
		_, err := coll.InsertMany(ctx, docs, w.insertOpts)
		if err != nil {
			atomic.AddInt64(&w.stats.Errors, 1)
			w.log.Error("insertMany failed",
				zap.String("db", target.DB),
				zap.String("collection", target.Collection),
				zap.Int64("seq", seq),
				zap.Error(err),
			)
			continue
		}
		atomic.AddInt64(&w.stats.DocsInserted, int64(len(docs)))
	}
}

// runProgressLogger periodically logs write throughput.
func (w *Writer) runProgressLogger(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(w.cfg.LogInterval)
	defer ticker.Stop()

	var lastDocs int64
	lastTime := time.Now()

	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			docs := atomic.LoadInt64(&w.stats.DocsInserted)
			errs := atomic.LoadInt64(&w.stats.Errors)
			elapsed := t.Sub(lastTime).Seconds()
			rate := float64(docs-lastDocs) / elapsed
			lastDocs = docs
			lastTime = t

			w.log.Info("progress",
				zap.Int64("docs_total", docs),
				zap.Float64("rate_per_sec", rate),
				zap.Int64("errors", errs),
			)
		}
	}
}

// logFinalStats emits a summary at the end of the run.
func (w *Writer) logFinalStats() {
	w.log.Info("run complete",
		zap.Int64("docs_inserted", atomic.LoadInt64(&w.stats.DocsInserted)),
		zap.Int64("errors", atomic.LoadInt64(&w.stats.Errors)),
	)
}

// printDistributionTable prints the target distribution to the logger before writing starts.
func (w *Writer) printDistributionTable() {
	targets := w.dist.Targets()
	total := len(targets)
	w.log.Info("distribution plan",
		zap.String("mode", w.cfg.Distribution),
		zap.Int("databases", w.cfg.DBs),
		zap.Int("collections_per_db", w.cfg.Collections),
		zap.Int("total_targets", total),
	)

	// Print a compact ASCII table showing DB→collections.
	w.log.Info("--- distribution table ---")
	for d := 0; d < w.cfg.DBs; d++ {
		w.log.Info("db assignment",
			zap.String("database", "db_"+itoa(d)),
			zap.Int("collections", w.cfg.Collections),
			zap.Float64("share_pct", 100.0/float64(w.cfg.DBs)),
		)
	}
	w.log.Info("--------------------------")
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
