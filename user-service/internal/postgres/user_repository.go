package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OubeidHanini/ecommerce-GO-microservices/user-service/internal/user"
)

const userSchema = `
CREATE TABLE IF NOT EXISTS users (
	id         BIGSERIAL PRIMARY KEY,
	name       TEXT        NOT NULL,
	email      TEXT        NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Migrate(ctx context.Context) error {
	_, err := r.db.Exec(ctx, userSchema)
	return err
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at`,
		u.Name, u.Email,
	).Scan(&u.ID, &u.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return user.ErrEmailTaken
	}
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (user.User, error) {
	var u user.User
	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, email, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []user.User{}
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
