package handlers

import "errors"

var (
	// common error code
	ErrInternalServer = errors.New("INTERNAL_SERVER_ERROR")
	ErrInvalidRequest = errors.New("VALIDATION_FAILED")
	ErrInvalidJson    = errors.New("INVALID_JSON_FORMAT")
	ErrMissingParam   = errors.New("MISSING_PARAM")
	ErrDb             = errors.New("DB_ERROR")

	// auth error code
	ErrAuthFailed     = errors.New("AUTH_FAILED")
	ErrMissingToken   = errors.New("MISSING_TOKEN")
	ErrMissingCookie  = errors.New("MISSING_COOKIE")
	ErrInvalidToken   = errors.New("INVALID_TOKEN")
	ErrTokenGenFailed = errors.New("TOKEN_GENERATION_FAILED")
	ErrToken          = errors.New("TOKEN_ERROR")
	ErrLogout         = errors.New("LOGOUT_ERROR")
	ErrAuthMiss       = errors.New("AUTH_MISSING")

	// user error code
	ErrUserNotFound = errors.New("USER_NOT_FOUND")

	// bid error code
	ErrBidLow          = errors.New("BID_TOO_LOW")
	ErrSelfBidding     = errors.New("SELF_BIDDING_NOT_ALLOWED")
	ErrBidCreateFailed = errors.New("BID_CREATION_FAILED")

	// file error code
	ErrInvalidForm   = errors.New("INVALID_FORM")
	ErrMissingFiles  = errors.New("MISSING_FILES")
	ErrLargeFile     = errors.New("FILE_TO_LARGE")
	ErrFileOpen      = errors.New("FILE_OPEN_ERROR")
	ErrFileReadError = errors.New("FILE_READ_ERROR")
	ErrInvalidFile   = errors.New("INVALID_FILE_TYPE")
	ErrUploadFailed  = errors.New("UPLOAD_FAILED")

	//products error code
	ErrProductNotFound = errors.New("PRODUCT_NOT_FOUND")
	ErrUrlsNotFound    = errors.New("PRODUCT_URLS_NOT_FOUND")
)
