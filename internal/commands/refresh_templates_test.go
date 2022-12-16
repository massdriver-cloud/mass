package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/template_cache"
)

func TestRefreshTemplates(t *testing.T) {
	cacheClient := &template_cache.MockCacheClient{
		Calls: make(map[string]*template_cache.CallTracker),
	}

	commands.RefreshTemplates(cacheClient)

	got := cacheClient.Calls["RefreshTemplates"].Calls
	want := 1

	if got != want {
		t.Errorf("Expected bundle cache client to be called %d times but it was called %d", want, got)
	}
}
