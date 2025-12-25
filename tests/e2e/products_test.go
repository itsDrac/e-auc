package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/handlers"
	"github.com/itsDrac/e-auc/internal/middleware"
	"github.com/itsDrac/e-auc/internal/model"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/internal/storage"
	"github.com/itsDrac/e-auc/tests"
)

// 1. Define a Local Storage Implementation
// This exists ONLY for this test file. It satisfies the service.Storager interface.
type testMinioStorage struct {
	client     *minio.Client
	bucketName string
}

func (s *testMinioStorage) UploadProductImage(ctx context.Context, fileName string, data []byte) (string, error) {
	// Simple upload logic
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucketName, fileName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", err
	}
	// we just need to verify the upload succeeded without error.
	return fmt.Sprintf("http://localhost:9000/%s/%s", s.bucketName, fileName), nil
}

func TestProductJourney(t *testing.T) {
	// Sets dummy JWT secrets for the test session
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-secret")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret")

	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	conn, cleanup := tests.SetupTestDB(t)
	defer cleanup()

	// 2. Initialize Dependencies
	store := db.New(conn)

	// storage
	// B. Setup MinIO (STANDALONE - No external helpers)
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"server", "/data"},
		Env:          map[string]string{"MINIO_ROOT_USER": "admin", "MINIO_ROOT_PASSWORD": "password"},
		WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start minio: %s", err)
	}
	defer minioContainer.Terminate(ctx)

	minioStorage, err := storage.NewMinioStorage()
	if err != nil {
		t.Fatalf("failed to start minioStorage")
	}

	authSvc, _ := service.NewAuthService(store)
	userSvc, _ := service.NewUserService(store)
	productSvc, _ := service.NewProductService(store, minioStorage)

	userHandler, _ := handlers.NewUserHandler(userSvc, authSvc)
	productHandler, _ := handlers.NewProductHandler(productSvc)

	// 3. Setup Router (Mirroring your Server.ProductRoutes logic)
	r := chi.NewRouter()

	// Auth Routes (Needed to get tokens)
	r.Post("/auth/register", userHandler.RegisterUser)
	r.Post("/auth/login", userHandler.LoginUser)

	// Product Routes Wiring

	// 2. PUBLIC Product Routes (Explicitly defined outside any group)
	r.Get("/products/{productId}", productHandler.GetProductByID)

	r.Route("/products", func(r chi.Router) {
		// Protected Routes
		r.Group(func(protected chi.Router) {
			protected.Use(middleware.AuthMiddleware(authSvc))
			protected.Post("/upload-images", productHandler.UploadImages)
			protected.Post("/", productHandler.CreateProduct)
			protected.Patch("/{productId}/bid", productHandler.PlaceBid)
			protected.Get("/{sellerId}", productHandler.ProductsBySellerID)
		})
	})

	// --- HELPERS ---
	// Helper to create a user and return their Access Token
	getAuthToken := func(username, email string) string {
		// Register
		regBody, _ := json.Marshal(model.CreateUserRequest{
			Username: username, Email: email, Password: "password123",
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody)))

		// Login
		loginBody, _ := json.Marshal(model.LoginUserRequest{
			Username: username, Password: "password123",
		})
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		return data["access_token"].(string)
	}

	// --- START JOURNEY ---

	sellerToken := getAuthToken("SellerUser", "seller@test.com")
	bidderToken := getAuthToken("BidderUser", "bidder@test.com")

	var productID string
	var imageURL string

	t.Run("1. Upload Image", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create the field "images" (Must match handler's r.MultipartForm.File["images"])
		part, err := writer.CreateFormFile("images", "test.jpg")
		assert.NoError(t, err)

		_, err = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01})
		assert.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest("POST", "/products/upload-images", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+sellerToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("Upload Failed Response: %s", rr.Body.String())
		}

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if rr.Code == 200 {
			data := resp["data"].(map[string]any)
			urls := data["image_urls"].([]any)
			imageURL = urls[0].(string)
		} else {
			imageURL = "http://localhost:9000/fallback.jpg"
		}
	})

	description := "Old camera"
	t.Run("2. Create Product", func(t *testing.T) {
		payload := model.CreateProductRequest{
			Title:        "Vintage Camera",
			Description:  &description,
			MinPrice:     100.0,
			CurrentPrice: 100.0,
			Images:       []string{imageURL},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/products", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+sellerToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		productID = data["product_id"].(string)

		assert.NotEmpty(t, productID)
	})

	t.Run("3. Get Product Publicly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/products/"+productID, nil)
		req.Header.Del("Authorization")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Logf("Public Get Failed: %s", rr.Body.String())
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Vintage Camera")
	})

	t.Run("4. Bidder Places Bid", func(t *testing.T) {
		payload := model.PlaceBidRequest{
			BidAmount: 150.0,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/products/"+productID+"/bid", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+bidderToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("5. Verify New Price", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/products/"+productID, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Contains(t, rr.Body.String(), "150")
	})

	t.Run("6. Seller Cannot Bid", func(t *testing.T) {
		payload := model.PlaceBidRequest{
			BidAmount: 200.0,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/products/"+productID+"/bid", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+sellerToken) // Using Seller Token!
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.NotEqual(t, http.StatusOK, rr.Code)
		assert.Contains(t, strings.ToLower(rr.Body.String()), "own product")
	})
}
