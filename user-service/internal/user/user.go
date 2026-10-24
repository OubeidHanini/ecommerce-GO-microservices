package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already used")
	ErrInvalidInput = errors.New("invalid input")
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository is declared here, on the domain side: the service does not depend on
// PostgreSQL and can be tested with an in-memory implementation.
type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id int64) (User, error)
	List(ctx context.Context) ([]User, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, name, email string) (User, error) {
	u := User{
		Name:  strings.TrimSpace(name),
		Email: strings.ToLower(strings.TrimSpace(email)),
	}
	if u.Name == "" {
		return User{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return User{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}

	if err := s.repo.Create(ctx, &u); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}
