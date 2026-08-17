package router

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"disapp/internal/controller"
	"disapp/pkg/web"
)

// Routes assembles all routes and static file serving.
func Routes(c *controller.Controller, dist fs.FS) http.Handler {
	mux := http.NewServeMux()

	pub := []web.Middleware{web.Recoverer, web.Logger}
	admin := append([]web.Middleware{}, pub...)
	admin = append(admin, c.RequireAuth)

	login := append([]web.Middleware{}, pub...)
	login = append(login, web.RateLimit(10, time.Minute))
	verify := append([]web.Middleware{}, pub...)
	verify = append(verify, web.RateLimit(30, time.Minute))

	mux.HandleFunc("POST /api/v1/auth/login", web.Chain(login...)(c.Login))
	mux.HandleFunc("PUT /api/v1/auth/password", web.Chain(admin...)(c.ChangePassword))

	mux.HandleFunc("GET /api/v1/apps/{id}", web.Chain(pub...)(c.AppDetail))
	mux.HandleFunc("POST /api/v1/versions/{id}/verify", web.Chain(verify...)(c.VerifyAccess))
	mux.HandleFunc("GET /api/v1/versions/{id}/install", web.Chain(pub...)(c.Install))
	mux.HandleFunc("GET /api/v1/versions/{id}/download", web.Chain(pub...)(c.Download))
	mux.HandleFunc("POST /api/v1/files", web.Chain(admin...)(c.PresignFile))
	mux.HandleFunc("POST /api/v1/files/upload", web.Chain(pub...)(c.FileUpload))
	mux.HandleFunc("GET /api/v1/files/preview", web.Chain(pub...)(c.FilePreview))
	// Authenticated display variant (icons/screenshots) behind the admin chain.
	mux.HandleFunc("GET /api/v1/admin/files/preview", web.Chain(admin...)(c.FilePreview))

	mux.HandleFunc("GET /api/v1/admin/apps", web.Chain(admin...)(c.AppsList))
	mux.HandleFunc("GET /api/v1/admin/apps/{id}", web.Chain(admin...)(c.AppDetailAdmin))
	mux.HandleFunc("POST /api/v1/admin/apps", web.Chain(admin...)(c.CreateApp))
	mux.HandleFunc("PUT /api/v1/admin/apps/{id}", web.Chain(admin...)(c.UpdateApp))
	mux.HandleFunc("DELETE /api/v1/admin/apps/{id}/screenshots", web.Chain(admin...)(c.DeleteAppScreenshot))
	mux.HandleFunc("DELETE /api/v1/admin/apps/{id}", web.Chain(admin...)(c.DeleteApp))
	mux.HandleFunc("GET /api/v1/admin/apps/{id}/members", web.Chain(admin...)(c.AppMembersAdmin))
	mux.HandleFunc("PUT /api/v1/admin/apps/{id}/members", web.Chain(admin...)(c.SetAppMembersAdmin))
	mux.HandleFunc("GET /api/v1/admin/apps/{id}/downloads", web.Chain(admin...)(c.DownloadsTimeSeries))
	mux.HandleFunc("GET /api/v1/admin/users", web.Chain(admin...)(c.UsersList))
	mux.HandleFunc("POST /api/v1/admin/users", web.Chain(admin...)(c.CreateUser))
	mux.HandleFunc("PUT /api/v1/admin/users/{id}", web.Chain(admin...)(c.UpdateUser))
	mux.HandleFunc("DELETE /api/v1/admin/users/{id}", web.Chain(admin...)(c.DeleteUser))
	mux.HandleFunc("POST /api/v1/admin/apps/{id}/current", web.Chain(admin...)(c.SetCurrentVersion))
	mux.HandleFunc("POST /api/v1/admin/versions", web.Chain(admin...)(c.CreateVersion))
	mux.HandleFunc("DELETE /api/v1/admin/versions/{id}", web.Chain(admin...)(c.DeleteVersion))
	mux.HandleFunc("GET /api/v1/admin/versions/{id}/stats", web.Chain(admin...)(c.VersionStats))

	// API key management (JWT-authenticated).
	mux.HandleFunc("GET /api/v1/admin/apps/manageable", web.Chain(admin...)(c.ManageableApps))
	mux.HandleFunc("GET /api/v1/admin/keys", web.Chain(admin...)(c.KeysList))
	mux.HandleFunc("POST /api/v1/admin/keys", web.Chain(admin...)(c.CreateKey))
	mux.HandleFunc("PUT /api/v1/admin/keys/{id}", web.Chain(admin...)(c.UpdateKey))
	mux.HandleFunc("DELETE /api/v1/admin/keys/{id}", web.Chain(admin...)(c.DeleteKey))

	// Notification bots (webhook subscriptions).
	mux.HandleFunc("GET /api/v1/admin/subscriptions", web.Chain(admin...)(c.SubscriptionsList))
	mux.HandleFunc("POST /api/v1/admin/subscriptions", web.Chain(admin...)(c.CreateSubscription))
	mux.HandleFunc("PUT /api/v1/admin/subscriptions/{id}", web.Chain(admin...)(c.UpdateSubscription))
	mux.HandleFunc("DELETE /api/v1/admin/subscriptions/{id}", web.Chain(admin...)(c.DeleteSubscription))
	mux.HandleFunc("POST /api/v1/admin/subscriptions/{id}/test", web.Chain(admin...)(c.TestSubscription))
	mux.HandleFunc("POST /api/v1/admin/subscriptions/test", web.Chain(admin...)(c.TestSubscriptionConfig))
	mux.HandleFunc("GET /api/v1/admin/subscriptions/{id}/logs", web.Chain(admin...)(c.SubscriptionLogs))

	// API-key-authenticated programmatic endpoints (key via `?apikey=`).
	mux.HandleFunc("POST /api/v1/keys/{app_id}/files", web.Chain(pub...)(c.PresignKeyFile))
	mux.HandleFunc("POST /api/v1/keys/{app_id}/versions", web.Chain(pub...)(c.UploadKeyVersion))
	mux.HandleFunc("POST /api/v1/keys/{app_id}/current", web.Chain(pub...)(c.SetKeyCurrentVersion))
	mux.HandleFunc("GET /api/v1/keys/{app_id}/versions", web.Chain(pub...)(c.KeyVersionsList))
	mux.HandleFunc("GET /api/v1/keys/{app_id}/current", web.Chain(pub...)(c.KeyCurrentVersion))
	mux.HandleFunc("GET /api/v1/keys/{app_id}/current/download", web.Chain(pub...)(c.KeyCurrentDownload))

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
