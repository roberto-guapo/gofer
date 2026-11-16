package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cristianadrielbraun/gofer/internal/auth"
	"github.com/cristianadrielbraun/gofer/internal/systemmonitor"
)

func TestSystemMonitorAccessAndOptIn(t *testing.T) {
	h, db := newAccountOwnershipTestHandler(t)
	reads := 0
	sample := func(path string) systemmonitor.Snapshot {
		reads++
		if path != filepath.Dir(db.Path()) {
			t.Errorf("disk path = %q", path)
		}
		return systemmonitor.Snapshot{Supported: true}
	}
	request := func(ctx context.Context) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/system/monitor", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.serveSystemMonitor(w, r, sample)
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("statistics can be cached")
		}
		return w
	}
	owner := auth.ContextWithUser(t.Context(), &auth.User{ID: "owner", UserType: auth.UserTypeWebmail})
	personal := auth.ContextWithMode(owner, auth.ModePersonal)
	if w := request(personal); w.Code != http.StatusNoContent {
		t.Fatalf("off by default: %d", w.Code)
	}
	if reads != 0 {
		t.Fatal("disabled monitor collected statistics")
	}
	if err := db.MergeUISettings(personal, "owner", map[string]string{systemmonitor.SettingKey: "true"}); err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []context.Context{t.Context(), auth.ContextWithMode(t.Context(), auth.ModePersonal), owner, auth.ContextWithMode(owner, auth.ModeManaged)} {
		if w := request(ctx); w.Code != http.StatusNotFound {
			t.Errorf("unauthorized request: %d", w.Code)
		}
	}
	if reads != 0 {
		t.Fatal("unauthorized request collected statistics")
	}
	for _, mode := range []auth.Mode{auth.ModePersonal, auth.ModeOpen} {
		w := request(auth.ContextWithMode(owner, mode))
		var s systemmonitor.Snapshot
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &s) != nil || !s.Supported {
			t.Fatalf("enabled owner: %d %s", w.Code, w.Body.String())
		}
	}
	if err := db.MergeUISettings(personal, "owner", map[string]string{systemmonitor.SettingKey: "false"}); err != nil {
		t.Fatal(err)
	}
	before := reads
	if w := request(personal); w.Code != http.StatusNoContent || reads != before {
		t.Fatal("disabled monitor still collected statistics")
	}
}

func TestSystemMonitorUserStorageRoute(t *testing.T) {
	assertUserStorageRoutes(t, []struct{ method, path string }{{http.MethodGet, "/api/system/monitor"}})
}
