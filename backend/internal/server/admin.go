package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"disapp/internal/model"
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
