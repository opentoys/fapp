// Package oci is the minimal Oracle Object Storage surface the app consumes.
// It signs request URLs with AWS SigV4 against the S3-compatible endpoint
// (`{bucket}.{namespace}.compat.objectstorage.{region}.oraclecloud.com`)
// and depends only on net/http — no official SDK.
package oci

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Store is the three operations the platform needs from object storage:
// presigned upload/download URLs and delete.
type Store interface {
	UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error)
}

type client struct {
	accessKey string
	secretKey string
	region    string
	host      string
	hc        *http.Client
}

// NewFromConfig builds the OCI Object Storage client. Keys sign request URLs
// directly (AWS SigV4 against the S3-compatible endpoint). baseURL is
// optional; when empty it derives from bucket+namespace+region.
func NewFromConfig(accessKey, secretKey, namespace, bucket, region, baseURL string) (Store, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("oci access and secret keys are required")
	}
	if namespace == "" || bucket == "" || region == "" {
		return nil, fmt.Errorf("oci namespace, bucket and region are required")
	}
	base := baseURL
	if base == "" {
		base = "https://" + bucket + "." + namespace + ".compat.objectstorage." + region + ".oraclecloud.com"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &client{
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		host:      u.Host,
		hc:        &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// UploadURL presigns a PUT for the client to push bytes straight to storage.
func (c *client) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	return c.presignURL(ctx, "put", key, nil, expire), nil
}

// Delete removes the object with a DELETE issued by the server, signed into
// the URL and executed over HTTPS.
func (c *client) Delete(ctx context.Context, key string) error {
	u := c.presignURL(ctx, "delete", key, nil, 5*time.Minute)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("oci delete %s: %s", key, resp.Status)
	}
	return nil
}

// DownloadURL presigns a GET that streams the object as an attachment. The
// response-content-disposition param is signed into the URL.
func (c *client) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	params := url.Values{}
	params.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return c.presignURL(ctx, "get", key, params, expire), nil
}
