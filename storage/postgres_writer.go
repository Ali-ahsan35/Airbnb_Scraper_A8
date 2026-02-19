package storage

import (
	"airbnb-scraper/config"
	"airbnb-scraper/models"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresWriter struct {
	pool *pgxpool.Pool
}

func NewPostgresWriter(cfg *config.Config) (*PostgresWriter, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect postgres: %w", err)
	}

	return &PostgresWriter{pool: pool}, nil
}

func (w *PostgresWriter) Close() {
	if w.pool != nil {
		w.pool.Close()
	}
}

func (w *PostgresWriter) EnsureSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sql := `
	CREATE TABLE IF NOT EXISTS listings (
		id BIGSERIAL PRIMARY KEY,
		platform TEXT NOT NULL,
		title TEXT NOT NULL,
		price NUMERIC(12,2),
		raw_price TEXT,
		location TEXT,
		rating NUMERIC(3,2),
		url TEXT NOT NULL UNIQUE,
		description TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_listings_price ON listings(price);
	CREATE INDEX IF NOT EXISTS idx_listings_location ON listings(location);
	`

	if _, err := w.pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}

	return nil
}

func (w *PostgresWriter) WriteBatch(listings []models.Listing) error {
	if len(listings) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	insertSQL := `
	INSERT INTO listings (platform, title, price, raw_price, location, rating, url, description)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (url) DO NOTHING;
	`

	enqueued := 0
	for _, l := range listings {
		title := strings.TrimSpace(l.Title)
		url := strings.TrimSpace(l.URL)
		if title == "" || url == "" {
			continue
		}

		batch.Queue(
			insertSQL,
			strings.TrimSpace(strings.ToLower(l.Platform)),
			title,
			l.Price,
			strings.TrimSpace(l.RawPrice),
			strings.TrimSpace(l.Location),
			l.Rating,
			url,
			strings.TrimSpace(l.Description),
		)
		enqueued++
	}

	if enqueued == 0 {
		return nil
	}

	results := w.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < enqueued; i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch insert failed at row %d: %w", i, err)
		}
	}

	return nil
}
