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

// Apps returns app list with latest enabled version summary.
func (s *Server) Apps(w http.ResponseWriter, r *http.Request) {
	var apps []model.App
	if err := s.DB.Order("id desc").Find(&apps).Error; err != nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}
	ids := make([]int64, 0, len(apps))
	for _, a := range apps {
		ids = append(ids, a.ID)
	}
	var versions []model.Version
	if len(ids) > 0 {
		s.DB.Where("app_id IN ? AND enabled = ? AND published = ?", ids, true, true).Order("version_code desc").Find(&versions)
	}
	out := make([]appSummary, 0, len(apps))
	for _, a := range apps {
		sum := appSummary{App: a}
		for _, v := range versions {
			if v.AppID == a.ID {
				sum.LatestVersion = &v
				break
			}
		}
		out = append(out, sum)
	}
	web.SendJson(w, out)
}

// AppDetail returns app detail: channels + enabled versions (secret fields hidden via json:"-").
func (s *Server) AppDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad id")
		return
	}
	var app model.App
	if err := s.DB.First(&app, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}
	var versions []model.Version
	s.DB.Where("app_id = ? AND enabled = ? AND published = ?", id, true, true).
		Order("version_code desc").Find(&versions)

	web.SendJson(w, map[string]any{
		"app":      app,
		"versions": versions,
	})
}

func (s *Server) checkAccess(v *model.Version, pwd string) error {
	if !v.Published {
		return &webErr{web.CodeForbidden, "该版本未上架"}
	}
	if !v.Enabled {
		return &webErr{web.CodeForbidden, "该版本已下架"}
	}
	switch v.AccessMode {
	case model.AccessPassword:
		if !password.Verify(pwd, v.PasswordHash, v.Salt) {
			return &webErr{web.CodeUnauthorized, "密码错误"}
		}
	case model.AccessExpiry:
		if v.ExpiresAt != nil && time.Now().After(*v.ExpiresAt) {
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

func (s *Server) downloadURL(r *http.Request, v *model.Version) (string, error) {
	pwd := r.URL.Query().Get("password")
	if err := s.checkAccess(v, pwd); err != nil {
		return "", err
	}
	return s.Storage.DownloadURL(r.Context(), v.StorageKey, v.FileName, 15*time.Minute)
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
