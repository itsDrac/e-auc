package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) UserRoutes(mux *chi.Mux) {
	mux.HandleFunc("POST /api/v1/users", s.RegisterUser)
}

func (s *Server) CommonRoutes(mux *chi.Mux) {
	mux.HandleFunc("GET /api/v1/health", healthCheck)
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
