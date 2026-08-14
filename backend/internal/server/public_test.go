package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"disapp/internal/model"
)

func seedApp(t *testing.T, s *Server) *model.App {
	t.Helper()
	app := model.App{Name: "测试应用", Description: "desc"}
	if err := s.DB.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	v := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 1,
		FileName: "app.apk", FileType: "apk", FileSize: 100, AccessMode: model.AccessPublic,
		Published: true, Enabled: true, StorageKey: "1/2/app.apk", StorageBackend: "local",
	}
	if err := s.DB.Create(&v).Error; err != nil {
		t.Fatal(err)
	}
	disabled := model.Version{
		AppID: app.ID, VersionName: "0.9.0", VersionCode: 0,
		FileName: "old.apk", FileType: "apk", AccessMode: model.AccessPublic,
		Published: true, Enabled: true,
	}
	if err := s.DB.Create(&disabled).Error; err != nil {
		t.Fatal(err)
	}
	// GORM default:true overrides Enabled=false on Create, so update after.
	s.DB.Model(&disabled).Update("enabled", false)
	// A draft (published=false, enabled=true) must stay hidden publicly.
	draft := model.Version{
		AppID: app.ID, VersionName: "2.0.0", VersionCode: 2,
		FileName: "new.apk", FileType: "apk", AccessMode: model.AccessPublic,
		Published: false, Enabled: true, StorageKey: "1/4/new.apk", StorageBackend: "local",
	}
	if err := s.DB.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	return &app
}

func TestPublicApps(t *testing.T) {
	s := testServer(t)
	seedApp(t, s)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	s.Apps(w, req)

	var res struct {
		Code int `json:"code"`
		Data []struct {
			Name           string `json:"name"`
			LatestVersion  *struct {
				VersionName string `json:"version_name"`
			} `json:"latest_version"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data) != 1 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data[0].LatestVersion == nil || res.Data[0].LatestVersion.VersionName != "1.0.0" {
		t.Fatalf("latest version wrong: %+v", res.Data[0].LatestVersion)
	}
}

func TestPublicAppDetailHidesDisabledAndSecret(t *testing.T) {
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
	if len(res.Data.Versions) != 1 {
		t.Fatalf("should only show enabled version, got %d", len(res.Data.Versions))
	}
	if res.Data.Versions[0].StorageKey != "" || res.Data.Versions[0].PasswordHash != "" {
		t.Fatal("secret fields leaked")
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
