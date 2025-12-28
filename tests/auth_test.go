package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsDrac/e-auc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser holds user credentials and tokens for testing
type TestUser struct {
	UserID       uuid.UUID
	Email        string
	Username     string
	Password     string
	AccessToken  string
	RefreshToken string
}

// Global storage for test users
var TestUsers []TestUser

func TestMain(m *testing.M) {
	// Setup containers once for all tests in this package
	exitCode := Setup(m)
	os.Exit(exitCode)
}

// TestAuthFlowIntegration tests complete authentication flows and creates test users
func TestAuthFlowIntegration(t *testing.T) {
	env := GetTestEnv()

	// Create 10 test users
	for i := 0; i < 10; i++ {
		timestamp := time.Now().UnixNano()
		email := fmt.Sprintf("testuser%d-%d@example.com", i, timestamp)
		username := fmt.Sprintf("testuser%d-%d", i, timestamp)
		password := "password123"

		t.Run(fmt.Sprintf("User_%d_Complete_Auth_Flow", i), func(t *testing.T) {
			// Step 1: Register
			registerReq := map[string]interface{}{
				"email":    email,
				"username": username,
				"password": password,
			}
			reqBody, _ := json.Marshal(registerReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			env.Dependencies.UserHandler.RegisterUser(w, req)
			assert.Equal(t, http.StatusCreated, w.Code)

			var registerResp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&registerResp)
			data := registerResp["data"].(map[string]interface{})
			userIDStr := data["user_id"].(string)
			userID, err := uuid.Parse(userIDStr)
			require.NoError(t, err)

			// Step 2: Login
			loginReq := map[string]interface{}{
				"username": username,
				"password": password,
			}
			reqBody, _ = json.Marshal(loginReq)
			req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()

			env.Dependencies.UserHandler.LoginUser(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			var loginResp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&loginResp)
			data = loginResp["data"].(map[string]interface{})
			accessToken := data["access_token"].(string)

			// Get refresh token from cookies
			var refreshToken string
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == config.RefreshTokenCookieName {
					refreshToken = cookie.Value
					break
				}
			}
			require.NotEmpty(t, refreshToken, "refresh token should be set")

			// Step 3: Refresh Token
			req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
			req.AddCookie(&http.Cookie{
				Name:  config.RefreshTokenCookieName,
				Value: refreshToken,
			})
			w = httptest.NewRecorder()

			env.Dependencies.UserHandler.RefreshToken(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			var refreshResp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&refreshResp)
			data = refreshResp["data"].(map[string]interface{})
			newAccessToken := data["access_token"].(string)
			assert.NotEmpty(t, newAccessToken)
			assert.NotEqual(t, accessToken, newAccessToken, "new access token should be different")

			// Get new refresh token from cookies
			var newRefreshToken string
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == config.RefreshTokenCookieName {
					newRefreshToken = cookie.Value
					break
				}
			}
			require.NotEmpty(t, newRefreshToken, "new refresh token should be set")

			// Step 4: Logout
			req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			req.Header.Set("Authorization", "Bearer "+newAccessToken)
			w = httptest.NewRecorder()

			env.Dependencies.UserHandler.LogoutUser(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			// Verify cookie was cleared
			var clearedCookie *http.Cookie
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == config.RefreshTokenCookieName {
					clearedCookie = cookie
					break
				}
			}
			require.NotNil(t, clearedCookie)
			assert.Empty(t, clearedCookie.Value)

			// Step 5: Login again to get fresh tokens for storage
			reqBody, _ = json.Marshal(loginReq)
			req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()

			env.Dependencies.UserHandler.LoginUser(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			json.NewDecoder(w.Body).Decode(&loginResp)
			data = loginResp["data"].(map[string]interface{})
			finalAccessToken := data["access_token"].(string)

			var finalRefreshToken string
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == config.RefreshTokenCookieName {
					finalRefreshToken = cookie.Value
					break
				}
			}

			// Store user in global storage
			testUser := TestUser{
				UserID:       userID,
				Email:        email,
				Username:     username,
				Password:     password,
				AccessToken:  finalAccessToken,
				RefreshToken: finalRefreshToken,
			}
			TestUsers = append(TestUsers, testUser)

			t.Logf("Created test user: %s (ID: %s)", username, userID.String())
		})
	}

	// Verify all users were created
	assert.Equal(t, 10, len(TestUsers), "Should have created 10 test users")
	t.Logf("Total test users created and stored: %d", len(TestUsers))
}

// GetTestUser returns a test user by index (0-9)
func GetTestUser(index int) *TestUser {
	if index < 0 || index >= len(TestUsers) {
		return nil
	}
	return &TestUsers[index]
}

// GetRandomTestUser returns a random test user
func GetRandomTestUser() *TestUser {
	if len(TestUsers) == 0 {
		return nil
	}
	return &TestUsers[time.Now().UnixNano()%int64(len(TestUsers))]
}

// GetAllTestUsers returns all test users
func GetAllTestUsers() []TestUser {
	return TestUsers
}
