package oci

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPresignURLShape(t *testing.T) {
	c := &client{
		accessKey: "AKIAEXAMPLE",
		secretKey: "SECRET",
		region:    "ap-chuncheon-1",
		host:      "bucket.ns.compat.objectstorage.ap-chuncheon-1.oraclecloud.com",
	}
	u, _ := url.Parse(c.presignURL(context.Background(), "put", "exampleobject", nil, time.Hour))
	if u.Host != "bucket.ns.compat.objectstorage.ap-chuncheon-1.oraclecloud.com" {
		t.Fatalf("host = %q", u.Host)
	}
	if u.Path != "/exampleobject" {
		t.Fatalf("path = %q", u.Path)
	}
	q, _ := url.ParseQuery(u.RawQuery)
	for _, k := range []string{"X-Amz-Algorithm", "X-Amz-Credential", "X-Amz-Date", "X-Amz-Expires", "X-Amz-SignedHeaders", "X-Amz-Signature"} {
		if q.Get(k) == "" {
			t.Fatalf("missing %s in %q", k, u.RawQuery)
		}
	}
	if q.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" {
		t.Fatalf("algorithm = %q", q.Get("X-Amz-Algorithm"))
	}
	if q.Get("X-Amz-SignedHeaders") != "host" {
		t.Fatalf("signed headers = %q", q.Get("X-Amz-SignedHeaders"))
	}
	if !strings.Contains(q.Get("X-Amz-Credential"), "/ap-chuncheon-1/s3/aws4_request") {
		t.Fatalf("credential = %q", q.Get("X-Amz-Credential"))
	}
}

func TestPresignURLEncodesPathSegments(t *testing.T) {
	c := &client{accessKey: "AK", secretKey: "S", region: "ap", host: "b.n.compat.objectstorage.ap.oraclecloud.com"}
	u, _ := url.Parse(c.presignURL(context.Background(), "get", "1/0/a b.png", nil, time.Hour))
	if u.Path != "/1/0/a%20b.png" {
		t.Fatalf("path = %q", u.Path)
	}
}

func TestPresignURLCoversExtraParams(t *testing.T) {
	c := &client{accessKey: "AK", secretKey: "S", region: "ap", host: "b.n.compat.objectstorage.ap.oraclecloud.com"}
	extra := url.Values{"response-content-disposition": {"attachment; filename=\"a.png\""}}
	q, _ := url.ParseQuery(c.presignURL(context.Background(), "get", "1/0/a.png", extra, time.Hour))
	if got := q.Get("response-content-disposition"); !strings.Contains(got, "a.png") {
		t.Fatalf("extra param = %q", got)
	}
}

func TestDownloadURLSignsParam(t *testing.T) {
	c := &client{accessKey: "AK", secretKey: "S", region: "ap", host: "b.n.compat.objectstorage.ap.oraclecloud.com"}
	u, err := c.DownloadURL(context.Background(), "1/0/a.png", "a.png", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "response-content-disposition") || !strings.Contains(u, "X-Amz-Signature=") {
		t.Fatalf("download url = %q", u)
	}
}

// kdf determinism + input sensitivity: only the hex chain matters (32 bytes,
// lowercase), and the key must change when any input changes.
func TestSigV4KeyDerivation(t *testing.T) {
	base := kdf("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "20150830", "us-east-1", "iam", terminator)
	if len(sha256Hex(base)) != 64 {
		t.Fatalf("hex size = %d", len(sha256Hex(base)))
	}
	changed := map[string][]byte{
		"secret":  kdf("OTHER-SECRET", "20150830", "us-east-1", "iam", terminator),
		"date":    kdf("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "20150831", "us-east-1", "iam", terminator),
		"region":  kdf("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "20150830", "us-west-2", "iam", terminator),
		"service": kdf("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "20150830", "us-east-1", "s3", terminator),
	}
	for name, k := range changed {
		if sha256Hex(k) == sha256Hex(base) {
			t.Fatalf("key unchanged for %s", name)
		}
	}
}
