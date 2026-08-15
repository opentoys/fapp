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

	"disapp/internal/resources/store/model"
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
	app := model.App{Name: "a", Platform: "android"}
	s.DB.Create(&app)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("app_id", itoa(app.ID))
	mw.WriteField("release_type", "production")
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

	var v model.Version
	s.DB.Last(&v)
	// The version's platform is forced from the app (single-platform apps),
	// regardless of any platform value the client sends.
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

func TestUploadVersionForcesAppPlatform(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "ios"}
	s.DB.Create(&app)

	// Client sends a conflicting platform; the server must ignore it and use
	// the app's platform.
	req := uploadVersionReq(t, map[string]string{
		"app_id":       itoa(app.ID),
		"version_name": "1.0.0",
		"platform":     "android",
	}, "app.ipa", []byte("ipa"))
	w := httptest.NewRecorder()
	s.UploadVersion(w, req)

	var v model.Version
	s.DB.Last(&v)
	if v.Platform != "ios" {
		t.Fatalf("platform = %q, want ios", v.Platform)
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

	// Access fields from the multipart body must be ignored; the access
	// scope is configured at the app level, never per version.
	s.DB.First(&app, app.ID)
	if app.AccessMode != "" {
		t.Fatalf("upload must not touch app access, got %+v", app)
	}
}

// The app's appid (package_name) is locked on the first version upload that
// carries one; afterwards the app row stores it.
func TestUploadVersionLocksAppidOnFirstUpload(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "android"}
	s.DB.Create(&app)

	req := uploadVersionReq(t, map[string]string{
		"app_id":       itoa(app.ID),
		"version_name": "1.0.0",
		"appid": "com.example.app",
	}, "app.apk", []byte("apk"))
	s.UploadVersion(httptest.NewRecorder(), req)

	s.DB.First(&app, app.ID)
	if app.PackageName == nil || *app.PackageName != "com.example.app" {
		t.Fatalf("appid not locked: %v", app.PackageName)
	}
}

// Once locked, an upload carrying a different (or no) appid must be rejected.
func TestUploadVersionRejectsMismatchedAppid(t *testing.T) {
	s := testServer(t)
	pkg := "com.example.app"
	app := model.App{Name: "a", Platform: "android", PackageName: &pkg}
	s.DB.Create(&app)

	upload := func(extra map[string]string) int {
		fields := map[string]string{"app_id": itoa(app.ID), "version_name": "1.0.0"}
		for k, v := range extra {
			fields[k] = v
		}
		req := uploadVersionReq(t, fields, "app.apk", []byte("apk"))
		w := httptest.NewRecorder()
		s.UploadVersion(w, req)
		var res struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		return res.Code
	}

	if code := upload(map[string]string{"appid": "com.example.app"}); code != 0 {
		t.Fatalf("matching appid rejected: %d", code)
	}
	if code := upload(map[string]string{"appid": "com.other.app"}); code == 0 {
		t.Fatal("mismatched appid must be rejected")
	}
	if code := upload(nil); code == 0 {
		t.Fatal("missing appid on a locked app must be rejected")
	}
}

// Locking an appid must respect the (platform, appid) uniqueness rule: a second
// app on the same platform cannot lock the same appid.
func TestUploadVersionLockRespectsUnique(t *testing.T) {
	s := testServer(t)
	app1 := model.App{Name: "a", Platform: "android"}
	s.DB.Create(&app1)
	app2 := model.App{Name: "b", Platform: "android"}
	s.DB.Create(&app2)

	upload := func(appID int64, pkg string) int {
		fields := map[string]string{"app_id": itoa(appID), "version_name": "1.0.0"}
		if pkg != "" {
			fields["appid"] = pkg
		}
		req := uploadVersionReq(t, fields, "app.apk", []byte("apk"))
		w := httptest.NewRecorder()
		s.UploadVersion(w, req)
		var res struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		return res.Code
	}

	if code := upload(app1.ID, "com.example.app"); code != 0 {
		t.Fatalf("first lock failed: %d", code)
	}
	if code := upload(app2.ID, "com.example.app"); code == 0 {
		t.Fatal("duplicate appid lock on same platform must be rejected")
	}
	if code := upload(app2.ID, "com.other.app"); code != 0 {
		t.Fatalf("distinct appid lock failed: %d", code)
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

func TestDeleteVersion(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1,
		StorageKey: "1/2/a.apk",
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
	mw.WriteField("appid", "com.test.app")
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
			PackageName string `json:"appid"`
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
		StorageKey: "1/2/a.apk", DownloadCount: 3, InstallCount: 1,
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
