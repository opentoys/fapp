package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"disapp/pkg/token"
	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
	"disapp/pkg/web"
)

// SuperAdminUID is the user id assigned to the super-admin in JWT claims.
// The super-admin lives in config.json and is never stored in the users
// table; any audit-style "operator" field that picks up the JWT uid will
// see -1 for actions performed by the super-admin.
const SuperAdminUID int64 = -1

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}

	// Super-admin: authenticated against config, never against the DB.
	// The issued token carries SuperAdminUID so downstream code can
	// attribute actions to the super-admin without needing a DB row.
	if s.Config.Admin.Username != "" && req.Username == s.Config.Admin.Username {
		if req.Password != s.Config.Admin.Password {
			web.SendError(w, web.CodeUnauthorized, "用户名或密码错误")
			return
		}
		token, err := token.CreateToken(s.Config.JWT.Secret, SuperAdminUID, s.Config.Admin.Username, s.Config.JWTExpire())
		if err != nil {
			web.SendError(w, web.CodeInternal, "生成 token 失败")
			return
		}
		web.SendJson(w, map[string]any{"token": token})
		return
	}

	// Fallback: ordinary users in the users table (no admin endpoints
	// create them today; kept for future account-management features).
	var u model.User
	if err := s.DB.Where("username = ?", req.Username).First(&u).Error; err != nil {
		web.SendError(w, web.CodeUnauthorized, "用户名或密码错误")
		return
	}
	if !pwd.Verify(req.Password, u.PasswordHash, u.Salt) {
		web.SendError(w, web.CodeUnauthorized, "用户名或密码错误")
		return
	}
	token, err := token.CreateToken(s.Config.JWT.Secret, u.ID, u.Username, s.Config.JWTExpire())
	if err != nil {
		web.SendError(w, web.CodeInternal, "生成 token 失败")
		return
	}
	web.SendJson(w, map[string]any{"token": token})
}

// ChangePassword lets the authenticated user change their own pwd.
// Super-admin (uid=-1) is rejected — their password is managed in config.json.
func (s *Server) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	if user.UserID == SuperAdminUID {
		web.SendError(w, web.CodeBadRequest, "超管密码请在 config.json 中修改")
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
	if req.OldPassword == "" || req.NewPassword == "" {
		web.SendError(w, web.CodeBadRequest, "密码不能为空")
		return
	}
	var u model.User
	if err := s.DB.First(&u, user.UserID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "用户不存在")
		return
	}
	if !pwd.Verify(req.OldPassword, u.PasswordHash, u.Salt) {
		web.SendError(w, web.CodeUnauthorized, "原密码错误")
		return
	}
	hash, salt := pwd.Hash(req.NewPassword)
	u.PasswordHash, u.Salt = hash, salt
	if err := s.DB.Save(&u).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// RequireAuth validates Bearer JWT middleware.
func (s *Server) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := token.ParseToken(s.Config.JWT.Secret, raw)
		if err != nil {
			web.SendStatus(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		r = r.WithContext(withUser(r.Context(), claims))
		next(w, r)
	}
}
