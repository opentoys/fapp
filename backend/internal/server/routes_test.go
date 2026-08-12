package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutesUnknownAPI(t *testing.T) {
	s := testServer(t)
	h := s.Routes(nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	// Should not crash; either 404 or a JSON error response
}

func TestRoutesAppsPath(t *testing.T) {
	s := testServer(t)
	h := s.Routes(nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
}