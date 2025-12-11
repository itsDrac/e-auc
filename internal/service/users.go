package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type UserServicer interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
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

func (us *UserService) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	exists, err := us.db.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if exists.ID != uuid.Nil {
		return "", errors.New("email already exists")
	}

	pwHash, err := utils.HashPassword(password)
	if err != nil {
		return "", err
	}
	userCreateParams := db.CreateUserParams{
		Name:     name,
		Email:    email,
		Password: pwHash,
	}
	user, err := us.db.CreateUser(ctx, userCreateParams)
	if err != nil {
		return "", err
	}

	return user.ID.String(), nil

}

func (us *UserService) GetUserByID(ctx context.Context, id string) (db.User, error) {
	if id == "" {
		return db.User{}, fmt.Errorf("id is empty")
	}
	userId, err := uuid.Parse(id)
	if err != nil {
		return db.User{}, fmt.Errorf("failed to parse user id")
	}
	user, err := us.db.GetUserByID(ctx, userId)
	if err != nil {
		slog.Error("error while getting user data from ID", "error", err.Error())
		return db.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}
