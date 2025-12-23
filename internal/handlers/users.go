package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	// "github.com/itsDrac/e-auc/internal/types"
	"github.com/go-playground/validator/v10"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/model"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/pkg/config"
	valid "github.com/itsDrac/e-auc/pkg/validator"
)

var validate = valid.GetValidator()

type UserHandler struct {
	userService service.UserServicer
	authService service.AuthServicer
}

func NewUserHandler(userSvc service.UserServicer, authSvc service.AuthServicer) (*UserHandler, error) {
	return &UserHandler{
		userService: userSvc,
		authService: authSvc,
	}, nil
}

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
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidJson.Error(), "Invalid JSON format", nil)
		return
	}
	if err := validate.Struct(req); err != nil {
		var details []model.ErrorDetails
		if validErrs, ok := err.(validator.ValidationErrors); ok {
			for _, vErr := range validErrs {
				details = append(details, model.ErrorDetails{
					Field: vErr.Field(),
					Issue: fmt.Sprintf("failed on tage '%s' with param '%s'", vErr.Tag(), vErr.Param()),
				})
			}
		}
		RespondErrorJSON(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "Input validation failed", details)
		return
	}

	// users details
	user := db.User{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	}

	userId, err := h.authService.AddUser(r.Context(), user)
	if err != nil {
		if err.Error() == service.ErrUserExists.Error() {
			RespondErrorJSON(w, r, http.StatusConflict, "USER_EXISTS", "user already exists with same email", nil)
			return
		}
		slog.Error("Internal Error", "error", err.Error())
		RespondErrorJSON(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Something went wrong", nil)
		return
	}
	resp := map[string]any{
		"user_id": userId.String(),
	}
	RespondSuccessJSON(w, r, http.StatusCreated, "user registered successfully", resp)
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
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req model.LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidJson.Error(), "Invalid JSON format", nil)
		return
	}

	if err := validate.Struct(req); err != nil {
		var details []model.ErrorDetails
		if validErrs, ok := err.(validator.ValidationErrors); ok {
			for _, vErr := range validErrs {
				details = append(details, model.ErrorDetails{
					Field: vErr.Field(),
					Issue: fmt.Sprintf("failed on tag '%s' with param '%s'", vErr.Tag(), vErr.Param()),
				})
			}
		}
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidRequest.Error(), "Input validation failed", details)
		return
	}

	// User request validation.
	user := db.User{
		Username: req.Username,
		Password: req.Password,
	}

	tokens, err := h.authService.ValidateUser(r.Context(), user)
	if err != nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrAuthFailed.Error(), "Invalid username or password", nil)
		return
	}

	// calculate cookie expiry by validating the refresh Token
	refreshClaims, _ := h.authService.ValidateRefreshToken(tokens.RefreshToken)
	expiry := refreshClaims.ExpiresAt.Time

	// set the refresh token cookie
	setRefreshTokenCookie(w, tokens.RefreshToken, expiry)

	resp := map[string]any{
		"access_token": tokens.AccessToken,
	}
	RespondSuccessJSON(w, r, http.StatusOK, "Login successful", resp)

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
func (h *UserHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// extract cookies
	cookie, err := r.Cookie(config.RefreshTokenCookieName)
	if err != nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrMissingCookie.Error(), "Refresh token cookie missing", nil)
		return
	}
	refreshTokenString := cookie.Value

	// validate token
	refreshClaims, err := h.authService.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrInvalidToken.Error(), "Invalid or expired refresh token", nil)
		return
	}

	// verify user still exists
	user, err := h.userService.GetUserByID(r.Context(), refreshClaims.UserID.String())
	if err != nil {
		slog.Error("refresh token error", "error", err.Error())
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrUserNotFound.Error(), "User account not found", nil)
		return
	}

	// issue new tokens
	tokens, err := h.authService.IssueTokenPair(user.ID.String())
	if err != nil {
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrTokenGenFailed.Error(), "failed to generate tokens", nil)
		return
	}
	claims, err := h.authService.ValidateRefreshToken(tokens.RefreshToken)
	if err != nil {
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrToken.Error(), "Error validating new token", nil)
		return
	}
	setRefreshTokenCookie(w, tokens.RefreshToken, claims.ExpiresAt.Time)

	resp := map[string]any{
		"access_token": tokens.AccessToken,
	}

	RespondSuccessJSON(w, r, http.StatusOK, "token refreshed successfully", resp)
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
func (h *UserHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	// extract the access token
	accessTokenString := ""
	authHeader := r.Header.Get("Authorization")
	if parts := strings.Split(authHeader, " "); len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		accessTokenString = parts[1]
	}

	// blacklist the token (if exists)
	// Even if the token is missing, we usually proceed to clear the cookie to ensure the client is "logged out"
	if accessTokenString != "" {
		if err := h.authService.BlacklistUserToken(r.Context(), accessTokenString); err != nil {
			// Log the error but decide if you want to block logout.
			// Usually, we log it and proceed, or return an error.
			// Here is how to return the error if that is your policy:
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrLogout.Error(), "Failed to blacklist token", nil)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     config.RefreshTokenCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	RespondSuccessJSON(w, r, http.StatusOK, "Logout out successfully", "")
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
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	// get claims
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrAuthFailed.Error(), "user claims not found in context", nil)
		return
	}

	userID := claims.UserID

	user, err := h.userService.GetUserByID(r.Context(), userID.String())
	if err != nil {
		RespondErrorJSON(w, r, http.StatusForbidden, ErrUserNotFound.Error(), "user profile could not be retrieved", nil)
		return
	}

	RespondSuccessJSON(w, r, http.StatusOK, "Profile data fetched successfully", user)
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
