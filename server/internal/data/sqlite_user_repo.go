package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// SQLiteUserRepo implements domain.UserRepo using SQLite
type SQLiteUserRepo struct {
	db *sql.DB
}

// NewSQLiteUserRepo creates a new SQLite user repository
func NewSQLiteUserRepo(db *sql.DB) *SQLiteUserRepo {
	return &SQLiteUserRepo{db: db}
}

func (r *SQLiteUserRepo) Create(ctx context.Context, username string) (*domain.User, error) {
	query := `INSERT INTO users (username, created_at, updated_at) VALUES (?, ?, ?)`
	now := time.Now()

	result, err := r.db.ExecContext(ctx, query, username, now, now)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &domain.User{
		ID:        id,
		Username:  username,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *SQLiteUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT id, username, created_at, updated_at FROM users WHERE id = ?`

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &u, nil
}

func (r *SQLiteUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, created_at, updated_at FROM users WHERE username = ?`

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &u, nil
}
