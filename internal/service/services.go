package service

import db "github.com/itsDrac/e-auc/internal/database"

type Services struct {
	UserService UserServicer
	AuthService AuthServicer
}

func NewServices(db db.Querier) (*Services, error) {
	authService, err := NewAuthService(db)
	if err != nil {
		return nil, err
	}

	userService, err := NewUserService(db)
	if err != nil {
		return nil, err
	}
	return &Services{
		UserService: userService,
		AuthService: authService,
	}, err
}
