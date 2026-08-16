package local

import (
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"disapp/internal/resources/storage"
)

func TestValidKey(t *testing.T) {
	valid := []string{"1/2/app.apk", "12/345/foo bar.zip"}
	invalid := []string{"", "../x", "a/b/c", "1/2/../../etc", "1//x", "/1/2/x", "1/2/"}
	for _, k := range valid {
		if !storage.ValidKey(k) {
			t.Errorf("expected valid: %q", k)
		}
	}
	for _, k := range invalid {
		if storage.ValidKey(k) {
			t.Errorf("expected invalid: %q", k)
		}
	}
}

func TestLocalSaveOpenDeleteSize(t *testing.T) {
	s, err := NewLocal(filepath.Join(t.TempDir(), "files"), "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	key := "1/2/app.apk"
	size, err := s.Save(nil, key, strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if size != 11 {
		t.Fatalf("size = %d", size)
	}
	rc, err := s.Open(nil, key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello world" {
		t.Fatalf("data = %q", data)
	}
	got, err := s.Size(nil, key)
	if err != nil || got != 11 {
		t.Fatalf("size = %d, err = %v", got, err)
	}
	if err := s.Delete(nil, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(nil, key); err == nil {
		t.Fatal("expected error after delete")
	}
}

// TestUploadURL checks the signed upload endpoint shape and that the sign
// round-trips through the ticket validator.
func TestUploadURL(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "jwt-secret")
	u, err := s.UploadURL(nil, "1/2/app.apk", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	q, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	if q.Path != "/api/v1/files/upload" {
		t.Fatalf("path = %q", q.Path)
	}
	ttl, err := strconv.ParseInt(q.Query().Get("ttl"), 10, 64)
	if err != nil {
		t.Fatalf("ttl missing: %v", err)
	}
	if !ValidUploadTicket("jwt-secret", ttl, q.Query().Get("sign")) {
		t.Fatal("signature did not validate")
	}
	if ValidUploadTicket("jwt-secret", ttl+1, q.Query().Get("sign")) {
		t.Fatal("signature validated with wrong ttl")
	}
	if ValidUploadTicket("other-secret", ttl, q.Query().Get("sign")) {
		t.Fatal("signature validated with wrong secret")
	}
}

// TestDownloadURL checks the signed preview ticket round-trips.
func TestDownloadURL(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), "jwt-secret")
	u, err := s.DownloadURL(nil, "1/2/app.apk", "app.apk", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	q, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	if q.Path != "/api/v1/files/preview" {
		t.Fatalf("path = %q", q.Path)
	}
	ttl, err := strconv.ParseInt(q.Query().Get("ttl"), 10, 64)
	if err != nil {
		t.Fatalf("ttl missing: %v", err)
	}
	if !ValidPreviewTicket("jwt-secret", q.Query().Get("key"), ttl, q.Query().Get("sign")) {
		t.Fatal("preview signature did not validate")
	}
	if ValidPreviewTicket("jwt-secret", "1/9/other.apk", ttl, q.Query().Get("sign")) {
		t.Fatal("preview signature validated with wrong key")
	}
}

// TestCheckSign ensures tampered signatures are rejected (constant time path).
func TestCheckSign(t *testing.T) {
	secret := "s"
	ttl := int64(12345)
	sign := signUpload(secret, ttl)
	if !CheckSign(secret, sign, []byte(strconv.FormatInt(ttl, 10))) {
		t.Fatal("expected match")
	}
	if CheckSign(secret, sign, []byte("12346")) {
		t.Fatal("expected mismatch on message")
	}
	if CheckSign("other", sign, []byte(strconv.FormatInt(ttl, 10))) {
		t.Fatal("expected mismatch on secret")
	}
	if CheckSign(secret, strings.ToUpper(sign), []byte(strconv.FormatInt(ttl, 10))) {
		t.Fatal("expected mismatch on garbage sign")
	}
}