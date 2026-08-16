package cos

import (
	"context"
	"fmt"
	"time"

	pkgcos "disapp/pkg/cos"
	"disapp/internal/resources/storage"
)

// COSStorage adapts the platform storage.Storage interface to the minimal
// COS Store. All SDK specifics (presign options, STS transport, HEAD) live in
// pkg/cos; this layer only validates keys and forwards.
type COSStorage struct {
	client pkgcos.Store
}

func NewCOS(client pkgcos.Store) *COSStorage {
	return &COSStorage{client: client}
}

func (s *COSStorage) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return s.client.UploadURL(ctx, key, contentType, expire)
}

func (s *COSStorage) Delete(ctx context.Context, key string) error {
	if !storage.ValidKey(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	return s.client.Delete(ctx, key)
}

func (s *COSStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	return s.client.DownloadURL(ctx, key, filename, expire)
}