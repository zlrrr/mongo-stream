package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/zlrrr/mongo-stream/internal/config"
	"github.com/zlrrr/mongo-stream/internal/connector"
	"github.com/zlrrr/mongo-stream/internal/distributor"
	"github.com/zlrrr/mongo-stream/internal/generator"
	"github.com/zlrrr/mongo-stream/internal/logger"
	"github.com/zlrrr/mongo-stream/internal/writer"
)

var cfg = config.Default()

// rootCmd is the main cobra command.
var rootCmd = &cobra.Command{
	Use:   "mongo-stream",
	Short: "Stream random documents into MongoDB for load testing",
	Long: `mongo-stream connects to a MongoDB deployment and continuously writes
randomly-generated documents across a configurable set of databases and collections.

Distribution modes:
  uniform   – writes are spread evenly across all targets (default)
  gaussian  – coming soon
  longtail  – coming soon`,
	RunE: runStream,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.Flags()
	f.StringVar(&cfg.URI, "uri", cfg.URI, "MongoDB connection URI")
	f.StringVar(&cfg.Username, "username", cfg.Username, "Auth username")
	f.StringVar(&cfg.Password, "password", cfg.Password, "Auth password")
	f.StringVar(&cfg.AuthSource, "auth-source", cfg.AuthSource, "Auth source database")
	f.IntVar(&cfg.DBs, "dbs", cfg.DBs, "Number of databases")
	f.IntVar(&cfg.Collections, "collections", cfg.Collections, "Collections per database")
	f.Int64Var(&cfg.Total, "total", cfg.Total, "Total documents to insert (0 = unlimited)")
	f.IntVar(&cfg.BatchSize, "batch", cfg.BatchSize, "Insert batch size")
	f.IntVar(&cfg.Concurrency, "concurrency", cfg.Concurrency, "Parallel writer goroutines")
	f.DurationVar(&cfg.LogInterval, "log-interval", cfg.LogInterval, "Progress log interval")
	f.StringVar(&cfg.Distribution, "distribution", cfg.Distribution, "Distribution mode: uniform|gaussian|longtail")
}

func runStream(cmd *cobra.Command, _ []string) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	log := logger.Must(false)
	defer func() { _ = log.Sync() }()

	log.Info("mongo-stream starting",
		zap.String("uri", sanitize(cfg.URI)),
		zap.Int("dbs", cfg.DBs),
		zap.Int("collections", cfg.Collections),
		zap.Int64("total", cfg.Total),
		zap.String("distribution", cfg.Distribution),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := connector.New(ctx, cfg.URI, cfg.Username, cfg.Password, cfg.AuthSource)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(disconnectCtx)
	}()

	log.Info("connected to MongoDB")

	var dist distributor.Distributor
	switch cfg.Distribution {
	case "uniform":
		dist = distributor.NewUniform(cfg.DBs, cfg.Collections)
	default:
		return fmt.Errorf("distribution %q not yet implemented", cfg.Distribution)
	}

	gen := generator.New(0) // time-seeded
	w := writer.New(client, cfg, dist, gen, log)

	if err := w.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("writer: %w", err)
	}

	return nil
}

// sanitize removes credentials from a URI for logging.
func sanitize(uri string) string {
	for i, ch := range uri {
		if ch == '@' {
			for j := 0; j < i; j++ {
				if uri[j] == '/' && j+1 < len(uri) && uri[j+1] == '/' {
					return uri[:j+2] + "***" + uri[i:]
				}
			}
		}
	}
	return uri
}
