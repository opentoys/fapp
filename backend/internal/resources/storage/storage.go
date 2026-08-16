package storage

import (
	"context"
	"regexp"
	"time"
)

// Storage is the file storage abstraction. All access goes through URLs:
// uploads are pushed directly to UploadURL, downloads read via DownloadURL.
type Storage interface {
	// UploadURL returns a URL the client can push bytes to directly. COS: a
	// presigned PUT URL. local: a server upload endpoint that writes the
	// request body under the given key.
	UploadURL(ctx context.Context, key, contentType string, expire time.Duration) (string, error)
	// Delete removes the object at key.
	Delete(ctx context.Context, key string) error
	// DownloadURL returns a signed URL to read the object at key. local
	// returns a signed /api/v1/files/preview URL that streams the file.
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
