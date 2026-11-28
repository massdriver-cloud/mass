package dashboard_test

import (
	"fmt"
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

func TestStashFirstReadOfANewEnvironmentIsNotNews(t *testing.T) {
	api := withStash()
	api.stash = append(api.stash, object(env("sandbox-demo"), "md/v1/banner", dashboard.Values{"message": "Demo at 3pm"}))
	m := loaded(t, api, config(&opener{}), 200, 48)
	m, cmd := keys(m, "/", "d", "e", "m", "o", "enter")
	m = drive(t, m, cmd, 30)
	if m.Focus() != "sandbox-demo" {
		t.Fatalf("focus = %q", m.Focus())
	}
	if out := view(m); !strings.Contains(out, "Demo at 3pm") || strings.Contains(out, "stash set") {
		t.Errorf("an environment new to the view should show its stash without putting it on the ticker:\n%s", out)
	}
}

func TestWideViewsPollInsteadOfStreaming(t *testing.T) {
	hub := dashboard.Environment{ID: "hub-dev", Name: "Dev", Project: ref("hub", "Hub"), Instances: []dashboard.Instance{{ID: "hub-dev-db", Status: "PROVISIONED"}}}
	envs := []dashboard.Environment{hub}
	for i := range 21 {
		id := fmt.Sprintf("app%d-dev", i)
		envs = append(envs, dashboard.Environment{ID: id, Name: "Dev", Project: ref(fmt.Sprintf("app%d", i), "App"), Instances: []dashboard.Instance{
			{ID: id + "-svc", Status: "PROVISIONED", Dependencies: []dashboard.Dependency{dependsOn("hub-dev-db", "hub-dev")}},
		}})
		envs[0].Instances[0].Resources = append(envs[0].Instances[0].Resources, usedBy(id+"-svc", id))
	}
	api := &fakeAPI{envPages: [][]dashboard.Environment{envs}, recent: "hub-dev", streamed: map[string][]dashboard.StashEvent{}}
	m := loaded(t, api, config(&opener{}), 200, 48)
	if !strings.Contains(view(m), "22 environments") {
		t.Fatalf("every consumer should be in view:\n%s", view(m))
	}
	if len(api.subscribed) != 0 {
		t.Errorf("subscribed %d environments; a view this wide should poll", len(api.subscribed))
	}
}

func TestStashFirstReadAfterAFailureIsNotNews(t *testing.T) {
	api := withStash()
	api.stash = append(api.stash, object(env("sandbox-demo"), "md/v1/banner", dashboard.Values{"message": "Demo at 3pm"}))
	m := loaded(t, api, config(&opener{}), 200, 48)
	api.failScope = "sandbox"
	m, cmd := keys(m, "/", "d", "e", "m", "o", "enter")
	m = drive(t, m, cmd, 30)
	if strings.Contains(view(m), "Demo at 3pm") {
		t.Fatal("the read after the refocus should have failed")
	}
	api.failScope = ""
	m = refreshed(t, m)
	if out := view(m); !strings.Contains(out, "Demo at 3pm") || strings.Contains(out, "stash set") {
		t.Errorf("a scope read for the first time after a failed read is not news:\n%s", out)
	}
}
