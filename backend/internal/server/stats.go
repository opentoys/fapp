package server

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"disapp/internal/model"
	"disapp/internal/web"
)

// DownloadsTimeSeries returns daily download counts for an app, optionally
// filtered by platform and/or version. "total" aggregates all of the app's
// versions; "selected" is the filtered subset (null when no filter is set).
// Dates are zero-filled across the app's full download history so the two
// series always share the same x axis.
//
// The driver stores created_at as RFC3339 text and reformats any date-like
// string it scans, so day bucketing happens in Go on the parsed time.Time
// (which GORM restores with the server's local location).
func (s *Server) DownloadsTimeSeries(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		web.SendError(w, web.CodeNotFound, "应用不存在")
		return
	}

	platform := r.URL.Query().Get("platform")
	versionID, _ := strconv.ParseInt(r.URL.Query().Get("version_id"), 10, 64)

	// Each call starts a fresh query chain; GORM shares the statement across
	// chained calls, so reusing one base would accumulate JOIN/WHERE clauses.
	query := func(platform string, versionID int64) map[string]int {
		q := s.DB.Table("download_logs l").
			Joins("JOIN versions v ON v.id = l.version_id").
			Where("v.app_id = ?", appID)
		if platform != "" {
			q = q.Where("v.platform = ?", platform)
		}
		if versionID != 0 {
			q = q.Where("l.version_id = ?", versionID)
		}
		var rows []struct {
			CreatedAt time.Time
		}
		if err := q.Select("l.created_at").Scan(&rows).Error; err != nil {
			return nil
		}
		m := make(map[string]int, len(rows))
		for _, r := range rows {
			m[r.CreatedAt.Format("2006-01-02")]++
		}
		return m
	}

	total := query(platform, 0)
	if total == nil {
		web.SendError(w, web.CodeInternal, "查询失败")
		return
	}

	selected := map[string]int(nil)
	if platform != "" || versionID != 0 {
		selected = query(platform, versionID)
		if selected == nil {
			web.SendError(w, web.CodeInternal, "查询失败")
			return
		}
	}

	dates, totals, sels := buildDailySeries(total, selected)
	web.SendJson(w, map[string]any{
		"dates":    dates,
		"total":    totals,
		"selected": sels, // null when no filter active
	})
}

// buildDailySeries zero-fills the map from its earliest to its latest day.
// sel is nil when the chart should only show the total line.
func buildDailySeries(total, sel map[string]int) (dates []string, totals, sels []int) {
	if len(total) == 0 {
		return []string{}, []int{}, nil
	}
	keys := make([]string, 0, len(total))
	for d := range total {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	cur, _ := time.Parse("2006-01-02", keys[0])
	end, _ := time.Parse("2006-01-02", keys[len(keys)-1])
	for ; !cur.After(end); cur = cur.AddDate(0, 0, 1) {
		day := cur.Format("2006-01-02")
		dates = append(dates, day)
		totals = append(totals, total[day])
		if sel != nil {
			sels = append(sels, sel[day])
		}
	}
	return dates, totals, sels
}
