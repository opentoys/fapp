package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

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

// UploadAppIcon stores the app-level icon (multipart file "icon").
func (c *Controller) UploadAppIcon(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	file, header, err := r.FormFile("icon")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少图标文件")
		return
	}
	defer file.Close()
	app, err := c.SVC.SaveAppIcon(r.Context(), id, header.Header.Get("Content-Type"), header.Filename, file)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// UploadAppScreenshot stores one app screenshot (multipart file "screenshot").
func (c *Controller) UploadAppScreenshot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	file, header, err := r.FormFile("screenshot")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少截图文件")
		return
	}
	defer file.Close()
	app, err := c.SVC.SaveAppScreenshot(r.Context(), id, header.Header.Get("Content-Type"), header.Filename, file)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
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

// UploadVersion handles multipart file upload for a new version.
func (c *Controller) UploadVersion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		web.SendError(w, web.CodeBadRequest, "multipart 解析失败")
		return
	}
	appID, _ := strconv.ParseInt(r.FormValue("app_id"), 10, 64)
	if appID == 0 {
		web.SendError(w, web.CodeBadRequest, "app_id 必填")
		return
	}
	c.uploadForApp(w, r, appID)
}

// uploadForApp assembles and persists a new version. The multipart body is
// parsed by the caller (JWT-admin upload and API-key upload share this).
func (c *Controller) uploadForApp(w http.ResponseWriter, r *http.Request, appID int64) {
	file, header, err := r.FormFile("file")
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "缺少文件")
		return
	}
	defer file.Close()

	versionCode, _ := strconv.Atoi(r.FormValue("version_code"))

	var icon io.Reader
	if iconFile, _, err := r.FormFile("icon"); err == nil {
		defer iconFile.Close()
		icon = iconFile
	}
	v, err := c.SVC.SaveVersion(r.Context(), appID, service.SaveVersionInput{
		VersionCode: versionCode,
		VersionName: r.FormValue("version_name"),
		ReleaseType: r.FormValue("release_type"),
		Arch:        r.FormValue("arch"),
		PackageName: r.FormValue("appid"),
		AppName:     r.FormValue("app_name"),
		Changelog:   r.FormValue("changelog"),
		FileName:    header.Filename,
		File:        file,
		Icon:        icon,
	})
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, v)
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