package cos

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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
	return "https://bucket.cos/upload?x=1", nil
}

func (f *fakeStore) Delete(ctx context.Context, key string) error {
	return f.deleteErr
}

func (f *fakeStore) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if f.downloadURL != nil {
		return f.downloadURL(ctx, key, filename, expire)
	}
	return "https://bucket.cos/download?x=1", nil
}

func TestCOSUploadPresign(t *testing.T) {
	c := NewCOS(&fakeStore{uploadURL: func(_ context.Context, key, _ string, _ time.Duration) (string, error) {
		return "https://bucket.cos/" + key + "?upload-sign=x", nil
	}})
	u, err := c.UploadURL(context.Background(), "wechat/1/2/app.apk", "application/vnd.android.package-archive", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "upload-sign=x") {
		t.Fatalf("upload url = %q", u)
	}
}

func TestCOSDownloadPresign(t *testing.T) {
	c := NewCOS(&fakeStore{downloadURL: func(_ context.Context, key, filename string, _ time.Duration) (string, error) {
		return "https://bucket.cos/" + key + "?download-sign=x&name=" + filename, nil
	}})
	du, err := c.DownloadURL(context.Background(), "wechat/1/2/app.apk", "app.apk", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(du, "download-sign=x") {
		t.Fatalf("download url = %q", du)
	}
}


func TestCOSDeleteError(t *testing.T) {
	c := NewCOS(&fakeStore{deleteErr: errors.New("boom")})
	if err := c.Delete(context.Background(), "wechat/1/2/x.apk"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestCOSInvalidKey(t *testing.T) {
	c := NewCOS(&fakeStore{})
	if _, err := c.UploadURL(context.Background(), "../x", "", time.Hour); err == nil {
		t.Fatal("expected invalid key error")
	}
}

