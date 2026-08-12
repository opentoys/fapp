package model

import "time"

const (
	AccessPublic   = "public"
	AccessPassword = "password"
	AccessExpiry   = "expiry"
)

type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type App struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128" json:"name"`
	Icon        string    `gorm:"size:512" json:"icon"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Channel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	AppID     int64     `gorm:"index" json:"app_id"`
	Name      string    `gorm:"size:64" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Version struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AppID          int64      `gorm:"index" json:"app_id"`
	ChannelID      int64      `gorm:"index" json:"channel_id"`
	VersionName    string     `gorm:"size:64" json:"version_name"`
	VersionCode    int        `json:"version_code"`
	FileType       string     `gorm:"size:16" json:"file_type"`
	FileName       string     `gorm:"size:256" json:"file_name"`
	FileSize       int64      `json:"file_size"`
	StorageKey     string     `gorm:"size:512" json:"-"`
	StorageBackend string     `gorm:"size:16" json:"-"`
	SHA256         string     `gorm:"size:64" json:"sha256"`
	Changelog      string     `gorm:"type:text" json:"changelog"`
	AccessMode     string     `gorm:"size:16" json:"access_mode"`
	PasswordHash   string     `json:"-"`
	Salt           string     `json:"-"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Enabled        bool       `gorm:"default:true" json:"enabled"`
	DownloadCount  int64      `json:"download_count"`
	InstallCount   int64      `json:"install_count"`
	CreatedAt      time.Time  `json:"created_at"`

	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
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
