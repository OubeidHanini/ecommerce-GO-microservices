package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("product not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Prices are integer cents: floats accumulate rounding errors on money.
type Product struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	PriceCents int64     `json:"price_cents"`
	Stock      int       `json:"stock"`
	CreatedAt  time.Time `json:"created_at"`
}

// Line is an order line to reserve.
type Line struct {
	ProductID int64
	Quantity  int
}

type Repository interface {
	Create(ctx context.Context, p *Product) error
	FindByID(ctx context.Context, id int64) (Product, error)
	List(ctx context.Context) ([]Product, error)
	// ReserveStock reserves every line or none, and records the outcome event
	// in the same transaction. Replaying the same order has no effect.
	ReserveStock(ctx context.Context, orderID int64, lines []Line) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, name string, priceCents int64, stock int) (Product, error) {
	p := Product{Name: strings.TrimSpace(name), PriceCents: priceCents, Stock: stock}
	if p.Name == "" {
		return Product{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if p.PriceCents <= 0 {
		return Product{}, fmt.Errorf("%w: price_cents must be positive", ErrInvalidInput)
	}
	if p.Stock < 0 {
		return Product{}, fmt.Errorf("%w: stock cannot be negative", ErrInvalidInput)
	}

	if err := s.repo.Create(ctx, &p); err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repo.List(ctx)
}

func (s *Service) Reserve(ctx context.Context, orderID int64, lines []Line) error {
	if orderID <= 0 || len(lines) == 0 {
		return fmt.Errorf("%w: order %d has no lines", ErrInvalidInput, orderID)
	}
	for _, l := range lines {
		if l.Quantity <= 0 {
			return fmt.Errorf("%w: quantity must be positive", ErrInvalidInput)
		}
	}
	return s.repo.ReserveStock(ctx, orderID, lines)
}
