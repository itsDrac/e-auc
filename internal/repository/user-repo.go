package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/itsDrac/e-auc/internal/db"
	"github.com/itsDrac/e-auc/internal/types"
)

type IUserrepo interface {
	GetUserByEmail(ctx context.Context, email string) (*types.User, error)
	CreateUser(ctx context.Context, email, password, name string) (string, error)
}

type Userrepo struct {
	db *db.DB
}

func NewUserrepo(db *db.DB) *Userrepo {
	return &Userrepo{
		db: db,
	}
}

func (ur *Userrepo) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	const q = `
		SELECT
			u.id,
			u.email,
			u.password,
			u.name,
			u.created_at,
			u.updated_at,
			u.deleted_at 
		FROM users u
		WHERE u.email = $1
		LIMIT 1;
	`

	var (
		u         types.User
		pw        sql.NullString
		deletedAt sql.NullTime
	)

	err := ur.db.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.Email,
		&pw,
		&u.Name,
		&u.CreatedAt,
		&u.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Set password (or mask it)
	if pw.Valid {
		u.Password = pw.String
	}

	if deletedAt.Valid {
		t := deletedAt.Time
		u.DeletedAt = &t
	}

	return &u, nil
}

func (ur *Userrepo) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	const q = `
		INSERT INTO users (
			email,
			password,
			name,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id;
	`

	var userID string

	err := ur.db.QueryRow(ctx, q, email, password, name).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}
