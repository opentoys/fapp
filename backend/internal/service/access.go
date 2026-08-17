package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"disapp/internal/resources/store/model"
)

// KeyPrefix is prepended to every generated API key.
const KeyPrefix = "dk_"

// CanManage reports whether userID may administer appID. Super-admin manages
// everything; ordinary users must be an app member.
func (s *Service) CanManage(userID, appID int64) bool {
	if userID == SuperAdminUID {
		return true
	}
	var n int64
	s.DB.Model(&model.AppMember{}).Where("user_id = ? AND app_id = ?", userID, appID).Count(&n)
	return n > 0
}

// manageableAppIDs returns the app ids userID can administer.
func (s *Service) manageableAppIDs(userID int64) []int64 {
	if userID == SuperAdminUID {
		var ids []int64
		s.DB.Model(&model.App{}).Pluck("id", &ids)
		return ids
	}
	var ids []int64
	s.DB.Model(&model.AppMember{}).Where("user_id = ?", userID).Pluck("app_id", &ids)
	return ids
}

// ManageableApps returns the apps the current user can manage (super-admin: all).
func (s *Service) ManageableApps(ctx context.Context, userID int64) ([]model.App, error) {
	var apps []model.App
	if userID == SuperAdminUID {
		s.DB.Order("id desc").Find(&apps)
	} else {
		s.DB.Where("id IN ?", s.manageableAppIDs(userID)).Order("id desc").Find(&apps)
	}
	return apps, nil
}

// KeysList returns the caller's own keys; the super-admin sees all keys and
// their owner usernames.
func (s *Service) KeysList(ctx context.Context, userID int64) ([]map[string]any, error) {
	var keys []model.ApiKey
	if userID == SuperAdminUID {
		s.DB.Order("id desc").Find(&keys)
	} else {
		s.DB.Where("user_id = ?", userID).Order("id desc").Find(&keys)
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		row := map[string]any{
			"id": k.ID, "name": k.Name, "key": k.Key, "user_id": k.UserID,
			"scope": k.Scope, "expires_at": k.ExpiresAt, "last_used_at": k.LastUsedAt,
			"created_at": k.CreatedAt,
		}
		if userID == SuperAdminUID {
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
	return out, nil
}

// CreateKey creates a key for the authenticated user.
func (s *Service) CreateKey(ctx context.Context, userID int64, name, scope string, expiresAt *time.Time) (*model.ApiKey, error) {
	if name == "" {
		return nil, &Error{http.StatusBadRequest, "key 名称不能为空"}
	}
	if scope != model.KeyScopeRead && scope != model.KeyScopeRun {
		return nil, &Error{http.StatusBadRequest, "scope 必须为 read 或 run"}
	}
	raw, err := newKey()
	if err != nil {
		return nil, &Error{http.StatusInternalServerError, "生成 key 失败"}
	}
	key := model.ApiKey{
		Name:      name,
		Key:       raw,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: normalizeExpiry(expiresAt),
	}
	if err := s.DB.Create(&key).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "创建失败"}
	}
	return &key, nil
}

// UpdateKeyInput carries the PATCH fields for a key.
type UpdateKeyInput struct {
	Name        *string
	Scope       *string
	ExpiresAt   *time.Time
	ClearExpiry bool
}

// UpdateKey edits name/scope/expires_at. The plain-text key is immutable.
func (s *Service) UpdateKey(ctx context.Context, userID, id int64, in UpdateKeyInput) (*model.ApiKey, error) {
	var key model.ApiKey
	if err := s.DB.First(&key, id).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "key 不存在"}
	}
	if userID != SuperAdminUID && key.UserID != userID {
		return nil, &Error{http.StatusForbidden, "无权操作该 key"}
	}
	if in.Name != nil {
		if *in.Name == "" {
			return nil, &Error{http.StatusBadRequest, "key 名称不能为空"}
		}
		key.Name = *in.Name
	}
	if in.Scope != nil {
		if *in.Scope != model.KeyScopeRead && *in.Scope != model.KeyScopeRun {
			return nil, &Error{http.StatusBadRequest, "scope 必须为 read 或 run"}
		}
		key.Scope = *in.Scope
	}
	if in.ClearExpiry {
		key.ExpiresAt = nil
	} else if in.ExpiresAt != nil {
		key.ExpiresAt = normalizeExpiry(in.ExpiresAt)
	}
	if err := s.DB.Save(&key).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "保存失败"}
	}
	return &key, nil
}

// DeleteKey removes a key; owner or super-admin only.
func (s *Service) DeleteKey(ctx context.Context, userID, id int64) error {
	var key model.ApiKey
	if err := s.DB.First(&key, id).Error; err != nil {
		return &Error{http.StatusNotFound, "key 不存在"}
	}
	if userID != SuperAdminUID && key.UserID != userID {
		return &Error{http.StatusForbidden, "无权操作该 key"}
	}
	if err := s.DB.Delete(&model.ApiKey{}, id).Error; err != nil {
		return &Error{http.StatusInternalServerError, "删除失败"}
	}
	return nil
}

// AuthorizeKey resolves a raw `apikey` query value to a live (unexpired) key.
func (s *Service) AuthorizeKey(ctx context.Context, raw string) (*model.ApiKey, error) {
	if raw == "" {
		return nil, &Error{http.StatusUnauthorized, "缺少 apikey 参数"}
	}
	var key model.ApiKey
	if err := s.DB.Where("key = ?", raw).First(&key).Error; err != nil {
		return nil, &Error{http.StatusUnauthorized, "apikey 无效"}
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, &Error{http.StatusUnauthorized, "apikey 已过期"}
	}
	return &key, nil
}

// AuthorizeKeyApp authenticates a key against an app the key's owner can
// manage, enforcing the minimum scope (requireRun).
func (s *Service) AuthorizeKeyApp(ctx context.Context, raw string, appID int64, requireRun bool) (*model.ApiKey, error) {
	key, err := s.AuthorizeKey(ctx, raw)
	if err != nil {
		return nil, err
	}
	if !s.CanManage(key.UserID, appID) {
		return nil, &Error{http.StatusForbidden, "无权访问该应用"}
	}
	if requireRun && key.Scope != model.KeyScopeRun {
		return nil, &Error{http.StatusForbidden, "该 key 需要 run 权限"}
	}
	return key, nil
}

// TouchKey records the last call time of a key (best-effort).
func (s *Service) TouchKey(ctx context.Context, id int64) {
	s.DB.Model(&model.ApiKey{}).Where("id = ?", id).Update("last_used_at", time.Now())
}

func newKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return KeyPrefix + hex.EncodeToString(b), nil
}
