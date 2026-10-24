package order

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	created []Order
}

func (r *fakeRepo) Create(_ context.Context, o *Order) error {
	o.ID = int64(len(r.created) + 1)
	r.created = append(r.created, *o)
	return nil
}
func (r *fakeRepo) FindByID(context.Context, int64) (Order, error)       { return Order{}, ErrNotFound }
func (r *fakeRepo) List(context.Context) ([]Order, error)                { return nil, nil }
func (r *fakeRepo) Resolve(context.Context, int64, Status, string) error { return nil }

type fakeUsers struct {
	exists bool
	err    error
}

func (u fakeUsers) Exists(context.Context, int64) (bool, error) { return u.exists, u.err }

type fakeCatalog map[int64]int64 // productID -> price in cents

func (c fakeCatalog) Price(_ context.Context, id int64) (int64, bool, error) {
	price, ok := c[id]
	return price, ok, nil
}

func TestPlaceComputesTotalWithPricesFromCatalog(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeUsers{exists: true}, fakeCatalog{1: 1000, 2: 250})

	o, err := svc.Place(context.Background(), 7, []Line{{ProductID: 1, Quantity: 2}, {ProductID: 2, Quantity: 3}})
	if err != nil {
		t.Fatalf("Place: %v", err)
	}
	if o.TotalCents != 2750 {
		t.Errorf("total = %d, want 2750", o.TotalCents)
	}
	if o.Status != StatusPending {
		t.Errorf("status = %q, want %q", o.Status, StatusPending)
	}
	if len(repo.created) != 1 {
		t.Errorf("orders saved = %d, want 1", len(repo.created))
	}
}

func TestPlaceRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		users   fakeUsers
		lines   []Line
		wantErr error
	}{
		{"no items", fakeUsers{exists: true}, nil, ErrInvalidInput},
		{"zero quantity", fakeUsers{exists: true}, []Line{{ProductID: 1, Quantity: 0}}, ErrInvalidInput},
		{"unknown user", fakeUsers{exists: false}, []Line{{ProductID: 1, Quantity: 1}}, ErrInvalidInput},
		{"unknown product", fakeUsers{exists: true}, []Line{{ProductID: 99, Quantity: 1}}, ErrInvalidInput},
		{"user service down", fakeUsers{err: errors.New("timeout")}, []Line{{ProductID: 1, Quantity: 1}}, ErrUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := NewService(repo, tt.users, fakeCatalog{1: 1000})

			_, err := svc.Place(context.Background(), 7, tt.lines)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
			if len(repo.created) != 0 {
				t.Error("an invalid order must not be saved")
			}
		})
	}
}
