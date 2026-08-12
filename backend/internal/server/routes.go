package server

import (
	"io/fs"
	"net/http"
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

// staticHandler serves SPA: fallback to index.html for unknown paths.
func staticHandler(dist fs.FS) http.HandlerFunc {
	fileServer := http.FileServerFS(dist)
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			p = "index.html"
		}
		r.URL.Path = "/" + p
		fileServer.ServeHTTP(w, r)
	}
}
