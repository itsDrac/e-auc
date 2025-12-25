package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/itsDrac/e-auc/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) routes() *chi.Mux {
	mux := chi.NewMux()

	// global middlewares
	mux.Use(chiMiddleware.Logger)
	mux.Use(chiMiddleware.Recoverer)

	// swagger documentation
	mux.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8000/swagger/doc.json"), // The url pointing to API definition
	))

	// api v1 routes
	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		s.AuthRoutes(r)
		s.UserRoutes(r)
		s.ProductRoutes(r)
	})

	return mux
}

// AuthRoutes registers authentication endpoints (public)
func (s *Server) AuthRoutes(router chi.Router) {
	userHandler := s.Dependencies.UserHandler
	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", userHandler.RegisterUser)
		r.Post("/login", userHandler.LoginUser)
		r.Post("/refresh", userHandler.RefreshToken)
		r.Post("/logout", userHandler.LogoutUser)
	})
}

// UserRoutes registers user endpoints (protected)
func (s *Server) UserRoutes(router chi.Router) {
	userHandler := s.Dependencies.UserHandler
	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(s.Dependencies.Services.AuthService))
		r.Route("/users", func(r chi.Router) {
			r.Get("/me", userHandler.Profile)
		})
	})
}

// ProductRoutes registers product endpoints (protected)
func (s *Server) ProductRoutes(router chi.Router) {
	var productHandler = s.Dependencies.ProductHandler
		// Not protected routes
		router.Route("/products", func(r chi.Router) {
			r.Get("/images", productHandler.GetProductImageUrls)
			r.Get("/{productId}", productHandler.GetProductByID)
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuthMiddleware(s.Dependencies.Services.AuthService))
				r.Post("/upload-images", productHandler.UploadImages)
				r.Post("/", productHandler.CreateProduct)
				r.Patch("/{productId}/bid", productHandler.PlaceBid)
				r.Get("/seller/{sellerId}", productHandler.ProductsBySellerID)
			})
		})
}

// Healthcheck godoc
// @Summary      Health Check
// @Description  Check if the server is running
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/health [get]
func healthCheck(w http.ResponseWriter, r *http.Request) {

	resp := map[string]any{
		"message": "ok",
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)

}
