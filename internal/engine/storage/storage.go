package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

type FileItem struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

type StorageTarget interface {
	Save(ctx context.Context, relPath string, r io.Reader, size int64) error
	Open(ctx context.Context, relPath string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]FileItem, error)
	Delete(ctx context.Context, relPath string) error
	Type() string
}

func NewStorageTarget(cfg config.DestinationConfig) (StorageTarget, error) {
	switch cfg.Type {
	case config.DestinationTypeFilesystem:
		return NewFilesystemStorage(cfg)
	case config.DestinationTypeS3:
		return NewS3Storage(cfg)
	default:
		return nil, fmt.Errorf("unsupported destination type: %s", cfg.Type)
	}
}
