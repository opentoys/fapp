package service

import (
	"context"
	"net/http"
	"sort"
	"time"

	"disapp/internal/resources/store/model"
)

// DownloadsTimeSeries returns daily download counts for an app, optionally
// filtered by version. "total" aggregates all versions; "selected" is the
// filtered subset (null when no filter). Dates are zero-filled across the
// app's full download history.
func (s *Service) DownloadsTimeSeries(ctx context.Context, appID, versionID int64) (map[string]any, error) {
	var app model.App
	if err := s.DB.First(&app, appID).Error; err != nil {
		return nil, &Error{http.StatusNotFound, "应用不存在"}
	}
	query := func(versionID int64) map[string]int {
		q := s.DB.Table("download_logs l").
			Joins("JOIN versions v ON v.id = l.version_id").
			Where("v.app_id = ?", appID)
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

	total := query(0)
	if total == nil {
		return nil, &Error{http.StatusInternalServerError, "查询失败"}
	}
	selected := map[string]int(nil)
	if versionID != 0 {
		selected = query(versionID)
		if selected == nil {
			return nil, &Error{http.StatusInternalServerError, "查询失败"}
		}
	}
	dates, totals, sels := buildDailySeries(total, selected)
	return map[string]any{
		"dates":    dates,
		"total":    totals,
		"selected": sels,
	}, nil
}

// buildDailySeries zero-fills the map from its earliest to its latest day.
// sel is nil when only the total line should be shown.
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
