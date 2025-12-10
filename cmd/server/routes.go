package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) routes() *chi.Mux {
	mux := chi.NewMux()

	// global middlewares
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)

		r.Route("/users", func(ur chi.Router) {
			ur.Post("/", s.RegisterUser)
		})
	})

	return mux
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	
	resp := map[string]any{
		"message": "ok",
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)

}
