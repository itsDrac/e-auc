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

// CreateProduct godoc
//
//	@Summary		Create a new Product
//	@Description	Create a new product listing
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			product	body		CreateProductRequest	true	"Product details"
//	@Success		201		{object}	map[string]any
//	@Failure		400		{object}	map[string]any
//	@Failure		401		{object}	map[string]any
//	@Router			/products [post]
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

// UploadImages godoc
//
//	@Summary		Upload Product Images
//	@Description	Upload images for a product
//	@Tags			Products
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			images	formData	file	true	"Product images"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]any
//	@Failure		401		{object}	map[string]any
//	@Router			/products/upload-images [post]
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

// GetProductImageUrls godoc
//
//	@Summary		Get Product Image URLs
//	@Description	Retrieve image URLs for a specific product by the given product ID
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			productId	path		string	true	"Product ID"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Router			/products/{productId}/images [get]
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

// GetProductByID godoc
//
//	@Summary		Get Product by ID
//	@Description	Retrieve a specific product by the given product ID
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			productId	path		string	true	"Product ID"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Router			/products/{productId} [get]
func (s *Server) GetProductByID(w http.ResponseWriter, r *http.Request) {
	productId := chi.URLParam(r, "productId")
	if productId == "" {
		RespondError(w, http.StatusBadRequest, "product ID is required")
		return
	}

	product, err := s.Services.ProductService.GetProductByID(r.Context(), productId)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"product": product,
	}
	RespondJson(w, http.StatusOK, resp)
}

// PlaceBid godoc
//
//	@Summary		Place a Bid on a Product
//	@Description	Place a bid(update current price) on a specific product by the given product ID
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			productId	path		string			true	"Product ID"
//	@Param			bid			body		PlaceBidRequest	true	"Bid details"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	map[string]any
//	@Failure		401			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Router			/products/{productId}/bid [patch]
func (s *Server) PlaceBid(w http.ResponseWriter, r *http.Request) {
	productId := chi.URLParam(r, "productId")
	if productId == "" {
		RespondError(w, http.StatusBadRequest, "product ID is required")
		return
	}

	var req PlaceBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// Get Current userClaim form request context
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := s.Services.ProductService.PlaceBid(r.Context(), productId, claims.UserID, req.BidAmount)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"message": "Bid placed successfully",
	}
	RespondJson(w, http.StatusOK, resp)
}

// ProductsBySellerID godoc
//
//	@Summary		Get Products by Seller ID
//	@Description	Retrieve products listed by a specific seller. If no seller ID is provided, retrieves products for the current user.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			sellerId	path		string	false	"Seller ID"
//	@Param			limit		query		int		false	"Number of products to return"
//	@Param			offset		query		int		false	"Number of products to skip"
//	@Success		200			{object}	map[string]any
//	@Failure		400			{object}	map[string]any
//	@Failure		401			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Router			/products/{sellerId} [get]
func (s *Server) ProductsBySellerID(w http.ResponseWriter, r *http.Request) {
	// If seller id is not given in the URL params, then set the seller id to the current user id
	var sellerId string
	sellerId = chi.URLParam(r, "sellerId")
	if sellerId == "" {
		claims := GetUserClaims(r.Context())
		if claims == nil {
			RespondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sellerId = claims.UserID.String()
	}
	// Get limit and offset from query params for pagination
	// Default limit is 10 and offset is 0
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")
	var limit uint = 10
	var offset uint = 0
	if limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}
	if offsetParam != "" {
		fmt.Sscanf(offsetParam, "%d", &offset)
	}

	products, err := s.Services.ProductService.GetProductsBySellerID(r.Context(), sellerId, limit, offset)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"products": products,
	}
	RespondJson(w, http.StatusOK, resp)
}
