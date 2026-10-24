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

// Relay publishes pending events until ctx is cancelled.
// Delivery is at-least-once: consumers must be idempotent.
func Relay(ctx context.Context, db *pgxpool.Pool, pub Publisher) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := flush(ctx, db, pub); err != nil && ctx.Err() == nil {
			log.Printf("outbox relay: %v", err)
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

func flush(ctx context.Context, db *pgxpool.Pool, pub Publisher) error {
	var publishErr error
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		// SKIP LOCKED: several service instances can relay without publishing the same batch twice.
		rows, err := tx.Query(ctx,
			`SELECT id, routing_key, payload FROM outbox ORDER BY id LIMIT 100 FOR UPDATE SKIP LOCKED`)
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
			if _, err := tx.Exec(ctx, `DELETE FROM outbox WHERE id = $1`, p.id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return publishErr
}
