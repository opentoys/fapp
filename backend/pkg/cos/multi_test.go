package cos

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPresignURLMultiHeaderJoin(t *testing.T) {
	c := &client{secretID: "AKID", secretKey: "SECRET", baseURL: "https://b.cos.region.myqcloud.com"}
	// Space in key forces PathEscape; exercises the exact path used in signing.
	hdr := http.Header{}
	hdr.Set("x-cos-forbid-overwrite", "true")
	u, err := c.presignURL(context.Background(), "put", "1/0/a b.apk", nil, hdr, time.Hour,
		&stsCredentials{secretID: "tmpAK", secretKey: "tmpSK", sessionToken: "TOK"})
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(u.RawQuery)
	if got := q.Get("q-header-list"); !strings.Contains(got, "x-cos-forbid-overwrite") {
		t.Fatalf("q-header-list = %q", got)
	}
	// Reconstruct the canonical HttpString as COS does: headers joined with '&'.
	block := "host=" + c.host + "&x-cos-forbid-overwrite=true"
	httpString := "put\n/1/0/a%20b.apk\nx-cos-security-token=TOK\n" + block + "\n"
	want := fmt.Sprintf("%x", sha1.Sum([]byte(httpString)))
	// The URL's q-signature must be computed over that exact sha1. We can't
	// read the signer's internal hash, but we can recompute the HMAC chain.
	signTime := q.Get("q-sign-time")
	stringToSign := "sha1\n" + signTime + "\n" + want + "\n"
	signKey := hex.EncodeToString(hmacSHA1("tmpSK", signTime))
	signature := hex.EncodeToString(hmacSHA1(signKey, stringToSign))
	if q.Get("q-signature") != signature {
		t.Fatalf("signature mismatch: got %s want %s (join=_; if this fails, separator '&' not used)", q.Get("q-signature"), signature)
	}
}
