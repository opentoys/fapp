package cos

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// presignURL builds a COS presigned (query-signed) URL for the given object
// key. method is the lowercase HTTP verb. extra are additional query params
// that must be covered by the signature (e.g. response-content-disposition).
// creds carries the STS session (temp secretID/secretKey + token).
func (c *client) presignURL(ctx context.Context, method, key string, extra url.Values, expire time.Duration, creds *stsCredentials) (*url.URL, error) {
	if method == "" {
		method = "get"
	}
	path := objectPath(key)
	start := time.Now().Unix()
	if c.now != nil {
		start = c.now().Unix()
	}
	end := start + int64(expire.Seconds())
	signTime := fmt.Sprintf("%d;%d", start, end)

	query := url.Values{}
	for k, vv := range extra {
		for _, v := range vv {
			query.Add(k, v)
		}
	}
	// STS credentials require the session token as a signed query param.
	if creds != nil && creds.sessionToken != "" {
		query.Set("x-cos-security-token", creds.sessionToken)
	}
	// Params owned by the caller must be listed in q-url-param-list and
	// signed. The q-* params we add below are appended after signing.
	paramList := sortedParamNames(query)

	// FormatString = method\npath\n<query>\n<headers>\n. COS presigning always
	// signs the host header, so headers is "host=<bucket-host>". The SDK signs
	// the SHA1 hex of the formatString (with a trailing newline), not the
	// formatString itself.
	headerBlock := "host=" + c.host
	formatString := method + "\n" + path + "\n" + encodeQuery(query) + "\n" + headerBlock + "\n"
	stringToSign := "sha1\n" + signTime + "\n" + sha1Hex(formatString) + "\n"
	// Sign with the STS temp keys so COS accepts the token that matches them.
	credentialID, credentialKey := c.secretID, c.secretKey
	if creds != nil {
		credentialID, credentialKey = creds.secretID, creds.secretKey
	}
	signKey := hex.EncodeToString(hmacSHA1(credentialKey, signTime))
	signature := hex.EncodeToString(hmacSHA1(signKey, stringToSign))

	q := url.Values{}
	q.Set("q-sign-algorithm", "sha1")
	q.Set("q-ak", credentialID)
	q.Set("q-sign-time", signTime)
	q.Set("q-key-time", signTime)
	q.Set("q-header-list", "host")
	q.Set("q-url-param-list", paramList)
	q.Set("q-signature", signature)

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
// COS canonical form uses %20 for spaces, not the '+' of QueryEscape.
func encodeQuery(v url.Values) string {
	parts := make([]string, 0, len(v))
	for k, vv := range v {
		for _, val := range vv {
			parts = append(parts, cosQueryEscape(k)+"="+cosQueryEscape(val))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

// cosQueryEscape URI-encodes with space as %20 per the COS sign spec.
func cosQueryEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
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

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
