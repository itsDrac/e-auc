package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Generic type for any data structure sent in a response
type JSONResponse map[string]any

// RespondJson sends a Json response to the client,
// handles Content-Type to JSON
func RespondJson(w http.ResponseWriter, status int, data JSONResponse) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to write JSON response", "status", status, "error", err)
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
