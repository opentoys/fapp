package oci

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	aws4HmacSHA256 = "AWS4-HMAC-SHA256"
	service        = "s3"
	terminator     = "aws4_request"
)

// presignURL builds an AWS SigV4 query-presigned URL for the given object key
// against the S3-compatible endpoint. method is the lowercase HTTP verb. extra
// are additional query params that must be covered by the signature (e.g.
// response-content-disposition).
func (c *client) presignURL(ctx context.Context, method, key string, extra url.Values, expire time.Duration) string {
	if method == "" {
		method = "get"
	}
	path := objectPath(key)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	// Query params: sortable keys (X-Amz-*) required for presigned URLs, plus
	// the caller-owned params that must be covered by the signature.
	query := url.Values{}
	query.Set("X-Amz-Algorithm", aws4HmacSHA256)
	query.Set("X-Amz-Credential", c.accessKey+"/"+dateStamp+"/"+c.region+"/"+service+"/"+terminator)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", int64(expire.Seconds())))
	query.Set("X-Amz-SignedHeaders", "host")
	for k, vv := range extra {
		for _, v := range vv {
			query.Add(k, v)
		}
	}

	canonicalQuery := encodeQuery(query)
	canonicalHeaders := "host:" + c.host + "\n"
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"
	canonicalRequest := strings.Join([]string{
		strings.ToUpper(method),
		path,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := dateStamp + "/" + c.region + "/" + service + "/" + terminator
	credential := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	keyVal := kdf(c.secretKey, dateStamp, c.region, service, terminator)
	signature := hex.EncodeToString(hmacSHA256(keyVal, credential))

	query.Set("X-Amz-Signature", signature)
	return (&url.URL{
		Scheme:   "https",
		Host:     c.host,
		Path:     path,
		RawQuery: query.Encode(),
	}).String()
}

// objectPath URL-encodes each key segment, preserving "/".
func objectPath(key string) string {
	segments := strings.Split(key, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return "/" + strings.Join(segments, "/")
}

// encodeQuery renders key=value pairs sorted by key, no trailing newline.
func encodeQuery(v url.Values) string {
	parts := make([]string, 0, len(v))
	for k, vv := range v {
		for _, val := range vv {
			parts = append(parts, k+"="+val)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

// kdf derives the SigV4 signing key.
func kdf(secret, date, region, service, terminator string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, terminator)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}
