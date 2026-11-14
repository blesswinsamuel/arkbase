package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/blesswinsamuel/arkbase/internal/engine/storage"
)

type fakeTarget struct {
	items   []storage.FileItem
	saveErr error
	saved   []string
	deleted []string
}

func (f *fakeTarget) Save(_ context.Context, relPath string, r io.Reader, size int64) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	f.saved = append(f.saved, relPath)
	f.items = append(f.items, storage.FileItem{Path: relPath, Size: size, ModTime: time.Now().UTC()})
	return nil
}

func (f *fakeTarget) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTarget) List(context.Context, string) ([]storage.FileItem, error) {
	return f.items, nil
}

func (f *fakeTarget) Delete(_ context.Context, relPath string) error {
	f.deleted = append(f.deleted, relPath)
	return nil
}

func (f *fakeTarget) Type() string { return "fake" }

func newTestRunner(t *testing.T, retention *config.RetentionPolicy, local, s3 storage.StorageTarget) *Runner {
	t.Helper()

	cfg := &config.Config{
		Destinations: map[string]config.DestinationConfig{
			"local": {Type: config.DestinationTypeFilesystem, Path: filepath.Join(t.TempDir(), "{{ .Database }}-{{ .Timestamp }}.sql.gz"), Retention: retention},
			"s3":    {Type: config.DestinationTypeS3, Path: "backups/{{ .Database }}/{{ .Database }}-{{ .Timestamp }}.sql.gz.enc", Retention: retention},
		},
		Databases: map[string]config.DatabaseConfig{
			"db": {Engine: "postgres", Destinations: []string{"local", "s3"}},
		},
	}

	return &Runner{
		cfg:          cfg,
		destinations: map[string]storage.StorageTarget{"local": local, "s3": s3},
	}
}

func stagedDump(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dump.raw")
	if err := os.WriteFile(path, []byte("dumpdata"), 0o600); err != nil {
		t.Fatalf("write staged dump: %v", err)
	}
	return path
}

func oldBackup(name string) storage.FileItem {
	return storage.FileItem{Path: "backups/db/" + name, Size: 10, ModTime: time.Now().UTC().Add(-30 * 24 * time.Hour)}
}

func TestDeliverDumpUploadsToEveryDestinationAndPrunes(t *testing.T) {
	ctx := context.Background()
	local := &fakeTarget{items: []storage.FileItem{oldBackup("db-old-local.sql.gz")}}
	s3 := &fakeTarget{items: []storage.FileItem{oldBackup("db-old-s3.sql.gz.enc")}}
	r := newTestRunner(t, &config.RetentionPolicy{KeepLast: 1}, local, s3)

	succeeded, failures := r.deliverDump(ctx, "db", r.cfg.Databases["db"], "run1", "20261114-020000", stagedDump(t), 8, t.TempDir(), &bytes.Buffer{})

	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if strings.Join(succeeded, ",") != "local,s3" {
		t.Fatalf("expected both destinations to succeed, got %v", succeeded)
	}
	for name, target := range map[string]*fakeTarget{"local": local, "s3": s3} {
		if len(target.saved) != 1 {
			t.Fatalf("%s: expected 1 upload, got %v", name, target.saved)
		}
		if len(target.deleted) != 1 {
			t.Fatalf("%s: expected exactly one prune, got %v", name, target.deleted)
		}
		if target.deleted[0] != target.items[0].Path {
			t.Fatalf("%s: expected the old backup pruned, got %v", name, target.deleted)
		}
	}
}

func TestDeliverDumpPrunesEvenWhenUploadFails(t *testing.T) {
	ctx := context.Background()
	local := &fakeTarget{}
	s3 := &fakeTarget{
		items: []storage.FileItem{
			oldBackup("db-older.s3.sql.gz.enc"),
			{Path: "backups/db/db-old.s3.sql.gz.enc", Size: 10, ModTime: time.Now().UTC().Add(-20 * 24 * time.Hour)},
		},
		saveErr: errors.New("storage cap exceeded"),
	}
	r := newTestRunner(t, &config.RetentionPolicy{KeepLast: 1}, local, s3)

	succeeded, failures := r.deliverDump(ctx, "db", r.cfg.Databases["db"], "run1", "20261114-020000", stagedDump(t), 8, t.TempDir(), &bytes.Buffer{})

	if strings.Join(succeeded, ",") != "local" {
		t.Fatalf("expected only local to succeed, got %v", succeeded)
	}
	if len(failures) != 1 || !strings.Contains(failures[0], "storage cap exceeded") {
		t.Fatalf("expected the s3 failure to be reported, got %v", failures)
	}
	if len(s3.deleted) != 1 || s3.deleted[0] != "backups/db/db-older.s3.sql.gz.enc" {
		t.Fatalf("expected the failed destination to still be pruned, got %v", s3.deleted)
	}
}
