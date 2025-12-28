package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itsDrac/e-auc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addAuthContext adds user claims to the request context for authenticated requests
func addAuthContext(req *http.Request, testUser *TestUser) *http.Request {
	claims := &config.UserClaims{
		UserID: testUser.UserID,
	}
	ctx := context.WithValue(req.Context(), config.UserClaimKey, claims)
	return req.WithContext(ctx)
}

// TestUserProfile tests the user profile retrieval
func TestUserProfile(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	// Get a test user with valid tokens
	testUser := GetRandomTestUser()
	require.NotNil(t, testUser, "Should have at least one test user")
	require.NotEmpty(t, testUser.AccessToken, "Test user should have access token")

	tests := []struct {
		name           string
		accessToken    string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Valid Access Token",
			accessToken:    testUser.AccessToken,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Profile data fetched successfully", body["message"])
				data, ok := body["data"].(map[string]interface{})
				require.True(t, ok, "Response data should be a map")

				// Verify user details are present
				assert.NotEmpty(t, data["id"])
				assert.NotEmpty(t, data["email"])
				assert.NotEmpty(t, data["username"])
				assert.NotEmpty(t, data["created_at"])
			},
		},
		{
			name:           "Missing Access Token",
			accessToken:    "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorData, ok := body["error"].(map[string]interface{})
				require.True(t, ok, "Response should have error field")
				assert.Contains(t, errorData["code"], "AUTH")
			},
		},
		{
			name:           "Invalid Access Token",
			accessToken:    "invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorData, ok := body["error"].(map[string]interface{})
				require.True(t, ok, "Response should have error field")
				assert.Contains(t, errorData["code"], "AUTH")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

			// Add Authorization header and context if token provided
			if tt.accessToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.accessToken)
				// For valid access token, add auth context
				if tt.accessToken == testUser.AccessToken {
					req = addAuthContext(req, testUser)
				}
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler directly
			env.Dependencies.UserHandler.Profile(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Parse response body
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Should be able to decode response")

			// Run custom response checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// TestUserProfileWithDifferentUsers tests profile access with different test users
func TestUserProfileWithDifferentUsers(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	allUsers := GetAllTestUsers()
	require.GreaterOrEqual(t, len(allUsers), 3, "Should have at least 3 test users")

	// Test profile access for first 3 users
	for i := 0; i < 3 && i < len(allUsers); i++ {
		testUser := GetTestUser(i)
		require.NotNil(t, testUser, "Test user should exist")

		t.Run("User_"+string(rune('A'+i))+"_Profile", func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+testUser.AccessToken)
			req = addAuthContext(req, testUser)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			env.Dependencies.UserHandler.Profile(w, req)

			// Check status code
			assert.Equal(t, http.StatusOK, w.Code, "Status code should be 200 OK")

			// Parse response
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Should decode response")

			// Verify response contains user data
			data, ok := response["data"].(map[string]interface{})
			require.True(t, ok, "Response should contain data")

			// Verify the returned data matches the test user
			assert.Equal(t, testUser.Email, data["email"])
			assert.Equal(t, testUser.Username, data["username"])
			assert.Equal(t, testUser.UserID.String(), data["id"])
		})
	}
}

// TestUserProfileAfterTokenRefresh tests that profile works after token refresh
func TestUserProfileAfterTokenRefresh(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	testUser := GetTestUser(0)
	require.NotNil(t, testUser, "Should have test user at index 0")

	// Step 1: Access profile with current token (should work)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req1.Header.Set("Authorization", "Bearer "+testUser.AccessToken)
	req1 = addAuthContext(req1, testUser)
	w1 := httptest.NewRecorder()
	env.Dependencies.UserHandler.Profile(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "Profile access should work with current token")

	// Step 2: Refresh token
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(&http.Cookie{
		Name:  config.RefreshTokenCookieName,
		Value: testUser.RefreshToken,
	})
	refreshW := httptest.NewRecorder()
	env.Dependencies.UserHandler.RefreshToken(refreshW, refreshReq)
	require.Equal(t, http.StatusOK, refreshW.Code, "Token refresh should succeed")

	// Parse new access token
	var refreshResponse map[string]interface{}
	err := json.NewDecoder(refreshW.Body).Decode(&refreshResponse)
	require.NoError(t, err)
	data, ok := refreshResponse["data"].(map[string]interface{})
	require.True(t, ok)
	newAccessToken := data["access_token"].(string)
	require.NotEmpty(t, newAccessToken, "Should receive new access token")

	// Step 3: Access profile with new token (should work)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req2.Header.Set("Authorization", "Bearer "+newAccessToken)
	req2 = addAuthContext(req2, testUser) // Use same user context
	w2 := httptest.NewRecorder()
	env.Dependencies.UserHandler.Profile(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "Profile access should work with new token")

	// Verify profile data is consistent
	var response2 map[string]interface{}
	err = json.NewDecoder(w2.Body).Decode(&response2)
	require.NoError(t, err)
	profileData, ok := response2["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, testUser.Email, profileData["email"])
	assert.Equal(t, testUser.Username, profileData["username"])
}

// TestConcurrentProfileAccess tests concurrent access to user profiles
func TestConcurrentProfileAccess(t *testing.T) {
	t.Skip("Skipping concurrent test due to database connection pool limitations in test environment")

	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	allUsers := GetAllTestUsers()
	require.GreaterOrEqual(t, len(allUsers), 5, "Should have at least 5 test users for concurrent testing")

	// Use first 5 users for concurrent testing
	numConcurrent := 5
	done := make(chan bool, numConcurrent)
	errors := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(userIndex int) {
			testUser := GetTestUser(userIndex)
			if testUser == nil {
				errors <- assert.AnError
				done <- false
				return
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+testUser.AccessToken)
			req = addAuthContext(req, testUser)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			env.Dependencies.UserHandler.Profile(w, req)

			// Check if successful
			if w.Code != http.StatusOK {
				errors <- assert.AnError
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		if <-done {
			successCount++
		}
	}
	close(errors)

	// Verify all requests succeeded
	assert.Equal(t, numConcurrent, successCount, "All concurrent profile requests should succeed")
	assert.Empty(t, errors, "No errors should occur during concurrent access")
}
