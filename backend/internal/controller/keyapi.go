package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"disapp/internal/service"
	"disapp/pkg/web"
)

// authKeyApp authenticates the `apikey` query param against an app that the
// key's owner can manage, enforcing the minimum scope. The service handles
// authn/authz+touch; returns the key id.
func (c *Controller) authKeyApp(w http.ResponseWriter, r *http.Request, appID int64, requireRun bool) (int64, bool) {
	key, err := c.SVC.AuthorizeKeyApp(r.Context(), r.URL.Query().Get("apikey"), appID, requireRun)
	if err != nil {
		sendErr(w, err)
		return 0, false
	}
	c.SVC.TouchKey(r.Context(), key.ID)
	return key.ID, true
}

// PresignKeyFile issues an upload ticket {url, key} for an app via API key
// (scope run). The caller pushes the bytes to url, then submits key (with size
// + sha256) to POST /keys/{app_id}/versions.
func (c *Controller) PresignKeyFile(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, true); !ok {
		return
	}
	var req struct {
		FileName string `json:"file_name"`
		SHA256   string `json:"sha256"`
		FileSize int64  `json:"file_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if !isValidSHA256(req.SHA256) || req.FileSize <= 0 {
		web.SendError(w, web.CodeBadRequest, "sha256 与 file_size 必填")
		return
	}
	ticket, err := c.presignFor(r.Context(), appID, req.FileName, req.SHA256, req.FileSize)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, ticket)
}

// UploadKeyVersion records a version via API key (scope run). The bytes were
// already pushed to the submitted storage key.
func (c *Controller) UploadKeyVersion(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, true); !ok {
		return
	}
	var req struct {
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
	v, err := c.SVC.CreateVersion(r.Context(), appID, service.CreateVersionInput{
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

// SetKeyCurrentVersion sets the app's current version via API key (scope run).
func (c *Controller) SetKeyCurrentVersion(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, true); !ok {
		return
	}
	var req struct {
		VersionID int64 `json:"version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	app, err := c.SVC.SetCurrentVersion(r.Context(), appID, req.VersionID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, app)
}

// KeyVersionsList returns all versions of the app (admin detail shape).
func (c *Controller) KeyVersionsList(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, false); !ok {
		return
	}
	app, versions, err := c.SVC.AppDetail(r.Context(), appID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"app": app, "versions": versions})
}

// KeyCurrentVersion returns the app plus its current version (if any).
func (c *Controller) KeyCurrentVersion(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, false); !ok {
		return
	}
	app, versions, err := c.SVC.CurrentVersion(r.Context(), appID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"app": app, "versions": versions})
}

// KeyCurrentDownload returns a signed download URL for the app's current
// version. Unlike the public endpoint it does not check published/access —
// this is a management channel behind a key with manage rights.
func (c *Controller) KeyCurrentDownload(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("app_id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if _, ok := c.authKeyApp(w, r, appID, false); !ok {
		return
	}
	rel, err := c.SVC.CurrentDownloadURL(r.Context(), appID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"url": absURL(r, rel)})
}