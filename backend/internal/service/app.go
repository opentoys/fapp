package service

import (
	"context"
	"fmt"
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
	app := model.App{Name: in.Name, Platform: in.Platform, PackageName: pkg, Icon: in.Icon, Description: in.Description, AccessMode: model.AccessPublic}
	if err := s.DB.Create(&app).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, &Error{StatusBadRequest, "该平台下已存在相同包名（appid）的应用"}
		}
		return nil, &Error{StatusInternal, "创建失败"}
	}
	return &app, nil
}

// UpdateAppInput carries the pointer-based PATCH fields. AccessMode keeps the
// explicit 公开/密码 choice; the download-link expiry is a separate,
// independent setting.
type UpdateAppInput struct {
	Name        *string    `json:"name"`
	Icon        *string    `json:"icon"`
	Screenshots []string   `json:"screenshots"`
	Description *string    `json:"description"`
	AccessMode  *string    `json:"access_mode"` // public | password
	Password    *string    `json:"password"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ClearExpiry bool       `json:"clear_expiry"`
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
	if in.Screenshots != nil {
		app.Screenshots = in.Screenshots
	}
	if in.Description != nil {
		app.Description = *in.Description
	}
	wasPublished := app.Published
	if in.Published != nil {
		app.Published = *in.Published
	}
	if in.AccessMode != nil {
		switch *in.AccessMode {
		case model.AccessPassword:
			if in.Password == nil || *in.Password == "" {
				return nil, &Error{StatusBadRequest, "密码模式下需要填写下载密码"}
			}
			app.AccessMode = model.AccessPassword
		case model.AccessPublic:
			app.AccessMode = model.AccessPublic
		default:
			return nil, &Error{StatusBadRequest, "access_mode 必须为 public 或 password"}
		}
	}
	if in.Password != nil && *in.Password != "" {
		h, salt := pwd.Hash(*in.Password)
		app.PasswordHash, app.Salt = h, salt
	}
	// Public mode has no download password.
	if app.AccessMode != model.AccessPassword {
		app.PasswordHash, app.Salt = "", ""
	}
	if in.ExpiresAt != nil {
		at := in.ExpiresAt.In(time.Local)
		app.ExpiresAt = &at
	}
	if in.ClearExpiry {
		app.ExpiresAt = nil
	}
	if err := s.DB.Save(&app).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	// Notify on publish/下架 changes.
	if in.Published != nil && *in.Published != wasPublished {
		s.NotifyEventParams(ctx, app.ID, model.EventAppPublish, app.Name, NotifyParams{
			"published":  fmt.Sprintf("%t", app.Published),
			"expires_at": expiresAtString(app.ExpiresAt),
		})
	}
	return &app, nil
}

func expiresAtString(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.In(time.Local).Format("2006-01-02 15:04:05")
}

func (s *Service) DeleteApp(ctx context.Context, appID int64) error {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return &Error{StatusNotFound, "应用不存在"}
	}
	for _, key := range app.Screenshots {
		s.Storage.Delete(ctx, key)
	}
	if app.Icon != "" {
		s.Storage.Delete(ctx, app.Icon)
	}
	if err := s.DB.Delete(&model.App{}, appID).Error; err != nil {
		return &Error{StatusInternal, "删除失败"}
	}
	return nil
}

// AppName returns the app's name, used to build the storage key folder.
func (s *Service) AppName(ctx context.Context, appID int64) (string, error) {
	var app model.App
	if err := s.DB.Select("name").First(&app, appID).Error; err != nil {
		return "", &Error{StatusNotFound, "应用不存在"}
	}
	return app.Name, nil
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
	s.NotifyEventParams(context.Background(), app.ID, model.EventVersionCurrent, app.Name, s.versionParams(&v))
	return &app, nil
}

// DeleteAppScreenshot removes a screenshot by its storage key and deletes the
// underlying file.
func (s *Service) DeleteAppScreenshot(ctx context.Context, appID int64, key string) (*model.App, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{StatusNotFound, "应用不存在"}
	}
	if key == "" || !storage.ValidKey(key) {
		return nil, &Error{StatusBadRequest, "无效的截图地址"}
	}
	kept := app.Screenshots[:0]
	for _, k := range app.Screenshots {
		if k != key {
			kept = append(kept, k)
		}
	}
	app.Screenshots = kept
	s.Storage.Delete(ctx, key)
	if err := s.DB.Model(&app).Update("screenshots", app.Screenshots).Error; err != nil {
		return nil, &Error{StatusInternal, "保存失败"}
	}
	return &app, nil
}

