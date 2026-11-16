package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/cristianadrielbraun/gofer/internal/auth"
	"github.com/cristianadrielbraun/gofer/internal/i18n"
	"github.com/cristianadrielbraun/gofer/internal/storage"
	"github.com/cristianadrielbraun/gofer/internal/systemmonitor"
)

func (h *Handler) systemMonitorHandler() http.HandlerFunc {
	collector := systemmonitor.New()
	return func(w http.ResponseWriter, r *http.Request) {
		h.serveSystemMonitor(w, r, collector.Sample)
	}
}

func (h *Handler) serveSystemMonitor(w http.ResponseWriter, r *http.Request, sample func(string) systemmonitor.Snapshot) {
	w.Header().Set("Cache-Control", "no-store")
	// The monitor belongs to the owner, just like owner diagnostics. A forged
	// preference cannot expose host details to a managed-mode webmail user.
	if !h.ownerInstance() || !auth.OwnerInstance(r.Context()) || auth.GetCurrentUser(r.Context()) == nil {
		http.NotFound(w, r)
		return
	}
	var enabled bool
	var diskPath string
	err := h.withUserDB(r.Context(), h.userID(r.Context()), func(db *storage.DB) error {
		enabled = db.GetUISettings(r.Context(), h.userID(r.Context()))[systemmonitor.SettingKey] == "true"
		diskPath = filepath.Dir(db.Path())
		return nil
	})
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "server.error.settings_unavailable"), http.StatusServiceUnavailable)
		return
	}
	if !enabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sample(diskPath))
}
