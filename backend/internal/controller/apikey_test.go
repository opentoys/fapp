package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"disapp/internal/resources/store/model"
	"disapp/internal/service"
)

// seedMember binds userID to appID in app_members.
func seedMember(t *testing.T, s *Controller, userID, appID int64) {
	t.Helper()
	if err := s.SVC.DB.Create(&model.AppMember{UserID: userID, AppID: appID}).Error; err != nil {
		t.Fatal(err)
	}
}

// callKeys invokes a key-management handler behind RequireAuth (the real
// middleware that fills the user context) and returns the JSON body.
func callKeys(t *testing.T, s *Controller, h http.HandlerFunc, method, path, token string, body []byte) (int, []byte) {
	t.Helper()
	req := authReq(method, path, token, body)
	// Last path segment is the {id} param for PUT/DELETE routes.
	if i := strings.LastIndex(path, "/"); i >= 0 {
		req.SetPathValue("id", path[i+1:])
	}
	w := httptest.NewRecorder()
	s.RequireAuth(h)(w, req)
	return w.Code, w.Body.Bytes()
}

// makeKey creates a key directly in DB (helper) and returns plain text.
func makeKey(t *testing.T, s *Controller, name, scope string, uid int64) string {
	t.Helper()
	k := model.ApiKey{Name: name, Key: "dk_" + name, UserID: uid, Scope: scope}
	if err := s.SVC.DB.Create(&k).Error; err != nil {
		t.Fatal(err)
	}
	return k.Key
}

func TestCreateKeyValidatesScopeAndReturnsPlaintext(t *testing.T) {
	s := testServer(t)
	token := adminLogin(t, s)

	code, body := callKeys(t, s, s.CreateKey, http.MethodPost, "/api/v1/admin/keys", token,
		[]byte(`{"name":"ci","scope":"run"}`))
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	var res struct {
		Code int `json:"code"`
		Data struct {
			Key   string `json:"key"`
			Scope string `json:"scope"`
		} `json:"data"`
	}
	json.Unmarshal(body, &res)
	if res.Code != 0 || res.Data.Key == "" || len(res.Data.Key) < 10 {
		t.Fatalf("res=%s", body)
	}
	if !bytes.HasPrefix([]byte(res.Data.Key), []byte(service.KeyPrefix)) {
		t.Fatalf("key must start with %s, got %q", service.KeyPrefix, res.Data.Key)
	}

	// Invalid scope rejected.
	code, body = callKeys(t, s, s.CreateKey, http.MethodPost, "/api/v1/admin/keys", token,
		[]byte(`{"name":"x","scope":"admin"}`))
	var res2 struct {
		Code int `json:"code"`
	}
	json.Unmarshal(body, &res2)
	if res2.Code == 0 {
		t.Fatalf("invalid scope must be rejected: %s", body)
	}
}

func TestKeysVisibility(t *testing.T) {
	s := testServer(t)
	admin := adminLogin(t, s) // adminLogin creates a DB user "admin"
	var u model.User
	s.SVC.DB.Where("username = ?", "admin").First(&u)

	makeKey(t, s, "k1", model.KeyScopeRun, u.ID)
	makeKey(t, s, "k2", model.KeyScopeRead, 999) // other user

	// Ordinary user sees only own keys.
	code, body := callKeys(t, s, s.KeysList, http.MethodGet, "/api/v1/admin/keys", admin, nil)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	var res struct {
		Data []map[string]any `json:"data"`
	}
	json.Unmarshal(body, &res)
	if len(res.Data) != 1 || res.Data[0]["name"] != "k1" {
		t.Fatalf("user should see only own key, got %s", body)
	}
}

func TestKeyOwnershipEdit(t *testing.T) {
	s := testServer(t)
	admin := adminLogin(t, s)
	var u model.User
	s.SVC.DB.Where("username = ?", "admin").First(&u)

	k := model.ApiKey{Name: "ci", Key: "dk_ci", UserID: u.ID, Scope: model.KeyScopeRun}
	s.SVC.DB.Create(&k)
	// Other user's key.
	other := model.ApiKey{Name: "other", Key: "dk_other", UserID: 555, Scope: model.KeyScopeRun}
	s.SVC.DB.Create(&other)

	// Owner edits own key.
	code, body := callKeys(t, s, s.UpdateKey, http.MethodPut, "/api/v1/admin/keys/"+itoa(k.ID), admin,
		[]byte(`{"scope":"read"}`))
	if code != http.StatusOK {
		t.Fatalf("owner edit failed: %s", body)
	}

	// Editing someone else's key is forbidden.
	code, body = callKeys(t, s, s.UpdateKey, http.MethodPut, "/api/v1/admin/keys/"+itoa(other.ID), admin,
		[]byte(`{"name":"hijack"}`))
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(body, &res)
	if res.Code != 403 {
		t.Fatalf("cross-user edit must be 403, got %s", body)
	}
}

// --- external key API ---

func TestKeyApiScopeGating(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	readKey := makeKey(t, s, "read", model.KeyScopeRead, service.SuperAdminUID)

	// Super-admin can manage everything; read-scope key cannot write.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/"+itoa(app.ID)+"/current?apikey="+readKey,
		bytes.NewBufferString(`{"version_id":1}`))
	req.SetPathValue("app_id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.SetKeyCurrentVersion(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 403 {
		t.Fatalf("read key must not set current version, got %s", w.Body.String())
	}
}

func TestKeyApiUnauthorizedForUnmanagedApp(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	runKey := makeKey(t, s, "run", model.KeyScopeRun, 4242) // not super, not member

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+itoa(app.ID)+"/versions?apikey="+runKey, nil)
	req.SetPathValue("app_id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.KeyVersionsList(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 403 {
		t.Fatalf("unmanaged app must be 403, got %s", w.Body.String())
	}
}

func TestKeyApiExpiredHashed(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	past := time.Now().Add(-time.Hour)
	k := model.ApiKey{Name: "old", Key: "dk_old", UserID: service.SuperAdminUID, Scope: model.KeyScopeRun, ExpiresAt: &past}
	s.SVC.DB.Create(&k)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+itoa(app.ID)+"/current?apikey=dk_old", nil)
	req.SetPathValue("app_id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.KeyCurrentVersion(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 401 {
		t.Fatalf("expired key must be 401, got %s", w.Body.String())
	}
}

func TestKeyApiSetCurrentAndTouch(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	runKey := makeKey(t, s, "run", model.KeyScopeRun, service.SuperAdminUID)
	var k model.ApiKey
	s.SVC.DB.Where("key = ?", runKey).First(&k)

	v := model.Version{AppID: app.ID, VersionName: "1.0.0", VersionCode: 1, FileName: "a.apk", FileType: "apk", StorageKey: "wechat/1/2/a.apk"}
	s.SVC.DB.Create(&v)

	body := `{"version_id":` + itoa(v.ID) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/"+itoa(app.ID)+"/current?apikey="+runKey, bytes.NewBufferString(body))
	req.SetPathValue("app_id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.SetKeyCurrentVersion(w, req)
	var res struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 {
		t.Fatalf("set-current failed: %s", w.Body.String())
	}

	var reload model.App
	s.SVC.DB.First(&reload, app.ID)
	if reload.CurrentVersionID != v.ID {
		t.Fatalf("current_version_id = %d", reload.CurrentVersionID)
	}
	s.SVC.DB.First(&k, k.ID)
	if k.LastUsedAt == nil {
		t.Fatal("last_used_at should be touched")
	}
}

func TestKeyApiVersionsList(t *testing.T) {
	s := testServer(t)
	app := seedApp(t, s)
	runKey := makeKey(t, s, "run", model.KeyScopeRun, service.SuperAdminUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys/"+itoa(app.ID)+"/versions?apikey="+runKey, nil)
	req.SetPathValue("app_id", itoa(app.ID))
	w := httptest.NewRecorder()
	s.KeyVersionsList(w, req)
	var res struct {
		Code int    `json:"code"`
		Data struct {
			Versions []model.Version `json:"versions"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data.Versions) != 3 {
		t.Fatalf("expected 3 versions (seedApp), got %s", w.Body.String())
	}
}