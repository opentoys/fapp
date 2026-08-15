package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"disapp/internal/model"
	"disapp/internal/password"
	"disapp/internal/web"
)

type appSummary struct {
	model.App
	LatestVersion *model.Version `json:"latest_version"`
}

// Apps returns the published app list; each app's latest_version is its
// current version (the single publicly downloadable version).
func (s *Server) Apps(w http.ResponseWriter, r *http.Request) {
	var apps []model.App
	if err := s.DB.Where("published = ?", true).Order("id desc").Find(&apps).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	// Collect the current version of each app in one query.
	versionIDs := make([]int64, 0, len(apps))
	appCurrent := make(map[int64]int64, len(apps))
	for _, a := range apps {
		if a.CurrentVersionID > 0 {
			appCurrent[a.ID] = a.CurrentVersionID
			versionIDs = append(versionIDs, a.CurrentVersionID)
		}
	}
	versions := make(map[int64]model.Version)
	if len(versionIDs) > 0 {
		var rows []model.Version
		s.DB.Where("id IN ?", versionIDs).Find(&rows)
		for _, v := range rows {
			versions[v.ID] = v
		}
	}
	out := make([]appSummary, 0, len(apps))
	for _, a := range apps {
		sum := appSummary{App: a}
		if id, ok := appCurrent[a.ID]; ok {
			if v, ok := versions[id]; ok {
				sum.LatestVersion = &v
			}
		}
		out = append(out, sum)
	}
	web.SendJson(w, out)
}

// AppDetail returns app detail. An unpublished app behaves as if it doesn't
// exist (404). The public share link is name-based (`/app/{name}`), so the
// path segment resolves by name first and falls back to the numeric id. Only
// the app's current version is exposed (secret fields hidden via json:"-").
func (s *Server) AppDetail(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("id")
	var app model.App
	err := s.DB.Where("name = ?", key).First(&app).Error
	if err != nil {
		if n, perr := strconv.ParseInt(key, 10, 64); perr == nil {
			err = s.DB.First(&app, n).Error
		}
	}
	if err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	if !app.Published {
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
	web.SendJson(w, map[string]any{
		"app":      app,
		"versions": versions,
	})
}

// checkAccess enforces that the version is publicly downloadable and then the
// app-level access permission. A version is public only when its app is
// published and the version is the app's current version. The password/expiry
// scope lives on the app, not per version.
func (s *Server) checkAccess(v *model.Version, pwd string) error {
	var app model.App
	if err := s.DB.First(&app, v.AppID).Error; err != nil {
		return &webErr{web.CodeNotFound, "应用不存在"}
	}
	if !app.Published {
		return &webErr{web.CodeNotFound, "应用不存在"}
	}
	if app.CurrentVersionID != v.ID {
		return &webErr{web.CodeForbidden, "该版本不可下载"}
	}
	switch app.AccessMode {
	case model.AccessPassword:
		if !password.Verify(pwd, app.PasswordHash, app.Salt) {
			return &webErr{web.CodeUnauthorized, "密码错误"}
		}
	case model.AccessExpiry:
		if app.ExpiresAt != nil && time.Now().After(*app.ExpiresAt) {
			return &webErr{web.CodeForbidden, "下载链接已过期"}
		}
	}
	return nil
}

// VerifyAccess checks access permission (password mode submits password).
func (s *Server) VerifyAccess(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad id")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return
	}
	if err := s.checkAccess(&v, req.Password); err != nil {
		we := err.(*webErr)
		web.SendError(w, we.code, we.msg)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// absURL prefixes an absolute scheme+host onto a Storage-returned path. The
// host comes from the request Host header (so reverse proxies must forward the
// real host), falling back to X-Forwarded-Host / Forwarded when present.
func absURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	if host == "" {
		return path
	}
	return scheme + "://" + host + path
}

// downloadURL returns an absolute download URL for the request host.
func (s *Server) downloadURL(r *http.Request, v *model.Version) (string, error) {
	pwd := r.URL.Query().Get("password")
	if err := s.checkAccess(v, pwd); err != nil {
		return "", err
	}
	rel, err := s.Storage.DownloadURL(r.Context(), v.StorageKey, v.FileName, 15*time.Minute)
	if err != nil {
		return "", err
	}
	return absURL(r, rel), nil
}

// Download returns download URL, increments download_count and logs.
func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
	v, urlStr, err := s.resolveAndURL(w, r)
	if err != nil {
		return
	}
	s.DB.Model(v).UpdateColumn("download_count", v.DownloadCount+1)
	s.DB.Create(&model.DownloadLog{
		VersionID: v.ID, IP: clientIP(r), UserAgent: r.UserAgent(),
	})
	web.SendJson(w, map[string]any{"url": urlStr})
}

// Install reports installation, increments install_count.
func (s *Server) Install(w http.ResponseWriter, r *http.Request) {
	v, urlStr, err := s.resolveAndURL(w, r)
	if err != nil {
		return
	}
	s.DB.Model(v).UpdateColumn("install_count", v.InstallCount+1)
	web.SendJson(w, map[string]any{"url": urlStr})
}

func (s *Server) resolveAndURL(w http.ResponseWriter, r *http.Request) (*model.Version, string, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad id")
		return nil, "", err
	}
	var v model.Version
	if err := s.DB.First(&v, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "版本不存在")
		return nil, "", err
	}
	urlStr, err := s.downloadURL(r, &v)
	if err != nil {
		we := err.(*webErr)
		web.SendError(w, we.code, we.msg)
		return nil, "", err
	}
	return &v, urlStr, nil
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}
