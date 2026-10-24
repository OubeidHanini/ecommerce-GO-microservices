package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect retries several times: while the stack starts, the database
// may not accept connections yet.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	const attempts = 10
	var err error
	for i := 1; i <= attempts; i++ {
		var pool *pgxpool.Pool
		pool, err = pgxpool.New(ctx, cfg.url())
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				return pool, nil
			}
			pool.Close()
		}
		log.Printf("postgres not ready (attempt %d/%d): %v", i, attempts, err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("connect to postgres: %w", err)
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
