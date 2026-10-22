package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
}

// UserStore finds or creates the account behind a verified phone number.
type UserStore interface {
	UpsertByPhone(ctx context.Context, phone string) (User, error)
	ByID(ctx context.Context, id string) (User, error)
}

type PGUsers struct{ Pool *pgxpool.Pool }

func (p PGUsers) UpsertByPhone(ctx context.Context, phone string) (User, error) {
	var u User
	err := p.Pool.QueryRow(ctx, `
		INSERT INTO users (phone) VALUES ($1)
		ON CONFLICT (phone) DO UPDATE SET last_login = now()
		RETURNING id::text, phone`, phone).Scan(&u.ID, &u.Phone)
	return u, err
}

func (p PGUsers) ByID(ctx context.Context, id string) (User, error) {
	var u User
	err := p.Pool.QueryRow(ctx, `SELECT id::text, phone FROM users WHERE id = $1::uuid`, id).Scan(&u.ID, &u.Phone)
	return u, err
}
