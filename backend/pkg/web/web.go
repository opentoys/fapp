package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	CodeOK           = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeInternal     = 500
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Chain(mws ...Middleware) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

func SendJson(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"code": CodeOK,
		"msg":  "ok",
		"data": data,
	})
}

func SendError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"code": code,
		"msg":  msg,
	})
}

// Recoverer catches panics and returns 500.
func Recoverer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				SendError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next(w, r)
	}
}

// Logger logs each request method, path, and duration.
func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	}
}

// RateLimit simple per-IP fixed-window rate limiter. Returns 429 when over limit.
func RateLimit(max int, window time.Duration) Middleware {
	type bucket struct {
		count int
		at    time.Time
	}
	var mu sync.Mutex
	lim := make(map[string]*bucket)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			mu.Lock()
			b := lim[r.RemoteAddr]
			if b == nil {
				b = &bucket{}
				lim[r.RemoteAddr] = b
			}
			if now.Sub(b.at) > window {
				b.count = 0
				b.at = now
			}
			b.count++
			over := b.count > max
			mu.Unlock()
			if over {
				SendError(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next(w, r)
		}
	}
}
