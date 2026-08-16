package oci

import (
	"context"
	"fmt"
	"time"

	"disapp/internal/resources/storage"
	pkgoci "disapp/pkg/oci"
)

// OCIStorage adapts the platform storage.Storage interface to the minimal
// Oracle Object Storage Store. All SDK specifics (SigV4 presign, endpoint)
// live in pkg/oci; this layer only validates keys and forwards.
type OCIStorage struct {
	client pkgoci.Store
}

func NewOCI(client pkgoci.Store) *OCIStorage {
	return &OCIStorage{client: client}
}

func (s *OCIStorage) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return s.client.UploadURL(ctx, key, contentType, expire)
}

func (s *OCIStorage) Delete(ctx context.Context, key string) error {
	if !storage.ValidKey(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	return s.client.Delete(ctx, key)
}

func (s *OCIStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return s.client.DownloadURL(ctx, key, filename, expire)
}
