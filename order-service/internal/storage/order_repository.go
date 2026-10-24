package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OubeidHanini/ecommerce-GO-microservices/order-service/internal/order"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/events"
	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/outbox"
)

const orderSchema = `
CREATE TABLE IF NOT EXISTS orders (
	id          BIGSERIAL PRIMARY KEY,
	user_id     BIGINT      NOT NULL,
	status      TEXT        NOT NULL,
	reason      TEXT        NOT NULL DEFAULT '',
	total_cents BIGINT      NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
	order_id         BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	product_id       BIGINT NOT NULL,
	quantity         INT    NOT NULL CHECK (quantity > 0),
	unit_price_cents BIGINT NOT NULL
)`

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Migrate(ctx context.Context) error {
	if _, err := r.db.Exec(ctx, orderSchema); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, outbox.Schema)
	return err
}

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO orders (user_id, status, total_cents) VALUES ($1, $2, $3) RETURNING id, created_at`,
			o.UserID, o.Status, o.TotalCents,
		).Scan(&o.ID, &o.CreatedAt)
		if err != nil {
			return err
		}

		evt := events.OrderCreatedEvent{OrderID: o.ID}
		for _, it := range o.Items {
			_, err := tx.Exec(ctx,
				`INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES ($1, $2, $3, $4)`,
				o.ID, it.ProductID, it.Quantity, it.UnitPriceCents)
			if err != nil {
				return err
			}
			evt.Items = append(evt.Items, events.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
		}
		return outbox.Add(ctx, tx, events.OrderCreated, evt)
	})
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (order.Order, error) {
	var o order.Order
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, status, reason, total_cents, created_at FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.Reason, &o.TotalCents, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return order.Order{}, order.ErrNotFound
	}
	if err != nil {
		return order.Order{}, err
	}

	items, err := r.items(ctx, `WHERE order_id = $1`, id)
	if err != nil {
		return order.Order{}, err
	}
	o.Items = items[o.ID]
	return o, nil
}

func (r *OrderRepository) List(ctx context.Context) ([]order.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, status, reason, total_cents, created_at FROM orders ORDER BY id`)
	if err != nil {
		return nil, err
	}
	orders, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (order.Order, error) {
		var o order.Order
		err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.Reason, &o.TotalCents, &o.CreatedAt)
		return o, err
	})
	if err != nil {
		return nil, err
	}

	// One query for all lines rather than one per order (N+1 problem).
	items, err := r.items(ctx, "")
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Items = items[orders[i].ID]
	}
	return orders, nil
}

// items returns order lines grouped by order.
func (r *OrderRepository) items(ctx context.Context, where string, args ...any) (map[int64][]order.Item, error) {
	rows, err := r.db.Query(ctx,
		`SELECT order_id, product_id, quantity, unit_price_cents FROM order_items `+where+` ORDER BY order_id`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byOrder := map[int64][]order.Item{}
	for rows.Next() {
		var orderID int64
		var it order.Item
		if err := rows.Scan(&orderID, &it.ProductID, &it.Quantity, &it.UnitPriceCents); err != nil {
			return nil, err
		}
		byOrder[orderID] = append(byOrder[orderID], it)
	}
	return byOrder, rows.Err()
}

func (r *OrderRepository) Resolve(ctx context.Context, id int64, status order.Status, reason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $2, reason = $3 WHERE id = $1 AND status = $4`,
		id, status, reason, order.StatusPending)
	return err
}
