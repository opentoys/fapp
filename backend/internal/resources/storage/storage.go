package storage

import (
	"context"
	"io"
	"regexp"
	"time"
)

// Storage is the file storage abstraction.
type Storage interface {
	Save(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	DownloadURL(ctx context.Context, key, filename string, expire time.Duration) (string, error)
}

// Key generates storage key: {app_id}/{version_id}/{filename}.
func Key(appID, versionID int64, filename string) string {
	return itoa(appID) + "/" + itoa(versionID) + "/" + filename
}

// Only allow number/number/filename-without-slashes, preventing directory traversal.
var keyRe = regexp.MustCompile(`^[0-9]+/[0-9]+/[^/]+$`)

func ValidKey(k string) bool {
	return keyRe.MatchString(k)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
