package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
	"disapp/internal/resources/storage"
	"disapp/pkg/web"

	"gorm.io/gorm"
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

// AppDetailAdmin returns app detail with all versions (admin side). The public
// AppDetail endpoint only exposes the app's current version (and only when the
// app itself is published).
func (s *Server) AppDetailAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var versions []model.Version
	if err := s.DB.Where("app_id = ?", app.ID).Order("version_code desc").Find(&versions).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	web.SendJson(w, map[string]any{
		"app":      app,
		"versions": versions,
	})
}

// CreateApp creates a new app.
func (s *Server) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Platform    string  `json:"platform"`
		Icon        string  `json:"icon"`
		Description string  `json:"description"`
		PackageName *string `json:"appid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Name == "" {
		web.SendError(w, web.CodeBadRequest, "应用名不能为空")
		return
	}
	// Platform is chosen at creation and immutable afterwards.
	if req.Platform != "ios" && req.Platform != "android" {
		web.SendError(w, web.CodeBadRequest, "平台必须为 ios 或 android")
		return
	}
	// appid = Android package / iOS bundle id. NULL when unknown so that
	// manually-created apps without a package don't collide. Same platform +
	// same appid must be unique (enforced below and by a DB unique index).
	var pkg *string
	if req.PackageName != nil {
		p := strings.TrimSpace(*req.PackageName)
		if p != "" {
			pkg = &p
		}
	}
	if pkg != nil {
		var n int64
		s.DB.Model(&model.App{}).Where("platform = ? AND package_name = ?", req.Platform, *pkg).Count(&n)
		if n > 0 {
			web.SendError(w, web.CodeBadRequest, "该平台下已存在相同包名（appid）的应用")
			return
		}
	}
	app := model.App{Name: req.Name, Platform: req.Platform, PackageName: pkg, Icon: req.Icon, Description: req.Description}
	if err := s.DB.Create(&app).Error; err != nil {
		// Belt-and-suspenders for a concurrent duplicate create.
		if strings.Contains(err.Error(), "UNIQUE") {
			web.SendError(w, web.CodeBadRequest, "该平台下已存在相同包名（appid）的应用")
			return
		}
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
		Name        *string    `json:"name"`
		Icon        *string    `json:"icon"`
		Description *string    `json:"description"`
		AccessMode  *string    `json:"access_mode"`
		Password    *string    `json:"password"`
		ExpiresAt   *time.Time `json:"expires_at"`
		Published   *bool      `json:"published"`
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
	if req.Published != nil {
		app.Published = *req.Published
	}
	if req.AccessMode != nil {
		app.AccessMode = *req.AccessMode
		// A non-password scope no longer needs a stored credential.
		if app.AccessMode != model.AccessPassword {
			app.PasswordHash, app.Salt = "", ""
		}
	}
	if req.Password != nil && *req.Password != "" {
		h, salt := pwd.Hash(*req.Password)
		app.PasswordHash, app.Salt = h, salt
	}
	if req.ExpiresAt != nil {
		// Normalize client-provided expiry to the server's default timezone so
		// it is stored/displayed as server-local wall-clock time.
		at := req.ExpiresAt.In(time.Local)
		app.ExpiresAt = &at
	}
	if err := s.DB.Save(&app).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

// SetCurrentVersion marks one version as the app's current version. An app has
// exactly one current version (a single public download), so setting one simply
// overwrites app.CurrentVersionID. The version must belong to the app.
func (s *Server) SetCurrentVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var req struct {
		VersionID int64 `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.VersionID <= 0 {
		web.SendError(w, web.CodeBadRequest, "version_id 必填")
		return
	}
	var v model.Version
	if err := s.DB.First(&v, req.VersionID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return
	}
	if v.AppID != app.ID {
		web.SendError(w, web.CodeBadRequest, "版本不属于该应用")
		return
	}
	app.CurrentVersionID = v.ID
	if err := s.DB.Save(&app).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

// DeleteApp deletes an app (cascades versions).
func (s *Server) DeleteApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	// Best-effort cleanup of the app-level icon and screenshot files.
	if key := strings.TrimPrefix(app.Icon, "/api/v1/files/"); key != app.Icon {
		s.Storage.Delete(r.Context(), key)
	}
	for _, url := range app.Screenshots {
		if key := strings.TrimPrefix(url, "/api/v1/files/"); key != url {
			s.Storage.Delete(r.Context(), key)
		}
	}
	if err := s.DB.Delete(&model.App{}, id).Error; err != nil {
		web.SendError(w, web.CodeInternal, "删除失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// UploadAppIcon stores the app-level icon (multipart file "icon"). It is
// saved under the fixed key {app_id}/0/icon.png so re-uploads overwrite the
// previous file instead of accumulating orphans, and the File handler serves
// it through the usual /api/v1/files/{key} proxy (local + COS).
func (s *Server) UploadAppIcon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	file, header, err := r.FormFile("icon")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少图标文件")
		return
	}
	defer file.Close()
	if !isImageUpload(header.Header.Get("Content-Type"), header.Filename) {
		web.SendError(w, web.CodeBadRequest, "仅支持图片文件")
		return
	}
	key := storage.Key(app.ID, 0, "icon.png")
	if _, err := s.Storage.Save(r.Context(), key, file); err != nil {
		web.SendError(w, web.CodeInternal, "存储失败")
		return
	}
	app.Icon = "/api/v1/files/" + key
	if err := s.DB.Model(&app).Update("icon", app.Icon).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

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

// UploadAppScreenshot stores one app screenshot (multipart file "screenshot")
// under {app_id}/0/shot-<nano>.ext and appends its URL to App.Screenshots.
func (s *Server) UploadAppScreenshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	file, header, err := r.FormFile("screenshot")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少截图文件")
		return
	}
	defer file.Close()
	if !isImageUpload(header.Header.Get("Content-Type"), header.Filename) {
		web.SendError(w, web.CodeBadRequest, "仅支持图片文件")
		return
	}
	key := storage.Key(app.ID, 0, fmt.Sprintf("shot-%d%s", time.Now().UnixNano(), filepath.Ext(header.Filename)))
	if _, err := s.Storage.Save(r.Context(), key, file); err != nil {
		web.SendError(w, web.CodeInternal, "存储失败")
		return
	}
	app.Screenshots = append(app.Screenshots, "/api/v1/files/"+key)
	if err := s.DB.Model(&app).Update("screenshots", app.Screenshots).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

// DeleteAppScreenshot removes a screenshot by its exposed URL and deletes the
// underlying storage file.
func (s *Server) DeleteAppScreenshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	url := r.URL.Query().Get("url")
	key := strings.TrimPrefix(url, "/api/v1/files/")
	if key == url || key == "" {
		web.SendError(w, web.CodeBadRequest, "无效的截图地址")
		return
	}
	kept := app.Screenshots[:0]
	for _, u := range app.Screenshots {
		if u != url {
			kept = append(kept, u)
		}
	}
	app.Screenshots = kept
	s.Storage.Delete(r.Context(), key)
	if err := s.DB.Model(&app).Update("screenshots", app.Screenshots).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

// UploadVersion handles multipart file upload for a new version.
func (s *Server) UploadVersion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		web.SendError(w, web.CodeBadRequest, "multipart 解析失败")
		return
	}
	appID, _ := strconv.ParseInt(r.FormValue("app_id"), 10, 64)
	if appID == 0 {
		web.SendError(w, web.CodeBadRequest, "app_id 必填")
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	s.uploadForApp(w, r, &app)
}

// uploadForApp assembles and persists a new version for an already-loaded
// app. Shared by the JWT-admin upload and the API-key upload (which resolves
// the app from the URL path instead of the form). The multipart body is
// parsed by the caller.
func (s *Server) uploadForApp(w http.ResponseWriter, r *http.Request, app *model.App) {
	file, header, err := r.FormFile("file")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少文件")
		return
	}
	defer file.Close()

	versionCode, _ := strconv.Atoi(r.FormValue("version_code"))
	versionName := r.FormValue("version_name")
	if versionName == "" {
		web.SendError(w, web.CodeBadRequest, "version_name 必填")
		return
	}

	// Appid lock: the app's package_name (appid) is locked on the first version
	// upload that carries one. After that, every uploaded package must expose
	// the same appid (+ the app's platform, which is already enforced), so an
	// app can never silently accumulate versions of unrelated apps. An app
	// without a lock stays unlocked when the package exposes no appid.
	pkg := strings.TrimSpace(r.FormValue("appid"))
	if app.PackageName == nil || *app.PackageName == "" {
		if pkg != "" {
			var n int64
			s.DB.Model(&model.App{}).
				Where("platform = ? AND package_name = ? AND id != ?", app.Platform, pkg, app.ID).
				Count(&n)
			if n > 0 {
				web.SendError(w, web.CodeBadRequest, "该平台下已存在相同包名（appid）的应用")
				return
			}
			app.PackageName = &pkg
			if err := s.DB.Save(app).Error; err != nil {
				// Belt-and-suspenders for a concurrent duplicate lock.
				if strings.Contains(err.Error(), "UNIQUE") {
					web.SendError(w, web.CodeBadRequest, "该平台下已存在相同包名（appid）的应用")
					return
				}
				web.SendError(w, web.CodeInternal, "保存失败")
				return
			}
		}
	} else if pkg != *app.PackageName {
		web.SendError(w, web.CodeBadRequest, "安装包 appid 与应用不一致")
		return
	}

	// Upload always creates a draft: a version only becomes publicly visible
	// once it is set as the app's current version and the app itself is
	// published. Metadata such as package_name/app_name is parsed in the
	// browser and sent here. The platform is always taken from the app
	// (single-platform apps), so a version can never drift onto a different
	// platform than its app.
	fileType := model.FileType(header.Filename)
	v := model.Version{
		AppID:          app.ID,
		ReleaseType:    r.FormValue("release_type"),
		Platform:       app.Platform,
		Arch:           r.FormValue("arch"),
		VersionName:    versionName,
		VersionCode:    versionCode,
		PackageName:    pkg,
		AppName:        r.FormValue("app_name"),
		FileName:       header.Filename,
		FileType:       fileType,
		Changelog:      r.FormValue("changelog"),
		StorageBackend: storageBackendName(s),
	}
	if err := s.DB.Create(&v).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建版本失败")
		return
	}

	// Compute sha256 + size while writing to storage.
	key := storage.Key(app.ID, v.ID, header.Filename)
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

	// Best-effort: store the browser-parsed app icon (always PNG) and expose
	// its URL. The File handler proxies /api/v1/files/{key} via Storage.Open,
	// so this works for both local and COS backends.
	if icon, _, err := r.FormFile("icon"); err == nil {
		defer icon.Close()
		iconKey := storage.Key(app.ID, v.ID, "icon.png")
		if _, err := s.Storage.Save(r.Context(), iconKey, icon); err != nil {
			log.Printf("store icon failed: %v", err)
		} else {
			v.IconURL = "/api/v1/files/" + iconKey
			s.DB.Model(&v).Update("icon_url", v.IconURL)
		}
	}

	web.SendJson(w, v)
}

// DeleteVersion deletes a version, optionally deleting the storage file. If
// the deleted version is the app's current version, the app's current version
// pointer is reset so the app no longer exposes a public download.
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
	s.DB.Model(&model.App{}).Where("id = ? AND current_version_id = ?", v.AppID, v.ID).
		Update("current_version_id", 0)
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

// AppMembersAdmin lists the app's members (users who can manage it).
func (s *Server) AppMembersAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var rows []model.AppMember
	s.DB.Where("app_id = ?", app.ID).Find(&rows)
	uids := make([]int64, 0, len(rows))
	for _, row := range rows {
		uids = append(uids, row.UserID)
	}
	web.SendJson(w, uids)
}

// SetAppMembersAdmin replaces the app's member list. Only the super-admin may
// manage membership; other admins keep their own access and cannot add/remove
// anyone.
func (s *Server) SetAppMembersAdmin(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil || user.UserID != SuperAdminUID {
		web.SendError(w, web.CodeForbidden, "仅超管可管理成员")
		return
	}
	id := r.PathValue("id")
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var uids []int64
	if err := json.NewDecoder(r.Body).Decode(&uids); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	// Reject unknown user ids so a bad client can't create dangling rows.
	if len(uids) > 0 {
		var n int64
		s.DB.Model(&model.User{}).Where("id IN ?", uids).Count(&n)
		if n != int64(len(uids)) {
			web.SendError(w, web.CodeBadRequest, "用户不存在")
			return
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
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, uids)
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
	hash, salt := pwd.Hash(req.Password)
	u := model.User{Username: req.Username, PasswordHash: hash, Salt: salt}
	if err := s.DB.Create(&u).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建失败")
		return
	}
	web.SendJson(w, u)
}

// UpdateUser resets a user's pwd.
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "用户不存在")
		return
	}
	var req struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Username != nil && *req.Username != "" {
		u.Username = *req.Username
	}
	if req.Password != nil && *req.Password != "" {
		hash, salt := pwd.Hash(*req.Password)
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
