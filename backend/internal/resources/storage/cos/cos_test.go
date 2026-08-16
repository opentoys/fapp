package cos

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type fakeObject struct {
	deleteErr error
	headSize  int64
	headErr   error
}

func (f *fakeObject) Delete(ctx context.Context, key string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
	return &cos.Response{}, f.deleteErr
}

func (f *fakeObject) GetPresignedURL2(ctx context.Context, httpMethod, key string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
	u := "https://bucket.cos/" + key + "?download-sign=x"
	if httpMethod == http.MethodPut {
		u = "https://bucket.cos/" + key + "?upload-sign=x"
	}
	if o, ok := opt.(*cos.PresignedURLOptions); ok && o.Query != nil {
		u += "&response-content-disposition=" + o.Query.Get("response-content-disposition")
	}
	return url.Parse(u)
}

func (f *fakeObject) Head(ctx context.Context, key string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
	return &cos.Response{
		Response: &http.Response{StatusCode: http.StatusOK, ContentLength: f.headSize},
	}, f.headErr
}

func TestCOSUploadPresign(t *testing.T) {
	obj := &fakeObject{}
	c := NewCOS(obj, "bucket", "https://bucket.cos")
	key := "1/2/app.apk"

	u, err := c.UploadURL(context.Background(), key, "application/vnd.android.package-archive", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "upload-sign=x") {
		t.Fatalf("upload url = %q", u)
	}

	du, err := c.DownloadURL(context.Background(), key, "app.apk", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(du, "download-sign=x") {
		t.Fatalf("download url = %q", du)
	}
	if !strings.Contains(du, "response-content-disposition") {
		t.Fatalf("download url missing attachment param: %q", du)
	}

	if _, err := c.Size(context.Background(), key); err != nil {
		t.Fatal(err)
	}
}

func TestCOSSize(t *testing.T) {
	obj := &fakeObject{headSize: 123}
	c := NewCOS(obj, "bucket", "https://bucket.cos")
	n, err := c.Size(context.Background(), "1/2/app.apk")
	if err != nil || n != 123 {
		t.Fatalf("size = %d, err = %v", n, err)
	}
}

func TestCOSDeleteError(t *testing.T) {
	obj := &fakeObject{deleteErr: errors.New("boom")}
	c := NewCOS(obj, "bucket", "https://bucket.cos")
	if err := c.Delete(context.Background(), "1/2/x.apk"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestCOSInvalidKey(t *testing.T) {
	c := NewCOS(&fakeObject{}, "bucket", "https://bucket.cos")
	if _, err := c.UploadURL(context.Background(), "../x", "", time.Hour); err == nil {
		t.Fatal("expected invalid key error")
	}
}