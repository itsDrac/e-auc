package config

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// Token Expiration Durations
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour

	// Context Keys
	UserClaimKey = "user_claims"

	//HttpOnly Cookie Name
	RefreshTokenCookieName = "refresh_token"
)

// UserClaims is the payload for the Access Token
type UserClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// RefreshClaims is the payload for the Refresh token
type RefreshClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}
