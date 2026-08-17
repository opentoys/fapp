package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/model"
	"disapp/pkg/pwd"
)

func adminLogin(t *testing.T, s *Controller) string {
	t.Helper()
	hash, salt := pwd.Hash("pass123")
	s.SVC.DB.Create(&model.User{Username: "admin", PasswordHash: hash, Salt: salt})
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
	s.SVC.DB.Model(&model.App{}).Count(&count)
	if count != 1 {
		t.Fatalf("apps = %d", count)
	}
	var app model.App
	s.SVC.DB.First(&app)
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
	s.SVC.DB.Create(&app)
	other := model.App{Name: "b", Platform: "android"}
	s.SVC.DB.Create(&other)
	v1 := model.Version{AppID: app.ID, VersionName: "1.0.0"}
	v2 := model.Version{AppID: app.ID, VersionName: "2.0.0"}
	s.SVC.DB.Create(&v1)
	s.SVC.DB.Create(&v2)
	foreign := model.Version{AppID: other.ID, VersionName: "1.0.0"}
	s.SVC.DB.Create(&foreign)

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
	s.SVC.DB.First(&reload, app.ID)
	if reload.CurrentVersionID != v1.ID {
		t.Fatalf("current_version_id = %d", reload.CurrentVersionID)
	}

	// Switching to another version overwrites (still one current version).
	if code := set(itoa(app.ID), `{"version_id":`+itoa(v2.ID)+`}`); code != 0 {
		t.Fatalf("switch current failed: %d", code)
	}
	s.SVC.DB.First(&reload, app.ID)
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
	s.SVC.DB.Create(&app)

	pub := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"published":true}`))
	pub.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), pub)
	s.SVC.DB.First(&app, app.ID)
	if !app.Published {
		t.Fatal("app should be published")
	}

	down := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"published":false}`))
	down.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), down)
	s.SVC.DB.First(&app, app.ID)
	if app.Published {
		t.Fatal("app should be unpublished")
	}
}

func TestAdminRequiresAuth(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apps", nil)
	w := httptest.NewRecorder()
	s.RequireAuth(s.AppsList)(w, req)
	if codeOf(w) != 401 {
		t.Fatal("should be unauthorized")
	}
}

func TestAdminAppsListAndDelete(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

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
	s.SVC.DB.Model(&model.App{}).Count(&count)
	if count != 0 {
		t.Fatalf("apps after delete = %d", count)
	}
}

func TestAdminUploadAndSetIcon(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

	// The single /files presign returns a key scoped to the app; bytes are
	// pushed there by the client, then the icon key is submitted on update.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader(`{"app_id":`+itoa(app.ID)+`,"file_name":"icon.png"}`))
	w := httptest.NewRecorder()
	s.PresignFile(w, req)

	var res struct {
		Code int    `json:"code"`
		Data ticket `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("res = %s", w.Body.String())
	}
	if res.Data.URL == "" || res.Data.Key == "" {
		t.Fatalf("ticket missing fields: %+v", res.Data)
	}
	if want := "disapp/" + itoa(app.ID) + "/0/"; !strings.HasPrefix(res.Data.Key, want) {
		t.Fatalf("key = %q, want prefix %q", res.Data.Key, want)
	}
	loc, _ := s.SVC.Storage.(*local.LocalStorage)
	storeBytes(t, s, res.Data.Key, pngData)

	upReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"icon":"`+res.Data.Key+`"}`))
	upReq.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), upReq)

	var reload model.App
	s.SVC.DB.First(&reload, app.ID)
	if reload.Icon != res.Data.Key {
		t.Fatalf("icon key = %q, want %q", reload.Icon, res.Data.Key)
	}
	rc, err := loc.Open(context.TODO(), res.Data.Key)
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
	if _, err := loc.Open(context.TODO(), res.Data.Key); err == nil {
		t.Fatal("icon file should be deleted with the app")
	}
}

func TestAdminPresignRejectsMissingAppID(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader(`{"file_name":"icon.png"}`))
	w := httptest.NewRecorder()
	s.PresignFile(w, req)
	if codeOf(w) == 0 {
		t.Fatalf("expected rejection, res = %s", w.Body.String())
	}
}

func TestUpdateAppAccessPassword(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)

	// Set the password; the credential must be hashed and stored.
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
	s.SVC.DB.First(&reload, app.ID)
	if reload.PasswordHash == "" || reload.Salt == "" {
		t.Fatalf("reload = %+v", reload)
	}

	// Switching back to public clears the stored credential.
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"access_mode":"public"}`))
	req2.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), req2)
	s.SVC.DB.First(&reload, app.ID)
	if reload.PasswordHash != "" || reload.Salt != "" {
		t.Fatalf("reload after clear = %+v", reload)
	}
}

func TestAdminScreenshotPresignAndDelete(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)
	loc, _ := s.SVC.Storage.(*local.LocalStorage)

	// Presign + write bytes at the returned key, then submit it on update.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader(`{"app_id":`+itoa(app.ID)+`,"file_name":"shot.png"}`))
	w := httptest.NewRecorder()
	s.PresignFile(w, req)
	var res struct {
		Code int    `json:"code"`
		Data ticket `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.Key == "" {
		t.Fatalf("res = %s", w.Body.String())
	}
	key := res.Data.Key
	storeBytes(t, s, key, pngData)
	if rc, err := loc.Open(context.TODO(), key); err != nil {
		t.Fatalf("screenshot not stored: %v", err)
	} else {
		got, _ := io.ReadAll(rc)
		rc.Close()
		if !bytes.Equal(got, pngData) {
			t.Fatalf("screenshot bytes mismatch: got %d bytes", len(got))
		}
	}
	upReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"screenshots":["`+key+`"]}`))
	upReq.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), upReq)
	var reload model.App
	s.SVC.DB.First(&reload, app.ID)
	if len(reload.Screenshots) != 1 || reload.Screenshots[0] != key {
		t.Fatalf("screenshots = %v", reload.Screenshots)
	}

	// Deleting the app removes the screenshot file too.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID), nil)
	delReq.SetPathValue("id", itoa(app.ID))
	s.DeleteApp(httptest.NewRecorder(), delReq)
	if _, err := loc.Open(context.TODO(), key); err == nil {
		t.Fatal("screenshot file should be deleted with the app")
	}
}

func TestAdminDeleteAppScreenshot(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "a"}
	s.SVC.DB.Create(&app)
	loc, _ := s.SVC.Storage.(*local.LocalStorage)

	// Presign two, write bytes, submit both, then delete one by key.
	keys := []string{}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader(`{"app_id":`+itoa(app.ID)+`,"file_name":"shot.png"}`))
		w := httptest.NewRecorder()
		s.PresignFile(w, req)
		var res struct {
			Code int    `json:"code"`
			Data ticket `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		if res.Code != 0 {
			t.Fatalf("res = %s", w.Body.String())
		}
		keys = append(keys, res.Data.Key)
		storeBytes(t, s, res.Data.Key, pngData)
	}
	if len(keys) != 2 {
		t.Fatalf("keys = %d", len(keys))
	}
	upReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/apps/"+itoa(app.ID), strings.NewReader(`{"screenshots":["`+keys[0]+`","`+keys[1]+`"]}`))
	upReq.SetPathValue("id", itoa(app.ID))
	s.UpdateApp(httptest.NewRecorder(), upReq)
	var app2 model.App
	s.SVC.DB.First(&app2, app.ID)
	if len(app2.Screenshots) != 2 {
		t.Fatalf("screenshots = %v", app2.Screenshots)
	}

	del := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/apps/"+itoa(app.ID)+"/screenshots?url="+keys[0], nil)
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
	if res.Code != 0 || len(res.Data.Screenshots) != 1 || res.Data.Screenshots[0] != keys[1] {
		t.Fatalf("res = %s", w.Body.String())
	}
	if _, err := loc.Open(context.TODO(), keys[0]); err == nil {
		t.Fatal("deleted screenshot file should be gone")
	}
}
