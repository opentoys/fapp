package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"disapp/internal/controller"
	"disapp/internal/resources/config"
	"disapp/internal/resources/storage/local"
	"disapp/internal/resources/store/db"
	"disapp/internal/service"
)

func testController(t *testing.T) *controller.Controller {
	t.Helper()
	gdb, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	st, err := local.NewLocal(t.TempDir(), config.Default().JWT.Secret)
	if err != nil {
		t.Fatal(err)
	}
	return controller.New(service.New(gdb, st, config.Default()))
}

func TestRoutesUnknownAPI(t *testing.T) {
	c := testController(t)
	h := Routes(c, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	// Should not crash; either 404 or a JSON error response
}

// The public app list endpoint was removed; only the detail path remains.
func TestRoutesAppsListRemoved(t *testing.T) {
	c := testController(t)
	h := Routes(c, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", w.Code)
	}
}