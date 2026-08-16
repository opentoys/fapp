package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"disapp/internal/resources/store/model"
	"disapp/pkg/token"
)

// superLogin authenticates as the config super-admin (uid=-1), so subscription
// management tests exercise the full admin permission path.
func superLogin(t *testing.T, s *Controller) string {
	t.Helper()
	s.SVC.Config.Admin.Username = "root"
	s.SVC.Config.Admin.Password = "s3cret"
	body := bytes.NewBufferString(`{"username":"root","password":"s3cret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	w := httptest.NewRecorder()
	s.Login(w, req)
	var res struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.Token == "" {
		t.Fatalf("no token: %s", w.Body.String())
	}
	return res.Data.Token
}

func createBotBody(appID int64) string {
	return `{"name":"钉钉","app_id":` + itoa(appID) + `,"method":"POST","url":"https://example.com/hook","headers":["X-App: demo"],"body_template":"{\"e\":\"{{event}}\"}","events":["version_uploaded","app_expire"]}`
}

func TestSubscriptionCRUD(t *testing.T) {
	s := testServer(t)
	token := superLogin(t, s)
	app := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app)

	// Create
	req := authReq(http.MethodPost, "/api/v1/admin/subscriptions", token, []byte(createBotBody(app.ID)))
	w := httptest.NewRecorder()
	s.RequireAuth(s.CreateSubscription)(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			ID    int64    `json:"id"`
			Name  string   `json:"name"`
			Events []string `json:"events"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || res.Data.Name != "钉钉" || len(res.Data.Events) != 2 {
		t.Fatalf("create res = %s", w.Body.String())
	}
	botID := res.Data.ID

	// List
	req = authReq(http.MethodGet, "/api/v1/admin/subscriptions", token, nil)
	w = httptest.NewRecorder()
	s.RequireAuth(s.SubscriptionsList)(w, req)
	var list struct {
		Code int `json:"code"`
		Data []model.NotificationBot `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if list.Code != 0 || len(list.Data) != 1 {
		t.Fatalf("list res = %s", w.Body.String())
	}

	// Update
	upd := `{"name":"企业微信","app_id":` + itoa(app.ID) + `,"method":"GET","url":"https://example.com/h2","events":["version_current"]}`
	req = authReq(http.MethodPut, "/api/v1/admin/subscriptions/"+itoa(botID), token, []byte(upd))
	req.SetPathValue("id", itoa(botID))
	w = httptest.NewRecorder()
	s.RequireAuth(s.UpdateSubscription)(w, req)
	var updRes struct {
		Code int `json:"code"`
		Data struct {
			Name   string `json:"name"`
			Method string `json:"method"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &updRes)
	if updRes.Code != 0 || updRes.Data.Name != "企业微信" || updRes.Data.Method != "GET" {
		t.Fatalf("update res = %s", w.Body.String())
	}

	// Delete
	req = authReq(http.MethodDelete, "/api/v1/admin/subscriptions/"+itoa(botID), token, nil)
	req.SetPathValue("id", itoa(botID))
	w = httptest.NewRecorder()
	s.RequireAuth(s.DeleteSubscription)(w, req)
	var count int64
	s.SVC.DB.Model(&model.NotificationBot{}).Count(&count)
	if count != 0 {
		t.Fatalf("bots after delete = %d", count)
	}
}

func TestSubscriptionValidation(t *testing.T) {
	s := testServer(t)
	token := superLogin(t, s)
	app := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app)

	cases := []struct{ name, body string }{
		{"missing name", `{"app_id":` + itoa(app.ID) + `,"method":"POST","url":"http://x","events":["version_uploaded"]}`},
		{"bad method", `{"name":"n","app_id":` + itoa(app.ID) + `,"method":"DELETE","url":"http://x","events":["version_uploaded"]}`},
		{"non-http url", `{"name":"n","app_id":` + itoa(app.ID) + `,"method":"POST","url":"ftp://x","events":["version_uploaded"]}`},
		{"no events", `{"name":"n","app_id":` + itoa(app.ID) + `,"method":"POST","url":"http://x"}`},
		{"unknown event", `{"name":"n","app_id":` + itoa(app.ID) + `,"method":"POST","url":"http://x","events":["nope"]}`},
		{"missing app", `{"name":"n","app_id":99999,"method":"POST","url":"http://x","events":["version_uploaded"]}`},
	}
	for _, c := range cases {
		req := authReq(http.MethodPost, "/api/v1/admin/subscriptions", token, []byte(c.body))
		w := httptest.NewRecorder()
		s.RequireAuth(s.CreateSubscription)(w, req)
		if codeOf(w) == 0 {
			t.Fatalf("%s: expected rejection, got %s", c.name, w.Body.String())
		}
	}
}

func TestSubscriptionRequiresAuth(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/subscriptions", nil)
	w := httptest.NewRecorder()
	s.SubscriptionsList(w, req)
	if codeOf(w) != 401 {
		t.Fatalf("unauthenticated list code = %d, want 401", codeOf(w))
	}
}

// waitFor polls cond until it returns true or the deadline passes.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestSubscriptionPermission(t *testing.T) {
	s := testServer(t)
	superTok := superLogin(t, s)
	app := model.App{Name: "a", Platform: "android"}
	s.SVC.DB.Create(&app)

	// A non-super user who is NOT a member cannot create a bot on the app.
	user := model.User{Username: "bob", PasswordHash: "h", Salt: "s"}
	s.SVC.DB.Create(&user)
	bobTok, err := token.CreateToken(s.SVC.Config.JWT.Secret, user.ID, "bob", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := authReq(http.MethodPost, "/api/v1/admin/subscriptions", bobTok, []byte(createBotBody(app.ID)))
	w := httptest.NewRecorder()
	s.RequireAuth(s.CreateSubscription)(w, req)
	if codeOf(w) != 403 {
		t.Fatalf("non-member create code = %d, want 403", codeOf(w))
	}

	// The super-admin's own list still shows nothing (no bots created).
	req = authReq(http.MethodGet, "/api/v1/admin/subscriptions", superTok, nil)
	w = httptest.NewRecorder()
	s.RequireAuth(s.SubscriptionsList)(w, req)
	var list struct {
		Code int `json:"code"`
		Data []model.NotificationBot `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if list.Code != 0 || len(list.Data) != 0 {
		t.Fatalf("list res = %s", w.Body.String())
	}
}

func TestSubscriptionVersionUploadFires(t *testing.T) {
	s := testServer(t)
	app := model.App{Name: "微信", Platform: "android"}
	s.SVC.DB.Create(&app)

	got := make(chan string, 2)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 4096)
		n, _ := r.Body.Read(b)
		got <- r.URL.Path + "|" + string(b[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	s.SVC.DB.Create(&model.NotificationBot{
		Name: "upload-hook", AppID: app.ID, Method: "POST",
		URL: hook.URL + "/{{app_id}}", BodyTemplate: `{"name":"{{app_name}}"}`,
		Events: []string{model.EventVersionUploaded},
	})

	createVersion(t, s, app.ID, map[string]string{"version_name": "3.1.4"})

	select {
	case msg := <-got:
		if !bytes.Contains([]byte(msg), []byte(itoa(app.ID))) {
			t.Fatalf("hook path missing app id: %s", msg)
		}
		if !bytes.Contains([]byte(msg), []byte("微信")) {
			t.Fatalf("hook body missing app name: %s", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("hook not called after version upload")
	}

	// The send goroutine writes the NotificationLog row after the webhook
	// returns; wait for it so the test's temp dir stays alive.
	waitFor(t, func() bool {
		var n int64
		s.SVC.DB.Model(&model.NotificationLog{}).Where("event = ?", model.EventVersionUploaded).Count(&n)
		return n == 1
	})
}
