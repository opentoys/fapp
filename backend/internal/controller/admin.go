package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"disapp/internal/resources/storage"
	"disapp/internal/service"
	"disapp/pkg/web"
)

// AppsList returns app list (admin side).
func (c *Controller) AppsList(w http.ResponseWriter, r *http.Request) {
	apps, err := c.SVC.AdminApps(r.Context())
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, apps)
}

// AppDetailAdmin returns app detail with all versions (admin side).
func (c *Controller) AppDetailAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	app, versions, err := c.SVC.AppDetail(r.Context(), id)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"app": app, "versions": versions})
}

// CreateApp creates a new app.
func (c *Controller) CreateApp(w http.ResponseWriter, r *http.Request) {
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
	app, err := c.SVC.CreateApp(r.Context(), service.CreateAppInput{
		Name: req.Name, Platform: req.Platform, Icon: req.Icon,
		Description: req.Description, PackageName: req.PackageName,
	})
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// UpdateApp modifies an app.
func (c *Controller) UpdateApp(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	var req service.UpdateAppInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	app, err := c.SVC.UpdateApp(r.Context(), id, req)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// SetCurrentVersion marks one version as the app's current version.
func (c *Controller) SetCurrentVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
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
	app, err := c.SVC.SetCurrentVersion(r.Context(), id, req.VersionID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// DeleteApp deletes an app (cascades versions).
func (c *Controller) DeleteApp(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if err := c.SVC.DeleteApp(r.Context(), id); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// uploadTicket is the shared {url, key} response for presigned uploads.
type uploadTicket struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// PresignFile is the single presigned-upload endpoint for every file kind
// (version package, icon, screenshot). The caller sends the target app id and
// the upload file name; it gets back {url, key} where key is
// {app_id}/0/{file_name}. The caller pushes the bytes to url, then submits
// key (with size + sha256) when saving the entity it belongs to.
func (c *Controller) PresignFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID    int64  `json:"app_id"`
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.AppID <= 0 {
		web.SendError(w, web.CodeBadRequest, "app_id 必填")
		return
	}
	name := fmt.Sprintf("%d-%s", time.Now().UnixNano(), path.Base(req.FileName))
	key := storage.Key(req.AppID, 0, name)
	if !storage.ValidKey(key) {
		web.SendError(w, web.CodeBadRequest, "无效的 file_name")
		return
	}
	url, err := c.SVC.Storage.UploadURL(r.Context(), key, "", service.UploadExpiry)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, uploadTicket{URL: url, Key: key})
}

// CreateVersion records a version whose bytes were already uploaded to the
// client-submitted storage key.
func (c *Controller) CreateVersion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID       int64  `json:"app_id"`
		VersionCode int    `json:"version_code"`
		VersionName string `json:"version_name"`
		ReleaseType string `json:"release_type"`
		Arch        string `json:"arch"`
		Appid       string `json:"appid"`
		AppName     string `json:"app_name"`
		Changelog   string `json:"changelog"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
		SHA256      string `json:"sha256"`
		FileSize    int64  `json:"file_size"`
		Key         string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	v, err := c.SVC.CreateVersion(r.Context(), req.AppID, service.CreateVersionInput{
		VersionCode: req.VersionCode,
		VersionName: req.VersionName,
		ReleaseType: req.ReleaseType,
		Arch:        req.Arch,
		PackageName: req.Appid,
		AppName:     req.AppName,
		Changelog:   req.Changelog,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		SHA256:      req.SHA256,
		FileSize:    req.FileSize,
		StorageKey:  req.Key,
	})
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, v)
}

// DeleteAppScreenshot removes a screenshot by its exposed URL.
func (c *Controller) DeleteAppScreenshot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	app, err := c.SVC.DeleteAppScreenshot(r.Context(), id, r.URL.Query().Get("url"))
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// DeleteVersion deletes a version, optionally deleting the storage file.
func (c *Controller) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if err := c.SVC.DeleteVersion(r.Context(), id, r.URL.Query().Get("delete_file") == "true"); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// VersionStats returns download/install stats for a version.
func (c *Controller) VersionStats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	out, err := c.SVC.VersionStats(r.Context(), id)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, out)
}

// AppMembersAdmin lists the app's members (users who can manage it).
func (c *Controller) AppMembersAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	uids, err := c.SVC.AppMembers(r.Context(), id)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, uids)
}

// SetAppMembersAdmin replaces the app's member list. Only the super-admin may
// manage membership.
func (c *Controller) SetAppMembersAdmin(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil || user.UserID != service.SuperAdminUID {
		web.SendError(w, web.CodeForbidden, "仅超管可管理成员")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	var uids []int64
	if err := json.NewDecoder(r.Body).Decode(&uids); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if err := c.SVC.SetAppMembers(r.Context(), id, uids); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, uids)
}

// UsersList returns all non-super-admin accounts.
func (c *Controller) UsersList(w http.ResponseWriter, r *http.Request) {
	users, err := c.SVC.UsersList(r.Context())
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, users)
}

// CreateUser creates a new account.
func (c *Controller) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	u, err := c.SVC.CreateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, u)
}

// UpdateUser resets a user's account.
func (c *Controller) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
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
	u, err := c.SVC.UpdateUser(r.Context(), id, req.Username, req.Password)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, u)
}

// DeleteUser removes a user.
func (c *Controller) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if err := c.SVC.DeleteUser(r.Context(), id); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}