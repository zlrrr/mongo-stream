package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() *Server {
	return New("mongodb://localhost:27017", "uniform", 1000, 100, func() (int64, int64) {
		return 500, 2
	})
}

func TestHandleStatus_Fields(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)

	var resp statusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI = %q", resp.MongoURI)
	}
	if resp.Distribution != "uniform" {
		t.Errorf("Distribution = %q", resp.Distribution)
	}
	if resp.DocsInserted != 500 {
		t.Errorf("DocsInserted = %d, want 500", resp.DocsInserted)
	}
	if resp.Errors != 2 {
		t.Errorf("Errors = %d, want 2", resp.Errors)
	}
	if resp.Total != 1000 {
		t.Errorf("Total = %d, want 1000", resp.Total)
	}
	if resp.ProgressPct < 49 || resp.ProgressPct > 51 {
		t.Errorf("ProgressPct = %f, want ~50", resp.ProgressPct)
	}
	if resp.BatchesWritten != 5 {
		t.Errorf("BatchesWritten = %d, want 5", resp.BatchesWritten)
	}
	if resp.RunStatus != "running" {
		t.Errorf("RunStatus = %q, want running", resp.RunStatus)
	}
}

func TestHandleStatus_ContentType(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestHandleIndex_HTML(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.handleIndex(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Error("expected non-empty HTML body")
	}
}

func TestSetStatus(t *testing.T) {
	srv := newTestServer()
	srv.SetStatus("done")
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)
	var resp statusResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.RunStatus != "done" {
		t.Errorf("RunStatus = %q, want done", resp.RunStatus)
	}
}

func TestUnlimitedTotal_NoPct(t *testing.T) {
	srv := New("mongodb://localhost:27017", "uniform", 0, 100, func() (int64, int64) { return 999, 0 })
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)
	var resp statusResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.ProgressPct != 0 {
		t.Errorf("ProgressPct = %f, want 0 for unlimited run", resp.ProgressPct)
	}
}

func TestRun_ServesOnPort(t *testing.T) {
	srv := newTestServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx, 0) }()

	// Cancel immediately; server should shut down without error.
	cancel()
	if err := <-errCh; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
