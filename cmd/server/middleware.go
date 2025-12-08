package server

import (
	"net/http"
	"time"

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
			s.Logger.Infof(
				"%3d | %v | %-7s \"%s\"",
				status,
				latency,
				method,
				path,
			)
		})
	}
}
