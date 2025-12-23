package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/itsDrac/e-auc/internal/handlers"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/pkg/config"
)

func AuthMiddleware(s service.AuthServicer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			parts := strings.Split(authHeader, " ")

			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				handlers.RespondErrorJSON(w, r, http.StatusUnauthorized, handlers.ErrMissingToken.Error(), "Missing token in the Authorization header", nil)
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}
			accessTokenString := parts[1]

			claims, err := s.ValidateAccessToken(accessTokenString)
			if err != nil {
				handlers.RespondErrorJSON(w, r, http.StatusUnauthorized, handlers.ErrToken.Error(), "Token is either revoked or invalid.", nil)
				return
			}

			ctx := context.WithValue(r.Context(), config.UserClaimKey, claims)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
