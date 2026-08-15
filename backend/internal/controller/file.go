package controller

import (
	"fmt"
	"io"
	"net/http"
	"path"
)

// File serves local storage files via streaming proxy.
func (c *Controller) File(w http.ResponseWriter, r *http.Request) {
	rc, err := c.SVC.OpenFile(r.Context(), r.PathValue("key"))
	if err != nil {
		sendStatusErr(w, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(r.PathValue("key"))))
	io.Copy(w, rc)
}