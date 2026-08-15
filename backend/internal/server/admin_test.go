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

	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
)

func adminLogin(t *testing.T, s *Server) string {
	t.Helper()
	hash, salt := pwd.Hash("pass123")
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

	req := authReq(http.MethodPost, "/api/v1/admin/apps", token, []byte(`{"name":"新应用","description":"d","platform":"android","appid":"com.example.app"}`))
	w := httptest.NewRecorder()
	s.CreateApp(w, req)

	var count int64
	s.DB.Model(&model.App{}).Count(&count)
	if count != 1 {
		t.Fatalf("apps = %d", count)
	}
	var app model.App
	s.DB.First(&app)
	if app.Platform != "android" {
		t.Fatalf("platform = %q", app.Platform)
	}
	if app.PackageName == nil || *app.PackageName != "com.example.app" {
		t.Fatalf("package = %v", app.PackageName)
	}
}

func TestAdminCreateAppPackageUnique(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)

	create := func(body string) int {
		req := authReq(http.MethodPost, "/api/v1/admin/apps", token, []byte(body))
		w := httptest.NewRecorder()
		s.CreateApp(w, req)
		var res struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		return res.Code
	}

	// Same platform + same package must be rejected.
	if code := create(`{"name":"a","platform":"android","appid":"com.example.app"}`); code != 0 {
		t.Fatalf("first create rejected: %d", code)
	}
	if code := create(`{"name":"b","platform":"android","appid":"com.example.app"}`); code == 0 {
		t.Fatal("duplicate platform+package should be rejected")
	}
	// Same package on a different platform is allowed.
	if code := create(`{"name":"c","platform":"ios","appid":"com.example.app"}`); code != 0 {
		t.Fatalf("same package on other platform rejected: %d", code)
	}
	// Same platform, different package is allowed.
	if code := create(`{"name":"d","platform":"android","appid":"com.example.other"}`); code != 0 {
		t.Fatalf("different package rejected: %d", code)
	}
	// Manually-created apps without a package coexist on one platform (NULL).
	if code := create(`{"name":"e","platform":"android"}`); code != 0 {
		t.Fatalf("no-package create rejected: %d", code)
	}
	if code := create(`{"name":"f","platform":"android"}`); code != 0 {
		t.Fatalf("second no-package create rejected: %d", code)
	}
}

func TestAdminCreateAppRequiresPlatform(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)

	// Missing / invalid platform must be rejected.
	for _, body := range []string{
		`{"name":"a","platform":""}`,
		`{"name":"a","platform":"windows"}`,
	} {
		req := authReq(http.MethodPost, "/api/v1/admin/apps", token, []byte(body))
		w := httptest.NewRecorder()
		s.CreateApp(w, req)
		var res struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		if res.Code == 0 {
			t.Fatalf("expected rejection for %s, got %s", body, w.Body.String())
		}
	}
}

func TestAdminAppDetailIncludesDrafts(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apps/"+itoa(app.ID), nil)
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.AppDetailAdmin(w, req)

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
	// seedApp creates 3 versions (1.0.0 current, 0.9.0, 2.0.0); the admin
	// detail exposes all of them regardless of current/published state.
	if len(res.Data.Versions) != 3 {
		t.Fatalf("admin detail should include all versions, got %d", len(res.Data.Versions))
	}
}

func TestSetCurrentVersion(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a", Platform: "android"}
	s.DB.Create(&app)
	other := model.App{Name: "b", Platform: "android"}
	s.DB.Create(&other)
	v1 := model.Version{AppID: app.ID, VersionName: "1.0.0"}
	v2 := model.Version{AppID: app.ID, VersionName: "2.0.0"}
	s.DB.Create(&v1)
	s.DB.Create(&v2)
	foreign := model.Version{AppID: other.ID, VersionName: "1.0.0"}
	s.DB.Create(&foreign)

	set := func(id, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/apps/"+id+"/current", strings.NewReader(body))
		req.SetPathValue("id", id)
		w := httptest.NewRecorder()
		s.SetCurrentVersion(w, req)
		var res struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		return res.Code
	}

	if code := set(itoa(app.ID), `{"version_id":`+itoa(v1.ID)+`}`); code != 0 {
		t.Fatalf("set current failed: %d", code)
	}
	var reload model.App
	s.DB.First(&reload, app.ID)
	if reload.CurrentVersionID != v1.ID {
		t.Fatalf("current_version_id = %d", reload.CurrentVersionID)
	}

	// Switching to another version overwrites (still one current version).
	if code := set(itoa(app.ID), `{"version_id":`+itoa(v2.ID)+`}`); code != 0 {
		t.Fatalf("switch current failed: %d", code)
	}
	s.DB.First(&reload, app.ID)
	if reload.CurrentVersionID != v2.ID {
		t.Fatalf("current_version_id after switch = %d", reload.CurrentVersionID)
	}

	// A version of a different app must be rejected.
	if code := set(itoa(app.ID), `{"version_id":`+itoa(foreign.ID)+`}`); code == 0 {
		t.Fatal("foreign version must be rejected")
	}
	// Missing version_id rejected.
	if code := set(itoa(app.ID), `{}`); code == 0 {
		t.Fatal("missing version_id must be rejected")
	}
}

func TestUpdateAppPublished(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	pub := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"published":true}`))
	pub.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), pub)
	s.DB.First(&app, app.ID)
	if !app.Published {
		t.Fatal("app should be published")
	}

	down := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"published":false}`))
	down.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), down)
	s.DB.First(&app, app.ID)
	if app.Published {
		t.Fatal("app should be unpublished")
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
	if res.Data[0].Screenshots == nil || len(res.Data[0].Screenshots) != 0 {
		t.Fatalf("screenshots must serialize as an empty array, got %v", res.Data[0].Screenshots)
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

func TestAdminUploadAppIcon(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("icon", "icon.png")
	fw.Write(pngData)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/apps/"+itoa(app.ID)+"/icon", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UploadAppIcon(w, req)

	var res struct {
		Code int `json:"code"`
		Data struct {
			Icon string `json:"icon"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.Icon == "" {
		t.Fatal("icon url missing")
	}

	// Bytes stored at the exposed key.
	key := strings.TrimPrefix(res.Data.Icon, "/api/v1/files/")
	rc, err := s.Storage.Open(nil, key)
	if err != nil {
		t.Fatalf("icon not stored: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, pngData) {
		t.Fatalf("icon bytes mismatch: got %d bytes", len(got))
	}

	// Deleting the app removes the icon file too.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID), nil)
	delReq.SetPathValue("id", itoa(app.ID))
	s.DeleteApp(httptest.NewRecorder(), delReq)
	if _, err := s.Storage.Open(nil, key); err == nil {
		t.Fatal("icon file should be deleted with the app")
	}
}

func TestUpdateAppAccessPassword(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// Set password scope; the credential must be hashed and stored.
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"access_mode":"password","password":"secret"}`))
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UpdateApp(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	var reload model.App
	s.DB.First(&reload, app.ID)
	if reload.AccessMode != model.AccessPassword || reload.PasswordHash == "" {
		t.Fatalf("reload = %+v", reload)
	}

	// Switching back to public clears the stored credential.
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"access_mode":"public"}`))
	req2.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), req2)
	s.DB.First(&reload, app.ID)
	if reload.AccessMode != model.AccessPublic || reload.PasswordHash != "" || reload.Salt != "" {
		t.Fatalf("reload after public = %+v", reload)
	}
}

func TestAdminScreenshotUploadAndDelete(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// Upload a screenshot.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("screenshot", "shot.png")
	fw.Write(pngData)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/apps/"+itoa(app.ID)+"/screenshots", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UploadAppScreenshot(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			Screenshots []string `json:"screenshots"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data.Screenshots) != 1 {
		t.Fatalf("res = %s", w.Body.String())
	}
	url := res.Data.Screenshots[0]
	key := strings.TrimPrefix(url, "/api/v1/files/")
	if rc, err := s.Storage.Open(nil, key); err != nil {
		t.Fatalf("screenshot not stored: %v", err)
	} else {
		got, _ := io.ReadAll(rc)
		rc.Close()
		if !bytes.Equal(got, pngData) {
			t.Fatalf("screenshot bytes mismatch: got %d bytes", len(got))
		}
	}

	// Deleting the app removes the screenshot file too.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID), nil)
	delReq.SetPathValue("id", itoa(app.ID))
	s.DeleteApp(httptest.NewRecorder(), delReq)
	if _, err := s.Storage.Open(nil, key); err == nil {
		t.Fatal("screenshot file should be deleted with the app")
	}
}

func TestAdminDeleteAppScreenshot(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	// Upload two, delete one.
	urls := []string{}
	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("screenshot", "shot.png")
		fw.Write(pngData)
		mw.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/apps/"+itoa(app.ID)+"/screenshots", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.SetPathValue("id", itoa(app.ID))
		w := httptest.NewRecorder()
		s.UploadAppScreenshot(w, req)
		var res struct {
			Code int `json:"code"`
			Data struct {
				Screenshots []string `json:"screenshots"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		if res.Code != 0 {
			t.Fatalf("res = %s", w.Body.String())
		}
		urls = res.Data.Screenshots
	}
	if len(urls) != 2 {
		t.Fatalf("screenshots = %d", len(urls))
	}

	del := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID)+"/screenshots?url="+urls[0], nil)
	del.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.DeleteAppScreenshot(w, del)
	var res struct {
		Code int `json:"code"`
		Data struct {
			Screenshots []string `json:"screenshots"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data.Screenshots) != 1 || res.Data.Screenshots[0] != urls[1] {
		t.Fatalf("res = %s", w.Body.String())
	}
	if _, err := s.Storage.Open(nil, strings.TrimPrefix(urls[0], "/api/v1/files/")); err == nil {
		t.Fatal("deleted screenshot file should be gone")
	}
}

func TestAdminUploadAppIconRejectsNonImage(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.DB.Create(&app)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("icon", "app.txt")
	fw.Write([]byte("not an image"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/apps/"+itoa(app.ID)+"/icon", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.UploadAppIcon(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code == 0 {
		t.Fatalf("expected rejection, got %s", w.Body.String())
	}
}
