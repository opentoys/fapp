package service

import (
	"context"
	"encoding/hex"
	"io"
	"log"
	"strings"
	"time"

	"disapp/internal/resources/storage"
	"disapp/internal/resources/store/model"
)

// SaveVersionInput groups the multipart-derived fields for a new version.
type SaveVersionInput struct {
	VersionCode int
	VersionName string
	ReleaseType string
	Arch        string
	PackageName string
	AppName     string
	Changelog   string
	FileName    string
	File        io.Reader
	Icon        io.Reader
}

// SaveVersion persists a new version for an app. It locks the app's appid on
// the first upload that carries one, creates a draft, writes the file to
// storage (computing sha256+size), and rolls back on write failure. Best-effort
// stores the parsed icon.
func (s *Service) SaveVersion(ctx context.Context, appID int64, in SaveVersionInput) (*model.Version, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	if in.VersionName == "" {
		return nil, &Error{StatusBadRequest, "version_name 必填"}
	}
	pkg := strings.TrimSpace(in.PackageName)
	if app.PackageName == nil || *app.PackageName == "" {
		if pkg != "" {
			var n int64
			s.DB.Model(&model.App{}).
				Where("platform = ? AND package_name = ? AND id != ?", app.Platform, pkg, app.ID).
				Count(&n)
			if n > 0 {
				return nil, &Error{StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
			}
			app.PackageName = &pkg
			if err := s.DB.Save(app).Error; err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					return nil, &Error{StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
				}
				return nil, &Error{StatusInternal, "保存失败"}
			}
		}
	} else if pkg != *app.PackageName {
		return nil, &Error{StatusBadRequest, "安装包 appid 与应用不一致"}
	}

	v := model.Version{
		AppID:          app.ID,
		ReleaseType:    in.ReleaseType,
		Platform:       app.Platform,
		Arch:           in.Arch,
		VersionName:    in.VersionName,
		VersionCode:    in.VersionCode,
		PackageName:    pkg,
		AppName:        in.AppName,
		FileName:       in.FileName,
		FileType:       model.FileType(in.FileName),
		Changelog:      in.Changelog,
		StorageBackend: storageBackendName(s),
	}
	if err := s.DB.Create(&v).Error; err != nil {
		return nil, &Error{StatusInternal, "创建版本失败"}
	}

	key := storage.Key(app.ID, v.ID, in.FileName)
	hr := newHashReader(in.File)
	if _, err := s.Storage.Save(ctx, key, hr); err != nil {
		s.DB.Delete(&v)
		return nil, &Error{StatusInternal, "存储写入失败"}
	}
	s.DB.Model(&v).Updates(map[string]any{
		"storage_key": key,
		"file_size":   hr.n,
		"sha256":      hex.EncodeToString(hr.h.Sum(nil)),
	})

	if in.Icon != nil {
		iconKey := storage.Key(app.ID, v.ID, "icon.png")
		if _, err := s.Storage.Save(ctx, iconKey, in.Icon); err != nil {
			log.Printf("store icon failed: %v", err)
		} else {
			v.IconURL = "/api/v1/files/" + iconKey
			s.DB.Model(&v).Update("icon_url", v.IconURL)
		}
	}

	return &v, nil
}

// DeleteVersion deletes a version, optionally deleting the storage file. If the
// deleted version was current, the app's current pointer is reset.
func (s *Service) DeleteVersion(ctx context.Context, id int64, deleteFile bool) error {
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		return &Error{StatusNotFound, "版本不存在"}
	}
	if deleteFile && v.StorageKey != "" {
		s.Storage.Delete(ctx, v.StorageKey)
	}
	if err := s.DB.Delete(&model.Version{}, id).Error; err != nil {
		return &Error{StatusInternal, "删除失败"}
	}
	s.DB.Model(&model.App{}).Where("id = ? AND current_version_id = ?", v.AppID, v.ID).
		Update("current_version_id", 0)
	return nil
}

// VersionStats returns download/install totals and the most recent logs.
func (s *Service) VersionStats(ctx context.Context, id int64) (map[string]any, error) {
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		return nil, &Error{StatusNotFound, "版本不存在"}
	}
	var recent []model.DownloadLog
	s.DB.Where("version_id = ?", id).Order("id desc").Limit(20).Find(&recent)
	return map[string]any{
		"download_count": v.DownloadCount,
		"install_count":  v.InstallCount,
		"recent":         recent,
	}, nil
}

// CurrentVersion returns an app plus its current version (if any), admin shape.
func (s *Service) CurrentVersion(ctx context.Context, appID int64) (model.App, []model.Version, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return app, nil, &Error{StatusNotFound, "应用不存在"}
	}
	versions := make([]model.Version, 0, 1)
	if app.CurrentVersionID > 0 {
		var v model.Version
		if err := s.DB.First(&v, app.CurrentVersionID).Error; err == nil {
			versions = append(versions, v)
		}
	}
	return app, versions, nil
}

// CurrentDownloadURL returns the storage download URL of an app's current
// version, bypassing public access checks (management channel).
func (s *Service) CurrentDownloadURL(ctx context.Context, appID int64) (string, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return "", &Error{StatusNotFound, "应用不存在"}
	}
	if app.CurrentVersionID == 0 {
		return "", &Error{StatusNotFound, "应用尚未设置当前版本"}
	}
	var v model.Version
	if err := s.DB.First(&v, app.CurrentVersionID).Error; err != nil {
		return "", &Error{StatusNotFound, "当前版本不存在"}
	}
	rel, err := s.Storage.DownloadURL(ctx, v.StorageKey, v.FileName, 15*time.Minute)
	if err != nil {
		return "", &Error{StatusInternal, "生成下载链接失败"}
	}
	return rel, nil
}

func storageBackendName(s *Service) string {
	if s.Config.Storage.Backend == "" {
		return "local"
	}
	return s.Config.Storage.Backend
}