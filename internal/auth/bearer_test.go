package auth

import (
	"net/http"
	"strings"
	"testing"
)


func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer 8943729897-dkfljfkas")
	
	token, err := GetBearerToken(headers)
	if err != nil {
		t.Error("getbearertoken function returned error")
		return
	}
	if strings.Contains(token, "Bearer") {
		t.Error("bearer substring still present in token")
	}
	t.Logf("successful parsing of token from header\ntoken: %s", token)
}
