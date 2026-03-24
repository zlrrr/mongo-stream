package connector

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const connectTimeout = 10 * time.Second

// New creates and validates a MongoDB client.
// It applies optional username/password credentials when provided.
func New(ctx context.Context, uri, username, password, authSource string) (*mongo.Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongo URI must not be empty")
	}

	opts := options.Client().ApplyURI(uri)

	if username != "" {
		cred := options.Credential{
			Username:   username,
			Password:   password,
			AuthSource: authSource,
		}
		opts.SetAuth(cred)
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}

	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping failed (URI: %s): %w", sanitizeURI(uri), err)
	}

	return client, nil
}

// sanitizeURI removes credentials from a URI for safe logging.
func sanitizeURI(uri string) string {
	// Simple approach: if URI contains '@', hide the userinfo part.
	for i, ch := range uri {
		if ch == '@' {
			// find scheme end
			for j := 0; j < i; j++ {
				if uri[j] == '/' && j+1 < len(uri) && uri[j+1] == '/' {
					return uri[:j+2] + "***:***" + uri[i:]
				}
			}
		}
	}
	return uri
}
