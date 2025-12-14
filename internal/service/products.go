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
	return uuid.Nil, errors.New("Not implemented.")
}

func (ps *ProductService) UploadProductImage(ctx context.Context, filename string, data []byte) (string, error) {
	bucket := "product-images"
	info, err := ps.storage.SaveImage(bucket, filename, data)
	if err != nil {
		return "", err
	}
	return info.Key, nil
}