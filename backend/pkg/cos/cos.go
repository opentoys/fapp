// Package cos is the minimal COS surface the app consumes. It owns the only
// direct dependency on cos-go-sdk-v5: callers build an Object via NewFromConfig
// and never touch SDK types (presign options, responses, STS transport).
package cos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// Store is the four operations the platform needs from object storage:
// presigned upload/download URLs and delete.
type Store interface {
	UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error)
}

type client struct {
	obj *cos.ObjectService
}

// NewFromConfig builds the real client backed by STS temporary credentials.
// baseURL is optional; when empty it derives from bucket+region.
func NewFromConfig(secretID, secretKey, bucket, region, baseURL string) (Store, error) {
	if bucket == "" || region == "" {
		return nil, fmt.Errorf("cos bucket and region are required")
	}
	base := baseURL
	if base == "" {
		base = "https://" + bucket + ".cos." + region + ".myqcloud.com"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	transport := &cos.StsCredentialTransport{
		SecretID:  secretID,
		SecretKey: secretKey,
		Transport: http.DefaultTransport,
		Region:    region,
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{Transport: transport})
	return &client{obj: c.Object}, nil
}

// UploadURL presigns a PUT for the client to push bytes straight to storage.
func (c *client) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	u, err := c.obj.GetPresignedURL2(ctx, http.MethodPut, key, expire, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	_, err := c.obj.Delete(ctx, key)
	return err
}

// DownloadURL presigns a GET that streams the object as an attachment.
func (c *client) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	query := url.Values{"response-content-disposition": []string{
		fmt.Sprintf("attachment; filename=%q", filename),
	}}
	opt := &cos.PresignedURLOptions{Query: &query}
	u, err := c.obj.GetPresignedURL2(ctx, http.MethodGet, key, expire, opt)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}