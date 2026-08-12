package db

import (
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"disapp/internal/model"
)

// Open opens sqlite database and auto-migrates all tables.
func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(
		&model.User{}, &model.App{}, &model.Channel{}, &model.Version{}, &model.DownloadLog{},
	); err != nil {
		return nil, err
	}
	return gdb, nil
}
