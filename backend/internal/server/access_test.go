package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"disapp/internal/model"
	"disapp/internal/password"
)

// setAppAccess applies the app-level access scope for the test app.
func setAppAccess(s *Server, app *model.App, mode, pwd string, expiresAt *time.Time) {
	m := map[string]any{"access_mode": mode}
	if expiresAt != nil {
		m["expires_at"] = expiresAt
	}
	s.DB.Model(app).Updates(m)
	if pwd != "" {
		h, salt := password.Hash(pwd)
		s.DB.Model(app).Updates(map[string]any{"password_hash": h, "salt": salt})
	}
}

func TestVerifyPassword(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	setAppAccess(s, app, model.AccessPassword, "abc", nil)
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 2, FileName: "a.apk",
		FileType: "apk", Published: true, Enabled: true, StorageKey: "1/3/a.apk",
	}
	s.DB.Create(&v)

	body := bytes.NewBufferString(`{"password":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/versions/"+itoa(v.ID)+"/verify", body)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.VerifyAccess(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestDownloadWrongPassword(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	setAppAccess(s, app, model.AccessPassword, "abc", nil)
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 2, FileName: "a.apk",
		FileType: "apk", Published: true, Enabled: true, StorageKey: "1/3/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/"+itoa(v.ID)+"/download?password=wrong", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.Download(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 401 {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestDownloadExpired(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	past := time.Now().Add(-time.Hour)
	setAppAccess(s, app, model.AccessExpiry, "", &past)
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 2, FileName: "a.apk",
		FileType: "apk", Published: true, Enabled: true, StorageKey: "1/3/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/"+itoa(v.ID)+"/download", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.Download(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 403 {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestDownloadCounts(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 2, FileName: "a.apk",
		FileType: "apk", Published: true, Enabled: true, StorageKey: "1/3/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/"+itoa(v.ID)+"/download", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.Download(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || !strings.HasPrefix(res.Data.URL, "/api/v1/files/") {
		t.Fatalf("res = %s", w.Body.String())
	}

	var reload model.Version
	s.DB.First(&reload, v.ID)
	if reload.DownloadCount != 1 {
		t.Fatalf("download_count = %d", reload.DownloadCount)
	}
	var logs []model.DownloadLog
	s.DB.Find(&logs, "version_id = ?", v.ID)
	if len(logs) != 1 {
		t.Fatalf("logs = %d", len(logs))
	}
}

func TestInstallCounts(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 2, FileName: "a.apk",
		FileType: "apk", Published: true, Enabled: true, StorageKey: "1/3/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/versions/"+itoa(v.ID)+"/install", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.Install(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var reload model.Version
	s.DB.First(&reload, v.ID)
	if reload.InstallCount != 1 {
		t.Fatalf("install_count = %d", reload.InstallCount)
	}
}

func TestFileProxy(t *testing.T) {
	s := testServer(t)
	if _, err := s.Storage.Save(nil, "1/2/app.apk", strings.NewReader("binary-data")); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/1/2/app.apk", nil)
	req.SetPathValue("key", "1/2/app.apk")
	w := httptest.NewRecorder()
	s.File(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "binary-data" {
		t.Fatalf("code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestFileProxyRejectsTraversal(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/../../etc/passwd", nil)
	req.SetPathValue("key", "../../etc/passwd")
	w := httptest.NewRecorder()
	s.File(w, req)
	if w.Code == http.StatusOK {
		t.Fatal("traversal must be rejected")
	}
}
