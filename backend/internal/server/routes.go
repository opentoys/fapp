package server

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"disapp/internal/web"
)

// Routes assembles all routes and static file serving.
func (s *Server) Routes(dist fs.FS) http.Handler {
	mux := http.NewServeMux()

	pub := []web.Middleware{web.Recoverer, web.Logger}
	admin := append([]web.Middleware{}, pub...)
	admin = append(admin, s.RequireAuth)

	login := append([]web.Middleware{}, pub...)
	login = append(login, web.RateLimit(10, time.Minute))
	verify := append([]web.Middleware{}, pub...)
	verify = append(verify, web.RateLimit(30, time.Minute))

	mux.HandleFunc("POST /api/v1/auth/login", web.Chain(login...)(s.Login))

	mux.HandleFunc("GET /api/v1/apps", web.Chain(pub...)(s.Apps))
	mux.HandleFunc("GET /api/v1/apps/{id}", web.Chain(pub...)(s.AppDetail))
	mux.HandleFunc("POST /api/v1/versions/{id}/verify", web.Chain(verify...)(s.VerifyAccess))
	mux.HandleFunc("GET /api/v1/versions/{id}/install", web.Chain(pub...)(s.Install))
	mux.HandleFunc("GET /api/v1/versions/{id}/download", web.Chain(pub...)(s.Download))
	mux.HandleFunc("GET /api/v1/files/{key...}", web.Chain(pub...)(s.File))

	mux.HandleFunc("GET /api/v1/admin/apps", web.Chain(admin...)(s.AppsList))
	mux.HandleFunc("POST /api/v1/admin/apps", web.Chain(admin...)(s.CreateApp))
	mux.HandleFunc("PUT /api/v1/admin/apps/{id}", web.Chain(admin...)(s.UpdateApp))
	mux.HandleFunc("DELETE /api/v1/admin/apps/{id}", web.Chain(admin...)(s.DeleteApp))
	mux.HandleFunc("GET /api/v1/admin/users", web.Chain(admin...)(s.UsersList))
	mux.HandleFunc("POST /api/v1/admin/users", web.Chain(admin...)(s.CreateUser))
	mux.HandleFunc("PUT /api/v1/admin/users/{id}", web.Chain(admin...)(s.UpdateUser))
	mux.HandleFunc("DELETE /api/v1/admin/users/{id}", web.Chain(admin...)(s.DeleteUser))
	mux.HandleFunc("GET /api/v1/admin/channels", web.Chain(admin...)(s.ChannelsList))
	mux.HandleFunc("POST /api/v1/admin/channels", web.Chain(admin...)(s.CreateChannel))
	mux.HandleFunc("POST /api/v1/admin/versions", web.Chain(admin...)(s.UploadVersion))
	mux.HandleFunc("PUT /api/v1/admin/versions/{id}", web.Chain(admin...)(s.UpdateVersion))
	mux.HandleFunc("DELETE /api/v1/admin/versions/{id}", web.Chain(admin...)(s.DeleteVersion))
	mux.HandleFunc("GET /api/v1/admin/versions/{id}/stats", web.Chain(admin...)(s.VersionStats))

	if dist != nil {
		mux.Handle("/", staticHandler(dist))
	}
	return mux
}

// staticHandler serves SPA: serves the file if it exists, otherwise falls
// back to index.html. Uses embed.FS directly to avoid http.FileServerFS's
// internal-redirect quirks with re-rooted fs.FS.
func staticHandler(dist fs.FS) http.HandlerFunc {
	indexBytes := func() []byte {
		f, err := dist.Open("index.html")
		if err != nil {
			return nil
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		return b
	}
	idx := indexBytes()
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		// For paths with extensions, require the file to exist.
		if strings.Contains(p, ".") {
			f, err := dist.Open(p)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, _ := f.Stat()
			if ct := mime.TypeByExtension(filepath.Ext(p)); ct != "" {
				w.Header().Set("Content-Type", ct)
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
			}
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			io.Copy(w, f)
			return
		}
		// SPA fallback for paths without an extension.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(idx)))
		w.Write(idx)
	}
}
