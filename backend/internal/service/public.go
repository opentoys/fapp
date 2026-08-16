package service

import (
	"context"
	"strconv"
	"time"


	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"

	"gorm.io/gorm"
)

// resolveAppMedia replaces an app's bare icon/screenshot keys with signed
// download URLs for public display. Non-destructive: mutates the passed copy.
func (s *Service) resolveAppMedia(a *model.App) {
	const displayExpire = time.Hour
	if a.Icon != "" {
		if rel, err := s.Storage.DownloadURL(context.Background(), a.Icon, "icon.png", displayExpire); err == nil {
			a.Icon = rel
		}
	}
	for i, k := range a.Screenshots {
		if rel, err := s.Storage.DownloadURL(context.Background(), k, "shot.png", displayExpire); err == nil {
			a.Screenshots[i] = rel
		}
	}
}

// PublicAppDetail resolves an app by name (share link) or numeric id. An
// unpublished app behaves as if it doesn't exist. Only the current version is
// exposed.
func (s *Service) PublicAppDetail(ctx context.Context, key string) (model.App, []model.Version, error) {
	var app model.App
	err := s.DB.Where("name = ?", key).First(&app).Error
	if err != nil {
		if n, perr := strconv.ParseInt(key, 10, 64); perr == nil {
			err = s.DB.First(&app, n).Error
		}
	}
	if err != nil {
		return app, nil, &Error{StatusNotFound, "应用不存在"}
	}
	if hiddenApp(&app) {
		return app, nil, &Error{StatusNotFound, "应用不存在"}
	}
	versions := make([]model.Version, 0, 1)
	if app.CurrentVersionID > 0 {
		var v model.Version
		if err := s.DB.First(&v, app.CurrentVersionID).Error; err == nil {
			versions = append(versions, v)
		}
	}
	s.resolveAppMedia(&app)
	return app, versions, nil
}

// hiddenApp reports whether an app is absent from the public view: taken down
// or past its expiry. Both behave as "应用不存在", matching the下架 logic.
func hiddenApp(app *model.App) bool {
	return !app.Published || (app.ExpiresAt != nil && time.Now().After(*app.ExpiresAt))
}

// checkAccess enforces that a version is publicly downloadable: its app must
// be visible (published, not expired) and the version must be current. A
// password-protected app additionally requires the correct password.
func (s *Service) checkAccess(v *model.Version, password string) error {
	var app model.App
	if err := s.DB.First(&app, v.AppID).Error; err != nil {
		return &Error{StatusNotFound, "应用不存在"}
	}
	if hiddenApp(&app) {
		return &Error{StatusNotFound, "应用不存在"}
	}
	if app.CurrentVersionID != v.ID {
		return &Error{StatusForbidden, "该版本不可下载"}
	}
	if app.PasswordHash != "" {
		if !pwd.Verify(password, app.PasswordHash, app.Salt) {
			return &Error{StatusUnauthorized, "密码错误"}
		}
	}
	return nil
}

// VerifyAccess checks the app-level permission (password mode submits password).
func (s *Service) VerifyAccess(ctx context.Context, versionID int64, password string) error {
	var v model.Version
	if err := s.DB.First(&v, versionID).Error; err != nil {
		return &Error{StatusNotFound, "版本不存在"}
	}
	return s.checkAccess(&v, password)
}

// ResolveDownload loads a version, checks access, and returns the relative
// storage download URL. The controller makes it absolute.
func (s *Service) ResolveDownload(ctx context.Context, versionID int64, password string) (string, *model.Version, error) {
	var v model.Version
	if err := s.DB.First(&v, versionID).Error; err != nil {
		return "", nil, &Error{StatusNotFound, "版本不存在"}
	}
	if err := s.checkAccess(&v, password); err != nil {
		return "", nil, err
	}
	rel, err := s.Storage.DownloadURL(ctx, v.StorageKey, v.FileName, 15*time.Minute)
	if err != nil {
		return "", nil, &Error{StatusInternal, "生成下载链接失败"}
	}
	return rel, &v, nil
}

// RecordInstall increments the install counter.
func (s *Service) RecordInstall(ctx context.Context, versionID int64) {
	s.DB.Model(&model.Version{}).Where("id = ?", versionID).
		UpdateColumn("install_count", gorm.Expr("install_count + 1"))
}

// RecordDownload increments the download counter and logs the row.
func (s *Service) RecordDownload(ctx context.Context, versionID int64, ip, userAgent string) {
	s.DB.Model(&model.Version{}).Where("id = ?", versionID).
		UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	s.DB.Create(&model.DownloadLog{VersionID: versionID, IP: ip, UserAgent: userAgent})
}