package service

import "errors"

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrIDMissing    = errors.New("user id is missing")

	// products
	ErrSelfBidding     = errors.New("seller cannot bid on their own product")
	ErrProductNotFound = errors.New("product not found")
	ErrInsufficientBid = errors.New("bid must be greater than current price")
	ErrConsecutiveBid  = errors.New("cannot place consecutive bids on the same product")
	ErrUrlsNotFound    = errors.New("Image Urls not found")
)
