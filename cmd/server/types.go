package server

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email`
	Password string `json:"password" validate:"required,min=8"`
	Username string `json:"username" validate:"required"`
}

type LoginUserRequest struct {
	Username    string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}