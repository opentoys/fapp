package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"disapp/internal/controller"
	"disapp/internal/resources/config"
	"disapp/internal/router"
	"disapp/internal/resources/storage"
	"disapp/internal/resources/storage/cos"
	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/db"
	"disapp/internal/service"
	pkgcos "disapp/pkg/cos"
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
		obj, err := pkgcos.NewFromConfig(cfg.Storage.COS.SecretID, cfg.Storage.COS.SecretKey, cfg.Storage.COS.Bucket, cfg.Storage.COS.Region, cfg.Storage.COS.BaseURL)
		if err != nil {
			log.Fatalf("init cos: %v", err)
		}
		st = cos.NewCOS(obj)
	default:
		loc, err := local.NewLocal(cfg.Storage.Local.Dir, cfg.JWT.Secret)
		if err != nil {
			log.Fatalf("init local storage: %v", err)
		}
		st = loc
	}

	svc := service.New(gdb, st, cfg)
	ctrl := controller.New(svc)
	// static.Dist is the embedded frontend when built with `-tags dist`,
	// otherwise nil (no static serving).
	handler := router.Routes(ctrl, static.Dist)

	log.Printf("app-dist listening on %s (storage: %s, dist: %v)", cfg.Server.Addr, cfg.Storage.Backend, static.Dist != nil)
	if err := http.ListenAndServe(cfg.Server.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
