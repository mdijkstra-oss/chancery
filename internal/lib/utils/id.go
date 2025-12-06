package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

func GenerateID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 8))
	}
	return hex.EncodeToString(b)[:12]
}

func HashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func EstimateTokens(s string) int {
	words := len(strings.Fields(s))
	chars := len(s)
	byWords := int(float64(words) * 1.3)
	byChars := chars / 4
	if byWords > byChars {
		return byWords
	}
	return byChars
}
