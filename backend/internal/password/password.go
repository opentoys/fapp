package password

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Hash returns sha256(salt + password) hex hash and random salt.
func Hash(pwd string) (hash, salt string) {
	buf := make([]byte, 16)
	rand.Read(buf)
	salt = hex.EncodeToString(buf)
	return digest(salt + pwd), salt
}

// Verify checks the password.
func Verify(pwd, hash, salt string) bool {
	return digest(salt+pwd) == hash
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
