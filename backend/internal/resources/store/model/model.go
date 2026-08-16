package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

const (
	AccessPublic   = "public"
	AccessPassword = "password"
)

type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// JSONList is a []string stored as a JSON text column. It serializes as a
// JSON array in API responses.
type JSONList []string

func (l JSONList) Value() (driver.Value, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *JSONList) Scan(v any) error {
	switch t := v.(type) {
	case []byte:
		*l = nil
		if len(t) > 0 {
			return json.Unmarshal(t, l)
		}
	case string:
		*l = nil
		if t != "" {
			return json.Unmarshal([]byte(t), l)
		}
	default:
		*l = nil
	}
	return nil
}

// MarshalJSON emits an empty list (not null) so the frontend can always treat
// screenshots as an array.
func (l JSONList) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(l))
}

// App carries its own access permission (public/password/expiry), applied to
// every version on download. Screenshots are URLs of uploaded images.
type App struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"size:128" json:"name"`
	Platform         string     `gorm:"size:16;uniqueIndex:idx_app_platform_package" json:"platform"`    // ios / android，创建时确定，不可修改
	PackageName      *string    `gorm:"size:128;uniqueIndex:idx_app_platform_package" json:"appid"` // Android package / iOS bundle id（appid），同一平台下唯一；nil 表示未解析/手动创建，SQLite 视 NULL 为互不相同
	Icon             string     `gorm:"size:512" json:"icon"`
	Description      string     `gorm:"type:text" json:"description"`
	Screenshots      JSONList   `gorm:"type:text" json:"screenshots"`
	AccessMode       string     `gorm:"size:16" json:"access_mode"`
	PasswordHash     string     `json:"-"`
	Salt             string     `json:"-"`
	ExpiresAt        *time.Time `json:"expires_at"`
	Published        bool       `gorm:"default:false" json:"published"` // 应用级上下架：未上架时公开页返回"应用不存在"
	CurrentVersionID int64      `json:"current_version_id"`             // 当前版本 id，0 = 未设置；仅该版本对外可见
	CreatedAt        time.Time  `json:"created_at"`
}

// ReleaseType values.
const (
	ReleaseProduction = "production"
	ReleaseBeta       = "beta"
	ReleaseCanary     = "canary"
)

type Version struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AppID          int64      `gorm:"index" json:"app_id"`
	ReleaseType    string     `gorm:"size:16" json:"release_type"`
	Platform       string     `gorm:"size:16" json:"platform"`
	Arch           string     `gorm:"size:64" json:"arch"` // 逗号分隔的多架构，如 "arm64,armv7,x86"
	VersionName    string     `gorm:"size:64" json:"version_name"`
	VersionCode    int        `json:"version_code"`
	FileType       string     `gorm:"size:16" json:"file_type"`
	FileName       string     `gorm:"size:256" json:"file_name"`
	FileSize       int64      `json:"file_size"`
	PackageName    string     `gorm:"size:128" json:"appid"` // Android package / iOS bundle id（解析所得）
	AppName        string     `gorm:"size:128" json:"app_name"`     // 解析出的应用名称
	IconURL        string     `gorm:"size:512" json:"icon_url"`     // 解析出的图标 URL（仅 Android）
	StorageKey     string     `gorm:"size:512" json:"-"`
	StorageBackend string     `gorm:"size:16" json:"-"`
	SHA256         string     `gorm:"size:64" json:"sha256"`
	Changelog      string     `gorm:"type:text" json:"changelog"`
	DownloadCount  int64      `json:"download_count"`
	InstallCount   int64      `json:"install_count"`
	CreatedAt      time.Time  `json:"created_at"`
}

// AppMember links a user to an app they can manage (admin side). Only members
// may operate on the app once admin endpoints are permission-gated.
type AppMember struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"index:idx_member_user_app,unique" json:"user_id"`
	AppID     int64     `gorm:"index:idx_member_user_app,unique" json:"app_id"`
	CreatedAt time.Time `json:"created_at"`
}

// API key scopes.
const (
	KeyScopeRead = "read"
	KeyScopeRun  = "run"
)

// ApiKey authenticates programmatic access (query param `apikey`). It is not
// bound to specific apps: at call time its scope is "the creator's currently
// manageable apps" (super-admin can manage everything). Plain text storage was
// a deliberate product decision (simpler key lifecycle; drop + recreate if
// leaked).
type ApiKey struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:128" json:"name"`
	Key        string     `gorm:"uniqueIndex;size:128" json:"key"`
	UserID     int64      `gorm:"index" json:"user_id"` // 创建人 uid；超管 = -1
	Scope      string     `gorm:"size:16" json:"scope"` // read | run
	ExpiresAt  *time.Time `json:"expires_at"`           // nil = 永不过期
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type DownloadLog struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	VersionID int64     `gorm:"index" json:"version_id"`
	IP        string    `gorm:"size:64" json:"ip"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// FileType returns the file type based on filename extension.
func FileType(filename string) string {
	switch {
	case hasSuffix(filename, ".apk"):
		return "apk"
	case hasSuffix(filename, ".aab"):
		return "aab"
	case hasSuffix(filename, ".ipa"):
		return "ipa"
	case hasSuffix(filename, ".exe"):
		return "exe"
	case hasSuffix(filename, ".dmg"):
		return "dmg"
	default:
		return "other"
	}
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
