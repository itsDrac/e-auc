package service

import (
	"context"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/storage"
	"github.com/jackc/pgx/v5"
)

const bucketName = "product-images"

type ProductServicer interface {
	AddProduct(context.Context, db.Product) (uuid.UUID, error)
	UploadProductImage(context.Context, string, []byte) (string, error)
	GetProductUrls(context.Context, string) ([]string, error)
	GetProductByID(context.Context, string) (*db.Product, error)
	PlaceBid(context.Context, string, uuid.UUID, int32) error
	GetProductsBySellerID(context.Context, string, uint, uint) ([]db.Product, error)
	// Define methods related to product service here
}

type ProductService struct {
	db      db.Querier
	storage storage.Storager
}

func NewProductService(db db.Querier, s storage.Storager) (*ProductService, error) {
	return &ProductService{
		db:      db,
		storage: s,
	}, nil
}

func (ps *ProductService) AddProduct(ctx context.Context, p db.Product) (uuid.UUID, error) {
	arg := db.AddProductParams{
		Title:        p.Title,
		Description:  p.Description,
		SellerID:     p.SellerID,
		Images:       p.Images,
		MinPrice:     p.MinPrice,
		CurrentPrice: p.CurrentPrice,
	}
	product, err := ps.db.AddProduct(ctx, arg)
	if err != nil {
		return uuid.Nil, err
	}
	return product.ID, nil
}

func (ps *ProductService) UploadProductImage(ctx context.Context, filename string, data []byte) (string, error) {
	info, err := ps.storage.SaveImage(bucketName, filename, data)
	if err != nil {
		return "", err
	}
	return info.Key, nil
}

func (ps *ProductService) GetProductUrls(ctx context.Context, productId string) ([]string, error) {
	productUUID, err := uuid.Parse(productId)
	if err != nil {
		return nil, err
	}
	imagekeys, err := ps.db.GetProductImages(ctx, productUUID)
	if err != nil {
		return nil, err
	}
	if imagekeys == nil {
		return nil, ErrUrlsNotFound
	}
	urls := []string{}
	for _, imgKey := range imagekeys {
		url, err := ps.storage.GetFileUrl(bucketName, imgKey)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (ps *ProductService) GetProductByID(ctx context.Context, productId string) (*db.Product, error) {
	productUUID, err := uuid.Parse(productId)
	if err != nil {
		return nil, err
	}
	product, err := ps.db.GetProductByID(ctx, productUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (ps *ProductService) PlaceBid(ctx context.Context, productId string, bidderId uuid.UUID, bidAmount int32) error {
	productUUID, err := uuid.Parse(productId)
	if err != nil {
		return err
	}
	// Should we check if the product.SellerId == bidderId to prevent self-bidding?
	product, err := ps.db.GetProductByID(ctx, productUUID)
	if err != nil {
		return err
	}
	if product.SellerID == bidderId {
		return ErrSelfBidding
	}

	// TODO: Add check for threshold bidding amount for the product
	if bidAmount <= product.CurrentPrice {
		return ErrInsufficientBid
	}

	// Check if the last valid bidder is not the current bidder
	lastBid, err := ps.db.GetLatestBidForProduct(ctx, productUUID)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if err == nil && lastBid.UserID == bidderId {
		return ErrConsecutiveBid
	}

	// Store the bid in bids table
	err = ps.db.CreateBid(ctx, db.CreateBidParams{
		ProductID: productUUID,
		UserID:    bidderId,
		Price:     bidAmount,
		Comments:  nil,
	})
	if err != nil {
		return err
	}

	// TODO: Move this function to a separate worker or use a message queue for better scalability
	// TODO: Add retry mechanism in case of transient failures
	// Update product current price in a goroutine
	go func() {
		// Use background context to avoid cancellation when original context is done
		bgCtx := context.Background()
		_ = ps.db.UpdateProductCurrentPrice(bgCtx, db.UpdateProductCurrentPriceParams{
			ID:           productUUID,
			CurrentPrice: bidAmount,
		})
		// TODO: Add logging if update fails
		// TODO: Add code to check for seller threshold on bidding of its products.
		// TODO: If the bidding amount is higher than the threshold, notify the seller via email.
	}()

	return nil
}

func (ps *ProductService) GetProductsBySellerID(ctx context.Context, sellerId string, limit uint, offset uint) ([]db.Product, error) {
	sellerUUID, err := uuid.Parse(sellerId)
	if err != nil {
		return nil, err
	}
	args := db.GetProductsBySellerIDParams{
		SellerID: sellerUUID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	}
	products, err := ps.db.GetProductsBySellerID(ctx, args)
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = []db.Product{}
	}
	return products, nil
}
