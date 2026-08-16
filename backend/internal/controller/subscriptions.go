package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"disapp/internal/service"
	"disapp/pkg/web"
)

// SubscriptionsList returns the caller's visible webhook bots.
func (c *Controller) SubscriptionsList(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	bots, err := c.SVC.BotsList(r.Context(), user.UserID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, bots)
}

// CreateSubscription stores a new webhook bot.
func (c *Controller) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var in service.BotInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if user.UserID != service.SuperAdminUID && !c.SVC.CanManage(user.UserID, in.AppID) {
		web.SendError(w, web.CodeForbidden, "无权访问该应用")
		return
	}
	bot, err := c.SVC.CreateBot(r.Context(), in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, bot)
}

// UpdateSubscription replaces a webhook bot's configuration.
func (c *Controller) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in service.BotInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if user.UserID != service.SuperAdminUID && !c.SVC.CanManage(user.UserID, in.AppID) {
		web.SendError(w, web.CodeForbidden, "无权访问该应用")
		return
	}
	bot, err := c.SVC.UpdateBot(r.Context(), id, in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, bot)
}

// DeleteSubscription removes a webhook bot.
func (c *Controller) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := c.SVC.DeleteBot(r.Context(), user.UserID, id); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// TestSubscription fires a sample event to a bot so the admin can verify the
// webhook without waiting for a real event.
func (c *Controller) TestSubscription(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := c.SVC.TestBot(r.Context(), id, requestBase(r)); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// TestSubscriptionConfig fires a sample event to an unsaved bot configuration
// (the create/edit dialog tests its current form before saving).
func (c *Controller) TestSubscriptionConfig(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var in service.BotInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if user.UserID != service.SuperAdminUID && !c.SVC.CanManage(user.UserID, in.AppID) {
		web.SendError(w, web.CodeForbidden, "无权访问该应用")
		return
	}
	if err := c.SVC.TestBotInput(r.Context(), in, requestBase(r)); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// SubscriptionLogs returns recent webhook attempts for a bot (admin debug).
func (c *Controller) SubscriptionLogs(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	rows, err := c.SVC.BotLogs(r.Context(), id, limit)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, rows)
}