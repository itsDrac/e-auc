package handlers

import "errors"

var (
	// user and auth errors
	ErrInvalidRequest = errors.New("VALIDATION_FAILED")
	ErrInvalidJson    = errors.New("INVALID_JSON_FORMAT")
	ErrAuthFailed     = errors.New("AUTH_FAILED")
	ErrMissingCookie  = errors.New("MISSING_COOKIE")
	ErrInvalidToken   = errors.New("INVALID_TOKEN")
	ErrUserNotFound   = errors.New("USER_NOT_FOUND")
	ErrTokenGenFailed = errors.New("TOKEN_GENERATION_FAILED")
	ErrToken          = errors.New("TOKEN_ERROR")
	ErrLogout         = errors.New("LOGOUT_ERROR")
	ErrAuthMiss       = errors.New("AUTH_MISSING")

	// products errors
)
