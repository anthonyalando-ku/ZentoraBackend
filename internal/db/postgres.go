package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB() (*pgxpool.Pool, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// Parse config once — no point re-parsing on every retry attempt.
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	// ---------------------------------------------------------------------------
	// Pool sizing
	//
	// MaxConns: PostgreSQL's default max_connections is 100. With one app server
	// this cap of 25 leaves headroom for migrations, admin tools, and future
	// horizontal scaling without exhausting the server. If you run multiple
	// replicas behind a load balancer, set this via an env var and divide the
	// total Postgres budget across all instances.
	//
	// MinConns: Keep 2 warm connections so the first request after an idle period
	// does not pay connection establishment latency. Raising this above ~5 wastes
	// server file descriptors for no throughput gain.
	//
	// MaxConnLifetime: Recycle connections every 30 minutes. Prevents stale
	// connections accumulating after a Postgres restart or network hiccup.
	//
	// MaxConnIdleTime: Release connections idle for more than 5 minutes back to
	// the OS. Keeps the pool lean during off-peak hours.
	//
	// HealthCheckPeriod: pgxpool background-checks idle connections every minute
	// and evicts dead ones before your handlers ever try to use them.
	// ---------------------------------------------------------------------------
	config.MaxConns           = 25
	config.MinConns           = 2
	config.MaxConnLifetime    = 30 * time.Minute
	config.MaxConnIdleTime    = 5 * time.Minute
	config.HealthCheckPeriod  = 1 * time.Minute

	const (
		maxRetries   = 5
		initialDelay = 2 * time.Second
	)

	delay := initialDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[DB] attempt %d/%d: connecting to database", attempt, maxRetries)

		// Each attempt gets its own scoped context — a defer inside a loop would
		// only fire at function return, leaking all intermediate contexts.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, connectErr := pgxpool.NewWithConfig(ctx, config)
		cancel()

		if connectErr != nil {
			log.Printf("[DB] attempt %d failed (connect): %v", attempt, connectErr)
		} else {
			pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingErr := pool.Ping(pingCtx)
			pingCancel()

			if pingErr == nil {
				log.Printf("[DB] connected successfully on attempt %d", attempt)
				return pool, nil
			}

			pool.Close()
			log.Printf("[DB] attempt %d failed (ping): %v", attempt, pingErr)
		}

		if attempt < maxRetries {
			log.Printf("[DB] retrying in %s", delay)
			time.Sleep(delay)
			delay *= 2 // exponential backoff
		}
	}

	return nil, fmt.Errorf("db: failed to connect after %d attempts", maxRetries)
}