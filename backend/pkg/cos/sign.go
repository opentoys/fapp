package cos

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// presignURL builds a COS presigned (query-signed) URL for the given object
// key. method is the lowercase HTTP verb. extra are additional query params
// that must be covered by the signature (e.g. response-content-disposition).
func (c *client) presignURL(ctx context.Context, method, key string, extra url.Values, expire time.Duration) (*url.URL, error) {
	if method == "" {
		method = "get"
	}
	path := objectPath(key)
	start := time.Now().Unix()
	end := start + int64(expire.Seconds())
	signTime := fmt.Sprintf("%d;%d", start, end)

	query := url.Values{}
	for k, vv := range extra {
		for _, v := range vv {
			query.Add(k, v)
		}
	}
	// Params owned by the caller must be listed in q-url-param-list and
	// signed. The q-* params we add below are appended after signing.
	paramList := sortedParamNames(query)

	httpString := method + "\n" + path + "\n" + encodeQuery(query) + "\n\n"
	stringToSign := "sha1\n" + signTime + "\n" + httpString
	signKey := hmacSHA1(c.secretKey, signTime)
	signature := hmacSHA1(string(signKey), stringToSign)

	q := url.Values{}
	q.Set("q-sign-algorithm", "sha1")
	q.Set("q-ak", c.secretID)
	q.Set("q-sign-time", signTime)
	q.Set("q-key-time", signTime)
	q.Set("q-header-list", "")
	q.Set("q-url-param-list", paramList)
	q.Set("q-signature", base64.StdEncoding.EncodeToString(signature))

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	// Merge the signed extra params and the q-* auth params into the final URL.
	merged := url.Values{}
	for k, vv := range query {
		for _, v := range vv {
			merged.Add(k, v)
		}
	}
	for k, vv := range q {
		for _, v := range vv {
			merged.Add(k, v)
		}
	}
	return &url.URL{
		Scheme:   base.Scheme,
		Host:     base.Host,
		Path:     path,
		RawQuery: merged.Encode(),
	}, nil
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
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(val))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

// sortedParamNames returns the comma-joined, sorted, URL-escaped key list.
func sortedParamNames(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	esc := make([]string, 0, len(keys))
	for _, k := range keys {
		esc = append(esc, url.PathEscape(k))
	}
	return strings.Join(esc, ",")
}

func hmacSHA1(key, data string) []byte {
	m := hmac.New(sha1.New, []byte(key))
	m.Write([]byte(data))
	return m.Sum(nil)
}