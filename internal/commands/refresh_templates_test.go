package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/templatecache"
)

func TestRefreshTemplates(t *testing.T) {
	cacheClient := &templatecache.MockCacheClient{
		Calls: make(map[string]*templatecache.CallTracker),
	}

	err := commands.RefreshTemplates(cacheClient)

	if err != nil {
		t.Fatal(err)
	}

	got := cacheClient.Calls["RefreshTemplates"].Calls
	want := 1

	if got != want {
		t.Errorf("Expected bundle cache client to be called %d times but it was called %d", want, got)
	}
}
