package service

import (
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/storage"
)

type Services struct {
	UserService UserServicer
	AuthService AuthServicer
	ProductService ProductServicer
}

func NewServices(db db.Querier, s storage.Storager) (*Services, error) {
	authService, err := NewAuthService(db)
	if err != nil {
		return nil, err
	}

	userService, err := NewUserService(db)
	if err != nil {
		return nil, err
	}
	productService, err := NewProductService(db, s)
	if err != nil {
		return nil, err
	}
	return &Services{
		UserService: userService,
		AuthService: authService,
		ProductService: productService,
	}, err
}
