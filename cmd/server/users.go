package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	// "github.com/itsDrac/e-auc/internal/types"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/pkg/config"
)

// RegisterUser godoc
//
//	@Summary		Register a new User
//	@Description	Register a new user with email, username, and password
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			user	body		CreateUserRequest	true	"User registration details"
//	@Success		201		{object}	map[string]any
//	@Failure		400		{object}	map[string]any
//	@Failure		409		{object}	map[string]any
//	@Router			/auth/register [post]
func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// User request validation.
	err := validate.Struct(req)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user := db.User{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	}

	userId, err := s.Services.AuthService.AddUser(r.Context(), user)
	if err != nil {
		RespondError(w, http.StatusConflict, err.Error())
		return
	}
	resp := map[string]any{
		"user_id": userId.String(),
		"message": "User registered successfully",
	}
	RespondJson(w, http.StatusCreated, resp)
}

// LoginUser godoc
//
//	@Summary		Login a User
//	@Description	Login a user with username and password
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		LoginUserRequest	true	"User login credentials"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	map[string]any
//	@Failure		401			{object}	map[string]any
//	@Router			/auth/login [post]
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	err := validate.Struct(req)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// User request validation.
	user := db.User{
		Username: req.Username,
		Password: req.Password,
	}

	tokens, err := s.Services.AuthService.ValidateUser(r.Context(), user)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// calculate cookie expiry by validating the refresh Token
	refreshClaims, _ := s.Services.AuthService.ValidateRefreshToken(tokens.RefreshToken)
	expiry := refreshClaims.ExpiresAt.Time

	// set the refresh token cookie
	setRefreshTokenCookie(w, tokens.RefreshToken, expiry)

	resp := map[string]any{
		"access_token": tokens.AccessToken,
		"message":      "login successfully",
	}
	RespondJson(w, http.StatusOK, resp)

}

// RefreshToken godoc
//
//	@Summary		Refresh Access Token
//	@Description	Refresh the access token using a valid refresh token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	map[string]any
//	@Router			/auth/refresh [post]
func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(config.RefreshTokenCookieName)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	refreshTokenString := cookie.Value

	refreshClaims, err := s.Services.AuthService.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}
	user, err := s.Services.UserService.GetUserByID(r.Context(), refreshClaims.UserID.String())
	if err != nil {
		slog.Error("refresh token error", "error", err.Error())
		RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	tokens, err := s.Services.AuthService.IssueTokenPair(user.ID.String())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}
	claims, _ := s.Services.AuthService.ValidateRefreshToken(tokens.RefreshToken)
	setRefreshTokenCookie(w, tokens.RefreshToken, claims.ExpiresAt.Time)

	resp := map[string]any{
		"access_token": tokens.AccessToken,
	}

	RespondJson(w, http.StatusOK, resp)

}

// LogoutUser godoc
//
//	@Summary		Logout User
//	@Description	Logout the user by blacklisting the access token and clearing the refresh token cookie
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	map[string]any
//	@Router			/auth/logout [post]
func (s *Server) LogoutUser(w http.ResponseWriter, r *http.Request) {
	accessTokenString := ""
	authHeader := r.Header.Get("Authorization")
	if parts := strings.Split(authHeader, " "); len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		accessTokenString = parts[1]
	}

	if err := s.Services.AuthService.BlacklistUserToken(r.Context(), accessTokenString); err != nil {
		RespondError(w, http.StatusUnauthorized, err.Error())
	}

	http.SetCookie(w, &http.Cookie{
		Name:     config.RefreshTokenCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	})

	resp := map[string]any{
		"message": "logged out successfully",
	}

	RespondJson(w, http.StatusOK, resp)
}

// Profile godoc
//
//	@Summary		Get User Profile
//	@Description	Retrieve the profile information of the authenticated user
//	@Tags			Users
//	@Requirements	BearerAuth
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Router			/users/me [get]
func (s *Server) Profile(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusForbidden, "claims not found in context")
		return
	}

	userID := claims.UserID

	user, err := s.Services.UserService.GetUserByID(r.Context(), userID.String())

	if err != nil {
		RespondError(w, http.StatusForbidden, err.Error())
		return
	}

	resp := map[string]any{
		"data":    user,
		"message": "data fetched successfully",
	}

	RespondJson(w, http.StatusOK, resp)

}

func setRefreshTokenCookie(w http.ResponseWriter, token string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.RefreshTokenCookieName,
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}
