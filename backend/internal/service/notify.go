package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"disapp/internal/resources/store/model"
)

// NotifyParams is the fixed set of common notification parameters available in
// every webhook template. Each key corresponds to a {{.key}} placeholder.
type NotifyParams map[string]string

// EventParams returns the parameters for a notification event of appName.
func EventParams(event, appName string) NotifyParams {
	p := NotifyParams{}
	p["event"] = eventName(event)
	p["event_key"] = event
	p["app_name"] = appName
	p["app_id"] = "" // set per-bot from the subscribed app id
	p["time"] = time.Now().In(time.Local).Format("2006-01-02 15:04:05")
	p["version_name"] = ""
	p["version_code"] = ""
	p["version_id"] = ""
	p["file_name"] = ""
	p["file_size"] = ""
	p["published"] = ""
	p["expires_at"] = ""
	return p
}

func eventName(key string) string {
	switch key {
	case model.EventVersionUploaded:
		return "版本上传"
	case model.EventVersionCurrent:
		return "版本设为当前"
	case model.EventAppPublish:
		return "应用发布/下架"
	case model.EventAppExpire:
		return "应用到期"
	}
	return key
}

// fillParams renders s with p using text/template. {{.key}} placeholders map
// to p["key"]; unknown keys and missing values render as empty
// (missingkey=zero). A malformed template is left untouched.
func fillParams(s string, p NotifyParams) string {
	tmpl, err := template.New("notify").Option("missingkey=zero").Parse(s)
	if err != nil {
		return s
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, p); err != nil {
		return s
	}
	return out.String()
}

// compileForBot merges the event params with the bot's own substitutions
// (app_id from the subscription).
func compileForBot(bot *model.NotificationBot, params NotifyParams) NotifyParams {
	out := make(NotifyParams, len(params)+1)
	for k, v := range params {
		out[k] = v
	}
	out["app_id"] = fmt.Sprintf("%d", bot.AppID)
	return out
}

// headermap parses the bot's ["K: V", ...] lines into an http.Header.
func headermap(headers []string) http.Header {
	h := make(http.Header)
	for _, line := range headers {
		if i := strings.Index(line, ":"); i > 0 {
			if key := strings.TrimSpace(line[:i]); key != "" {
				h.Set(key, strings.TrimSpace(line[i+1:]))
			}
		}
	}
	return h
}

// logNotification records the outgoing webhook request for admin debugging.
func (s *Service) logNotification(ctx context.Context, bot *model.NotificationBot, params NotifyParams, status int, errMsg string) {
	p := compileForBot(bot, params)
	row := model.NotificationLog{
		BotID:  bot.ID,
		AppID:  bot.AppID,
		Event:  params["event_key"],
		URL:    fillParams(bot.URL, p),
		Body:   fillParams(bot.BodyTemplate, p),
		Status: status,
		Error:  errMsg,
	}
	s.DB.WithContext(ctx).Create(&row)
}

// SendNotification compiles the bot template with the event params and calls
// the webhook. The request is logged; the outcome is signalled via the error.
func (s *Service) SendNotification(ctx context.Context, bot *model.NotificationBot, params NotifyParams) error {
	p := compileForBot(bot, params)
	url := fillParams(bot.URL, p)
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(bot.Method), url, bytes.NewReader([]byte(fillParams(bot.BodyTemplate, p))))
	if err != nil {
		s.logNotification(ctx, bot, params, 0, err.Error())
		return err
	}
	req.Header = headermap(bot.Headers)
	req.Header.Set("User-Agent", "disapp-notification")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logNotification(ctx, bot, params, 0, err.Error())
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	s.logNotification(ctx, bot, params, resp.StatusCode, "")
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook 返回 %d", resp.StatusCode)
	}
	return nil
}

// NotifyEvent fans out to every bot subscribed to the event for appID. Sends
// run in goroutines (fire-and-forget); failures land in NotificationLog.
func (s *Service) NotifyEvent(ctx context.Context, appID int64, event, appName string) {
	s.NotifyEventParams(ctx, appID, event, appName, nil)
}

// NotifyEventParams is NotifyEvent with extra params merged over the base ones.
func (s *Service) NotifyEventParams(ctx context.Context, appID int64, event, appName string, extra NotifyParams) {
	var bots []model.NotificationBot
	if err := s.DB.Where("app_id = ?", appID).Find(&bots).Error; err != nil {
		return
	}
	if len(bots) == 0 {
		return
	}
	if appName == "" {
		var a model.App
		if err := s.DB.Select("name").First(&a, appID).Error; err == nil {
			appName = a.Name
		}
	}
	params := EventParams(event, appName)
	for k, v := range extra {
		params[k] = v
	}
	for i := range bots {
		bot := bots[i]
		if bot.URL == "" || !hasEvent(bot.Events, event) {
			continue
		}
		b := bot
		go func() { s.SendNotification(context.Background(), &b, params) }()
	}
}

func hasEvent(list []string, name string) bool {
	for _, e := range list {
		if e == name {
			return true
		}
	}
	return false
}

// ---- CRUD ----

// BotInput carries the create/update fields for a notification bot.
type BotInput struct {
	Name         string   `json:"name"`
	AppID        int64    `json:"app_id"`
	Method       string   `json:"method"` // POST | GET | PUT
	URL          string   `json:"url"`
	Headers      []string `json:"headers"` // ["K: V", ...]
	BodyTemplate string   `json:"body_template"`
	Events       []string `json:"events"`
}

var validBotMethods = map[string]bool{"POST": true, "GET": true, "PUT": true}

func (s *Service) validateBot(in *BotInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return &Error{http.StatusBadRequest, "机器人名称不能为空"}
	}
	in.Method = strings.ToUpper(strings.TrimSpace(in.Method))
	if in.Method == "" {
		in.Method = http.MethodPost
	}
	if !validBotMethods[in.Method] {
		return &Error{http.StatusBadRequest, "请求方法必须为 POST/GET/PUT"}
	}
	in.URL = strings.TrimSpace(in.URL)
	if in.URL == "" || !strings.HasPrefix(in.URL, "http") {
		return &Error{http.StatusBadRequest, "请求 url 必须为 http(s) 地址"}
	}
	if len(in.Events) == 0 {
		return &Error{http.StatusBadRequest, "至少订阅一个事件"}
	}
	for _, e := range in.Events {
		switch e {
		case model.EventVersionUploaded, model.EventVersionCurrent, model.EventAppPublish, model.EventAppExpire:
		default:
			return &Error{http.StatusBadRequest, "未知事件: " + e}
		}
	}
	var app model.App
	if err := s.DB.Select("id").First(&app, in.AppID).Error; err != nil {
		return &Error{http.StatusNotFound, "应用不存在"}
	}
	return nil
}

// CreateBot stores a new webhook bot.
func (s *Service) CreateBot(ctx context.Context, in BotInput) (*model.NotificationBot, error) {
	if err := s.validateBot(&in); err != nil {
		return nil, err
	}
	bot := model.NotificationBot{
		Name: in.Name, AppID: in.AppID, Method: in.Method, URL: in.URL,
		Headers: in.Headers, BodyTemplate: in.BodyTemplate, Events: in.Events,
	}
	if err := s.DB.Create(&bot).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "创建失败"}
	}
	return &bot, nil
}

// UpdateBot replaces a bot's configuration.
func (s *Service) UpdateBot(ctx context.Context, id int64, in BotInput) (*model.NotificationBot, error) {
	var bot model.NotificationBot
	if err := s.DB.First(&bot, id).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "机器人不存在"}
	}
	if err := s.validateBot(&in); err != nil {
		return nil, err
	}
	bot.Name, bot.AppID = in.Name, in.AppID
	bot.Method, bot.URL = in.Method, in.URL
	bot.Headers, bot.BodyTemplate, bot.Events = in.Headers, in.BodyTemplate, in.Events
	if err := s.DB.Save(&bot).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "保存失败"}
	}
	return &bot, nil
}

// BotsList returns the bots on apps the caller can manage (super-admin: all).
func (s *Service) BotsList(ctx context.Context, userID int64) ([]model.NotificationBot, error) {
	q := s.DB.Order("id desc")
	if userID != SuperAdminUID {
		appIDs := s.manageableAppIDs(userID)
		if len(appIDs) == 0 {
			return []model.NotificationBot{}, nil
		}
		q = q.Where("app_id IN ?", appIDs)
	}
	var bots []model.NotificationBot
	if err := q.Find(&bots).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "查询失败"}
	}
	return bots, nil
}

// DeleteBot removes a bot and its notification history.
func (s *Service) DeleteBot(ctx context.Context, userID, id int64) error {
	var bot model.NotificationBot
	if err := s.DB.First(&bot, id).Error; err != nil {
		return &Error{http.StatusNotFound, "机器人不存在"}
	}
	if userID != SuperAdminUID && !s.CanManage(userID, bot.AppID) {
		return &Error{http.StatusForbidden, "无权操作该机器人"}
	}
	if err := s.DB.Delete(&model.NotificationBot{}, id).Error; err != nil {
		return &Error{http.StatusInternalServerError, "删除失败"}
	}
	s.DB.Where("bot_id = ?", id).Delete(&model.NotificationLog{})
	return nil
}

// TestBot fires a sample "版本上传" event to the bot so the admin can verify
// the webhook without waiting for a real event. Uses the saved app name.
func (s *Service) TestBot(ctx context.Context, id int64) error {
	var bot model.NotificationBot
	if err := s.DB.First(&bot, id).Error; err != nil {
		return &Error{http.StatusNotFound, "机器人不存在"}
	}
	var app model.App
	if err := s.DB.Select("name").First(&app, bot.AppID).Error; err != nil {
		return &Error{http.StatusNotFound, "应用不存在"}
	}
	params := EventParams(model.EventVersionUploaded, app.Name)
	params["version_name"] = "（测试）"
	return s.SendNotification(ctx, &bot, params)
}

// TestBotInput fires a sample "版本上传" event to an unsaved bot configuration
// so the admin can verify the webhook before creating or updating it.
func (s *Service) TestBotInput(ctx context.Context, in BotInput) error {
	if err := s.validateBot(&in); err != nil {
		return err
	}
	var app model.App
	if err := s.DB.Select("name").First(&app, in.AppID).Error; err != nil {
		return &Error{http.StatusNotFound, "应用不存在"}
	}
	bot := model.NotificationBot{
		AppID: in.AppID, Method: in.Method, URL: in.URL,
		Headers: in.Headers, BodyTemplate: in.BodyTemplate, Events: in.Events,
	}
	params := EventParams(model.EventVersionUploaded, app.Name)
	params["version_name"] = "（测试）"
	return s.SendNotification(ctx, &bot, params)
}

// BotLogs returns the latest notification attempts for a bot (newest first).
func (s *Service) BotLogs(ctx context.Context, botID, limit int64) ([]model.NotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []model.NotificationLog
	if err := s.DB.Where("bot_id = ?", botID).Order("id desc").Limit(int(limit)).Find(&rows).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "查询失败"}
	}
	return rows, nil
}

// ---- App-expired background scan ----

// ExpiryScan fires EventAppExpire the first time an app passes its expiry. The
// dedupe is the persisted NotificationLog table, so restarts don't re-fire.
// Called from a background ticker; best-effort.
func (s *Service) ExpiryScan(ctx context.Context) {
	now := time.Now()
	var apps []model.App
	s.DB.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at <= ?", now).Find(&apps)
	for _, a := range apps {
		var n int64
		s.DB.WithContext(ctx).Model(&model.NotificationLog{}).
			Where("app_id = ? AND event = ?", a.ID, model.EventAppExpire).Count(&n)
		if n > 0 {
			continue
		}
		s.NotifyEvent(ctx, a.ID, model.EventAppExpire, a.Name)
	}
}
