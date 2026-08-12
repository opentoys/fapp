package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"disapp/internal/model"
	"disapp/internal/password"
)

func adminLogin(t *testing.T, s *Server) string {
	t.Helper()
	hash, salt := password.Hash("pass123")
	s.DB.Create(&model.User{Username: "admin", PasswordHash: hash, Salt: salt})
	body := bytes.NewBufferString(`{"username":"admin","password":"pass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	var res struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res.Data.Token
}

func authReq(method, path, token string, body []byte) *http.Request {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestAdminCreateApp(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)

	req := authReq(http.MethodPost, "/api/v1/admin/apps", token, []byte(`{"name":"新应用","description":"d"}`))
	w := httptest.NewRecorder()
	s.CreateApp(w, req)

	var count int64
	s.DB.Model(&model.App{}).Count(&count)
	if count != 1 {
		t.Fatalf("apps = %d", count)
	}
}

func TestAdminRequiresAuth(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apps", nil)
	w := httptest.NewRecorder()
	s.RequireAuth(s.AppsList)(w, req)
	if w.Code == http.StatusOK {
		t.Fatal("should be unauthorized")
	}
}

func TestAdminAppsListAndDelete(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	req := authReq(http.MethodGet, "/api/v1/admin/apps", token, nil)
	rec := httptest.NewRecorder()
	s.AppsList(rec, req)
	var res struct {
		Data []model.App `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Data) != 1 {
		t.Fatalf("apps = %d", len(res.Data))
	}

	delReq := authReq(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID), token, nil)
	delReq.SetPathValue("id", itoa(app.ID))
	delRec := httptest.NewRecorder()
	s.DeleteApp(delRec, delReq)
	var count int64
	s.DB.Model(&model.App{}).Count(&count)
	if count != 0 {
		t.Fatalf("apps after delete = %d", count)
	}
}

func TestAdminChannels(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	req := authReq(http.MethodPost, "/api/v1/admin/channels", token, []byte(`{"app_id":1,"name":"release"}`))
	w := httptest.NewRecorder()
	s.CreateChannel(w, req)
	var count int64
	s.DB.Model(&model.Channel{}).Count(&count)
	if count != 1 {
		t.Fatalf("channels = %d", count)
	}
}
