package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"disapp/internal/resources/storage"
	"disapp/internal/resources/store/model"
)

// CreateVersionInput carries the JSON metadata for a new version. The file
// itself is pushed directly to a presigned URL obtained separately (POST
// /api/v1/files); sha256 + size + the storage key are computed/assigned
// client-side and submitted here together.
type CreateVersionInput struct {
	VersionCode int
	VersionName string
	ReleaseType string
	Arch        string
	PackageName string
	AppName     string
	Changelog   string
	FileName    string
	ContentType string
	SHA256      string
	FileSize    int64
	StorageKey  string
}

// UploadExpiry bounds how long a presigned upload URL stays valid.
const UploadExpiry = 30 * time.Minute

// CreateVersion records a new version whose bytes were already uploaded to the
// client-supplied storage key. It locks the app's appid on the first upload
// that carries one.
func (s *Service) CreateVersion(ctx context.Context, appID int64, in CreateVersionInput) (*model.Version, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "应用不存在"}
	}
	if in.VersionName == "" {
		return nil, &Error{http.StatusBadRequest, "version_name 必填"}
	}
	if in.FileName == "" {
		return nil, &Error{http.StatusBadRequest, "file_name 必填"}
	}
	if !storage.ValidKey(in.StorageKey) {
		return nil, &Error{http.StatusBadRequest, "无效的上传 key"}
	}
	pkg := strings.TrimSpace(in.PackageName)
	if app.PackageName == nil || *app.PackageName == "" {
		if pkg != "" {
			var n int64
			s.DB.Model(&model.App{}).
				Where("platform = ? AND package_name = ? AND id != ?", app.Platform, pkg, app.ID).
				Count(&n)
			if n > 0 {
				return nil, &Error{http.StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
			}
			app.PackageName = &pkg
			if err := s.DB.Save(app).Error; err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					return nil, &Error{http.StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
				}
				return nil, &Error{http.StatusInternalServerError, "保存失败"}
			}
		}
	} else if pkg != *app.PackageName {
		return nil, &Error{http.StatusBadRequest, "安装包 appid 与应用不一致"}
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
		StorageKey:     in.StorageKey,
		FileSize:       in.FileSize,
		SHA256:         in.SHA256,
		StorageBackend: storageBackendName(s),
	}
	if err := s.DB.Create(&v).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "创建版本失败"}
	}
	// The app's first version becomes current automatically; later ones need
	// the explicit "set current" action.
	if app.CurrentVersionID == 0 {
		app.CurrentVersionID = v.ID
		if err := s.DB.Save(&app).Error; err != nil {
			return nil, &Error{http.StatusInternalServerError, "保存失败"}
		}
		s.NotifyEventParams(ctx, app.ID, model.EventVersionCurrent, app.Name, s.versionParams(&v))
	}
	s.NotifyEventParams(ctx, app.ID, model.EventVersionUploaded, app.Name, s.versionParams(&v))
	return &v, nil
}

// versionParams fills the version-related notification parameters.
func (s *Service) versionParams(v *model.Version) NotifyParams {
	return NotifyParams{
		"version_id":   fmt.Sprintf("%d", v.ID),
		"version_name": v.VersionName,
		"version_code": strconv.Itoa(v.VersionCode),
		"file_name":    v.FileName,
		"file_size":    fmt.Sprintf("%d", v.FileSize),
	}
}

// DeleteVersion deletes a version, optionally deleting the storage file. If the
// deleted version was current, the app's current pointer is reset.
func (s *Service) DeleteVersion(ctx context.Context, id int64, deleteFile bool) error {
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		return &Error{http.StatusNotFound, "版本不存在"}
	}
	if deleteFile && v.StorageKey != "" {
		s.Storage.Delete(ctx, v.StorageKey)
	}
	if err := s.DB.Delete(&model.Version{}, id).Error; err != nil {
		return &Error{http.StatusInternalServerError, "删除失败"}
	}
	s.DB.Model(&model.App{}).Where("id = ? AND current_version_id = ?", v.AppID, v.ID).
		Update("current_version_id", 0)
	return nil
}

// VersionStats returns download/install totals and the most recent logs.
func (s *Service) VersionStats(ctx context.Context, id int64) (map[string]any, error) {
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "版本不存在"}
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
		return app, nil, &Error{http.StatusNotFound, "应用不存在"}
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
		return "", &Error{http.StatusNotFound, "应用不存在"}
	}
	if app.CurrentVersionID == 0 {
		return "", &Error{http.StatusNotFound, "应用尚未设置当前版本"}
	}
	var v model.Version
	if err := s.DB.First(&v, app.CurrentVersionID).Error; err != nil {
		return "", &Error{http.StatusNotFound, "当前版本不存在"}
	}
	rel, err := s.Storage.DownloadURL(ctx, v.StorageKey, v.FileName, 15*time.Minute)
	if err != nil {
		return "", &Error{http.StatusInternalServerError, "生成下载链接失败"}
	}
	return rel, nil
}

// PreviewURL returns a fresh signed download URL for a storage key (used by
// the admin ?dl redirect). filename derives from the key.
func (s *Service) PreviewURL(ctx context.Context, key string) (string, error) {
	rel, err := s.Storage.DownloadURL(ctx, key, key[strings.LastIndex(key, "/")+1:], 15*time.Minute)
	if err != nil {
		return "", &Error{http.StatusInternalServerError, "生成下载链接失败"}
	}
	return rel, nil
}

func storageBackendName(s *Service) string {
	if s.Config.Storage.Backend == "" {
		return "local"
	}
	return s.Config.Storage.Backend
}
