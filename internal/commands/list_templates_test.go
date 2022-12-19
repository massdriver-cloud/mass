package commands_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/templatecache"
)

func TestListTemplates(t *testing.T) {
	cacheClient := &templatecache.MockCacheClient{
		Calls: make(map[string]*templatecache.CallTracker),
	}

	_, err := commands.ListTemplates(cacheClient)

	if err != nil {
		t.Fatal(err)
	}

	got := cacheClient.Calls["ListTemplates"].Calls
	want := 1

	if got != want {
		t.Errorf("Expected bundle cache client to be called %d times but it was called %d", want, got)
	}
}
