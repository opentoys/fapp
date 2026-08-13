package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"disapp/internal/config"
	"disapp/internal/db"
	"disapp/internal/model"
	"disapp/internal/password"
	"disapp/internal/server"
	"disapp/internal/storage"
	"disapp/static"
)

func main() {
	path := os.Getenv("APP_CONFIG")
	if path == "" {
		path = "config.json"
	}
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Ensure data directory exists for sqlite db.
	if dir := filepath.Dir(cfg.Database.DSN); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("create db dir: %v", err)
		}
	}

	gdb, err := db.Open(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	// Auto-create / reset default admin on every startup.
	// Dev convenience: password is reset to the env var (or default) so the
	// platform is always reachable. Override APP_ADMIN_USER / APP_ADMIN_PASS
	// before launching to pick your own credentials.
	adminUser := os.Getenv("APP_ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("APP_ADMIN_PASS")
	if adminPass == "" {
		adminPass = "admin123"
	}
	hash, salt := password.Hash(adminPass)
	var existing model.User
	if err := gdb.Where("username = ?", adminUser).First(&existing).Error; err == nil {
		gdb.Model(&existing).Updates(map[string]any{"password_hash": hash, "salt": salt})
		log.Printf("reset password for admin %q", adminUser)
	} else {
		gdb.Create(&model.User{Username: adminUser, PasswordHash: hash, Salt: salt})
		log.Printf("created default admin %q", adminUser)
	}
	log.Printf("admin credentials: %s / %s", adminUser, adminPass)

	var st storage.Storage
	switch cfg.Storage.Backend {
	case "cos":
		cosSt, err := storage.NewCOSFromConfig(cfg.Storage.COS.SecretID, cfg.Storage.COS.SecretKey, cfg.Storage.COS.Bucket, cfg.Storage.COS.Region, cfg.Storage.COS.BaseURL)
		if err != nil {
			log.Fatalf("init cos: %v", err)
		}
		st = cosSt
	default:
		loc, err := storage.NewLocal(cfg.Storage.Local.Dir)
		if err != nil {
			log.Fatalf("init local storage: %v", err)
		}
		st = loc
	}

	srv := server.New(gdb, st, cfg)
	// The embed.FS is rooted at the static package dir; re-root to "dist" so
	// http.FileServerFS can find index.html at the FS root.
	distRoot, err := fs.Sub(static.Dist, "dist")
	if err != nil {
		log.Fatalf("sub dist: %v", err)
	}
	handler := srv.Routes(distRoot)

	log.Printf("app-dist listening on %s (storage: %s)", cfg.Server.Addr, cfg.Storage.Backend)
	if err := http.ListenAndServe(cfg.Server.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
