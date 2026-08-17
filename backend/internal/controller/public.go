package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"disapp/pkg/web"
)

// AppDetail returns app detail. A password-protected app returns only
// {id, access_mode} until a valid password is supplied via query param;
// then the full detail. An unpublished app behaves as if it doesn't exist
// (404).
func (c *Controller) AppDetail(w http.ResponseWriter, r *http.Request) {
	app, versions, unlocked, err := c.SVC.PublicAppDetail(r.Context(), r.PathValue("id"), r.URL.Query().Get("password"))
	if err != nil {
		sendErr(w, err)
		return
	}
	out := map[string]any{"app": app}
	if unlocked {
		out["versions"] = versions
	}
	web.SendJson(w, out)
}

// VerifyAccess checks access permission (password mode submits password).
func (c *Controller) VerifyAccess(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad id")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := c.SVC.VerifyAccess(r.Context(), id, req.Password); err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"ok": true})
}

// resolveAndURL loads a version, checks access, and returns the absolute URL.
func (c *Controller) resolveAndURL(w http.ResponseWriter, r *http.Request) (int64, string, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "bad id")
		return 0, "", err
	}
	rel, v, err := c.SVC.ResolveDownload(r.Context(), id, r.URL.Query().Get("password"))
	if err != nil {
		sendErr(w, err)
		return 0, "", err
	}
	return v.ID, absURL(r, rel), nil
}

// Download returns download URL, increments download_count and logs.
func (c *Controller) Download(w http.ResponseWriter, r *http.Request) {
	vid, urlStr, err := c.resolveAndURL(w, r)
	if err != nil {
		return
	}
	c.SVC.RecordDownload(r.Context(), vid, clientIP(r), r.UserAgent())
	web.SendJson(w, map[string]any{"url": urlStr})
}

// Install reports installation, increments install_count.
func (c *Controller) Install(w http.ResponseWriter, r *http.Request) {
	vid, urlStr, err := c.resolveAndURL(w, r)
	if err != nil {
		return
	}
	c.SVC.RecordInstall(r.Context(), vid)
	web.SendJson(w, map[string]any{"url": urlStr})
}

// absURL prefixes an absolute scheme+host onto a Storage-returned path. The
// host comes from the request Host header (so reverse proxies must forward the
// real host), falling back to X-Forwarded-Host / Forwarded when present.
func absURL(r *http.Request, path string) string {
	base := requestBase(r)
	if base == "" {
		return path
	}
	return base + path
}

// requestBase returns the scheme://host origin of a request (X-Forwarded-Host
// wins over Host). Empty when no host is present.
func requestBase(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

// clientIP returns the caller IP, honoring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}