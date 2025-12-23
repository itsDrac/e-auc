package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
)

type UserServicer interface {
	GetUserByID(ctx context.Context, id string) (db.User, error)
}

type UserService struct {
	db db.Querier // We'll be using code genrated by sqlc here
}

func NewUserService(db db.Querier) (*UserService, error) {
	return &UserService{
		db: db,
	}, nil
}

func (us *UserService) GetUserByID(ctx context.Context, id string) (db.User, error) {
	if id == "" {
		return db.User{}, ErrIDMissing
	}
	userId, err := uuid.Parse(id)
	if err != nil {
		return db.User{}, fmt.Errorf("failed to parse user id")
	}
	user, err := us.db.GetUserByID(ctx, userId)
	if err != nil {
		slog.Error("error while getting user data from ID", "error", err.Error())
		return db.User{}, ErrUserNotFound
	}

	return user, nil
}
