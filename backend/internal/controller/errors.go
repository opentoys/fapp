package controller

import (
	"errors"
	"net/http"

	"disapp/internal/service"
	"disapp/pkg/web"
)

// sendErr maps a *service.Error to its HTTP status and JSON body; anything else
// becomes a 500.
func sendErr(w http.ResponseWriter, err error) {
	var se *service.Error
	if errors.As(err, &se) {
		web.SendError(w, se.Status, se.Msg)
		return
	}
	web.SendError(w, http.StatusInternalServerError, "服务器内部错误")
}
