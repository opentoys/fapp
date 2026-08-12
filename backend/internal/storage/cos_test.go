package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type fakeObject struct {
	putKey string
	putErr error
}

func (f *fakeObject) Put(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	f.putKey = key
	io.Copy(io.Discard, r)
	return &cos.Response{}, f.putErr
}

func (f *fakeObject) Get(ctx context.Context, key string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
	return &cos.Response{
		Response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("cos-data"))},
	}, nil
}

func (f *fakeObject) Delete(ctx context.Context, key string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
	return &cos.Response{}, nil
}

func (f *fakeObject) GetPresignedURL(ctx context.Context, method, key, akID, skID string, exp time.Duration, opt interface{}, signHost ...bool) (*url.URL, error) {
	return url.Parse("https://bucket.cos/" + key + "?sign=x")
}

func TestCOSSaveOpenDeletePresign(t *testing.T) {
	obj := &fakeObject{}
	c := NewCOS(obj, "ak", "sk", "bucket", "https://bucket.cos")
	key := "1/2/app.apk"

	n, err := c.Save(context.Background(), key, strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("n = %d", n)
	}
	if obj.putKey != key {
		t.Fatalf("put key = %q", obj.putKey)
	}

	rc, err := c.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "cos-data" {
		t.Fatalf("data = %q", data)
	}

	u, err := c.DownloadURL(context.Background(), key, "app.apk", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "sign=x") {
		t.Fatalf("url = %q", u)
	}

	if err := c.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
}

func TestCOSPutError(t *testing.T) {
	obj := &fakeObject{putErr: errors.New("boom")}
	c := NewCOS(obj, "ak", "sk", "bucket", "https://bucket.cos")
	if _, err := c.Save(context.Background(), "1/2/x.apk", strings.NewReader("hi")); err == nil {
		t.Fatal("expected put error")
	}
}