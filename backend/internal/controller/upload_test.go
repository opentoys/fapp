package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"disapp/internal/resources/storage/local"
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

// presignFor requests the app icon/screenshot presign and returns the {url,key}.
type ticket struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// createVersion posts JSON metadata (file already uploaded at key) and returns code.
func createVersion(t *testing.T, s *Controller, appID int64, extra map[string]string) int {
	t.Helper()
	body := map[string]any{
		"app_id":       appID,
		"version_name": "1.0.0",
		"version_code": 100,
		"file_name":    "app.apk",
		"key":          "wechat/1/2/app.apk",
		"file_size":    11,
		"sha256":       "abc",
	}
	for k, v := range extra {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", bytes.NewReader(b))
	w := httptest.NewRecorder()
	s.CreateVersion(w, req)
	return codeOf(w)
}

func codeOf(w *httptest.ResponseRecorder) int {
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res.Code
}

func TestCreateVersionSavesMetadata(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app)

	body := `{"app_id":` + itoa(app.ID) + `,"version_name":"1.2.3","version_code":123,"release_type":"production","arch":"arm64,x86_64","changelog":"修复 bug","file_name":"app.apk","key":"wechat/7/8/app.apk","file_size":14,"sha256":"feedbeef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.CreateVersion(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			ID          int64  `json:"id"`
			VersionName string `json:"version_name"`
			FileSize    int64  `json:"file_size"`
			SHA256      string `json:"sha256"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.VersionName != "1.2.3" || res.Data.FileSize != 14 || res.Data.SHA256 != "feedbeef" {
		t.Fatalf("data = %+v", res.Data)
	}

	var v model.Version
	s.SVC.DB.First(&v, res.Data.ID)
	// The version's platform is forced from the app (single-platform apps),
	// regardless of any platform value the client sends.
	if v.ReleaseType != "production" || v.Platform != "android" || v.Arch != "arm64,x86_64" {
		t.Fatalf("release_type/platform/arch not stored: %+v", v)
	}
	if v.StorageKey != "wechat/7/8/app.apk" {
		t.Fatalf("storage_key = %q", v.StorageKey)
	}
}

func TestCreateVersionForcesAppPlatform(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "ios"}
	s.SVC.DB.Create(&app)

	// Client sends a conflicting platform; the server must ignore it and use
	// the app's platform.
	body := `{"app_id":` + itoa(app.ID) + `,"version_name":"1.0.0","platform":"android","file_name":"app.ipa","key":"wechat/1/2/app.ipa"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/versions", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.CreateVersion(w, req)

	var v model.Version
	s.SVC.DB.Last(&v)
	if v.Platform != "ios" {
		t.Fatalf("platform = %q, want ios", v.Platform)
	}
}

func TestCreateVersionRequiresValidKey(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

	// A storage key that fails ValidKey must be rejected.
	code := createVersion(t, s, app.ID, map[string]string{"key": "../etc/passwd"})
	if code == 0 {
		t.Fatal("invalid key must be rejected")
	}
}

func TestCreateVersionRequiresVersionName(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

	code := createVersion(t, s, app.ID, map[string]string{"version_name": ""})
	if code == 0 {
		t.Fatal("missing version_name must be rejected")
	}
}

// The app's appid (package_name) is locked on the first version create that
// carries one; afterwards the app row stores it.
func TestCreateVersionLocksAppidOnFirstUpload(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app)

	createVersion(t, s, app.ID, map[string]string{"appid": "com.example.app"})

	s.SVC.DB.First(&app, app.ID)
	if app.PackageName == nil || *app.PackageName != "com.example.app" {
		t.Fatalf("appid not locked: %v", app.PackageName)
	}
}

// Once locked, a create carrying a different (or no) appid must be rejected.
func TestCreateVersionRejectsMismatchedAppid(t *testing.T) {
	s := testServer(t)
	pkg := "com.example.app"
	app := model.App{Name: "a", Platform: "android", PackageName: &pkg}
	s.SVC.DB.Create(&app)

	if code := createVersion(t, s, app.ID, map[string]string{"appid": "com.example.app"}); code != 0 {
		t.Fatalf("matching appid rejected: %d", code)
	}
	if code := createVersion(t, s, app.ID, map[string]string{"appid": "com.other.app"}); code == 0 {
		t.Fatal("mismatched appid must be rejected")
	}
	if code := createVersion(t, s, app.ID, nil); code == 0 {
		t.Fatal("missing appid on a locked app must be rejected")
	}
}

// Locking an appid must respect the (platform, appid) uniqueness rule: a second
// app on the same platform cannot lock the same appid.
func TestCreateVersionLockRespectsUnique(t *testing.T) {
	s := testServer(t)
	app1 := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app1)
	app2 := model.App{Name: "b", Platform: "android"}
	s.SVC.DB.Create(&app2)

	if code := createVersion(t, s, app1.ID, map[string]string{"appid": "com.example.app"}); code != 0 {
		t.Fatalf("first lock failed: %d", code)
	}
	if code := createVersion(t, s, app2.ID, map[string]string{"appid": "com.example.app"}); code == 0 {
		t.Fatal("duplicate appid lock on same platform must be rejected")
	}
	if code := createVersion(t, s, app2.ID, map[string]string{"appid": "com.other.app"}); code != 0 {
		t.Fatalf("distinct appid lock failed: %d", code)
	}
}

func TestUpdateAppNormalizesExpiryTZ(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

	// The client sends an absolute instant in UTC; the server should store it
	// with the server's default timezone offset (not UTC) so the frontend shows
	// server-local wall-clock time.
	body := `{"expires_at":"2026-08-20T02:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(body))
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UpdateApp(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}

	var reload model.App
	s.SVC.DB.First(&reload, app.ID)
	if reload.ExpiresAt == nil {
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
		StorageKey: "wechat/1/2/a.apk",
	}
	s.SVC.DB.Create(&v)
	storeBytes(t, s, "wechat/1/2/a.apk", []byte("x"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/versions/"+itoa(v.ID)+"?delete_file=true", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.DeleteVersion(w, req)
	var count int64
	s.SVC.DB.Model(&model.Version{}).Count(&count)
	if count != 0 {
		t.Fatalf("versions = %d", count)
	}
	if _, err := s.SVC.Storage.(*local.LocalStorage).Open(nil, "wechat/1/2/a.apk"); err == nil {
		t.Fatal("file should be deleted")
	}
}

func TestVersionStats(t *testing.T) {
	s := testServer(t)
	v := model.Version{
		AppID: 1, VersionName: "1.0", VersionCode: 1,
		StorageKey: "wechat/1/2/a.apk", DownloadCount: 3, InstallCount: 1,
	}
	s.SVC.DB.Create(&v)
	s.SVC.DB.Create(&model.DownloadLog{VersionID: v.ID, IP: "1.2.3.4", UserAgent: "curl"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/versions/"+itoa(v.ID)+"/stats", nil)
	req.SetPathValue("id", itoa(v.ID))
	w := httptest.NewRecorder()
	s.VersionStats(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			DownloadCount int64               `json:"download_count"`
			InstallCount  int64               `json:"install_count"`
			Recent        []model.DownloadLog `json:"recent"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.DownloadCount != 3 || len(res.Data.Recent) != 1 {
		t.Fatalf("res = %s", w.Body.String())
	}
}