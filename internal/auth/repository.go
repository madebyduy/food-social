package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

type Repository interface {
	CreateUser(ctx context.Context, q database.Querier, input CreateUserInput) (*User, error)
	GetUserByLogin(ctx context.Context, q database.Querier, login string) (*User, error)
	GetUserByID(ctx context.Context, q database.Querier, id int64) (*User, error)
	CreateSession(ctx context.Context, q database.Querier, input CreateSessionInput) error
	RevokeSession(ctx context.Context, q database.Querier, tokenHash string) error
	GetSessionUserID(ctx context.Context, q database.Querier, tokenHash string, now time.Time) (int64, error)
}

type repository struct{}

func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

type CreateUserInput struct {
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
}

type CreateSessionInput struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
}

func (r *repository) CreateUser(ctx context.Context, q database.Querier, input CreateUserInput) (*User, error) {
	const query = `
		INSERT INTO users (username, email, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		RETURNING
			id, username, email, password_hash, display_name, role, status,
			created_at, updated_at`

	var u User
	err := q.QueryRowContext(ctx, query,
		input.Username, input.Email, input.PasswordHash, input.DisplayName,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.Conflict("username hoặc email đã tồn tại")
		}
		return nil, fmt.Errorf("auth.CreateUser scan: %w", err)
	}
	return &u, nil
}

func (r *repository) GetUserByLogin(ctx context.Context, q database.Querier, login string) (*User, error) {
	const query = `
		SELECT
			id, username, email, password_hash, display_name, role, status,
			created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
		  AND (lower(username) = lower($1) OR lower(email) = lower($1))`

	return scanUser(q.QueryRowContext(ctx, query, login), "auth.GetUserByLogin scan")
}

func (r *repository) GetUserByID(ctx context.Context, q database.Querier, id int64) (*User, error) {
	const query = `
		SELECT
			id, username, email, password_hash, display_name, role, status,
			created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	return scanUser(q.QueryRowContext(ctx, query, id), "auth.GetUserByID scan")
}

func (r *repository) CreateSession(ctx context.Context, q database.Querier, input CreateSessionInput) error {
	const query = `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := q.ExecContext(ctx, query, input.UserID, input.TokenHash, input.ExpiresAt); err != nil {
		return fmt.Errorf("auth.CreateSession exec: %w", err)
	}
	return nil
}

func (r *repository) RevokeSession(ctx context.Context, q database.Querier, tokenHash string) error {
	const query = `
		UPDATE sessions
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := q.ExecContext(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("auth.RevokeSession exec: %w", err)
	}
	return nil
}

func (r *repository) GetSessionUserID(ctx context.Context, q database.Querier, tokenHash string, now time.Time) (int64, error) {
	const query = `
		SELECT s.user_id
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > $2
		  AND u.deleted_at IS NULL
		  AND u.status = 'ACTIVE'`

	var userID int64
	if err := q.QueryRowContext(ctx, query, tokenHash, now).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apperr.ErrNotFound
		}
		return 0, fmt.Errorf("auth.GetSessionUserID scan: %w", err)
	}
	return userID, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner, wrap string) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	return &u, nil
}
