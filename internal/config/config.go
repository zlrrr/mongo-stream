package config

import (
	"errors"
	"time"
)

// Config holds all runtime configuration for mongo-stream.
type Config struct {
	URI          string
	Username     string
	Password     string
	AuthSource   string
	DBs          int
	Collections  int
	Total        int64 // 0 = unlimited
	BatchSize    int
	Concurrency  int
	LogInterval  time.Duration
	Distribution string // "uniform" | "gaussian" | "longtail"
	WebUI        bool   // enable the web monitoring UI
	WebUIPort    int    // HTTP port for the web UI (default 8080)
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		URI:          "mongodb://localhost:27017",
		AuthSource:   "admin",
		DBs:          10,
		Collections:  20,
		Total:        0,
		BatchSize:    100,
		Concurrency:  4,
		LogInterval:  5 * time.Second,
		Distribution: "uniform",
		WebUI:        false,
		WebUIPort:    8080,
	}
}

// Validate checks that the config is self-consistent.
func (c *Config) Validate() error {
	if c.URI == "" {
		return errors.New("URI must not be empty")
	}
	if c.DBs <= 0 {
		return errors.New("--dbs must be > 0")
	}
	if c.Collections <= 0 {
		return errors.New("--collections must be > 0")
	}
	if c.BatchSize <= 0 {
		return errors.New("--batch must be > 0")
	}
	if c.Concurrency <= 0 {
		return errors.New("--concurrency must be > 0")
	}
	if c.LogInterval <= 0 {
		return errors.New("--log-interval must be > 0")
	}
	supported := map[string]bool{"uniform": true, "gaussian": true, "longtail": true}
	if !supported[c.Distribution] {
		return errors.New("--distribution must be one of: uniform, gaussian, longtail")
	}
	if c.Total < 0 {
		return errors.New("--total must be >= 0 (0 = unlimited)")
	}
	if c.WebUI && (c.WebUIPort < 1 || c.WebUIPort > 65535) {
		return errors.New("--webui-port must be between 1 and 65535")
	}
	return nil
}
