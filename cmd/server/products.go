package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
)

func (s *Server) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// Product request validation.
	err := validate.Struct(req)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get Current userClaim form request context
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	product := db.Product{
		Title:        req.Title,
		Description:  req.Description,
		SellerID:     claims.UserID,
		Images:       req.Images,
		MinPrice:     req.MinPrice,
		CurrentPrice: req.CurrentPrice,
	}

	productId, err := s.Services.ProductService.AddProduct(r.Context(), product)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := map[string]any{
		"product_id": productId.String(),
		"message":    "Product created successfully",
	}
	RespondJson(w, http.StatusCreated, resp)
}

func (s *Server) UploadImages(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20) // Limit request body to 50MB
	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Retrieve the files from the "images" form field
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		RespondError(w, http.StatusBadRequest, "no images uploaded")
		return
	}

	var imageURLs []string
	for _, fileHeader := range files {
		// Use fileHeader.Size to check file size if needed
		if fileHeader.Size > 10<<20 { // 10MB limit per file
			RespondError(w, http.StatusBadRequest, "file size exceeds 10MB limit")
			return
		}
		// Check the file type if needed (e.g., only allow JPEG/PNG)
		contentType := fileHeader.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			RespondError(w, http.StatusBadRequest, "only image files are allowed")
			return
		}
		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			RespondError(w, http.StatusInternalServerError, "failed to open uploaded file")
			return
		}
		defer file.Close()
		
		// Read file data
		fileData, err := io.ReadAll(file)
		if err != nil {
			RespondError(w, http.StatusInternalServerError, "failed to read uploaded file")
			return
		}

		// Generate unique filename using UUID and preserve the original extension
		ext := filepath.Ext(fileHeader.Filename)

		uniqueFilename := uuid.New().String() + ext
		fmt.Println("Generated unique filename:", uniqueFilename)
		// Upload to storage service
		imageURL, err := s.Services.ProductService.UploadProductImage(r.Context(), uniqueFilename, fileData)
		if err != nil {
			slog.Error("Error on uploading image", "err:", err.Error())
			RespondError(w, http.StatusInternalServerError, "failed to upload image")
			return
		}

		// Temporary dummy URL for demonstration
		imageURLs = append(imageURLs, imageURL)

		slog.Info("Uploaded image", "original_filename", fileHeader.Filename, "unique_filename", uniqueFilename, "url", imageURL)
	}

	resp := map[string]any{
		"image_urls": imageURLs,
		"message":    "Images uploaded successfully",
	}
	RespondJson(w, http.StatusOK, resp)
}

func (s *Server) GetProductImageUrls(w http.ResponseWriter, r *http.Request) {
	productId := chi.URLParam(r, "productId")
	if productId == "" {
		RespondError(w, http.StatusBadRequest, "product ID is required")
		return
	}

	imageUrls, err := s.Services.ProductService.GetProductUrls(r.Context(), productId)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"image_urls": imageUrls,
	}
	RespondJson(w, http.StatusOK, resp)
}