package server

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Username string `json:"username" validate:"required"`
}

type LoginUserRequest struct {
	Username    string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type CreateProductRequest struct {
	Title       string   `json:"title" validate:"required,max=200,min=3"`
	Description *string   `json:"description"`
	Images      []string `json:"images" validate:"required,min=1,max=5"`
	MinPrice    int32      `json:"min_price" validate:"required,gte=0"`
	CurrentPrice int32     `json:"current_price" validate:"required,gte=0"`
}

type PlaceBidRequest struct {
	BidAmount int32 `json:"bid_amount" validate:"required,gt=0"`
}