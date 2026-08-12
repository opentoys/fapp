package main

import (
	"log"
	"net/http"
	"os"

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

	gdb, err := db.Open(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	// Auto-create default admin from env vars if no users exist.
	if user, pass := os.Getenv("APP_ADMIN_USER"), os.Getenv("APP_ADMIN_PASS"); user != "" && pass != "" {
		var c int64
		gdb.Model(&model.User{}).Count(&c)
		if c == 0 {
			hash, salt := password.Hash(pass)
			gdb.Create(&model.User{Username: user, PasswordHash: hash, Salt: salt})
			log.Printf("created default admin %q", user)
		}
	}

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
	handler := srv.Routes(static.Dist)

	log.Printf("app-dist listening on %s (storage: %s)", cfg.Server.Addr, cfg.Storage.Backend)
	if err := http.ListenAndServe(cfg.Server.Addr, handler); err != nil {
		log.Fatal(err)
	}
}
