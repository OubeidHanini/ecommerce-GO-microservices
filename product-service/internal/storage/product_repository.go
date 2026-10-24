package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/outbox"
	"github.com/OubeidHanini/ecommerce-GO-microservices/product-service/internal/product"
)

const productSchema = `
CREATE TABLE IF NOT EXISTS products (
	id          BIGSERIAL PRIMARY KEY,
	name        TEXT        NOT NULL,
	price_cents BIGINT      NOT NULL CHECK (price_cents > 0),
	stock       INT         NOT NULL CHECK (stock >= 0),
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS processed_orders (
	order_id BIGINT PRIMARY KEY
)`

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Migrate(ctx context.Context) error {
	if _, err := r.db.Exec(ctx, productSchema); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, outbox.Schema)
	return err
}

func (r *ProductRepository) Create(ctx context.Context, p *product.Product) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO products (name, price_cents, stock) VALUES ($1, $2, $3) RETURNING id, created_at`,
		p.Name, p.PriceCents, p.Stock,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *ProductRepository) FindByID(ctx context.Context, id int64) (product.Product, error) {
	var p product.Product
	err := r.db.QueryRow(ctx,
		`SELECT id, name, price_cents, stock, created_at FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.PriceCents, &p.Stock, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return product.Product{}, product.ErrNotFound
	}
	return p, err
}

func (r *ProductRepository) List(ctx context.Context) ([]product.Product, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, price_cents, stock, created_at FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []product.Product{}
	for rows.Next() {
		var p product.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceCents, &p.Stock, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) ReserveStock(ctx context.Context, orderID int64, lines []product.Line) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// The primary key makes handling idempotent: RabbitMQ may deliver a message more than once.
		tag, err := tx.Exec(ctx,
			`INSERT INTO processed_orders (order_id) VALUES ($1) ON CONFLICT DO NOTHING`, orderID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}

		reason, err := decrementStock(ctx, tx, lines)
		if err != nil {
			return err
		}
		if reason != "" {
			return outbox.Add(ctx, tx, events.StockRejected, events.StockResultEvent{OrderID: orderID, Reason: reason})
		}
		return outbox.Add(ctx, tx, events.StockReserved, events.StockResultEvent{OrderID: orderID})
	})
}

// decrementStock takes stock for every line or none (savepoint) and returns the
// reason for a rejection. The "stock >= quantity" check lives in the UPDATE itself,
// so two concurrent orders cannot sell the same item.
func decrementStock(ctx context.Context, tx pgx.Tx, lines []product.Line) (reason string, err error) {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer sp.Rollback(ctx) // no-op once the savepoint is committed

	for _, l := range lines {
		tag, err := sp.Exec(ctx,
			`UPDATE products SET stock = stock - $2 WHERE id = $1 AND stock >= $2`, l.ProductID, l.Quantity)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Sprintf("product %d is unavailable or out of stock", l.ProductID), nil
		}
	}
	return "", sp.Commit(ctx)
}
