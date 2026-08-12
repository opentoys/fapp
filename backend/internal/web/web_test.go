package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendJson(t *testing.T) {
	w := httptest.NewRecorder()
	SendJson(w, map[string]string{"a": "b"})
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var body struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Msg != "ok" || body.Data["a"] != "b" {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	SendError(w, 404, "not found")
	if w.Code != http.StatusOK {
		t.Fatalf("SendError should keep HTTP 200, got %d", w.Code)
	}
	var body struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != 404 || body.Msg != "not found" {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestSendStatus(t *testing.T) {
	w := httptest.NewRecorder()
	SendStatus(w, http.StatusNotFound, "no")
	if w.Code != http.StatusNotFound {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestRateLimit(t *testing.T) {
	h := Chain(RateLimit(2, time.Minute))(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", "/", nil)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d should pass, got %d", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-in")
				next(w, r)
				order = append(order, name+"-out")
			}
		}
	}
	h := Chain(mk("a"), mk("b"))(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})
	h(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	want := []string{"a-in", "b-in", "handler", "b-out", "a-out"}
	if len(order) != len(want) {
		t.Fatalf("order = %v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %s, want %s", i, order[i], want[i])
		}
	}
}
