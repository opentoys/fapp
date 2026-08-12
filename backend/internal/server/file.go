package server

import (
	"fmt"
	"io"
	"net/http"
	"path"

	"disapp/internal/storage"
	"disapp/internal/web"
)

// File serves local storage files via streaming proxy.
func (s *Server) File(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !storage.ValidKey(key) {
		web.SendStatus(w, http.StatusBadRequest, "invalid key")
		return
	}
	rc, err := s.Storage.Open(r.Context(), key)
	if err != nil {
		web.SendStatus(w, http.StatusNotFound, "file not found")
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(key)))
	io.Copy(w, rc)
}
