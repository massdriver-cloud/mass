package dashboard_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
)

func TestStashStreams(t *testing.T) {
	api := withStash()
	down := text(inst("care-dev-app"), "md/v1/health", "down")
	down.UpdatedAt = fixedNow().Add(time.Second)
	api.streamed = map[string][]dashboard.StashEvent{
		"care-dev": {{Action: stash.EventSet, Scope: inst("care-dev-app"), Key: "md/v1/health", Entry: &down}},
	}
	var m tea.Model = dashboard.NewModel(api, config(&opener{}))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	model := pump(t, m, []tea.Cmd{m.Init()}, 40)
	if out := view(model); !strings.Contains(out, "md/v1/health = down") {
		t.Errorf("a streamed key should reach the ticker:\n%s", out)
	}
	for _, want := range []string{"care/care-dev", "platform/platform-dev"} {
		if !slices.Contains(api.subscribed, want) {
			t.Errorf("subscribed %v, want %s: every environment in view streams", api.subscribed, want)
		}
	}
}
