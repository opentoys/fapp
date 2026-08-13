package server

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"disapp/internal/model"
	"disapp/internal/password"
	"disapp/internal/storage"
	"disapp/internal/web"
)

// AppsList returns app list (admin side).
func (s *Server) AppsList(w http.ResponseWriter, r *http.Request) {
	var apps []model.App
	if err := s.DB.Order("id desc").Find(&apps).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	web.SendJson(w, apps)
}

// CreateApp creates a new app.
func (s *Server) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Icon        string `json:"icon"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Name == "" {
		web.SendError(w, web.CodeBadRequest, "应用名不能为空")
		return
	}
	app := model.App{Name: req.Name, Icon: req.Icon, Description: req.Description}
	if err := s.DB.Create(&app).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建失败")
		return
	}
	web.SendJson(w, app)
}

// UpdateApp modifies an app.
func (s *Server) UpdateApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Icon        *string `json:"icon"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Icon != nil {
		app.Icon = *req.Icon
	}
	if req.Description != nil {
		app.Description = *req.Description
	}
	s.DB.Save(&app)
	web.SendJson(w, app)
}

// DeleteApp deletes an app (cascades channels and versions).
func (s *Server) DeleteApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.DB.Delete(&model.App{}, id).Error; err != nil {
		web.SendError(w, web.CodeInternal, "删除失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// ChannelsList returns channels, filtered by ?app_id=.
func (s *Server) ChannelsList(w http.ResponseWriter, r *http.Request) {
	q := s.DB.Order("id asc")
	if aid := r.URL.Query().Get("app_id"); aid != "" {
		if n, err := strconv.ParseInt(aid, 10, 64); err == nil {
			q = q.Where("app_id = ?", n)
		}
	}
	var channels []model.Channel
	if err := q.Find(&channels).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	web.SendJson(w, channels)
}

// CreateChannel creates a channel.
func (s *Server) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID int64  `json:"app_id"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Name == "" {
		web.SendError(w, web.CodeBadRequest, "渠道名不能为空")
		return
	}
	ch := model.Channel{AppID: req.AppID, Name: req.Name}
	if err := s.DB.Create(&ch).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建失败")
		return
	}
	web.SendJson(w, ch)
}

// UploadVersion handles multipart file upload for a new version.
func (s *Server) UploadVersion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		web.SendError(w, web.CodeBadRequest, "multipart 解析失败")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少文件")
		return
	}
	defer file.Close()

	appID, _ := strconv.ParseInt(r.FormValue("app_id"), 10, 64)
	channelID, _ := strconv.ParseInt(r.FormValue("channel_id"), 10, 64)
	versionCode, _ := strconv.Atoi(r.FormValue("version_code"))
	versionName := r.FormValue("version_name")
	accessMode := r.FormValue("access_mode")
	if accessMode == "" {
		accessMode = model.AccessPublic
	}

	if versionName == "" || appID == 0 {
		web.SendError(w, web.CodeBadRequest, "app_id 与 version_name 必填")
		return
	}

	// Create record first to get version_id for storage key.
	v := model.Version{
		AppID:          appID,
		ChannelID:      channelID,
		VersionName:    versionName,
		VersionCode:    versionCode,
		FileName:       header.Filename,
		FileType:       model.FileType(header.Filename),
		AccessMode:     accessMode,
		Changelog:      r.FormValue("changelog"),
		Enabled:        true,
		StorageBackend: storageBackendName(s),
	}
	switch accessMode {
	case model.AccessPassword:
		hash, salt := password.Hash(r.FormValue("password"))
		v.PasswordHash, v.Salt = hash, salt
	case model.AccessExpiry:
		expiresAt, _ := time.Parse(time.RFC3339, r.FormValue("expires_at"))
		if !expiresAt.IsZero() {
			v.ExpiresAt = &expiresAt
		}
	}
	if err := s.DB.Create(&v).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建版本失败")
		return
	}

	// Compute sha256 + size while writing to storage.
	key := storage.Key(appID, v.ID, header.Filename)
	hr := newHashReader(file)
	if _, err := s.Storage.Save(r.Context(), key, hr); err != nil {
		s.DB.Delete(&v)
		web.SendError(w, web.CodeInternal, "存储写入失败")
		return
	}
	s.DB.Model(&v).Updates(map[string]any{
		"storage_key": key,
		"file_size":   hr.n,
		"sha256":      hex.EncodeToString(hr.h.Sum(nil)),
	})
	web.SendJson(w, v)
}

// UpdateVersion updates version info (changelog, access mode, enabled, etc).
func (s *Server) UpdateVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return
	}
	var req struct {
		Changelog  *string    `json:"changelog"`
		AccessMode *string    `json:"access_mode"`
		Password   *string    `json:"password"`
		ExpiresAt  *time.Time `json:"expires_at"`
		Enabled    *bool      `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Changelog != nil {
		v.Changelog = *req.Changelog
	}
	if req.AccessMode != nil {
		v.AccessMode = *req.AccessMode
	}
	if req.Password != nil && *req.Password != "" {
		h, salt := password.Hash(*req.Password)
		v.PasswordHash, v.Salt = h, salt
	}
	if req.ExpiresAt != nil {
		v.ExpiresAt = req.ExpiresAt
	}
	if req.Enabled != nil {
		v.Enabled = *req.Enabled
	}
	if err := s.DB.Save(&v).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, v)
}

// DeleteVersion deletes a version, optionally deleting the storage file.
func (s *Server) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return
	}
	if r.URL.Query().Get("delete_file") == "true" && v.StorageKey != "" {
		s.Storage.Delete(r.Context(), v.StorageKey)
	}
	if err := s.DB.Delete(&model.Version{}, id).Error; err != nil {
		web.SendError(w, web.CodeInternal, "删除失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// VersionStats returns download/install stats for a version.
func (s *Server) VersionStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return
	}
	var recent []model.DownloadLog
	s.DB.Where("version_id = ?", id).Order("id desc").Limit(20).Find(&recent)
	web.SendJson(w, map[string]any{
		"download_count": v.DownloadCount,
		"install_count":  v.InstallCount,
		"recent":         recent,
	})
}

func storageBackendName(s *Server) string {
	if s.Config.Storage.Backend == "" {
		return "local"
	}
	return s.Config.Storage.Backend
}

// UsersList returns all non-super-admin accounts.
func (s *Server) UsersList(w http.ResponseWriter, r *http.Request) {
	var users []model.User
	if err := s.DB.Order("id asc").Find(&users).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	web.SendJson(w, users)
}

// CreateUser creates a new account. The super-admin is in config.json and
// is never created here; if the requested username matches the configured
// super-admin, refuse to avoid a confusing duplicate.
func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Username == "" || req.Password == "" {
		web.SendError(w, web.CodeBadRequest, "用户名和密码不能为空")
		return
	}
	if s.Config.Admin.Username != "" && req.Username == s.Config.Admin.Username {
		web.SendError(w, web.CodeBadRequest, "该用户名为超管保留")
		return
	}
	hash, salt := password.Hash(req.Password)
	u := model.User{Username: req.Username, PasswordHash: hash, Salt: salt}
	if err := s.DB.Create(&u).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建失败")
		return
	}
	web.SendJson(w, u)
}

// UpdateUser resets a user's password.
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "用户不存在")
		return
	}
	var req struct {
		Password *string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Password != nil && *req.Password != "" {
		hash, salt := password.Hash(*req.Password)
		u.PasswordHash, u.Salt = hash, salt
	}
	if err := s.DB.Save(&u).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, u)
}

// DeleteUser removes a user. Refuses if it would be the configured
// super-admin (shouldn't happen, since we never insert that name into the
// table, but defend in depth).
func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "用户不存在")
		return
	}
	if s.Config.Admin.Username != "" && u.Username == s.Config.Admin.Username {
		web.SendError(w, web.CodeBadRequest, "不能删除超管账号")
		return
	}
	if err := s.DB.Delete(&model.User{}, id).Error; err != nil {
		web.SendError(w, web.CodeInternal, "删除失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}
