package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

type FilesystemStorage struct {
	baseDir string
}

func NewFilesystemStorage(cfg config.DestinationConfig) (*FilesystemStorage, error) {
	return &FilesystemStorage{baseDir: "/"}, nil
}

func (fs *FilesystemStorage) Type() string {
	return string(config.DestinationTypeFilesystem)
}

func (fs *FilesystemStorage) Save(ctx context.Context, targetPath string, r io.Reader, size int64) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	tmpFile := targetPath + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create file %s: %w", tmpFile, err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpFile)
	}()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tmpFile, targetPath); err != nil {
		return fmt.Errorf("rename to %s: %w", targetPath, err)
	}

	return nil
}

func (fs *FilesystemStorage) Open(ctx context.Context, targetPath string) (io.ReadCloser, error) {
	return os.Open(targetPath)
}

func (fs *FilesystemStorage) List(ctx context.Context, dirPath string) ([]FileItem, error) {
	entries, err := os.ReadDir(dirPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []FileItem
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			Path:    filepath.Join(dirPath, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return items, nil
}

func (fs *FilesystemStorage) Delete(ctx context.Context, targetPath string) error {
	return os.Remove(targetPath)
}
