package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"disapp/internal/resources/store/model"
)

// seedDownloads creates an app with two versions and download logs spread
// over a few days, returning the app and the two versions.
func seedDownloads(t *testing.T, s *Server) (*model.App, *model.Version, *model.Version) {
	t.Helper()
	app := model.App{Name: "统计测试"}
	if err := s.DB.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	android := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 1, Platform: "android",
		FileName: "a.apk", FileType: "apk",
	}
	ios := model.Version{
		AppID: app.ID, VersionName: "1.0.0", VersionCode: 1, Platform: "ios",
		FileName: "a.ipa", FileType: "ipa",
	}
	if err := s.DB.Create(&android).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&ios).Error; err != nil {
		t.Fatal(err)
	}

	// Day 1: android 2x, ios 1x  → 3 total
	// Day 3: ios 1x               → 1 total
	// Day 5: android 1x           → 1 total
	day := func(d int) time.Time { return time.Date(2026, 8, d, 10, 0, 0, 0, time.Local) }
	logs := []model.DownloadLog{
		{VersionID: android.ID, CreatedAt: day(10)},
		{VersionID: android.ID, CreatedAt: day(10)},
		{VersionID: ios.ID, CreatedAt: day(10)},
		{VersionID: ios.ID, CreatedAt: day(12)},
		{VersionID: android.ID, CreatedAt: day(14)},
	}
	for i := range logs {
		if err := s.DB.Create(&logs[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	return &app, &android, &ios
}

func TestDownloadsTimeSeries(t *testing.T) {
	s := testServer(t)
	app, _, ios := seedDownloads(t, s)

	get := func(qs string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apps/"+itoa(app.ID)+"/downloads"+qs, nil)
		req.SetPathValue("id", itoa(app.ID))
		w := httptest.NewRecorder()
		s.DownloadsTimeSeries(w, req)
		var res struct {
			Code int            `json:"code"`
			Data map[string]any `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		if res.Code != 0 {
			t.Fatalf("code=%d body=%s", res.Code, w.Body.String())
		}
		return res.Data
	}

	// Unfiltered: 5-day range (08-10..08-14), zero-filled, selected is null.
	d := get("")
	dates, _ := d["dates"].([]any)
	total, _ := d["total"].([]any)
	if len(dates) != 5 || dates[0] != "2026-08-10" || dates[4] != "2026-08-14" {
		t.Fatalf("dates = %v", dates)
	}
	if total[0].(float64) != 3 || total[1].(float64) != 0 || total[2].(float64) != 1 || total[4].(float64) != 1 {
		t.Fatalf("total = %v", total)
	}
	if d["selected"] != nil {
		t.Fatalf("selected should be null without filter, got %v", d["selected"])
	}

	// Filter by version: only that version's rows, still full date range.
	d = get("?version_id=" + itoa(ios.ID))
	sel, _ := d["selected"].([]any)
	if len(sel) != 5 || sel[0].(float64) != 1 || sel[2].(float64) != 1 || sel[4].(float64) != 0 {
		t.Fatalf("selected = %v", sel)
	}

	// Empty app → empty series, selected null.
	s2 := testServer(t)
	empty := model.App{Name: "empty"}
	s2.DB.Create(&empty)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/apps/"+itoa(empty.ID)+"/downloads", nil)
	req.SetPathValue("id", itoa(empty.ID))
	w := httptest.NewRecorder()
	s2.DownloadsTimeSeries(w, req)
	var res struct {
		Code int `json:"code"`
		Data struct {
			Dates []int `json:"dates"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Code != 0 || len(res.Data.Dates) != 0 {
		t.Fatalf("empty app: code=%d dates=%v", res.Code, res.Data.Dates)
	}
}
