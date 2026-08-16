package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"disapp/internal/resources/config"
	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/db"
	"disapp/internal/resources/store/model"
)

func testService(t *testing.T) *Service {
	t.Helper()
	gdb, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := local.NewLocal(filepath.Join(t.TempDir(), "files"), config.Default().JWT.Secret)
	if err != nil {
		t.Fatal(err)
	}
	return New(gdb, st, config.Default())
}

func TestFillParams(t *testing.T) {
	p := NotifyParams{"event": "版本上传", "app_name": "微信"}
	cases := []struct{ in, want string }{
		{"{{event}}", "版本上传"},
		{"hello {{ app_name }}", "hello 微信"},
		{"{{unknown}} stays", "{{unknown}} stays"},
		{"no placeholders here", "no placeholders here"},
		{"{{event}}-{{app_name}}", "版本上传-微信"},
		{"{{app_name}{{event}}", "{{app_name}{{event}}"},
		{"", ""},
	}
	for _, c := range cases {
		if got := fillParams(c.in, p); got != c.want {
			t.Fatalf("fillParams(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCompileForBotOverridesAppID(t *testing.T) {
	bot := &model.NotificationBot{AppID: 7}
	p := compileForBot(bot, NotifyParams{"app_id": "1", "app_name": "x"})
	if p["app_id"] != "7" {
		t.Fatalf("app_id = %q, want bot's app id", p["app_id"])
	}
	if p["app_name"] != "x" {
		t.Fatalf("app_name lost: %q", p["app_name"])
	}
}

func TestHeadermap(t *testing.T) {
	h := headermap([]string{"Authorization: token abc", "X-App: 微信", "BadLine"})
	if h.Get("Authorization") != "token abc" {
		t.Fatalf("Authorization = %q", h.Get("Authorization"))
	}
	if h.Get("X-App") != "微信" {
		t.Fatalf("X-App = %q", h.Get("X-App"))
	}
	if h.Get("BadLine") != "" {
		t.Fatal("malformed header line must be ignored")
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

func TestNotifyEventParamsTriggersMatchingBots(t *testing.T) {
	s := testService(t)
	app := model.App{Name: "微信", Platform: "android"}
	s.DB.Create(&app)

	var gotPath, gotBody string
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath = r.URL.Path
		gotBody = strings.TrimSpace(string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	// One bot subscribed to the fired event, one to a different event.
	s.DB.Create(&model.NotificationBot{Name: "match", AppID: app.ID, URL: hook.URL + "/{{app_id}}/{{event_key}}", Method: "POST", BodyTemplate: `{"version":"{{version_name}}"}` , Events: []string{model.EventVersionUploaded}})
	s.DB.Create(&model.NotificationBot{Name: "skip", AppID: app.ID, URL: hook.URL + "/skip", Method: "POST", Events: []string{model.EventAppPublish}})

	s.NotifyEventParams(context.Background(), app.ID, model.EventVersionUploaded, "微信", NotifyParams{"version_name": "9.9.9"})

	waitFor(t, func() bool { return gotPath != "" })
	if want := fmt.Sprintf("/%d/version_uploaded", app.ID); gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
	if !strings.Contains(gotBody, "9.9.9") {
		t.Fatalf("body missing version_name: %q", gotBody)
	}

	// Exactly one log row: the matched bot, not the skipped one.
	var n int64
	s.DB.Model(&model.NotificationLog{}).Count(&n)
	if n != 1 {
		t.Fatalf("log rows = %d, want 1", n)
	}
}

func TestExpiryScanFiresOncePerApp(t *testing.T) {
	s := testService(t)
	past := time.Now().Add(-time.Hour)
	app := model.App{Name: "过期应用", Platform: "ios", ExpiresAt: &past}
	s.DB.Create(&app)

	hit := make(chan struct{}, 4)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()
	s.DB.Create(&model.NotificationBot{Name: "exp", AppID: app.ID, URL: hook.URL, Method: "POST", Events: []string{model.EventAppExpire}})

	// First scan fires the app_expire notification.
	s.ExpiryScan(context.Background())
	waitFor(t, func() bool { return len(hit) > 0 })
	var n int64
	s.DB.Model(&model.NotificationLog{}).Where("app_id = ? AND event = ?", app.ID, model.EventAppExpire).Count(&n)
	if n != 1 {
		t.Fatalf("app_expire log rows = %d, want 1", n)
	}

	// Second scan must be deduped by the persisted log row (no re-fire).
	s.ExpiryScan(context.Background())
	time.Sleep(50 * time.Millisecond)
	s.DB.Model(&model.NotificationLog{}).Where("app_id = ? AND event = ?", app.ID, model.EventAppExpire).Count(&n)
	if n != 1 {
		t.Fatalf("app_expire log rows after re-scan = %d, want 1 (dedupe)", n)
	}
}

