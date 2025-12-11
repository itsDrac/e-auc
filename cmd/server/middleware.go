package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsDrac/e-auc/pkg/config"
)

func (s *Server) LoggerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)
			latency := time.Since(start)
			method := r.Method
			path := r.URL.Path
			status := ww.Status()
			slog.Info("Http Request",
				"status", status,
				"latency", latency,
				"method", method,
				"path", path,
			)
			// s.Logger.Infof("%s %s %d %s", method, path, status, latency) --- IGNORE ---
		})
	}
}

func (s *Server) AuthMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			parts := strings.Split(authHeader, " ")

			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}
			accessTokenString := parts[1]

			claims, err := s.Services.AuthService.ValidateAccessToken(accessTokenString)
			if err != nil {
				http.Error(w, "Invalid or revoked token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), config.UserClaimKey, claims)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserClaims(ctx context.Context) *config.UserClaims {
	claims, ok := ctx.Value(config.UserClaimKey).(*config.UserClaims)
	if !ok {
		return nil
	}
	return claims
}
