package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/itsDrac/e-auc/internal/model"
	"github.com/itsDrac/e-auc/pkg/config"
)

// Generic type for any data structure sent in a response
type JSONResponse map[string]any

var requestIDKey = "X-Request-ID"

// RespondJson sends a Json response to the client,
// handles Content-Type to JSON
func RespondJson(w http.ResponseWriter, status int, data JSONResponse) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to write JSON response", "status", status, "error", err)
	}
}

func writeJson(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("Failed to write json response", "status", status, "error", err)
	}
}

// RespondError wraps the Respond function for common error responses.
func RespondError(w http.ResponseWriter, status int, message string) {
	// Logs the error on the server side (best practice)
	slog.Warn("Responding with error", "status", status, "message", message)

	data := JSONResponse{
		"error": message,
	}
	RespondJson(w, status, data)
}

func GetUserClaims(ctx context.Context) *config.UserClaims {
	claims, ok := ctx.Value(config.UserClaimKey).(*config.UserClaims)
	if !ok {
		return nil
	}
	return claims
}

func RespondSuccessJSON[T any](w http.ResponseWriter, r *http.Request, status int, message string, data T) {

	// fetch request ID , if not found generate new UUID
	reqID := r.Header.Get(requestIDKey)
	if reqID == "" {
		reqID = uuid.NewString()
	}

	// This ensures the client gets the ID whether they sent it or we created it.
	w.Header().Set(requestIDKey, reqID)

	payload := model.APIResponse[T]{
		Status:  "success",
		Message: message,
		Metadata: model.Metadata{
			Timestamp: time.Now().UTC(),
			RequestID: reqID,
		},
		Data:  data,
		Error: nil,
	}
	writeJson(w, status, payload)
}

func RespondErrorJSON(w http.ResponseWriter, r *http.Request, status int, code string, message string, details []model.ErrorDetails) {
	// fetch request ID , if not found generate new UUID
	reqID := r.Header.Get(requestIDKey)
	if reqID == "" {
		reqID = uuid.NewString()
	}
	// This ensures the client gets the ID whether they sent it or we created it.
	w.Header().Set(requestIDKey, reqID)

	payload := model.APIResponse[any]{
		Status: "error",
		Metadata: model.Metadata{
			Timestamp: time.Now().UTC(),
			RequestID: reqID,
		},
		Error: &model.APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	writeJson(w, status, payload)
}
