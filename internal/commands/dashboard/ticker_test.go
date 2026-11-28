package dashboard_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestEventLabel(t *testing.T) {
	tests := map[string]string{
		"deployment.completed":      "deployment completed",
		"remote_reference.assigned": "remote reference assigned",
		"alarm":                     "alarm",
	}
	for typ, want := range tests {
		if got := dashboard.EventLabel(dashboard.Event{Type: typ}); got != want {
			t.Errorf("EventLabel(%s) = %q, want %q", typ, got, want)
		}
	}
}
