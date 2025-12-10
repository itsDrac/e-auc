package server

import (
	"encoding/json"
	"net/http"

	"github.com/itsDrac/e-auc/internal/types"
)

func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	user := types.User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userId, err := s.Services.UserService.CreateUser(r.Context(), user.Email, user.Password, user.Name)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(userId))
}
