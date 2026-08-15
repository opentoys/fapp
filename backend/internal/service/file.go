package service

import (
	"context"
	"io"

	"disapp/internal/resources/storage"
)

// OpenFile returns an open stream for a storage key. Validates the key shape.
func (s *Service) OpenFile(ctx context.Context, key string) (io.ReadCloser, error) {
	if !storage.ValidKey(key) {
		return nil, &Error{StatusBadRequest, "invalid key"}
	}
	rc, err := s.Storage.Open(ctx, key)
	if err != nil {
		return nil, &Error{StatusNotFound, "file not found"}
	}
	return rc, nil
}