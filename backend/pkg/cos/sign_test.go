package cos

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestPresignURLGolden locks the httpString layout (method\npath\nquery\nheaders\n)
// against a known COS string-to-sign hash. The hash must equal the value COS
// itself reported when it rejected the old malformed signature.
func TestPresignURLGolden(t *testing.T) {
	// Locks the httpString layout (method\npath\nquery\nheaders\n) against a
	// known COS string-to-sign hash. The hash must equal the value COS itself
	// reported when it rejected the old malformed signature.
	seg := "/7/0/1786846530268470000-tvbox.apk"
	http := "put\n" + seg + "\n\n\n"
	if got := fmt.Sprintf("%x", sha1.Sum([]byte(http))); got != "5d8723c495fee093b6365a78da70b2fa963448d4" {
		t.Fatalf("httpString sha1 = %s", got)
	}
}

func TestPresignURLShape(t *testing.T) {
	c := &client{
		secretID:  "AKIDEXAMPLE",
		secretKey: "SECRET",
		baseURL:   "https://test-1250000000.cos.ap-guangzhou.myqcloud.com",
	}
	u, err := c.presignURL(context.Background(), "put", "exampleobject", nil, nil, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "test-1250000000.cos.ap-guangzhou.myqcloud.com" {
		t.Fatalf("host = %q", u.Host)
	}
	if u.Path != "/exampleobject" {
		t.Fatalf("path = %q", u.Path)
	}
	q, _ := url.ParseQuery(u.RawQuery)
	for _, k := range []string{"q-sign-algorithm", "q-ak", "q-sign-time", "q-key-time", "q-signature"} {
		if q.Get(k) == "" {
			t.Fatalf("missing %s in %q", k, u.RawQuery)
		}
	}
	if q.Get("q-sign-algorithm") != "sha1" {
		t.Fatalf("algorithm = %q", q.Get("q-sign-algorithm"))
	}
	if q.Get("q-ak") != "AKIDEXAMPLE" {
		t.Fatalf("ak = %q", q.Get("q-ak"))
	}
	// Header list is empty by spec; url-param-list may be empty when no extra
	// params are signed.
	if _, ok := q["q-header-list"]; !ok {
		t.Fatalf("missing q-header-list in %q", u.RawQuery)
	}
}

func TestPresignURLAddsSessionToken(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	u, _ := c.presignURL(context.Background(), "put", "1/0/a.apk", nil, nil, time.Hour,
		&stsCredentials{secretID: "tmpAK", secretKey: "tmpSK", sessionToken: "v4/sessionfoo"})
	q, _ := url.ParseQuery(u.RawQuery)
	if got := q.Get("x-cos-security-token"); got != "v4/sessionfoo" {
		t.Fatalf("token = %q", got)
	}
	if got := q.Get("q-ak"); got != "tmpAK" {
		t.Fatalf("q-ak = %q (must be the temp id)", got)
	}
	// The token is a signed (non-q-*) param: it must appear in q-url-param-list.
	if !strings.Contains(q.Get("q-url-param-list"), "x-cos-security-token") {
		t.Fatalf("token not in param list: %q", q.Get("q-url-param-list"))
	}
}

func TestPresignURLNoTokenByDefault(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	u, _ := c.presignURL(context.Background(), "put", "1/0/a.apk", nil, nil, time.Hour, nil)
	q, _ := url.ParseQuery(u.RawQuery)
	if _, ok := q["x-cos-security-token"]; ok {
		t.Fatal("token present without session credentials")
	}
}

func TestPresignURLEncodesPathSegments(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	u, _ := c.presignURL(context.Background(), "get", "1/0/a b.png", nil, nil, time.Hour, nil)
	if u.Path != "/1/0/a%20b.png" {
		t.Fatalf("path = %q", u.Path)
	}
}

func TestPresignURLCoversExtraParams(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	extra := url.Values{"response-content-disposition": {"attachment; filename=\"a.png\""}}
	u, _ := c.presignURL(context.Background(), "get", "1/0/a.png", extra, nil, time.Hour, nil)
	q, _ := url.ParseQuery(u.RawQuery)
	if got := q.Get("response-content-disposition"); !strings.Contains(got, "a.png") {
		t.Fatalf("extra param = %q", got)
	}
	if got := q.Get("q-url-param-list"); !strings.Contains(got, "response-content-disposition") {
		t.Fatalf("param list = %q", got)
	}
}

func TestDownloadURLSignsParam(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	c.fetchSts = func(context.Context) (*stsCredentials, error) {
		return &stsCredentials{secretID: "AKID", secretKey: "SECRET", sessionToken: ""}, nil
	}
	u, err := c.DownloadURL(context.Background(), "1/0/a.png", "a.png", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "response-content-disposition") || !strings.Contains(u, "q-signature=") {
		t.Fatalf("download url = %q", u)
	}
}
func TestPresignURLSignsHeaders(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	hdr := http.Header{}
	hdr.Set("x-cos-forbid-overwrite", "true")
	u, err := c.presignURL(context.Background(), "put", "1/0/a.apk", nil, hdr, time.Hour,
		&stsCredentials{secretID: "tmpAK", secretKey: "tmpSK", sessionToken: ""})
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(u.RawQuery)
	// The header is covered by the signature: it appears in q-header-list.
	if got := q.Get("q-header-list"); !strings.Contains(got, "x-cos-forbid-overwrite") {
		t.Fatalf("q-header-list = %q", got)
	}
	// q-header-list starts with the mandatory host header.
	if got := q.Get("q-header-list"); !strings.HasPrefix(got, "host") {
		t.Fatalf("q-header-list = %q, want host first", got)
	}
}

func TestUploadURLSignsForbidOverwrite(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	c.fetchSts = func(context.Context) (*stsCredentials, error) {
		return &stsCredentials{secretID: "AKID", secretKey: "SECRET", sessionToken: ""}, nil
	}
	u, err := c.UploadURL(context.Background(), "1/0/a.apk", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(strings.SplitN(u, "?", 2)[1])
	if !strings.Contains(q.Get("q-header-list"), "x-cos-forbid-overwrite") {
		t.Fatalf("q-header-list = %q", q.Get("q-header-list"))
	}
}

func TestUploadURLSignsContentType(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	c.fetchSts = func(context.Context) (*stsCredentials, error) {
		return &stsCredentials{secretID: "AKID", secretKey: "SECRET", sessionToken: ""}, nil
	}
	u, err := c.UploadURL(context.Background(), "1/0/a.apk", "application/vnd.android.package-archive", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(strings.SplitN(u, "?", 2)[1])
	// content-type is lowercase and signed into the URL alongside the header.
	if !strings.Contains(q.Get("q-header-list"), "content-type") {
		t.Fatalf("q-header-list = %q", q.Get("q-header-list"))
	}
}
