package types

// RegisterRequest
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// LoginRequest
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
