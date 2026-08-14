package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"disapp/internal/model"
)

// pngData is a 1x1 PNG for icon tests.
var pngData = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0xff, 0xff, 0x39, 0x9b, 0x8a, 0xdc, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func uploadVersionReq(t *testing.T, fields map[string]string, filename string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	fw, _ := mw.CreateFormFile("file", filename)
	fw.Write(data)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadVersion(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("release_type", "production")
	mw.WriteField("platform", "android")
	mw.WriteField("arch", "arm64,x86_64")
	mw.WriteField("version_name", "1.2.3")
	mw.WriteField("version_code", "123")
	mw.WriteField("changelog", "修复 bug")
	fw, _ := mw.CreateFormFile("file", "app.apk")
	fw.Write([]byte("fake-apk-bytes"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			ID           int64  `json:"id"`
			VersionName  string `json:"version_name"`
			FileSize     int64  `json:"file_size"`
			SHA256       string `json:"sha256"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.VersionName != "1.2.3" || res.Data.FileSize != int64(len("fake-apk-bytes")) {
		t.Fatalf("data = %+v", res.Data)
	}
	if res.Data.SHA256 == "" {
		t.Fatal("sha256 missing")
	}

	// Upload must create a draft: not visible until published.
	var v model.Version
	s.DB.Last(&v)
	if v.Published {
		t.Fatalf("upload must create a draft, got published = %v", v.Published)
	}
	if v.ReleaseType != "production" || v.Platform != "android" || v.Arch != "arm64,x86_64" {
		t.Fatalf("release_type/platform/arch not stored: %+v", v)
	}
	// Verify file was actually written to local storage
	rc, err := s.Storage.Open(nil, itoa(app.ID)+"/"+itoa(v.ID)+"/app.apk")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "fake-apk-bytes" {
		t.Fatalf("stored = %q", data)
	}
}

func TestUploadVersionIgnoresAccessFields(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("version_name", "1.0")
	mw.WriteField("version_code", "10")
	mw.WriteField("access_mode", "password")
	mw.WriteField("password", "secret")
	fw, _ := mw.CreateFormFile("file", "x.apk")
	fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}

	// Access fields from the multipart body must be ignored; the upload is
	// a draft and the access scope is configured at the app level.
	var v model.Version
	s.DB.Last(&v)
	if v.Published {
		t.Fatalf("upload must not publish, got published = %v", v.Published)
	}
	s.DB.First(&app, v.AppID)
	if app.AccessMode != "" {
		t.Fatalf("upload must not touch app access, got %+v", app)
	}
}

func TestPublishVersion(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1, StorageKey: "1/2/a.apk",
	}
	s.DB.Create(&v)

	// Publishing flips visibility only; access scope is app-level and is set
	// on the app's Overview page, never per version.
	body := `{"published":true,"enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/versions/"+itoa(v.ID), strings.NewReader(body))
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.UpdateVersion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}

	var reload model.Version
	s.DB.First(&reload, v.ID)
	if !reload.Published || !reload.Enabled {
		t.Fatalf("reload = %+v", reload)
	}
}

func TestUpdateAppNormalizesExpiryTZ(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// The client sends an absolute instant in UTC; the server should store it
	// with the server's default timezone offset (not UTC) so the frontend shows
	// server-local wall-clock time.
	body := `{"access_mode":"expiry","expires_at":"2026-08-20T02:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(body))
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UpdateApp(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}

	var reload model.App
	s.DB.First(&reload, app.ID)
	if reload.AccessMode != model.AccessExpiry || reload.ExpiresAt == nil {
		t.Fatalf("reload = %+v", reload)
	}
	// Same absolute instant the client sent.
	want, _ := time.Parse(time.RFC3339, "2026-08-20T02:00:00Z")
	if !reload.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at instant = %v, want %v", reload.ExpiresAt, want)
	}
	// Stored with the server default timezone offset (not UTC).
	_, wantOff := want.In(time.Local).Zone()
	_, off := reload.ExpiresAt.Zone()
	if off != wantOff {
		t.Fatalf("expires_at offset = %d (%v), want server default %d", off, reload.ExpiresAt, wantOff)
	}
}

func TestUpdateVersionToggleDisabled(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1,
		Enabled: true, StorageKey: "1/2/a.apk",
	}
	s.DB.Create(&v)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/versions/"+itoa(v.ID), strings.NewReader(`{"enabled":false,"changelog":"下架"}`))
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.UpdateVersion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var reload model.Version
	s.DB.First(&reload, v.ID)
	if reload.Enabled || reload.Changelog != "下架" {
		t.Fatalf("reload = %+v", reload)
	}
}

func TestDeleteVersion(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1,
		Enabled: true, StorageKey: "1/2/a.apk",
	}
	s.DB.Create(&v)
	s.Storage.Save(nil, "1/2/a.apk", strings.NewReader("x"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/versions/"+itoa(v.ID)+"?delete_file=true", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.DeleteVersion(w, req)
	var count int64
	s.DB.Model(&model.Version{}).Count(&count)
	if count != 0 {
		t.Fatalf("versions = %d", count)
	}
	if _, err := s.Storage.Open(nil, "1/2/a.apk"); err == nil {
		t.Fatal("file should be deleted")
	}
}

func TestUploadVersionMetadata(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// The browser sends parsed package/app_name as fields and the icon as a
	// separate multipart file (always PNG). The handler just stores them.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("version_name", "2.0.0")
	mw.WriteField("version_code", "200")
	mw.WriteField("package_name", "com.test.app")
	mw.WriteField("app_name", "Test App")
	fw, _ := mw.CreateFormFile("file", "app.apk")
	fw.Write([]byte("apk"))
	icon, _ := mw.CreateFormFile("icon", "icon.png")
	icon.Write(pngData)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			ID          int64  `json:"id"`
			VersionName string `json:"version_name"`
			VersionCode int    `json:"version_code"`
			PackageName string `json:"package_name"`
			AppName     string `json:"app_name"`
			IconURL     string `json:"icon_url"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.VersionName != "2.0.0" || res.Data.VersionCode != 200 ||
		res.Data.PackageName != "com.test.app" || res.Data.AppName != "Test App" {
		t.Fatalf("metadata not applied: %+v", res.Data)
	}
	if res.Data.IconURL == "" {
		t.Fatal("icon_url missing")
	}

	// Icon bytes must be stored at the exposed key.
	key := strings.TrimPrefix(res.Data.IconURL, "/api/v1/files/")
	rc, err := s.Storage.Open(nil, key)
	if err != nil {
		t.Fatalf("icon not stored: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, pngData) {
		t.Fatalf("icon bytes mismatch: got %d bytes", len(got))
	}

	// DB row persists the fields.
	var v model.Version
	s.DB.First(&v, res.Data.ID)
	if v.PackageName != "com.test.app" || v.AppName != "Test App" || v.IconURL != res.Data.IconURL {
		t.Fatalf("db row = %+v", v)
	}
}

func TestUploadVersionWithoutIcon(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// No icon part -> upload succeeds with an empty icon_url.
	req := uploadVersionReq(t, map[string]string{
		"app_id":       itoa(app.ID),
		"version_name": "1.0.0",
	}, "app.apk", []byte("apk"))
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			IconURL string `json:"icon_url"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.IconURL != "" {
		t.Fatalf("expected empty icon_url, got %q", res.Data.IconURL)
	}
}

func TestUploadVersionRequiresVersionName(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	req := uploadVersionReq(t, map[string]string{"app_id": itoa(app.ID)}, "app.apk", []byte("apk"))
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code == 0 {
		t.Fatalf("expected failure, got %s", w.Body.String())
	}
}

func TestVersionStats(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1,
		Enabled: true, StorageKey: "1/2/a.apk", DownloadCount: 3, InstallCount: 1,
	}
	s.DB.Create(&v)
	s.DB.Create(&model.DownloadLog{VersionID: v.ID, IP: "1.2.3.4", UserAgent: "curl"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/versions/"+itoa(v.ID)+"/stats", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.VersionStats(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			DownloadCount int64              `json:"download_count"`
			InstallCount  int64              `json:"install_count"`
			Recent        []model.DownloadLog `json:"recent"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.DownloadCount != 3 || len(res.Data.Recent) != 1 {
		t.Fatalf("res = %s", w.Body.String())
	}
}
