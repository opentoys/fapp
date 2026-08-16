// Package cos is the minimal COS surface the app consumes. It signs request
// URLs with the permanent Tencent Cloud keys (SHA1, following the COS presign
// spec) and depends only on net/http — no cos-go-sdk-v5.
package cos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Store is the four operations the platform needs from object storage:
// presigned upload/download URLs and delete.
type Store interface {
	UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error)
}

type client struct {
	secretID  string
	secretKey string
	baseURL   string
	host      string
	hc        *http.Client
}

// NewFromConfig builds the COS client. Keys sign request URLs directly (no
// STS exchange). baseURL is optional; when empty it derives from bucket+region.
func NewFromConfig(secretID, secretKey, bucket, region, baseURL string) (Store, error) {
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("cos SecretID and SecretKey are required")
	}
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
	return &client{
		secretID:  secretID,
		secretKey: secretKey,
		baseURL:   base,
		host:      u.Host,
		hc:        &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// UploadURL presigns a PUT for the client to push bytes straight to storage.
// Non-q-* query params are signed into the URL so extra params are covered.
func (c *client) UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error) {
	u, err := c.presignURL(ctx, "put", key, nil, expire)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Delete removes the object with a DELETE issued by the server, signed into
// the URL and executed over HTTPS.
func (c *client) Delete(ctx context.Context, key string) error {
	u, err := c.presignURL(ctx, "delete", key, nil, 5*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cos delete %s: %s", key, resp.Status)
	}
	return nil
}

// DownloadURL presigns a GET that streams the object as an attachment. The
// response-content-disposition param is signed into the URL.
func (c *client) DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error) {
	params := url.Values{}
	params.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%q", filename))
	u, err := c.presignURL(ctx, "get", key, params, expire)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}