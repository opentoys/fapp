package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"disapp/internal/resources/config"
	"disapp/internal/resources/storage"
	"disapp/internal/resources/storage/cos"
	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/db"
	"disapp/internal/server"
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

	// Super-admin lives in config.json only; never written to the users
	// table. Dev convention: if the DB ever drifts out of sync with the
	// model, blow it away with `rm -rf data/` rather than carrying
	// migration code.
	if cfg.Admin.Username != "" {
		log.Printf("super-admin: %s (auth handled by config, uid = -1)", cfg.Admin.Username)
	} else {
		log.Printf("no super-admin configured; admin endpoints will be unreachable")
	}

	var st storage.Storage
	switch cfg.Storage.Backend {
	case "cos":
		cosSt, err := cos.NewCOSFromConfig(cfg.Storage.COS.SecretID, cfg.Storage.COS.SecretKey, cfg.Storage.COS.Bucket, cfg.Storage.COS.Region, cfg.Storage.COS.BaseURL)
		if err != nil {
			log.Fatalf("init cos: %v", err)
		}
		st = cosSt
	default:
		loc, err := local.NewLocal(cfg.Storage.Local.Dir)
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
