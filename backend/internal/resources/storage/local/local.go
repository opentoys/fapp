package local

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"disapp/internal/resources/storage"
)

type LocalStorage struct {
	dir    string
	secret string
}

// NewLocal creates local storage backed by the directory dir. secret is used
// to sign upload/preview URLs; pass the JWT secret.
func NewLocal(dir string, secret string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: dir, secret: secret}, nil
}

// path resolves a validated key to the on-disk location.
func (s *LocalStorage) path(key string) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return filepath.Join(s.dir, filepath.FromSlash(key)), nil
}

// Save writes the reader to key (concrete method used by the upload endpoint).
func (s *LocalStorage) Save(ctx context.Context, key string, r io.Reader) (int64, error) {
	full, err := s.path(key)
	if err != nil {
		return 0, err
	}
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

// Open returns a read stream for key (concrete method used by the preview
// endpoint).
func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	full, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

// Size returns the on-disk size of key (concrete method, optional capability).
func (s *LocalStorage) Size(ctx context.Context, key string) (int64, error) {
	full, err := s.path(key)
	if err != nil {
		return 0, err
	}
	st, err := os.Stat(full)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	full, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// UploadURL returns the signed server upload endpoint. The client POSTs the
// file body to this URL with ?key=<key>, and the server stores it under dir.
func (s *LocalStorage) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	ttl := time.Now().Add(expire).Unix()
	q := url.Values{
		"key": {key},
		"ttl": {strconv.FormatInt(ttl, 10)},
		"sign": {signUpload(s.secret, ttl)},
	}
	return "/api/v1/files/upload?" + q.Encode(), nil
}

// DownloadURL returns the signed preview URL. With ?dl the endpoint 307s to
// the actual streamed file; with the signature it streams directly.
func (s *LocalStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	ttl := time.Now().Add(expire).Unix()
	q := url.Values{"key": {key}, "ttl": {strconv.FormatInt(ttl, 10)}, "sign": {signPreview(s.secret, key, ttl)}}
	return "/api/v1/files/preview?" + q.Encode(), nil
}
