package service

import (
	"context"
	"errors"

	"github.com/itsDrac/e-auc/internal/repository"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type UserServicer interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
}

type Services struct {
	UserService UserServicer
	ProductService interface{}
}

func NewServices(users *repository.Userrepo) *Services {
	return &Services{
		UserService: NewUserService(users),
	}
}

type UserService struct {
	Users *repository.Userrepo // We'll be using code genrated by sqlc here
}

func NewUserService(users *repository.Userrepo) *UserService {
	return &UserService{
		Users: users,
	}
}

func (us *UserService) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	exists, err := us.Users.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if exists != nil {
		return "", errors.New("email already exists")
	}

	pwHash, err := utils.HashPassword(password)
	if err != nil {
		return "", err
	}

	userID, err := us.Users.CreateUser(ctx, email, pwHash, name)
	if err != nil {
		return "", err
	}

	return userID, nil

}
