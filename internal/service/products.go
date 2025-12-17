package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	db "github.com/itsDrac/e-auc/internal/database"
	"github.com/itsDrac/e-auc/internal/storage"
)

type ProductServicer interface {
	AddProduct(context.Context, db.Product) (uuid.UUID, error)
	UploadProductImage(context.Context, string, []byte) (string, error)
	GetProductUrls(context.Context, string) ([]string, error)
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
	product,  err := ps.db.AddProduct(ctx, arg)
	if err != nil {
		return uuid.Nil, err
	}
	return product.ID, nil
}

func (ps *ProductService) UploadProductImage(ctx context.Context, filename string, data []byte) (string, error) {
	bucket := "product-images"
	info, err := ps.storage.SaveImage(bucket, filename, data)
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
		return nil, errors.New("product not found")
	}
	urls := []string{}
	for _, imgKey := range imagekeys {
		url, err := ps.storage.GetFileUrl("product-images", imgKey)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}