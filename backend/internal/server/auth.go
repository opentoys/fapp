package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"disapp/internal/auth"
	"disapp/internal/model"
	"disapp/internal/password"
	"disapp/internal/web"
)

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	var u model.User
	if err := s.DB.Where("username = ?", req.Username).First(&u).Error; err != nil {
		web.SendError(w, web.CodeUnauthorized, "用户名或密码错误")
		return
	}
	if !password.Verify(req.Password, u.PasswordHash, u.Salt) {
		web.SendError(w, web.CodeUnauthorized, "用户名或密码错误")
		return
	}
	token, err := auth.CreateToken(s.Config.JWT.Secret, u.ID, u.Username, s.Config.JWTExpire())
	if err != nil {
		web.SendError(w, web.CodeInternal, "生成 token 失败")
		return
	}
	web.SendJson(w, map[string]any{"token": token})
}

// RequireAuth validates Bearer JWT middleware.
func (s *Server) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := auth.ParseToken(s.Config.JWT.Secret, raw)
		if err != nil {
			web.SendStatus(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		r = r.WithContext(withUser(r.Context(), claims))
		next(w, r)
	}
}
