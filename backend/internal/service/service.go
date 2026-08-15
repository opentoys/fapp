package service

import (
	"disapp/internal/resources/config"
	"disapp/internal/resources/storage"

	"gorm.io/gorm"
)

// Service holds the business layer: database, storage, configuration.
// Controllers call its methods and map errors to HTTP responses.
type Service struct {
	DB      *gorm.DB
	Storage storage.Storage
	Config  config.Config
}

func New(gdb *gorm.DB, st storage.Storage, cfg config.Config) *Service {
	return &Service{DB: gdb, Storage: st, Config: cfg}
}