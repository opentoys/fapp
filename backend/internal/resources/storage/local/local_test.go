package local

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

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

func TestLocalSaveOpenDelete(t *testing.T) {
	s, err := NewLocal(filepath.Join(t.TempDir(), "files"))
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
	url, err := s.DownloadURL(nil, key, "app.apk", 0)
	if err != nil {
		t.Fatal(err)
	}
	if url != "/api/v1/files/"+key {
		t.Fatalf("url = %q", url)
	}
	if err := s.Delete(nil, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(nil, key); err == nil {
		t.Fatal("expected error after delete")
	}
}
