package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type UserServicer interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
}

type Services struct {
	UserService UserServicer
	ProductService interface{}
}

func NewServices(db db.Querier) *Services {
	return &Services{
		UserService: NewUserService(db),
	}
}

type UserService struct {
	db db.Querier // We'll be using code genrated by sqlc here
}

func NewUserService(db db.Querier) *UserService {
	return &UserService{
		db: db,
	}
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
