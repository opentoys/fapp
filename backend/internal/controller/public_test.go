package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
)

func seedApp(t *testing.T, s *Controller) *model.App {
	t.Helper()
	app := model.App{Name: "测试应用", Description: "desc", Published: true}
	if err := s.SVC.DB.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	current := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 1,
		FileName: "app.apk", FileType: "apk", FileSize: 100,
		StorageKey: "wechat/1/2/app.apk", StorageBackend: "local",
	}
	if err := s.SVC.DB.Create(&current).Error; err != nil {
		t.Fatal(err)
	}
	old := model.Version{
		AppID: app.ID, VersionName: "0.9.0", VersionCode: 0,
		FileName: "old.apk", FileType: "apk",
	}
	if err := s.SVC.DB.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	newer := model.Version{
		AppID: app.ID, VersionName: "2.0.0", VersionCode: 2,
		FileName: "new.apk", FileType: "apk",
		StorageKey: "wechat/1/4/new.apk", StorageBackend: "local",
	}
	if err := s.SVC.DB.Create(&newer).Error; err != nil {
		t.Fatal(err)
	}
	// 1.0.0 is the current version; 0.9.0 and 2.0.0 stay hidden publicly.
	s.SVC.DB.Model(&app).Update("current_version_id", current.ID)
	return &app
}

// An app past its download-link expiry behaves as "应用不存在" on the detail
// path (the public list endpoint no longer exists).
func TestPublicAppExpiredNotFound(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	past := time.Now().Add(-time.Hour)
	s.SVC.DB.Model(app).Update("expires_at", past)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID), nil)
	req2.SetPathValue("id", itoa(app.ID))
	w2 := httptest.NewRecorder()
	s.AppDetail(w2, req2)
	var res2 struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w2.Body.Bytes(), &res2)
	if res2.Code != 404 {
		t.Fatalf("expired app detail must be 404, got %s", w2.Body.String())
	}
}

func TestPublicAppDetailShowsOnlyCurrent(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID), nil)
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.AppDetail(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			Versions []model.Version `json:"versions"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if len(res.Data.Versions) != 1 || res.Data.Versions[0].VersionName != "1.0.0" {
		t.Fatalf("should only show the current version, got %d", len(res.Data.Versions))
	}
	if res.Data.Versions[0].StorageKey != "" {
		t.Fatal("secret fields leaked")
	}
}

// A password-protected app returns only {id, access_mode} until the correct
// password is supplied; the full detail (versions) then comes back.
func TestPublicAppPasswordGate(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	s.SVC.DB.Model(app).Update("access_mode", "password")
	hash, salt := pwd.Hash("secret")
	s.SVC.DB.Model(app).Updates(map[string]any{"password_hash": hash, "salt": salt})

	// No password → minimal payload, no versions.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID), nil)
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.AppDetail(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			App struct {
				ID         int64           `json:"id"`
				AccessMode string          `json:"access_mode"`
				Name       string          `json:"name"`
				Icon       string          `json:"icon"`
				Versions   []model.Version `json:"versions"`
			} `json:"app"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.App.ID != app.ID || res.Data.App.AccessMode != "password" {
		t.Fatalf("locked detail = %s", w.Body.String())
	}
	if res.Data.App.Name != "" || res.Data.App.Icon != "" {
		t.Fatalf("locked payload must not leak identity, got %+v", res.Data.App)
	}

	// Wrong password → 403 (so the public client shows an error without
	// treating it as a session expiry / login redirect).
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID)+"?password=nope", nil)
	req2.SetPathValue("id", itoa(app.ID))
	w2 := httptest.NewRecorder()
	s.AppDetail(w2, req2)
	var res2 struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w2.Body.Bytes(), &res2)
	if res2.Code != 403 {
		t.Fatalf("wrong password must be 403, got %s", w2.Body.String())
	}

	// Correct password → full detail with the current version.
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID)+"?password=secret", nil)
	req3.SetPathValue("id", itoa(app.ID))
	w3 := httptest.NewRecorder()
	s.AppDetail(w3, req3)
	var res3 struct {
		Code int `json:"code"`
		Data struct {
			Versions []model.Version `json:"versions"`
		} `json:"data"`
	}
	json.Unmarshal(w3.Body.Bytes(), &res3)
	if res3.Code != 0 || len(res3.Data.Versions) != 1 {
		t.Fatalf("unlocked detail = %s", w3.Body.String())
	}
}

func TestPublicAppUnpublishedNotFound(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	s.SVC.DB.Model(app).Update("published", false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID), nil)
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.AppDetail(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 404 {
		t.Fatalf("unpublished app must return not-found, got code=%d body=%s", res.Code, w.Body.String())
	}
}

// Published but with no current version → detail shows an empty version list.
func TestPublicAppDetailNoCurrentVersion(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	// Remove the current version pointer; app stays published.
	s.SVC.DB.Model(app).Update("current_version_id", 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+itoa(app.ID), nil)
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.AppDetail(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			Versions []model.Version `json:"versions"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data.Versions) != 0 {
		t.Fatalf("no-current version must yield an empty array, got %s", w.Body.String())
	}
}

func TestPublicAppDetailResolvesByName(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/"+app.Name, nil)
	req.SetPathValue("id", app.Name)
	w := httptest.NewRecorder()
	s.AppDetail(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			App struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"app"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.App.ID != app.ID || res.Data.App.Name != app.Name {
		t.Fatalf("res = %s", w.Body.String())
	}
}

// storeBytes writes bytes at key through the local storage concrete method.
func storeBytes(t *testing.T, s *Controller, key string, data []byte) {
	t.Helper()
	loc, ok := s.SVC.Storage.(*local.LocalStorage)
	if !ok {
		t.Fatal("test requires local storage")
	}
	if _, err := loc.Save(context.TODO(), key, bytes.NewReader(data)); err != nil {
		t.Fatalf("store bytes: %v", err)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// hashA/hashB are distinct valid SHA-256 hex digests for content-addressed key
// tests.
const (
	hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hashB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)
