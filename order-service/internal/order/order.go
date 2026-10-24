package order

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound     = errors.New("order not found")
	ErrInvalidInput = errors.New("invalid input")
	// ErrUnavailable means a service we depend on is not responding: the client may retry.
	ErrUnavailable = errors.New("dependency unavailable")
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
)

type Item struct {
	ProductID      int64 `json:"product_id"`
	Quantity       int   `json:"quantity"`
	UnitPriceCents int64 `json:"unit_price_cents"`
}

type Order struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Status     Status    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	TotalCents int64     `json:"total_cents"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// Line is a line requested by the client, before its price is known.
type Line struct {
	ProductID int64
	Quantity  int
}

type Repository interface {
	// Create stores the order and its order.created event atomically.
	Create(ctx context.Context, o *Order) error
	FindByID(ctx context.Context, id int64) (Order, error)
	List(ctx context.Context) ([]Order, error)
	// Resolve moves a pending order to its final status; it is a no-op
	// if the order is already resolved (events may arrive twice).
	Resolve(ctx context.Context, id int64, status Status, reason string) error
}

// Users and Catalog are the other services as seen by the domain: the order service
// doesn't know they are reached over HTTP, and tests replace them with fakes.
type Users interface {
	Exists(ctx context.Context, id int64) (bool, error)
}

type Catalog interface {
	Price(ctx context.Context, productID int64) (cents int64, found bool, err error)
}

type Service struct {
	repo    Repository
	users   Users
	catalog Catalog
}

func NewService(repo Repository, users Users, catalog Catalog) *Service {
	return &Service{repo: repo, users: users, catalog: catalog}
}

// Place creates a pending order. Stock is reserved asynchronously by the
// product-service, which confirms or rejects through an event.
func (s *Service) Place(ctx context.Context, userID int64, lines []Line) (Order, error) {
	if userID <= 0 || len(lines) == 0 {
		return Order{}, fmt.Errorf("%w: user_id and at least one item are required", ErrInvalidInput)
	}
	for _, l := range lines {
		if l.Quantity <= 0 {
			return Order{}, fmt.Errorf("%w: quantity must be positive", ErrInvalidInput)
		}
	}

	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return Order{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if !exists {
		return Order{}, fmt.Errorf("%w: user %d does not exist", ErrInvalidInput, userID)
	}

	o := Order{UserID: userID, Status: StatusPending}
	for _, l := range lines {
		// The price is frozen at order time: it no longer changes if the catalog does.
		price, found, err := s.catalog.Price(ctx, l.ProductID)
		if err != nil {
			return Order{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		if !found {
			return Order{}, fmt.Errorf("%w: product %d does not exist", ErrInvalidInput, l.ProductID)
		}
		o.Items = append(o.Items, Item{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceCents: price})
		o.TotalCents += price * int64(l.Quantity)
	}

	if err := s.repo.Create(ctx, &o); err != nil {
		return Order{}, err
	}
	return o, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Order, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Order, error) {
	return s.repo.List(ctx)
}

func (s *Service) Confirm(ctx context.Context, id int64) error {
	return s.repo.Resolve(ctx, id, StatusConfirmed, "")
}

func (s *Service) Cancel(ctx context.Context, id int64, reason string) error {
	return s.repo.Resolve(ctx, id, StatusCancelled, reason)
}
