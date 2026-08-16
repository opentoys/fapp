package controller

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"disapp/internal/resources/storage"
	"disapp/internal/resources/storage/local"
	"disapp/pkg/web"
)

// FileUpload receives the direct POST body of a local-storage upload. The URL
// carries a signed ttl; the key is passed by the client as a query param.
func (c *Controller) FileUpload(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ttl, err := strconv.ParseInt(q.Get("ttl"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "无效的签名")
		return
	}
	key := q.Get("key")
	loc, ok := c.localStorage()
	if !ok {
		web.SendError(w, web.CodeBadRequest, "当前存储后端不支持直接上传")
		return
	}
	if !local.ValidUploadTicket(c.SVC.Config.JWT.Secret, ttl, q.Get("sign")) {
		web.SendError(w, web.CodeBadRequest, "签名无效或已过期")
		return
	}
	if !storage.ValidKey(key) {
		web.SendError(w, web.CodeBadRequest, "无效的 key")
		return
	}
	if _, err := loc.Save(r.Context(), key, r.Body); err != nil {
		web.SendError(w, web.CodeBadRequest, "写入失败")
		return
	}
	web.SendJson(w, map[string]any{"key": key})
}

// FilePreview streams a signed file (public, ?sign) or redirects an
// authenticated display request (?dl) to a freshly signed URL.
func (c *Controller) FilePreview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	key := q.Get("key")
	if !storage.ValidKey(key) {
		web.SendError(w, web.CodeBadRequest, "无效的 key")
		return
	}

	if q.Get("dl") == "1" {
		user := userFrom(r)
		if user == nil || !c.SVC.CanManage(user.UserID, appOfKey(key)) {
			web.SendError(w, web.CodeForbidden, "无权访问该文件")
			return
		}
		rel, err := c.SVC.PreviewURL(r.Context(), key)
		if err != nil {
			sendErr(w, err)
			return
		}
		http.Redirect(w, r, rel, http.StatusTemporaryRedirect)
		return
	}

	ttl, err := strconv.ParseInt(q.Get("ttl"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "无效的签名")
		return
	}
	loc, ok := c.localStorage()
	if !ok {
		web.SendError(w, web.CodeBadRequest, "当前存储后端不支持流式读取")
		return
	}
	if !local.ValidPreviewTicket(c.SVC.Config.JWT.Secret, key, ttl, q.Get("sign")) {
		web.SendError(w, web.CodeBadRequest, "签名无效或已过期")
		return
	}
	rc, err := loc.Open(r.Context(), key)
	if err != nil {
		web.SendError(w, web.CodeNotFound, "文件不存在")
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(key)))
	io.Copy(w, rc)
}

// localStorage asserts the underlying storage supports direct local I/O.
func (c *Controller) localStorage() (*local.LocalStorage, bool) {
	loc, ok := c.SVC.Storage.(*local.LocalStorage)
	return loc, ok
}

// appOfKey extracts the owning app id from a {app_id}/{version_id}/file key.
func appOfKey(key string) int64 {
	seg := key[:strings.Index(key, "/")]
	appID, err := strconv.ParseInt(seg, 10, 64)
	if err != nil {
		return 0
	}
	return appID
}
