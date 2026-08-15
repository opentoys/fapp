package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"disapp/internal/resources/storage"
)

type LocalStorage struct {
	dir string
}

// NewLocal creates local storage, creating the directory if needed.
func NewLocal(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: dir}, nil
}

func (s *LocalStorage) Save(ctx context.Context, key string, r io.Reader) (int64, error) {
	if !storage.ValidKey(key) {
		return 0, fmt.Errorf("invalid key %q", key)
	}
	full := filepath.Join(s.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return 0, err
	}
	f, err := os.Create(full)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return n, err
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !storage.ValidKey(key) {
		return nil, fmt.Errorf("invalid key %q", key)
	}
	full := filepath.Join(s.dir, filepath.FromSlash(key))
	return os.Open(full)
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if !storage.ValidKey(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	full := filepath.Join(s.dir, filepath.FromSlash(key))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	return "/api/v1/files/" + key, nil
}