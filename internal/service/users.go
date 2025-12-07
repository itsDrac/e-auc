package service

import (
	"context"
	"errors"

	"github.com/itsDrac/e-auc/internal/repository"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type UserService struct {
	Users *repository.Userrepo
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
