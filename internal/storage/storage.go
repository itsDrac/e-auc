package storage

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/itsDrac/e-auc/pkg/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storager interface {
	SaveImage(bucket string, objectKey string, data []byte) (*minio.UploadInfo, error)
	GetFile(bucket string, objectKey string) ([]byte, error)
	GetFileUrl(bucket string, objectKey string) (string, error)
	DeleteFile(bucket string, objectKey string) error
}

type MinioStorage struct {
	client *minio.Client
}

func NewMinioStorage() (*MinioStorage, error) {
	endpoint := utils.GetEnv("MINIO_ENDPOINT", "localhost:9000")
	accessKeyID := utils.GetEnv("MINIO_ACCESS_KEY", "minioadmin")
	secretAccessKey := utils.GetEnv("MINIO_SECRET_KEY", "minioadmin")
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioStorage{
		client: minioClient,
	}, nil
}

func (s *MinioStorage) SaveImage(bucket string, objectKey string, data []byte) (*minio.UploadInfo, error) {
	ctx := context.Background()

	// Check if bucket exists, create if not
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Upload the image
	// TODO: Upload image using FPutObject, for which we'll
	// need to save the file temporarily on disk first
	reader := bytes.NewReader(data)
	info, err := s.client.PutObject(ctx, bucket, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	slog.Info("Storage Layer: File uploaded", "info", info)
	return &info, nil
}

func (s *MinioStorage) GetFile(bucket string, objectKey string) ([]byte, error) {
	return nil, nil
}

func (s *MinioStorage) DeleteFile(bucket string, objectKey string) error {
	return nil
}

func (s *MinioStorage) GetFileUrl(bucket string, objectKey string) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(context.Background(), bucket, objectKey, 24*time.Hour, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}
