package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CreateHMACHeader creates the hmac token
func CreateHMACHeader(data []byte, key string) string {
	sig := hmac.New(sha256.New, []byte(key))
	sig.Write(data)
	return "sha256=" + hex.EncodeToString(sig.Sum(nil))
}
