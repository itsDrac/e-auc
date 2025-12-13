package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/pkg/config"
	"github.com/itsDrac/e-auc/pkg/jwt"
	"github.com/itsDrac/e-auc/pkg/utils"
)

type AuthServicer interface {
	AddUser(context.Context, db.User) (uuid.UUID, error)
	ValidateUser(context.Context, db.User) (jwt.Tokens, error)
	BlacklistUserToken(ctx context.Context, accessTokenString string) error
	ValidateRefreshToken(tokenString string) (*config.RefreshClaims, error)
	ValidateAccessToken(tokenString string) (*config.UserClaims, error)
	IssueTokenPair(userID string) (jwt.Tokens, error)
}

type AuthService struct {
	db db.Querier
	JM *jwt.JwtManager
}

func NewAuthService(db db.Querier) (*AuthService, error) {
	jwtManger, err := jwt.NewJwtManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AuthService: %w", err)
	}
	return &AuthService{
		db: db,
		JM: jwtManger,
	}, nil
}

// Register
func (as *AuthService) AddUser(ctx context.Context, u db.User) (uuid.UUID, error) {
	exists, _ := as.db.GetUserByEmail(ctx, u.Email)

	if exists.ID != uuid.Nil {
		return uuid.Nil, fmt.Errorf("user already exists with email: %s", u.Email)
	}

	hash, err := utils.HashPassword(u.Password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to hash password: %w", err)
	}

	params := db.CreateUserParams{
		Email:    u.Email,
		Username: u.Username,
		Password: string(hash),
	}

	user, err := as.db.CreateUser(ctx, params)
	if err != nil {
		return uuid.Nil, fmt.Errorf("Failed to create user: %w", err)
	}

	return user.ID, nil
}

func (as *AuthService) ValidateUser(ctx context.Context, u db.User) (jwt.Tokens, error) {
	user, err := as.db.GetUserByUsername(ctx, u.Username)
	if err != nil {
		return jwt.Tokens{}, fmt.Errorf("invalid credentials")
	}

	if err := utils.ComparePassword(u.Password, user.Password); err != nil {
		slog.Error(err.Error())
		return jwt.Tokens{}, fmt.Errorf("invalid credentials")
	}
	tokens, err := as.JM.GenerateTokenPair(user.ID)
	if err != nil {
		return jwt.Tokens{}, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

func (as *AuthService) BlacklistUserToken(ctx context.Context, accessTokenString string) error {
	var revocationError []error

	accessClaims, err := as.JM.ValidateAccessToken(accessTokenString)
	if err == nil {
		remainingDuration := time.Until(accessClaims.ExpiresAt.Time)
		if remainingDuration > 0 {
			if err := as.JM.AddToBlackList(accessClaims.ID, remainingDuration); err != nil {
				revocationError = append(revocationError, fmt.Errorf("failed to blacklist access token: %w", err))
			}
		}
	} else {

	}

	if len(revocationError) > 0 {
		return fmt.Errorf("logout completed with errors: %v", revocationError)
	}

	return nil
}

func (as *AuthService) ValidateRefreshToken(tokenString string) (*config.RefreshClaims, error) {
	return as.JM.ValidateRefreshToken(tokenString)
}

func (as *AuthService) ValidateAccessToken(tokenString string) (*config.UserClaims, error) {
	return as.JM.ValidateAccessToken(tokenString)
}

func (as *AuthService) IssueTokenPair(userID string) (jwt.Tokens, error) {
	return as.JM.GenerateTokenPair(uuid.MustParse(userID))
}
