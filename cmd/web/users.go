package web

import (
	"net/http"
	"encoding/json"
	
	"github.com/itsDrac/e-auc/internal/service"
)

func (h *ChiHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	user := service.User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	userId, err := h.service.CreateUser(r.Context(), user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(userId))
}