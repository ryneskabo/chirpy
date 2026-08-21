package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() string {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	
	return hex.EncodeToString(randBytes)
}
