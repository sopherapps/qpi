package db

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns a SHA-256 hex digest of the raw string value.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
