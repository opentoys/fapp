package cos

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPresignURLShape(t *testing.T) {
	c := &client{
		secretID:  "AKIDEXAMPLE",
		secretKey: "SECRET",
		baseURL:   "https://test-1250000000.cos.ap-guangzhou.myqcloud.com",
	}
	u, err := c.presignURL(context.Background(), "put", "exampleobject", nil, time.Hour)
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

func TestPresignURLEncodesPathSegments(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	u, _ := c.presignURL(context.Background(), "get", "1/0/a b.png", nil, time.Hour)
	if u.Path != "/1/0/a%20b.png" {
		t.Fatalf("path = %q", u.Path)
	}
}

func TestPresignURLCoversExtraParams(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	extra := url.Values{"response-content-disposition": {"attachment; filename=\"a.png\""}}
	u, _ := c.presignURL(context.Background(), "get", "1/0/a.png", extra, time.Hour)
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
	u, err := c.DownloadURL(context.Background(), "1/0/a.png", "a.png", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "response-content-disposition") || !strings.Contains(u, "q-signature=") {
		t.Fatalf("download url = %q", u)
	}
}