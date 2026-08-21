package auth

import (
	"errors"
	"strings"
	"net/http"
)

func GetApiKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header")
	}
	apiKey, found := strings.CutPrefix(authHeader, "ApiKey ")
	if !found {
		return "", errors.New("no api key found in authorization header")
	}
	return apiKey, nil
}
