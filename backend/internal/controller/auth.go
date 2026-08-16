package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"disapp/pkg/web"
)

// Login authenticates a user and returns a JWT.
func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	tok, err := c.SVC.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"token": tok})
}

// ChangePassword lets the authenticated user change their own password.
// Super-admin is rejected (managed in config.json).
func (c *Controller) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if err := c.SVC.ChangePassword(r.Context(), user.UserID, req.OldPassword, req.NewPassword); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// RequireAuth validates the JWT from the Authorization header or, failing
// that, the token query param (for non-JS resource fetches like <img>).
func (c *Controller) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			raw = r.URL.Query().Get("token")
		}
		claims, err := c.SVC.ParseToken(raw)
		if err != nil {
			web.SendStatus(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		r = r.WithContext(withUser(r.Context(), claims))
		next(w, r)
	}
}