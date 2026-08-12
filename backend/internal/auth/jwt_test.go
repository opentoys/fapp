package auth

import (
	"testing"
	"time"
)

func TestCreateParse(t *testing.T) {
	token, err := CreateToken("secret", 7, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken("secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 7 || claims.Username != "admin" {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, _ := CreateToken("a", 1, "u", time.Hour)
	if _, err := ParseToken("b", token); err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestParseExpired(t *testing.T) {
	token, _ := CreateToken("a", 1, "u", -time.Minute)
	if _, err := ParseToken("a", token); err == nil {
		t.Fatal("expected error with expired token")
	}
}
