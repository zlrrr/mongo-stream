// Package webui provides a lightweight HTTP monitoring server for mongo-stream.
// It has no external dependencies; everything is served from memory using the
// Go standard library.
package webui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

//go:embed ui/index.html
var indexHTML string

// StatsFn is a callback that returns the current (docsInserted, errors) counters.
// It is called from multiple goroutines and must be goroutine-safe.
type StatsFn func() (docsInserted, errors int64)

// Server is a read-only HTTP monitoring server.
// It is safe for concurrent use after construction.
type Server struct {
	mongoURI  string
	dist      string
	total     int64
	batchSize int
	statsFn   StatsFn
	startTime time.Time

	mu         sync.Mutex
	lastDocs   int64
	lastTime   time.Time
	ratePerSec float64
	runStatus  string
}

// New creates a Server.
//
//   - mongoURI   sanitised connection string shown in the UI
//   - dist       distribution mode ("uniform", "gaussian", "longtail")
//   - total      target document count (0 = unlimited)
//   - batchSize  insertMany batch size (used to compute batches-written)
//   - statsFn    callback returning live (docsInserted, errors) counters
func New(mongoURI, dist string, total int64, batchSize int, statsFn StatsFn) *Server {
	now := time.Now()
	return &Server{
		mongoURI:  mongoURI,
		dist:      dist,
		total:     total,
		batchSize: batchSize,
		statsFn:   statsFn,
		startTime: now,
		lastTime:  now,
		runStatus: "running",
	}
}

// SetStatus updates the status string displayed in the UI.
// Typical values: "running", "done", "stopped", "error".
func (s *Server) SetStatus(status string) {
	s.mu.Lock()
	s.runStatus = status
	s.mu.Unlock()
}

// Run starts the HTTP server on the given port and blocks until ctx is
// cancelled or the server encounters a fatal error.
func (s *Server) Run(ctx context.Context, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)

	go s.rateLoop(ctx)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Shutdown when ctx is cancelled.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webui ListenAndServe: %w", err)
	}
	return nil
}

// rateLoop samples stats every second to calculate docs/sec.
func (s *Server) rateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			docs, _ := s.statsFn()
			s.mu.Lock()
			elapsed := t.Sub(s.lastTime).Seconds()
			if elapsed > 0 {
				s.ratePerSec = float64(docs-s.lastDocs) / elapsed
			}
			s.lastDocs = docs
			s.lastTime = t
			s.mu.Unlock()
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

// statusResponse is the JSON payload served to the browser.
type statusResponse struct {
	MongoURI       string  `json:"mongo_uri"`
	Distribution   string  `json:"distribution"`
	Total          int64   `json:"total"`
	DocsInserted   int64   `json:"docs_inserted"`
	Errors         int64   `json:"errors"`
	RatePerSec     float64 `json:"rate_per_sec"`
	ElapsedSec     float64 `json:"elapsed_sec"`
	RunStatus      string  `json:"status"`
	ProgressPct    float64 `json:"progress_pct"`
	BatchesWritten int64   `json:"batches_written"`
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	docs, errs := s.statsFn()

	s.mu.Lock()
	rate := s.ratePerSec
	status := s.runStatus
	s.mu.Unlock()

	elapsed := time.Since(s.startTime).Seconds()

	var pct float64
	if s.total > 0 {
		pct = float64(docs) / float64(s.total) * 100
	}

	var batches int64
	if s.batchSize > 0 {
		batches = docs / int64(s.batchSize)
	}

	resp := statusResponse{
		MongoURI:       s.mongoURI,
		Distribution:   s.dist,
		Total:          s.total,
		DocsInserted:   docs,
		Errors:         errs,
		RatePerSec:     rate,
		ElapsedSec:     elapsed,
		RunStatus:      status,
		ProgressPct:    pct,
		BatchesWritten: batches,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	_ = json.NewEncoder(w).Encode(resp)
}
