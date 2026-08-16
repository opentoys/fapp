package local

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// signUpload signs an upload ticket: hash(secret, ttl). The upload key is
// carried separately by the client; the signature binds only the expiry.
func signUpload(secret string, ttl int64) string {
	return macHex(secret, []byte(fmt.Sprintf("%d", ttl)))
}

// signPreview signs a preview ticket: hash(secret, key + ttl).
func signPreview(secret string, key string, ttl int64) string {
	return macHex(secret, []byte(fmt.Sprintf("%s%d", key, ttl)))
}

func macHex(secret string, msg []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(msg)
	return hex.EncodeToString(m.Sum(nil))
}

// CheckSign verifies a signature over msg in constant time.
func CheckSign(secret, sign string, msg []byte) bool {
	want := macHex(secret, msg)
	return hmac.Equal([]byte(sign), []byte(want))
}

// ValidUploadTicket checks an upload URL's ttl + sign.
func ValidUploadTicket(secret string, ttl int64, sign string) bool {
	return ttl > time.Now().Unix() && CheckSign(secret, sign, []byte(fmt.Sprintf("%d", ttl)))
}

// ValidPreviewTicket checks a preview URL's key + ttl + sign.
func ValidPreviewTicket(secret, key string, ttl int64, sign string) bool {
	return ttl > time.Now().Unix() && CheckSign(secret, sign, []byte(fmt.Sprintf("%s%d", key, ttl)))
}
