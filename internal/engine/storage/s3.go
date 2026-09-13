package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewS3Storage(cfg config.DestinationConfig) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "s3.amazonaws.com"
	}
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	useSSL := !cfg.Insecure

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (s *S3Storage) Type() string {
	return string(config.DestinationTypeS3)
}

func (s *S3Storage) Save(ctx context.Context, targetPath string, r io.Reader, size int64) error {
	targetPath = strings.TrimPrefix(targetPath, "/")
	opts := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}
	// If size is <= 0 (e.g. streaming without known length), minio uses chunked upload with size -1
	if size <= 0 {
		size = -1
	}

	_, err := s.client.PutObject(ctx, s.bucket, targetPath, r, size, opts)
	if err != nil {
		return fmt.Errorf("s3 upload %s: %w", targetPath, err)
	}
	return nil
}

func (s *S3Storage) Open(ctx context.Context, targetPath string) (io.ReadCloser, error) {
	targetPath = strings.TrimPrefix(targetPath, "/")
	obj, err := s.client.GetObject(ctx, s.bucket, targetPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 get %s: %w", targetPath, err)
	}
	return obj, nil
}

func (s *S3Storage) List(ctx context.Context, prefix string) ([]FileItem, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	var items []FileItem

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return nil, obj.Err
		}
		items = append(items, FileItem{
			Path:    obj.Key,
			Size:    obj.Size,
			ModTime: obj.LastModified,
		})
	}

	return items, nil
}

func (s *S3Storage) Delete(ctx context.Context, targetPath string) error {
	targetPath = strings.TrimPrefix(targetPath, "/")
	return s.client.RemoveObject(ctx, s.bucket, targetPath, minio.RemoveObjectOptions{})
}
