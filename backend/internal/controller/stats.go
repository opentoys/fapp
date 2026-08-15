package controller

import (
	"net/http"
	"strconv"

	"disapp/pkg/web"
)

// DownloadsTimeSeries returns daily download counts for an app, optionally
// filtered by version.
func (c *Controller) DownloadsTimeSeries(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad request")
		return
	}
	versionID, _ := strconv.ParseInt(r.URL.Query().Get("version_id"), 10, 64)
	out, err := c.SVC.DownloadsTimeSeries(r.Context(), appID, versionID)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, out)
}