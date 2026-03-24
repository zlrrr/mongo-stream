package config

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.DBs != 10 {
		t.Fatalf("expected 10 dbs, got %d", c.DBs)
	}
	if c.Collections != 20 {
		t.Fatalf("expected 20 collections, got %d", c.Collections)
	}
	if c.Distribution != "uniform" {
		t.Fatalf("expected uniform distribution, got %s", c.Distribution)
	}
}

func TestValidate_OK(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"empty uri", func(c *Config) { c.URI = "" }, "URI must not be empty"},
		{"zero dbs", func(c *Config) { c.DBs = 0 }, "--dbs must be > 0"},
		{"neg collections", func(c *Config) { c.Collections = -1 }, "--collections must be > 0"},
		{"zero batch", func(c *Config) { c.BatchSize = 0 }, "--batch must be > 0"},
		{"zero concurrency", func(c *Config) { c.Concurrency = 0 }, "--concurrency must be > 0"},
		{"zero interval", func(c *Config) { c.LogInterval = 0 }, "--log-interval must be > 0"},
		{"bad distribution", func(c *Config) { c.Distribution = "random" }, "--distribution must be"},
		{"neg total", func(c *Config) { c.Total = -1 }, "--total must be >= 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Default()
			tt.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestValidate_UnlimitedTotal(t *testing.T) {
	c := Default()
	c.Total = 0
	c.LogInterval = time.Second
	if err := c.Validate(); err != nil {
		t.Fatalf("total=0 should be valid: %v", err)
	}
}
