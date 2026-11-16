package views

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/cristianadrielbraun/gofer/internal/auth"
	"github.com/cristianadrielbraun/gofer/internal/systemmonitor"
)

func TestSystemMonitorOwnerVisibility(t *testing.T) {
	user := auth.ContextWithUser(t.Context(), &auth.User{ID: "owner"})
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		allowed bool
	}{
		{"anonymous", auth.ContextWithMode(t.Context(), auth.ModePersonal), false},
		{"unknown mode", user, false},
		{"managed", auth.ContextWithMode(user, auth.ModeManaged), false},
		{"personal", auth.ContextWithMode(user, auth.ModePersonal), true},
		{"open", auth.ContextWithMode(user, auth.ModeOpen), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var settings, widget, scripts bytes.Buffer
			if err := SystemMonitorSettings(nil).Render(tc.ctx, &settings); err != nil {
				t.Fatal(err)
			}
			if err := SystemMonitor().Render(tc.ctx, &widget); err != nil {
				t.Fatal(err)
			}
			if err := SystemMonitorScripts().Render(tc.ctx, &scripts); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(settings.String(), "system-monitor-enabled") != tc.allowed {
				t.Fatal("setting shown to wrong audience")
			}
			if strings.Contains(scripts.String(), "system-monitor.js") != tc.allowed {
				t.Fatal("script shown to wrong audience")
			}
			if strings.Contains(widget.String(), "data-system-monitor") != (tc.allowed && systemmonitor.Supported()) {
				t.Fatal("widget shown to wrong audience/platform")
			}
			if tc.allowed {
				if strings.Contains(settings.String(), " checked") {
					t.Fatal("monitor enabled by default")
				}
				if systemmonitor.Supported() && !strings.Contains(widget.String(), " hidden") {
					t.Fatal("widget flashes before saved preferences load")
				}
			}
		})
	}
}
