package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{
		"server": {"addr": ":9090"},
		"database": {"dsn": "./data/t.db"},
		"storage": {"backend": "cos", "cos": {"bucket": "b1", "region": "ap-guangzhou"}},
		"jwt": {"secret": "s3", "expire": "1h"}
	}`), 0o644)

	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Addr != ":9090" {
		t.Errorf("addr = %q", c.Server.Addr)
	}
	if c.Storage.Backend != "cos" || c.Storage.COS.Bucket != "b1" {
		t.Errorf("storage = %+v", c.Storage)
	}
	if c.JWTExpire() != time.Hour {
		t.Errorf("expire = %v", c.JWTExpire())
	}
}

func TestLoadDefaultsForEmptyFields(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{}`), 0o644)

	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Addr != ":8080" {
		t.Errorf("default addr = %q", c.Server.Addr)
	}
	if c.Database.DSN != "./data/app.db" {
		t.Errorf("default dsn = %q", c.Database.DSN)
	}
	if c.Storage.Backend != "local" {
		t.Errorf("default backend = %q", c.Storage.Backend)
	}
	if c.JWTExpire() != 24*time.Hour {
		t.Errorf("default expire = %v", c.JWTExpire())
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Addr != ":8080" {
		t.Errorf("addr = %q", c.Server.Addr)
	}
}
