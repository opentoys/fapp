package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"disapp/internal/model"
	"disapp/internal/web"
)

// authorizeKeyApp authenticates the `apikey` query param against an app that
// the key's owner can manage, and enforces the minimum scope. Returns a
// *webErr for the caller to send.
func (s *Server) authorizeKeyApp(r *http.Request, appID int64, requireRun bool) (*model.ApiKey, error) {
	key, err := s.authorizeKey(r)
	if err != nil {
		return nil, err
	}
	if !s.canManage(key.UserID, appID) {
		return nil, &webErr{web.CodeForbidden, "无权访问该应用"}
	}
	if requireRun && key.Scope != model.KeyScopeRun {
		return nil, &webErr{web.CodeForbidden, "该 key 需要 run 权限"}
	}
	return key, nil
}

// sendKeyErr unwraps a *webErr into a JSON error response.
func sendKeyErr(w http.ResponseWriter, err error) {
	if we, ok := err.(*webErr); ok {
		web.SendError(w, we.code, we.msg)
		return
	}
	web.SendError(w, web.CodeInternal, "内部错误")
}

// UploadKeyVersion uploads a new version for an app via API key (scope run).
// Multipart shape matches the JWT-admin upload except app_id comes from the
// URL path.
func (s *Server) UploadKeyVersion(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	key, err := s.authorizeKeyApp(r, appID, true)
	if err != nil {
		sendKeyErr(w, err)
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	s.touchKey(key.ID)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		web.SendError(w, web.CodeBadRequest, "multipart 解析失败")
		return
	}
	s.uploadForApp(w, r, &app)
}

// SetKeyCurrentVersion sets the app's current version via API key (scope run).
func (s *Server) SetKeyCurrentVersion(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	key, err := s.authorizeKeyApp(r, appID, true)
	if err != nil {
		sendKeyErr(w, err)
		return
	}
	var req struct {
		VersionID int64 `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
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
	s.touchKey(key.ID)
	app.CurrentVersionID = v.ID
	if err := s.DB.Save(&app).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, app)
}

// KeyVersionsList returns all versions of the app (admin detail shape).
func (s *Server) KeyVersionsList(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	key, err := s.authorizeKeyApp(r, appID, false)
	if err != nil {
		sendKeyErr(w, err)
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var versions []model.Version
	if err := s.DB.Where("app_id = ?", app.ID).Order("version_code desc").Find(&versions).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	s.touchKey(key.ID)
	web.SendJson(w, map[string]any{
		"app":      app,
		"versions": versions,
	})
}

// KeyCurrentVersion returns the app plus its current version (if any).
func (s *Server) KeyCurrentVersion(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	key, err := s.authorizeKeyApp(r, appID, false)
	if err != nil {
		sendKeyErr(w, err)
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	versions := make([]model.Version, 0, 1)
	if app.CurrentVersionID > 0 {
		var v model.Version
		if err := s.DB.First(&v, app.CurrentVersionID).Error; err == nil {
			versions = append(versions, v)
		}
	}
	s.touchKey(key.ID)
	web.SendJson(w, map[string]any{
		"app":      app,
		"versions": versions,
	})
}

// KeyCurrentDownload returns a signed download URL for the app's current
// version. Unlike the public endpoint it does not check published/access —
// this is a management channel behind a key with manage rights.
func (s *Server) KeyCurrentDownload(w http.ResponseWriter, r *http.Request) {
	appID, _ := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	key, err := s.authorizeKeyApp(r, appID, false)
	if err != nil {
		sendKeyErr(w, err)
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	if app.CurrentVersionID == 0 {
		web.SendError(w, web.CodeNotFound, "应用尚未设置当前版本")
		return
	}
	var v model.Version
	if err := s.DB.First(&v, app.CurrentVersionID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "当前版本不存在")
		return
	}
	rel, err := s.Storage.DownloadURL(r.Context(), v.StorageKey, v.FileName, 15*time.Minute)
	if err != nil {
		web.SendError(w, web.CodeInternal, "生成下载链接失败")
		return
	}
	s.touchKey(key.ID)
	web.SendJson(w, map[string]any{"url": absURL(r, rel)})
}