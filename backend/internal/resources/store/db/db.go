package db

import (
	"time"

	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"disapp/internal/resources/store/model"
)

// Open opens sqlite database and auto-migrates all tables.
func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		// Store auto timestamps in the server's default timezone (time.Local).
		NowFunc: func() time.Time { return time.Now().In(time.Local) },
	})
	if err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(
		&model.User{}, &model.App{}, &model.Version{}, &model.DownloadLog{}, &model.AppMember{}, &model.ApiKey{},
		&model.NotificationBot{}, &model.NotificationLog{},
	); err != nil {
		return nil, err
	}
	return gdb, nil
}
