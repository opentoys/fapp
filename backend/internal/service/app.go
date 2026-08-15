package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"disapp/internal/resources/storage"
	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"

	"gorm.io/gorm"
)

// CreateAppInput carries validated create fields from the controller.
type CreateAppInput struct {
	Name        string
	Platform    string
	Icon        string
	Description string
	PackageName *string
}

func (s *Service) CreateApp(ctx context.Context, in CreateAppInput) (*model.App, error) {
	if in.Name == "" {
		return nil, &Error{StatusBadRequest, "应用名不能为空"}
	}
	if in.Platform != "ios" && in.Platform != "android" {
		return nil, &Error{StatusBadRequest, "平台必须为 ios 或 android"}
	}
	var pkg *string
	if in.PackageName != nil {
		p := strings.TrimSpace(*in.PackageName)
		if p != "" {
			pkg = &p
		}
	}
	if pkg != nil {
		var n int64
		s.DB.Model(&model.App{}).Where("platform = ? AND package_name = ?", in.Platform, *pkg).Count(&n)
		if n > 0 {
			return nil, &Error{StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
		}
	}
	app := model.App{Name: in.Name, Platform: in.Platform, PackageName: pkg, Icon: in.Icon, Description: in.Description}
	if err := s.DB.Create(&app).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, &Error{StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
		}
		return nil, &Error{StatusInternal, "创建失败"}
	}
	return &app, nil
}

// UpdateAppInput carries the pointer-based PATCH fields.
type UpdateAppInput struct {
	Name        *string    `json:"name"`
	Icon        *string    `json:"icon"`
	Description *string    `json:"description"`
	AccessMode  *string    `json:"access_mode"`
	Password    *string    `json:"password"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Published   *bool      `json:"published"`
}

func (s *Service) UpdateApp(ctx context.Context, id int64, in UpdateAppInput) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	if in.Name != nil {
		app.Name = *in.Name
	}
	if in.Icon != nil {
		app.Icon = *in.Icon
	}
	if in.Description != nil {
		app.Description = *in.Description
	}
	if in.Published != nil {
		app.Published = *in.Published
	}
	if in.AccessMode != nil {
		app.AccessMode = *in.AccessMode
		if app.AccessMode != model.AccessPassword {
			app.PasswordHash, app.Salt = "", ""
		}
	}
	if in.Password != nil && *in.Password != "" {
		h, salt := pwd.Hash(*in.Password)
		app.PasswordHash, app.Salt = h, salt
	}
	if in.ExpiresAt != nil {
		at := in.ExpiresAt.In(time.Local)
		app.ExpiresAt = &at
	}
	if err := s.DB.Save(&app).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}

func (s *Service) DeleteApp(ctx context.Context, appID int64) error {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return &Error{StatusNotFound, "应用不存在"}
	}
	if key := strings.TrimPrefix(app.Icon, "/api/v1/files/"); key != app.Icon {
		s.Storage.Delete(ctx, key)
	}
	for _, url := range app.Screenshots {
		if key := strings.TrimPrefix(url, "/api/v1/files/"); key != url {
			s.Storage.Delete(ctx, key)
		}
	}
	if err := s.DB.Delete(&model.App{}, appID).Error; err != nil {
		return &Error{StatusInternal, "删除失败"}
	}
	return nil
}

func (s *Service) AdminApps(ctx context.Context) ([]model.App, error) {
	var apps []model.App
	if err := s.DB.Order("id desc").Find(&apps).Error; err != nil {
		return nil, &Error{StatusInternal, "查询失败"}
	}
	return apps, nil
}

// AppDetail returns an app with its full version list (admin side), ordered by
// version_code desc.
func (s *Service) AppDetail(ctx context.Context, appID int64) (model.App, []model.Version, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return app, nil, &Error{StatusNotFound, "应用不存在"}
	}
	var versions []model.Version
	if err := s.DB.Where("app_id = ?", app.ID).Order("version_code desc").Find(&versions).Error; err != nil {
		return app, nil, &Error{StatusInternal, "查询失败"}
	}
	return app, versions, nil
}

// AppMembers returns the member user ids of an app.
func (s *Service) AppMembers(ctx context.Context, appID int64) ([]int64, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	var rows []model.AppMember
	s.DB.Where("app_id = ?", app.ID).Find(&rows)
	uids := make([]int64, 0, len(rows))
	for _, row := range rows {
		uids = append(uids, row.UserID)
	}
	return uids, nil
}

// SetAppMembers replaces an app's member list, rejecting unknown user ids so a
// bad client cannot create dangling rows.
func (s *Service) SetAppMembers(ctx context.Context, appID int64, uids []int64) error {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return &Error{StatusNotFound, "应用不存在"}
	}
	if len(uids) > 0 {
		var n int64
		s.DB.Model(&model.User{}).Where("id IN ?", uids).Count(&n)
		if n != int64(len(uids)) {
			return &Error{StatusBadRequest, "用户不存在"}
		}
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("app_id = ?", app.ID).Delete(&model.AppMember{}).Error; err != nil {
			return err
		}
		if len(uids) > 0 {
			rows := make([]model.AppMember, 0, len(uids))
			for _, uid := range uids {
				rows = append(rows, model.AppMember{UserID: uid, AppID: app.ID})
			}
			return tx.Create(&rows).Error
		}
		return nil
	})
	if err != nil {
		return &Error{StatusInternal, "保存失败"}
	}
	return nil
}

func (s *Service) UsersList(ctx context.Context) ([]model.User, error) {
	return s.Users(ctx)
}

// SetCurrentVersion marks one version as the app's current version. The
// version must belong to the app.
func (s *Service) SetCurrentVersion(ctx context.Context, appID, versionID int64) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	var v model.Version
	if err := s.DB.First(&v, versionID).Error; err != nil {
		return nil, &Error{StatusNotFound, "版本不存在"}
	}
	if v.AppID != app.ID {
		return nil, &Error{StatusBadRequest, "版本不属于该应用"}
	}
	app.CurrentVersionID = v.ID
	if err := s.DB.Save(&app).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}

// isImageUpload reports whether a multipart upload is an image, by content type
// or extension.
func isImageUpload(contentType, filename string) bool {
	if strings.HasPrefix(contentType, "image/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return true
	}
	return false
}

// SaveAppIcon stores the app-level icon under the fixed key {app_id}/0/icon.png
// so re-uploads overwrite the previous file.
func (s *Service) SaveAppIcon(ctx context.Context, appID int64, contentType, filename string, file io.Reader) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	if !isImageUpload(contentType, filename) {
		return nil, &Error{StatusBadRequest, "仅支持图片文件"}
	}
	key := storage.Key(app.ID, 0, "icon.png")
	if _, err := s.Storage.Save(ctx, key, file); err != nil {
		return nil, &Error{StatusInternal, "存储失败"}
	}
	app.Icon = "/api/v1/files/" + key
	if err := s.DB.Model(&app).Update("icon", app.Icon).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}

// SaveAppScreenshot stores one app screenshot under {app_id}/0/shot-<nano>.ext
// and appends its URL to App.Screenshots.
func (s *Service) SaveAppScreenshot(ctx context.Context, appID int64, contentType, filename string, file io.Reader) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	if !isImageUpload(contentType, filename) {
		return nil, &Error{StatusBadRequest, "仅支持图片文件"}
	}
	key := storage.Key(app.ID, 0, fmt.Sprintf("shot-%d%s", time.Now().UnixNano(), filepath.Ext(filename)))
	if _, err := s.Storage.Save(ctx, key, file); err != nil {
		return nil, &Error{StatusInternal, "存储失败"}
	}
	app.Screenshots = append(app.Screenshots, "/api/v1/files/"+key)
	if err := s.DB.Model(&app).Update("screenshots", app.Screenshots).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}

// DeleteAppScreenshot removes a screenshot by its exposed URL and deletes the
// underlying storage file.
func (s *Service) DeleteAppScreenshot(ctx context.Context, appID int64, rawURL string) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	key := strings.TrimPrefix(rawURL, "/api/v1/files/")
	if key == rawURL || key == "" {
		return nil, &Error{StatusBadRequest, "无效的截图地址"}
	}
	kept := app.Screenshots[:0]
	for _, u := range app.Screenshots {
		if u != rawURL {
			kept = append(kept, u)
		}
	}
	app.Screenshots = kept
	s.Storage.Delete(ctx, key)
	if err := s.DB.Model(&app).Update("screenshots", app.Screenshots).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}