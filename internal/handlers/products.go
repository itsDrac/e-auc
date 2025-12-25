package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/model"
	"github.com/itsDrac/e-auc/internal/service"
	"github.com/itsDrac/e-auc/internal/cache"
)

const (
	productParamKey string = "productId"
	sellerParamKey  string = "sellerId"
)

type ProductHandler struct {
	svc service.ProductServicer
	cache cache.Cacher
}

func NewProductHandler(sevc service.ProductServicer, c cache.Cacher) (*ProductHandler, error) {
	return &ProductHandler{
		svc: sevc,
		cache: c,
	}, nil
}

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
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidJson.Error(), "invalid json format", nil)
		return
	}

	// Product request validation.
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

	// Get Current userClaim form request context
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrAuthFailed.Error(), "user claims not found in context", nil)
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

	productId, err := h.svc.AddProduct(r.Context(), product)
	if err != nil {
		if err.Error() == service.ErrInsufficientBid.Error() {
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrSelfBidding.Error(), "you cannot bid on your own product", nil)
			return
		}
		if err.Error() == service.ErrInsufficientBid.Error() {
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrBidLow.Error(), "Your bid must be higher than the current price", nil)
			return
		}
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrInternalServer.Error(), "Internal server error", nil)
		return
	}
	// remove images from temp list after product creation
	for _, imgName := range req.Images {
		// AddImageNameToTempList
		h.cache.RemoveImageNameFromTempList(r.Context(), imgName)
	}
	resp := map[string]any{
		"product_id": productId.String(),
	}
	RespondSuccessJSON(w, r, http.StatusCreated, "Product created successfully", resp)
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
func (h *ProductHandler) UploadImages(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20) // Limit request body to 50MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidForm.Error(), "failed to parse multipart form", nil)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	// Retrieve the files from the "images" form field
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrMissingFiles.Error(), "No images uploaded", nil)
		return
	}

	var imageNames []string
	for _, fileHeader := range files {
		// Use fileHeader.Size to check file size if needed
		if fileHeader.Size > 10<<20 { // 10MB limit per file
			fileNameResp := fmt.Sprintf("File %s exceeds 10MB limit", fileHeader.Filename)
			RespondErrorJSON(w, r, http.StatusBadRequest, ErrLargeFile.Error(), fileNameResp, nil)
			return
		}

		// open file
		file, err := fileHeader.Open()
		if err != nil {
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrFileOpen.Error(), "Failed to process uploaded file", nil)
			return
		}

		// Read file data
		fileData, err := io.ReadAll(file)
		file.Close()

		if err != nil {
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrFileReadError.Error(), "failed to read uploaded file", nil)
			return
		}

		detectedType := http.DetectContentType(fileData)
		if !strings.HasPrefix(detectedType, "image/") {
			fileNameResp := fmt.Sprintf("File %s is not a valid image", fileHeader.Filename)
			RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidFile.Error(), fileNameResp, nil)
			return
		}

		// Generate unique filename using UUID and preserve the original extension
		ext := filepath.Ext(fileHeader.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		uniqueFilename := uuid.New().String() + ext
		fmt.Println("Generated unique filename:", uniqueFilename)
		// Upload to storage service
		imageName, err := h.svc.UploadProductImage(r.Context(), uniqueFilename, fileData)
		if err != nil {
			slog.Error("Error on uploading image", "err:", err.Error())
			RespondErrorJSON(w, r, http.StatusInternalServerError, ErrUploadFailed.Error(), "failed to store image", nil)
			return
		}

		// Temporary dummy URL for demonstration
		imageNames = append(imageNames, imageName)
		h.cache.AddImageNameToTempList(r.Context(), imageName)

		slog.Info("Uploaded image", "original_filename", fileHeader.Filename, "unique_filename", uniqueFilename, "stored_as", imageName)
	}

	resp := map[string]any{
		"image_names": imageNames,
	}
	RespondSuccessJSON(w, r, http.StatusOK, "Images uploaded successfully", resp)
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
func (h *ProductHandler) GetProductImageUrls(w http.ResponseWriter, r *http.Request) {
	// Get Product id form query params
	productId := r.URL.Query().Get(productParamKey)
	if productId == "" {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrMissingParam.Error(), "Product ID is required", nil)
		return
	}

	imageUrls, err := h.svc.GetProductUrls(r.Context(), productId)
	if err != nil {
		if errors.Is(err, service.ErrUrlsNotFound) {
			RespondErrorJSON(w, r, http.StatusNotFound, ErrUrlsNotFound.Error(), "Failed to retrieve images", nil)
			return
		}
		slog.Error("[DB] failed to fetch product images -> ", "product_id", productId, "error", err)
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrDb.Error(), "failed to retrieve product urls", nil)
		return
	}

	// fallback - if db returns a nil for a empty list
	if imageUrls == nil {
		imageUrls = []string{}
	}

	resp := map[string]any{
		"image_urls": imageUrls,
	}
	RespondSuccessJSON(w, r, http.StatusOK, "Images retrieved successfully", resp)
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
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	productId := chi.URLParam(r, productParamKey)
	if productId == "" {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrMissingParam.Error(), "Product ID is required", nil)
		return
	}

	product, err := h.svc.GetProductByID(r.Context(), productId)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			RespondErrorJSON(w, r, http.StatusNotFound, ErrProductNotFound.Error(), "Product not found", nil)
			return
		}
		slog.Error("[DB] failed to fetch product", "product_id", productId, "error", err)
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrDb.Error(), "failed to retrieve product", nil)
		return
	}

	resp := map[string]any{
		"product": product,
	}
	RespondSuccessJSON(w, r, http.StatusOK, "Product fetched successfully", resp)
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
func (h *ProductHandler) PlaceBid(w http.ResponseWriter, r *http.Request) {
	productId := chi.URLParam(r, productParamKey)
	if productId == "" {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrMissingParam.Error(), "Product ID is required", nil)
		return
	}

	var req model.PlaceBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondErrorJSON(w, r, http.StatusBadRequest, ErrInvalidJson.Error(), "Invalid JSON format", nil)
		return
	}

	// Get Current userClaim form request context
	claims := GetUserClaims(r.Context())
	if claims == nil {
		RespondErrorJSON(w, r, http.StatusUnauthorized, ErrAuthFailed.Error(), "user claims not found in context", nil)
		return
	}

	err := h.svc.PlaceBid(r.Context(), productId, claims.UserID, req.BidAmount)
	if err != nil {
		slog.Error("[DB] failed to create bid ->", "product_id", productId, "error", err)
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrBidCreateFailed.Error(), "failed to create bid", nil)
		return
	}

	RespondSuccessJSON(w, r, http.StatusOK, "Bid placed successfully", "")
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
func (h *ProductHandler) ProductsBySellerID(w http.ResponseWriter, r *http.Request) {
	// If seller id is not given in the URL params, then set the seller id to the current user id
	var sellerId string
	sellerId = chi.URLParam(r, sellerParamKey)
	if sellerId == "" {
		claims := GetUserClaims(r.Context())
		if claims == nil {
			RespondErrorJSON(w, r, http.StatusUnauthorized, ErrAuthFailed.Error(), "user claims not found in context", nil)
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

	products, err := h.svc.GetProductsBySellerID(r.Context(), sellerId, limit, offset)
	if err != nil {
		slog.Error("[DB] failed to found products -> ", "seller_id", sellerId, "error", err)
		RespondErrorJSON(w, r, http.StatusInternalServerError, ErrDb.Error(), "failed to retrieve products", nil)
		return
	}

	resp := map[string]any{
		"products": products,
	}
	RespondSuccessJSON(w, r, http.StatusOK, "products fetched successfully", resp)
}
