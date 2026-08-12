package password

import "testing"

func TestHashAndVerify(t *testing.T) {
	h, salt := Hash("secret123")
	if h == "" || salt == "" {
		t.Fatal("empty hash or salt")
	}
	if h == "secret123" {
		t.Fatal("hash must not be plaintext")
	}
	if !Verify("secret123", h, salt) {
		t.Fatal("correct password should verify")
	}
	if Verify("wrong", h, salt) {
		t.Fatal("wrong password should not verify")
	}
}

func TestHashSaltIsRandom(t *testing.T) {
	h1, s1 := Hash("x")
	h2, s2 := Hash("x")
	if s1 == s2 {
		t.Fatal("salt should be random")
	}
	if h1 == h2 {
		t.Fatal("hash should differ with random salt")
	}
}
