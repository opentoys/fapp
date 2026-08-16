package cos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"

	"disapp/internal/resources/storage"
)

// objectClient is a narrow interface for cos object operations, for test mocking.
type objectClient interface {
	Delete(ctx context.Context, key string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error)
	GetPresignedURL2(ctx context.Context, httpMethod, key string, expired time.Duration, opt interface{}, signHost ...bool) (*url.URL, error)
	Head(ctx context.Context, key string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)
}

type COSStorage struct {
	client  objectClient
	baseURL string
}

func NewCOS(client objectClient, bucket, baseURL string) *COSStorage {
	return &COSStorage{client: client, baseURL: baseURL}
}

// NewCOSFromConfig creates COSStorage from config parameters. The underlying
// client uses a StsCredentialTransport so presigned URLs are signed with
// short-lived temporary credentials (no server-side file writes needed).
func NewCOSFromConfig(secretID, secretKey, bucket, region, baseURL string) (*COSStorage, error) {
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
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{Transport: transport})
	return NewCOS(client.Object, bucket, base), nil
}

func (s *COSStorage) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	u, err := s.client.GetPresignedURL2(ctx, http.MethodPut, key, expire, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *COSStorage) Delete(ctx context.Context, key string) error {
	if !storage.ValidKey(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	_, err := s.client.Delete(ctx, key)
	return err
}

func (s *COSStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if !storage.ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	query := url.Values{"response-content-disposition": []string{
		fmt.Sprintf("attachment; filename=%q", filename),
	}}
	opt := &cos.PresignedURLOptions{Query: &query}
	u, err := s.client.GetPresignedURL2(ctx, http.MethodGet, key, expire, opt)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Size returns the object's content length via HEAD.
func (s *COSStorage) Size(ctx context.Context, key string) (int64, error) {
	if !storage.ValidKey(key) {
		return 0, fmt.Errorf("invalid key %q", key)
	}
	resp, err := s.client.Head(ctx, key, nil)
	if err != nil {
		return 0, err
	}
	if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusOK {
		return resp.ContentLength, nil
	}
	return 0, fmt.Errorf("head %s: unexpected status", key)
}
