package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// objectClient is a narrow interface for cos object operations, for test mocking.
type objectClient interface {
	Put(ctx context.Context, key string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	Get(ctx context.Context, key string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error)
	Delete(ctx context.Context, key string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error)
	GetPresignedURL(ctx context.Context, method, key, akID, skID string, exp time.Duration, opt interface{}, signHost ...bool) (*url.URL, error)
}

type COSStorage struct {
	client    objectClient
	secretID  string
	secretKey string
	baseURL   string
}

func NewCOS(client objectClient, secretID, secretKey, bucket, baseURL string) *COSStorage {
	return &COSStorage{client: client, secretID: secretID, secretKey: secretKey, baseURL: baseURL}
}

// NewCOSFromConfig creates COSStorage from config parameters.
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
	transport := &cos.AuthorizationTransport{
		SecretID:  secretID,
		SecretKey: secretKey,
		Transport: http.DefaultTransport,
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{Transport: transport})
	return NewCOS(client.Object, secretID, secretKey, bucket, base), nil
}

func (s *COSStorage) Save(ctx context.Context, key string, r io.Reader) (int64, error) {
	if !ValidKey(key) {
		return 0, fmt.Errorf("invalid key %q", key)
	}
	// Read all content to get size, then re-upload from buffer
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	size := int64(len(data))
	_, err = s.client.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentLength: size}})
	return size, err
}

func (s *COSStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !ValidKey(key) {
		return nil, fmt.Errorf("invalid key %q", key)
	}
	resp, err := s.client.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *COSStorage) Delete(ctx context.Context, key string) error {
	if !ValidKey(key) {
		return fmt.Errorf("invalid key %q", key)
	}
	_, err := s.client.Delete(ctx, key)
	return err
}

func (s *COSStorage) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	if !ValidKey(key) {
		return "", fmt.Errorf("invalid key %q", key)
	}
	query := url.Values{"response-content-disposition": []string{
		fmt.Sprintf("attachment; filename=%q", filename),
	}}
	opt := &cos.PresignedURLOptions{Query: &query}
	u, err := s.client.GetPresignedURL(ctx, http.MethodGet, key, s.secretID, s.secretKey, expire, opt)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}