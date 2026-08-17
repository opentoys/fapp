package service

import (
	"context"
	"net/http"
	"time"

	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
	"disapp/pkg/token"
)

// SuperAdminUID is the user id assigned to the super-admin in JWT claims.
// The super-admin lives in config.json and is never stored in the users table.
const SuperAdminUID int64 = -1

// Login authenticates a user. The super-admin is checked against config; any
// other user against the users table. Returns a JWT.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	if s.Config.Admin.Username != "" && username == s.Config.Admin.Username {
		if password != s.Config.Admin.Password {
			return "", &Error{http.StatusUnauthorized, "用户名或密码错误"}
		}
		t, err := token.CreateToken(s.Config.JWT.Secret, SuperAdminUID, s.Config.Admin.Username, s.Config.JWTExpire())
		if err != nil {
			return "", &Error{http.StatusInternalServerError, "生成 token 失败"}
		}
		return t, nil
	}
	var u model.User
	if err := s.DB.Where("username = ?", username).First(&u).Error; err != nil {
		return "", &Error{http.StatusUnauthorized, "用户名或密码错误"}
	}
	if !pwd.Verify(password, u.PasswordHash, u.Salt) {
		return "", &Error{http.StatusUnauthorized, "用户名或密码错误"}
	}
	t, err := token.CreateToken(s.Config.JWT.Secret, u.ID, u.Username, s.Config.JWTExpire())
	if err != nil {
		return "", &Error{http.StatusInternalServerError, "生成 token 失败"}
	}
	return t, nil
}

// ParseToken validates a JWT using the configured secret.
func (s *Service) ParseToken(raw string) (*token.Claims, error) {
	return token.ParseToken(s.Config.JWT.Secret, raw)
}

// ChangePassword resets the authenticated user's own password. Super-admin is
// rejected (managed in config.json).
func (s *Service) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	if userID == SuperAdminUID {
		return &Error{http.StatusBadRequest, "超管密码请在 config.json 中修改"}
	}
	if oldPassword == "" || newPassword == "" {
		return &Error{http.StatusBadRequest, "密码不能为空"}
	}
	var u model.User
	if err := s.DB.First(&u, userID).Error; err != nil {
		return &Error{http.StatusNotFound, "用户不存在"}
	}
	if !pwd.Verify(oldPassword, u.PasswordHash, u.Salt) {
		return &Error{http.StatusUnauthorized, "原密码错误"}
	}
	hash, salt := pwd.Hash(newPassword)
	u.PasswordHash, u.Salt = hash, salt
	if err := s.DB.Save(&u).Error; err != nil {
		return &Error{http.StatusInternalServerError, "保存失败"}
	}
	return nil
}

func (s *Service) Users(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := s.DB.Order("id asc").Find(&users).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "查询失败"}
	}
	return users, nil
}

func (s *Service) CreateUser(ctx context.Context, username, password string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, &Error{http.StatusBadRequest, "用户名和密码不能为空"}
	}
	if s.Config.Admin.Username != "" && username == s.Config.Admin.Username {
		return nil, &Error{http.StatusBadRequest, "该用户名为超管保留"}
	}
	hash, salt := pwd.Hash(password)
	u := model.User{Username: username, PasswordHash: hash, Salt: salt}
	if err := s.DB.Create(&u).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "创建失败"}
	}
	return &u, nil
}

func (s *Service) UpdateUser(ctx context.Context, id int64, username, password *string) (*model.User, error) {
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "用户不存在"}
	}
	if username != nil && *username != "" {
		u.Username = *username
	}
	if password != nil && *password != "" {
		hash, salt := pwd.Hash(*password)
		u.PasswordHash, u.Salt = hash, salt
	}
	if err := s.DB.Save(&u).Error; err != nil {
		return nil, &Error{http.StatusInternalServerError, "保存失败"}
	}
	return &u, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		return &Error{http.StatusNotFound, "用户不存在"}
	}
	if s.Config.Admin.Username != "" && u.Username == s.Config.Admin.Username {
		return &Error{http.StatusBadRequest, "不能删除超管账号"}
	}
	if err := s.DB.Delete(&model.User{}, id).Error; err != nil {
		return &Error{http.StatusInternalServerError, "删除失败"}
	}
	return nil
}

// normalizeExpiry clamps a client-provided expiry to server-local wall clock.
func normalizeExpiry(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	t := at.In(time.Local)
	return &t
}
