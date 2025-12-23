package service

import "errors"

var (
	ErrUserExists = errors.New("USER_EXISTS")

	// products
	ErrSelfBidding     = errors.New("seller cannot bid on their own product")
	ErrProductNotFound = errors.New("product not found")
	ErrInsufficientBid = errors.New("bid must be greater than current price")
	ErrUrlsNotFound    = errors.New("Image Urls not found")
)
