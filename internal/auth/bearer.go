package auth

import (
	"net/http"
	"errors"
	"strings"
)


func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header")
	} 
	token, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found {
		return "", errors.New("bearer not found in authorization header")
	}
	return token, nil
}
