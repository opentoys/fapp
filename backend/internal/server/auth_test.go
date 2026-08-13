package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"disapp/internal/auth"
	"disapp/internal/config"
	"disapp/internal/db"
	"disapp/internal/model"
	"disapp/internal/password"
	"disapp/internal/storage"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	gdb, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.NewLocal(filepath.Join(t.TempDir(), "files"))
	if err != nil {
		t.Fatal(err)
	}
	return New(gdb, st, config.Default())
}

func testServerWithAdmin(t *testing.T, user, pass string) *Server {
	t.Helper()
	s := testServer(t)
	s.Config.Admin.Username = user
	s.Config.Admin.Password = pass
	return s
}

func TestLoginOK(t *testing.T) {
	s := testServer(t)
	hash, salt := password.Hash("pass123")
	s.DB.Create(&model.User{Username: "admin", PasswordHash: hash, Salt: salt})

	body := bytes.NewBufferString(`{"username":"admin","password":"pass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var res struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.Token == "" {
		t.Fatalf("res = %s", w.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	s := testServer(t)
	hash, salt := password.Hash("pass123")
	s.DB.Create(&model.User{Username: "admin", PasswordHash: hash, Salt: salt})

	body := bytes.NewBufferString(`{"username":"admin","password":"nope"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 401 {
		t.Fatalf("code = %d, body = %s", res.Code, w.Body.String())
	}
}

func TestRequireAuth(t *testing.T) {
	s := testServer(t)
	h := s.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token should be 401, got %d", w.Code)
	}
}

// Super-admin lives in config, not in the users table. Login must accept
// the configured credentials without touching the DB and must issue a
// token whose uid is SuperAdminUID.
func TestSuperAdminLoginFromConfig(t *testing.T) {
	s := testServerWithAdmin(t, "root", "s3cret")

	// Deliberately put a *different* user in the DB to prove the login
	// didn't go through the DB path.
	hash, salt := password.Hash("dbpass")
	s.DB.Create(&model.User{Username: "root", PasswordHash: hash, Salt: salt})

	body := bytes.NewBufferString(`{"username":"root","password":"s3cret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}
	var res struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.Token == "" {
		t.Fatalf("res = %s", w.Body.String())
	}

	claims, err := auth.ParseToken(s.Config.JWT.Secret, res.Data.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != SuperAdminUID {
		t.Fatalf("super-admin uid = %d, want %d", claims.UserID, SuperAdminUID)
	}
}

func TestSuperAdminWrongPassword(t *testing.T) {
	s := testServerWithAdmin(t, "root", "s3cret")

	body := bytes.NewBufferString(`{"username":"root","password":"nope"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 401 {
		t.Fatalf("code = %d, body = %s", res.Code, w.Body.String())
	}
}
