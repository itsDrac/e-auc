package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/itsDrac/e-auc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadTestImages uploads test images and returns the image names
func uploadTestImages(t *testing.T, env *TestEnv, testUser *TestUser, imageFiles ...string) []string {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Get the project root to locate assets
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if cwd, err2 := os.Getwd(); err2 == nil && filepath.Base(cwd) == "tests" {
		projectRoot, _ = filepath.Abs("..")
	}

	for _, imageFile := range imageFiles {
		imagePath := filepath.Join(projectRoot, "tests", "assets", imageFile)
		file, err := os.Open(imagePath)
		require.NoError(t, err, "Should open test image file: %s", imagePath)
		defer file.Close()

		part, err := writer.CreateFormFile("images", filepath.Base(imagePath))
		require.NoError(t, err, "Should create form file")
		_, err = io.Copy(part, file)
		require.NoError(t, err, "Should copy file content")
	}
	writer.Close()

	// Create upload request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/upload-images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testUser.AccessToken)
	req = addProductAuthContext(req, testUser)

	// Upload images
	w := httptest.NewRecorder()
	env.Dependencies.ProductHandler.UploadImages(w, req)

	require.Equal(t, http.StatusOK, w.Code, "Image upload should succeed")

	// Parse response to get image names
	var uploadResp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&uploadResp)
	require.NoError(t, err, "Should decode upload response")

	data := uploadResp["data"].(map[string]interface{})
	imageNamesInterface := data["image_names"].([]interface{})

	imageNames := make([]string, len(imageNamesInterface))
	for i, name := range imageNamesInterface {
		imageNames[i] = name.(string)
	}

	return imageNames
}

// addProductAuthContext adds user claims to the request context for authenticated product requests
func addProductAuthContext(req *http.Request, testUser *TestUser) *http.Request {
	claims := &config.UserClaims{
		UserID: testUser.UserID,
	}
	ctx := context.WithValue(req.Context(), config.UserClaimKey, claims)
	return req.WithContext(ctx)
}

// addProductIDToContext adds productId URL param to chi context
func addProductIDToContext(req *http.Request, productID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("productId", productID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// addSellerIDToContext adds sellerId URL param to chi context
func addSellerIDToContext(req *http.Request, sellerID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sellerId", sellerID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// TestProductCreation tests product creation endpoint
func TestProductCreation(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	// Get a test user with valid tokens
	testUser := GetRandomTestUser()
	require.NotNil(t, testUser, "Should have at least one test user")
	require.NotEmpty(t, testUser.AccessToken, "Test user should have access token")

	// Upload test images first
	imageNames := uploadTestImages(t, env, testUser, "test_image_1.png")

	tests := []struct {
		name           string
		accessToken    string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "Valid Product Creation",
			accessToken: testUser.AccessToken,
			payload: map[string]interface{}{
				"title":         "Vintage Camera",
				"description":   "A beautiful vintage camera in excellent condition",
				"min_price":     100.00,
				"current_price": 100.00,
				"images":        imageNames,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Product created successfully", body["message"])
				data, ok := body["data"].(map[string]interface{})
				require.True(t, ok, "Response data should be a map")
				assert.NotEmpty(t, data["product_id"])
			},
		},
		{
			name:        "Missing Required Fields",
			accessToken: testUser.AccessToken,
			payload: map[string]interface{}{
				"description": "Missing title and prices",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorData, ok := body["error"].(map[string]interface{})
				require.True(t, ok, "Response should have error field")
				assert.Contains(t, errorData["code"], "VALIDATION")
			},
		},
		{
			name:        "Unauthorized - No Token",
			accessToken: "",
			payload: map[string]interface{}{
				"title":         "Test Product",
				"description":   "Test Description",
				"min_price":     50.00,
				"current_price": 50.00,
				"images":        imageNames,
			},
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
			// Marshal payload
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err, "Should marshal payload")

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add Authorization header if token provided
			if tt.accessToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.accessToken)
				// For valid access token, add auth context
				if tt.accessToken == testUser.AccessToken {
					req = addProductAuthContext(req, testUser)
				}
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			env.Dependencies.ProductHandler.CreateProduct(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Parse response body
			var response map[string]interface{}
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Should be able to decode response")

			// Run custom response checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// TestGetProductByID tests retrieving a product by its ID
func TestGetProductByID(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	testUser := GetRandomTestUser()
	require.NotNil(t, testUser, "Should have test user")

	// Upload images first
	imageNames := uploadTestImages(t, env, testUser, "test_image_2.png")

	// First, create a product
	createPayload := map[string]interface{}{
		"title":         "Test Product for Retrieval",
		"description":   "This product will be retrieved by ID",
		"min_price":     75.00,
		"current_price": 75.00,
		"images":        imageNames,
	}
	createPayloadBytes, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(createPayloadBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+testUser.AccessToken)
	createReq = addProductAuthContext(createReq, testUser)
	createW := httptest.NewRecorder()
	env.Dependencies.ProductHandler.CreateProduct(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code, "Product creation should succeed")

	// Parse product ID
	var createResponse map[string]interface{}
	json.NewDecoder(createW.Body).Decode(&createResponse)
	data := createResponse["data"].(map[string]interface{})
	productID := data["product_id"].(string)
	require.NotEmpty(t, productID, "Should have product ID")

	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Valid Product ID",
			productID:      productID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data")
				product, ok := data["product"].(map[string]interface{})
				require.True(t, ok, "Data should contain product")
				assert.Equal(t, productID, product["id"])
				assert.Equal(t, "Test Product for Retrieval", product["title"])
			},
		},
		{
			name:           "Non-existent Product ID",
			productID:      uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorData, ok := body["error"].(map[string]interface{})
				require.True(t, ok, "Response should have error field")
				assert.Contains(t, errorData["code"], "NOT_FOUND")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with chi URLParam simulation
			url := fmt.Sprintf("/api/v1/products/%s", tt.productID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = addProductIDToContext(req, tt.productID)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			env.Dependencies.ProductHandler.GetProductByID(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Parse response
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Should decode response")

			// Run custom response checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// TestGetProductsBySeller tests retrieving products by seller ID
func TestGetProductsBySeller(t *testing.T) {
	env := GetTestEnv()
	require.NotNil(t, env, "Test environment should be initialized")

	seller := GetTestUser(0)
	require.NotNil(t, seller, "Should have seller user")

	// Upload images once for all products
	imageNames := uploadTestImages(t, env, seller,
		"test_image_1.png",
		"test_image_2.png",
		"test_image_3.png")

	// Create 3 products for the seller
	for i := 0; i < 3; i++ {
		createPayload := map[string]interface{}{
			"title":         fmt.Sprintf("Seller Product %d", i+1),
			"description":   fmt.Sprintf("Product %d by seller", i+1),
			"min_price":     50.00,
			"current_price": 50.00,
			"images":        []string{imageNames[i]},
		}
		payloadBytes, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+seller.AccessToken)
		req = addProductAuthContext(req, seller)
		w := httptest.NewRecorder()
		env.Dependencies.ProductHandler.CreateProduct(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "Product creation should succeed")
	}

	tests := []struct {
		name           string
		sellerID       string
		accessToken    string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "Get Own Products",
			sellerID:       seller.UserID.String(),
			accessToken:    seller.AccessToken,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].(map[string]interface{})
				require.True(t, ok, "Response should contain data")
				products, ok := data["products"].([]interface{})
				require.True(t, ok, "Data should contain products array")
				assert.GreaterOrEqual(t, len(products), 3, "Should have at least 3 products")
			},
		},
		{
			name:           "Get Products with Pagination",
			sellerID:       seller.UserID.String(),
			accessToken:    seller.AccessToken,
			queryParams:    "?limit=2&offset=0",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				products := data["products"].([]interface{})
				assert.LessOrEqual(t, len(products), 2, "Should respect limit parameter")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := fmt.Sprintf("/api/v1/products/seller/%s%s", tt.sellerID, tt.queryParams)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = addSellerIDToContext(req, tt.sellerID)

			// Add Authorization header and context if token provided
			if tt.accessToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.accessToken)
				req = addProductAuthContext(req, seller)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			env.Dependencies.ProductHandler.ProductsBySellerID(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			if tt.expectedStatus == http.StatusOK {
				// Parse response
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err, "Should decode response")

				// Run custom response checks
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}
