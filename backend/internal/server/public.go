package server

import (
	"net/http"
	"strconv"

	"disapp/internal/model"
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
		s.DB.Where("app_id IN ? AND enabled = ?", ids, true).Order("version_code desc").Find(&versions)
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
	var channels []model.Channel
	s.DB.Where("app_id = ?", id).Order("id asc").Find(&channels)
	var versions []model.Version
	s.DB.Where("app_id = ? AND enabled = ?", id, true).
		Order("version_code desc").Preload("Channel").Find(&versions)

	web.SendJson(w, map[string]any{
		"app":      app,
		"channels": channels,
		"versions": versions,
	})
}
