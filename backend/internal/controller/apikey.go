package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"disapp/internal/service"
	"disapp/pkg/web"
)

// ManageableApps returns the apps the current user can manage (super-admin:
// all). Used by the key-scope hint in the key management UI.
func (c *Controller) ManageableApps(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	apps, err := c.SVC.ManageableApps(r.Context(), user.UserID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, apps)
}

// KeysList returns the caller's own keys; the super-admin sees all keys
// (super-admin rows also carry owner_username).
func (c *Controller) KeysList(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	keys, err := c.SVC.KeysList(r.Context(), user.UserID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, keys)
}

// CreateKey creates a new API key for the authenticated user. The plain-text
// key is returned once.
func (c *Controller) CreateKey(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var req struct {
		Name      string     `json:"name"`
		Scope     string     `json:"scope"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	key, err := c.SVC.CreateKey(r.Context(), user.UserID, req.Name, req.Scope, req.ExpiresAt)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, key)
}

// UpdateKey updates name/scope/expires_at. The key plain-text is immutable;
// only the owner or the super-admin may edit.
func (c *Controller) UpdateKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	// expires_at is raw: absent→unchanged, explicit null→clear, RFC3339→set.
	var req struct {
		Name      *string         `json:"name"`
		Scope     *string         `json:"scope"`
		ExpiresAt json.RawMessage `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	in := service.UpdateKeyInput{Name: req.Name, Scope: req.Scope}
	if len(req.ExpiresAt) > 0 {
		if string(req.ExpiresAt) == "null" {
			in.ClearExpiry = true
		} else {
			var ts *time.Time
			if err := json.Unmarshal(req.ExpiresAt, &ts); err != nil {
				web.SendError(w, web.CodeBadRequest, "expires_at 格式错误")
				return
			}
			in.ExpiresAt = ts
		}
	}
	key, err := c.SVC.UpdateKey(r.Context(), user.UserID, id, in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, key)
}

// DeleteKey removes a key. Owner or super-admin only.
func (c *Controller) DeleteKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	if err := c.SVC.DeleteKey(r.Context(), user.UserID, id); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}