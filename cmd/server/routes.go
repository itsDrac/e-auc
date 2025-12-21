package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) routes() *chi.Mux {
	mux := chi.NewMux()

	// global middlewares
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8000/swagger/doc.json"), // The url pointing to API definition
	))

	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.RegisterUser)
			r.Post("/login", s.LoginUser)
			r.Post("/refresh", s.RefreshToken)
			r.Post("/logout", s.LogoutUser)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.AuthMiddleware())
			r.Route("/users", func(r chi.Router) {
				r.Get("/me", s.Profile)
			})
			r.Route("/products", func(r chi.Router) {
				r.Post("/upload-images", s.UploadImages)
				r.Post("/", s.CreateProduct)
				r.Patch("/{productId}/bid", s.PlaceBid)
				r.Get("/{sellerId}", s.ProductsBySellerID)
			})
		})
		// r.Route("/products", func(r chi.Router) {
		// 			r.Get("/{productId}", s.GetProductByID)
		// 			r.Get("/{productId}/images", s.GetProductImageUrls)

		// 		})
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
