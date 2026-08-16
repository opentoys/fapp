package oci

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkgoci "disapp/pkg/oci"
)

type fakeStore struct {
	uploadURL   func(ctx context.Context, key, contentType string, expire time.Duration) (string, error)
	deleteErr   error
	downloadURL func(ctx context.Context, key, filename string, expire time.Duration) (string, error)
}

func (f *fakeStore) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	if f.uploadURL != nil {
		return f.uploadURL(ctx, key, contentType, expire)
	}
	return "https://bucket.ns.compat.objectstorage.ap.oraclecloud.com/upload?x=1", nil
}

func (f *fakeStore) Delete(ctx context.Context, key string) error {
	return f.deleteErr
}

func (f *fakeStore) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if f.downloadURL != nil {
		return f.downloadURL(ctx, key, filename, expire)
	}
	return "https://bucket.ns.compat.objectstorage.ap.oraclecloud.com/upload?x=1", nil
}

func TestUploadPresign(t *testing.T) {
	c := NewOCI(&fakeStore{uploadURL: func(_ context.Context, key, _ string, _ time.Duration) (string, error) {
		return "https://bucket.ns.compat.objectstorage.ap.oraclecloud.com/" + key + "?X-Amz-Signature=x", nil
	}})
	u, err := c.UploadURL(context.Background(), "1/2/app.apk", "application/vnd.android.package-archive", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "X-Amz-Signature=x") {
		t.Fatalf("upload url = %q", u)
	}
}

func TestDownloadPresign(t *testing.T) {
	c := NewOCI(&fakeStore{downloadURL: func(_ context.Context, key, filename string, _ time.Duration) (string, error) {
		return "https://bucket.ns.compat.objectstorage.ap.oraclecloud.com/" + key + "?X-Amz-Signature=x", nil
	}})
	du, err := c.DownloadURL(context.Background(), "1/2/app.apk", "app.apk", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(du, "X-Amz-Signature=x") {
		t.Fatalf("download url = %q", du)
	}
}

func TestDeleteError(t *testing.T) {
	c := NewOCI(&fakeStore{deleteErr: errors.New("boom")})
	if err := c.Delete(context.Background(), "1/2/x.apk"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestInvalidKey(t *testing.T) {
	c := NewOCI(&fakeStore{})
	if _, err := c.UploadURL(context.Background(), "../x", "", time.Hour); err == nil {
		t.Fatal("expected invalid key error")
	}
}

var _ pkgoci.Store = (*fakeStore)(nil)
