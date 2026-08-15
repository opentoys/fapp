package config

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Storage  StorageConfig  `json:"storage"`
	JWT      JWTConfig      `json:"jwt"`
	Admin    AdminConfig    `json:"admin"`
}

type ServerConfig struct {
	Addr string `json:"addr"`
}

type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

type StorageConfig struct {
	Backend string      `json:"backend"`
	Local   LocalConfig `json:"local"`
	COS     COSConfig   `json:"cos"`
}

type LocalConfig struct {
	Dir string `json:"dir"`
}

type COSConfig struct {
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"` // 如 app-dist-1250000000
	Region    string `json:"region"` // 如 ap-guangzhou
	BaseURL   string `json:"base_url"`
}

type JWTConfig struct {
	Secret string `json:"secret"`
	Expire string `json:"expire"` // Go duration 字符串，如 "24h"
}

type AdminConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Default() Config {
	return Config{
		Server:   ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{DSN: "./data/app.db"},
		Storage:  StorageConfig{Backend: "local", Local: LocalConfig{Dir: "./data/files"}},
		JWT:      JWTConfig{Secret: "change-me", Expire: "24h"},
		// Admin: 留空时启动不会自动创建任何管理员账号
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, err
	}
	c := Default()
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) JWTExpire() time.Duration {
	d, err := time.ParseDuration(c.JWT.Expire)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
