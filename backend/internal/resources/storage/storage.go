package storage

import (
	"context"
	"regexp"
	"strings"
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

// AppKey generates a storage key with a human-readable app-name folder as
// the top segment: {app_name}/{app_id}/{version_id}/{filename}.
func AppKey(name string, appID, versionID int64, filename string) string {
	return SlugName(name) + "/" + itoa(appID) + "/" + itoa(versionID) + "/" + filename
}

// SlugName sanitizes an app name to a safe folder segment: slash and spaces
// become dashes, leading/trailing dots and dashes are trimmed. Empty falls
// back to "app". Unicode (e.g. CJK) is preserved.
func SlugName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "app"
	}
	return s
}

// Only allow {segment}/number/number/filename, all without slashes or spaces,
// preventing directory traversal. Dots-only segments are rejected explicitly.
var keyRe = regexp.MustCompile(`^[^/ ]+/[0-9]+/[0-9]+/[^/ ]+$`)

func ValidKey(k string) bool {
	if !keyRe.MatchString(k) {
		return false
	}
	for _, seg := range strings.Split(k, "/") {
		if seg == "." || seg == ".." {
			return false
		}
	}
	return true
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
