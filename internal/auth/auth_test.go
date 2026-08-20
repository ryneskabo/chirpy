package auth

import (
	"testing"
)


func TestHashPassword(t *testing.T) {
	password := "password"
	hash, err := HashPassword(password)
	if hash == password || err != nil {
		t.Errorf("error hashing password %s\n outcome %s", password, hash)
		return
	}
	t.Logf("success hashing password: %s\noutcome: %s", password, hash)	

	match, err := CheckPasswordHash(password, hash)
	if !match || err != nil {
		t.Logf("error matching password: %s and hash: %s", password, hash)
		return
	}
	t.Logf("success matching password and hash")
}

func TestHashPasswordBlankPassword(t *testing.T) {
	password := ""
	hash, err := HashPassword(password)
	if hash == password || err != nil {
		t.Errorf("error hashing password %s\n outcome %s", password, hash)
		return
	}
	t.Logf("success hashing password: %s\noutcome: %s", password, hash)	

	match, err := CheckPasswordHash(password, hash)
	if !match || err != nil {
		t.Logf("error matching password: %s and hash: %s", password, hash)
		return
	}
	t.Logf("success matching password: %s and hash: %s", password, hash)
}
