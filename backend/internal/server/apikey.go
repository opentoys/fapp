package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"disapp/internal/model"
	"disapp/internal/web"
)

const keyPrefix = "dk_"

// canManage reports whether userID may administer appID. The super-admin
// (uid = -1) manages everything; ordinary users must be an app member.
func (s *Server) canManage(userID, appID int64) bool {
	if userID == SuperAdminUID {
		return true
	}
	var n int64
	s.DB.Model(&model.AppMember{}).Where("user_id = ? AND app_id = ?", userID, appID).Count(&n)
	return n > 0
}

// manageableAppIDs returns the app ids userID can administer. Super-admin
// gets all apps; otherwise membership in app_members. The current user's own
// keys are always listed in KeysList regardless (visibility is by owner).
func (s *Server) manageableAppIDs(userID int64) []int64 {
	if userID == SuperAdminUID {
		var ids []int64
		s.DB.Model(&model.App{}).Pluck("id", &ids)
		return ids
	}
	var ids []int64
	s.DB.Model(&model.AppMember{}).Where("user_id = ?", userID).Pluck("app_id", &ids)
	return ids
}

// ManageableApps returns the apps the current user can manage (super-admin:
// all). Used by the key-scope hint in the key management UI.
func (s *Server) ManageableApps(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var apps []model.App
	if user.UserID == SuperAdminUID {
		s.DB.Order("id desc").Find(&apps)
	} else {
		s.DB.Where("id IN ?", s.manageableAppIDs(user.UserID)).Order("id desc").Find(&apps)
	}
	web.SendJson(w, apps)
}

// KeysList returns the caller's own keys; the super-admin sees all keys.
// Super-admin rows also carry owner_username (uid=-1 keys show the configured
// admin name) so the admin table can show the creator.
func (s *Server) KeysList(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var keys []model.ApiKey
	if user.UserID == SuperAdminUID {
		s.DB.Order("id desc").Find(&keys)
	} else {
		s.DB.Where("user_id = ?", user.UserID).Order("id desc").Find(&keys)
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		row := map[string]any{
			"id": k.ID, "name": k.Name, "key": k.Key, "user_id": k.UserID,
			"scope": k.Scope, "expires_at": k.ExpiresAt, "last_used_at": k.LastUsedAt,
			"created_at": k.CreatedAt,
		}
		if user.UserID == SuperAdminUID {
			owner := ""
			if k.UserID == SuperAdminUID {
				owner = s.Config.Admin.Username
			} else {
				var u model.User
				if err := s.DB.First(&u, k.UserID).Error; err == nil {
					owner = u.Username
				}
			}
			row["owner_username"] = owner
		}
		out = append(out, row)
	}
	web.SendJson(w, out)
}

// CreateKey creates a new API key for the authenticated user (super-admin
// keys are owned by uid = -1). The plain-text key is returned once; short
// of the DB there is no other place it is shown.
func (s *Server) CreateKey(w http.ResponseWriter, r *http.Request) {
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
	if req.Name == "" {
		web.SendError(w, web.CodeBadRequest, "key 名称不能为空")
		return
	}
	if req.Scope != model.KeyScopeRead && req.Scope != model.KeyScopeRun {
		web.SendError(w, web.CodeBadRequest, "scope 必须为 read 或 run")
		return
	}
	raw, err := newKey()
	if err != nil {
		web.SendError(w, web.CodeInternal, "生成 key 失败")
		return
	}
	key := model.ApiKey{
		Name:      req.Name,
		Key:       raw,
		UserID:    user.UserID,
		Scope:     req.Scope,
		ExpiresAt: normalizeExpiry(req.ExpiresAt),
	}
	if err := s.DB.Create(&key).Error; err != nil {
		web.SendError(w, web.CodeInternal, "创建失败")
		return
	}
	web.SendJson(w, key)
}

// UpdateKey updates name/scope/expires_at. The key plain-text is immutable;
// only the owner or the super-admin may edit.
func (s *Server) UpdateKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var key model.ApiKey
	if err := s.DB.First(&key, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "key 不存在")
		return
	}
	if user.UserID != SuperAdminUID && key.UserID != user.UserID {
		web.SendError(w, web.CodeForbidden, "无权操作该 key")
		return
	}
	var req struct {
		Name      *string          `json:"name"`
		Scope     *string          `json:"scope"`
		ExpiresAt json.RawMessage  `json:"expires_at"` // raw: absent→unchanged, "null"→clear, RFC3339→set
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	if req.Name != nil {
		if *req.Name == "" {
			web.SendError(w, web.CodeBadRequest, "key 名称不能为空")
			return
		}
		key.Name = *req.Name
	}
	if req.Scope != nil {
		if *req.Scope != model.KeyScopeRead && *req.Scope != model.KeyScopeRun {
			web.SendError(w, web.CodeBadRequest, "scope 必须为 read 或 run")
			return
		}
		key.Scope = *req.Scope
	}
	if len(req.ExpiresAt) > 0 && string(req.ExpiresAt) != "null" {
		var ts *time.Time
		if err := json.Unmarshal(req.ExpiresAt, &ts); err != nil {
			web.SendError(w, web.CodeBadRequest, "expires_at 格式错误")
			return
		}
		key.ExpiresAt = normalizeExpiry(ts)
	} else if len(req.ExpiresAt) > 0 { // explicit null → clear expiry
		key.ExpiresAt = nil
	}
	if err := s.DB.Save(&key).Error; err != nil {
		web.SendError(w, web.CodeInternal, "保存失败")
		return
	}
	web.SendJson(w, key)
}

// DeleteKey removes a key. Owner or super-admin only.
func (s *Server) DeleteKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	user := userFrom(r)
	if user == nil {
		web.SendError(w, web.CodeUnauthorized, "未登录")
		return
	}
	var key model.ApiKey
	if err := s.DB.First(&key, id).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "key 不存在")
		return
	}
	if user.UserID != SuperAdminUID && key.UserID != user.UserID {
		web.SendError(w, web.CodeForbidden, "无权操作该 key")
		return
	}
	if err := s.DB.Delete(&model.ApiKey{}, id).Error; err != nil {
		web.SendError(w, web.CodeInternal, "删除失败")
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// authorizeKey resolves the `apikey` query param to a live key. Returns a
// *webErr-style error via the http response path.
func (s *Server) authorizeKey(r *http.Request) (*model.ApiKey, error) {
	raw := r.URL.Query().Get("apikey")
	if raw == "" {
		return nil, &webErr{web.CodeUnauthorized, "缺少 apikey 参数"}
	}
	var key model.ApiKey
	if err := s.DB.Where("key = ?", raw).First(&key).Error; err != nil {
		return nil, &webErr{web.CodeUnauthorized, "apikey 无效"}
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, &webErr{web.CodeUnauthorized, "apikey 已过期"}
	}
	return &key, nil
}

// touchKey records the last call time of a key (best-effort).
func (s *Server) touchKey(id int64) {
	s.DB.Model(&model.ApiKey{}).Where("id = ?", id).Update("last_used_at", time.Now())
}

func newKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return keyPrefix + hex.EncodeToString(b), nil
}

// normalizeExpiry clamps a client-provided expiry to server-local wall clock,
// matching how UpdateApp stores app expiries.
func normalizeExpiry(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	t := at.In(time.Local)
	return &t
}

