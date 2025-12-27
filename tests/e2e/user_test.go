package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/handlers"
	"github.com/itsDrac/e-auc/internal/middleware" // Import your middleware package
	"github.com/itsDrac/e-auc/internal/model"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/tests"
)

func TestUserJourney(t *testing.T) {

	t.Setenv("ACCESS_TOKEN_SECRET", "test-secret-key-123")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-key-456")
	// 1. Setup Infrastructure (Real DB via Docker)
	conn, cleanup := tests.SetupTestDB(t)
	defer cleanup()

	// 2. Initialize Dependencies
	store := db.New(conn)

	// Note: You might need your real config for signing keys here
	// For testing, we can often rely on defaults or inject a test key if your service supports it
	authService, err := service.NewAuthService(store)
	if err != nil {
		t.Fatalf("auth service intiialization failed %v", err)
	}
	userService, err := service.NewUserService(store)
	if err != nil {
		t.Fatalf("auth service intiialization failed %v", err)
	}
	userHandler, err := handlers.NewUserHandler(userService, authService)
	if err != nil {
		t.Fatalf("auth service intiialization failed %v", err)
	}

	// 3. Setup Router (Crucial Step!)
	// We create a real router to ensure Middleware runs correctly
	r := chi.NewRouter()

	// Public Routes
	r.Post("/auth/register", userHandler.RegisterUser)
	r.Post("/auth/login", userHandler.LoginUser)

	// Protected Routes (Apply the REAL Middleware)
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.AuthMiddleware(authService))
		protected.Get("/users/me", userHandler.Profile)
	})

	// --- TEST DATA ---
	testUser := model.CreateUserRequest{
		Email:    "mobasir@example.com",
		Username: "mobasir",
		Password: "StrongPassword123!",
	}
	var authToken string // We will store the token here after login

	// --- STEP 1: REGISTER ---
	t.Run("1. Register User", func(t *testing.T) {
		// Create JSON Body
		body, _ := json.Marshal(testUser)
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Record Response
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		// Assertions
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Contains(t, rr.Body.String(), "user registered successfully")
	})

	// --- STEP 2: LOGIN ---
	t.Run("2. Login User", func(t *testing.T) {
		loginPayload := map[string]string{
			"username": testUser.Username,
			"password": testUser.Password,
		}
		body, _ := json.Marshal(loginPayload)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Extract Access Token from Response
		var resp map[string]any // Or use your APIResponse struct
		json.Unmarshal(rr.Body.Bytes(), &resp)

		// Navigate the JSON structure: { "data": { "access_token": "..." } }
		data := resp["data"].(map[string]any)
		authToken = data["access_token"].(string)

		assert.NotEmpty(t, authToken, "Access token should not be empty")

		// Optional: Verify Cookie was set
		cookies := rr.Result().Cookies()
		assert.NotEmpty(t, cookies, "Refresh token cookie should be present")
	})

	// --- STEP 3: GET PROFILE (Protected) ---
	t.Run("3. Get Profile", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/users/me", nil)

		// Inject the Token we got from Step 2
		req.Header.Set("Authorization", "Bearer "+authToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify we got the correct user back
		assert.Contains(t, rr.Body.String(), testUser.Email)
		assert.Contains(t, rr.Body.String(), testUser.Username)
	})

	// --- STEP 4: FAILURE SCENARIO (Bad Password) ---
	t.Run("4. Login Fail", func(t *testing.T) {
		badPayload := map[string]string{
			"username": testUser.Username,
			"password": "WrongPassword",
		}
		body, _ := json.Marshal(badPayload)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
