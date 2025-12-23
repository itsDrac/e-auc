package model

import "time"

// Metadata for the response
type Metadata struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id"`
}

// Error details
type ErrorDetails struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details []ErrorDetails `json:"details"`
}

// Bid data
type BidData struct {
	BidID     string  `json:"bid_id"`
	ImageURL  string  `json:"image_url"`
	BidAmount float64 `json:"bid_amount"`
}

type APIResponse[T any] struct {
	Status   string    `json:"status"`
	Message  string    `json:"message,omitempty"`
	Metadata Metadata  `json:"metadata"`
	Error    *APIError `json:"error,omitempty"`
	Data     T         `json:"data,omitempty"`
}
