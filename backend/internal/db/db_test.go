package db

import (
	"path/filepath"
	"testing"
)

func TestOpenAutoMigrates(t *testing.T) {
	gdb, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if !gdb.Migrator().HasTable("users") {
		t.Fatal("users table missing")
	}
	if !gdb.Migrator().HasTable("apps") {
		t.Fatal("apps table missing")
	}
	if !gdb.Migrator().HasTable("channels") {
		t.Fatal("channels table missing")
	}
	if !gdb.Migrator().HasTable("versions") {
		t.Fatal("versions table missing")
	}
	if !gdb.Migrator().HasTable("download_logs") {
		t.Fatal("download_logs table missing")
	}
}
