// Package outbox guarantees an event is published if, and only if, the business
// transaction that produced it commits (transactional outbox pattern).
// Without it, a crash between COMMIT and publish would lose the event.
package outbox

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Schema = `
CREATE TABLE IF NOT EXISTS outbox (
	id          BIGSERIAL PRIMARY KEY,
	routing_key TEXT        NOT NULL,
	payload     JSONB       NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// Add records the event in the current transaction.
func Add(ctx context.Context, tx pgx.Tx, routingKey string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox (routing_key, payload) VALUES ($1, $2)`, routingKey, payload)
	return err
}

type Publisher interface {
	Publish(ctx context.Context, routingKey string, event any) error
}

const batchSize = 100

// Relay publishes pending events until ctx is cancelled.
// Delivery is at-least-once: consumers must be idempotent.
func Relay(ctx context.Context, db *pgxpool.Pool, pub Publisher) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		n, err := flush(ctx, db, pub)
		if err != nil && ctx.Err() == nil {
			log.Printf("outbox relay: %v", err)
		}
		// A full batch means more events are waiting: drain them right away instead of
		// waiting for the next tick, which capped throughput at 200 events/s under load.
		if n == batchSize && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type pending struct {
	id         int64
	routingKey string
	payload    json.RawMessage
}

// flush publishes one batch and returns how many events it published.
func flush(ctx context.Context, db *pgxpool.Pool, pub Publisher) (int, error) {
	var published []int64
	var publishErr error
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		// SKIP LOCKED: several service instances can relay without publishing the same batch twice.
		rows, err := tx.Query(ctx,
			`SELECT id, routing_key, payload FROM outbox ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, batchSize)
		if err != nil {
			return err
		}
		batch, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pending, error) {
			var p pending
			err := row.Scan(&p.id, &p.routingKey, &p.payload)
			return p, err
		})
		if err != nil {
			return err
		}

		for _, p := range batch {
			if publishErr = pub.Publish(ctx, p.routingKey, p.payload); publishErr != nil {
				break // still commit the deletion of the ones already published
			}
			published = append(published, p.id)
		}
		_, err = tx.Exec(ctx, `DELETE FROM outbox WHERE id = ANY($1)`, published)
		return err
	})
	if err != nil {
		return 0, err
	}
	return len(published), publishErr
}
