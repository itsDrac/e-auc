package jwt

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/itsDrac/e-auc/pkg/config"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTManager interface {
	GenerateTokenPair(userID uuid.UUID, role string) (Tokens, error)
	ValidateAccessToken(tokenString string) (*config.UserClaims, error)
	ValidateRefreshToken(tokenString string) (*config.RefreshClaims, error)
	GetAccessSecret() []byte
	AddToBlackList(tokenID string, expiration time.Duration)
	IsBlackListed(tokenID string) bool
}

type JwtManager struct {
	accessSecret  []byte
	refreshSecret []byte
	tokens        *sync.Map
}

func NewJwtManager() (*JwtManager, error) {
	accessSecret := utils.GetEnv("ACCESS_TOKEN_SECRET", "")
	refreshSecret := utils.GetEnv("REFRESH_TOKEN_SECRET", "")

	if accessSecret == "" || refreshSecret == "" {
		return nil, fmt.Errorf("JWT secrets must be set in environment: ACCESS_TOKEN_SECRET and REFRESH_TOKEN_SECRET")
	}

	return &JwtManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		tokens:        &sync.Map{},
	}, nil
}

// GetAccessSecret exposes the secret for middleware use
func (jm *JwtManager) GetAccessSecret() []byte {
	return jm.accessSecret
}

// GenerateTokenPair creates both an access token and a refresh token
func (jm *JwtManager) GenerateTokenPair(userID uuid.UUID) (Tokens, error) {
	now := time.Now()

	accessClaims := config.UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // unique jwt id for blacklist
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccessToken, err := accessToken.SignedString(jm.accessSecret)
	if err != nil {
		return Tokens{}, err
	}

	// refresh Token
	refreshClaims := config.RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // unique jwt id for rotation
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString(jm.refreshSecret)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}, nil

}

// ValidateAccessToken verifies and returns the claims from an access token string.
func (jm *JwtManager) ValidateAccessToken(tokenString string) (*config.UserClaims, error) {
	claims := &config.UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}
		return jm.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	return claims, nil
}

// ValidateRefreshToken verifies and returns the claims from a refresh token string.
func (jm *JwtManager) ValidateRefreshToken(tokenString string) (*config.RefreshClaims, error) {
	claims := &config.RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
		}
		return jm.refreshSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	return claims, nil
}

// Add blacklists a token ID for a specified duration
func (jm *JwtManager) AddToBlackList(tokenID string, duration time.Duration) error {
	jm.tokens.Store(tokenID, struct{}{})

	time.AfterFunc(duration, func() {
		jm.tokens.Delete(tokenID)
	})

	return nil
}

// IsBlackListed checks if a token ID exists in the blacklist.
func (jm *JwtManager) IsBlackListed(tokenID string) bool {
	_, found := jm.tokens.Load(tokenID)
	return found
}
