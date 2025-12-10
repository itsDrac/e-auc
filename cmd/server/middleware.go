package server

import (
	"net/http"
	"time"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
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
