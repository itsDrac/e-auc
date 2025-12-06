package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/itsDrac/e-auc/internal/service"
)

type Handler interface {
	CreateUser(http.ResponseWriter, *http.Request)
}

type ChiHandler struct {
	mux *chi.Mux
	service service.Servicer
}

func NewChiHandler() *ChiHandler {
	r := chi.NewRouter()
	return &ChiHandler{
		mux: r,
	}
}

func (h *ChiHandler) GetMux() http.Handler {
	return h.mux
}

func (h *ChiHandler) Mount() {
	r := h.mux
	r.Get("/health", h.Health)
	r.Post("/users", h.CreateUser)
}

func (h *ChiHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}