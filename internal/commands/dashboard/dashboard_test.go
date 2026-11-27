package dashboard_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/urls"
)

func TestRunSummary(t *testing.T) {
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		api     *fakeAPI
		want    []string
		wantErr bool
	}{
		{
			name: "organization",
			api:  fixture(),
			want: []string{
				"care-dev", "Care", "1 provisioned, 1 not deployed", "platform-dev, secret-prod",
				"CPU high", "platform-dev-db", "2 environments, 2 instances",
				"ecs", "1 of 2 instances",
			},
		},
		{name: "read fails", api: &fakeAPI{err: errors.New("boom")}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := dashboard.RunSummary(context.Background(), tc.api, &out, now)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			got := ansi.Strip(out.String())
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("summary is missing %q:\n%s", want, got)
				}
			}
			if strings.Contains(got, "\nsecret-prod") {
				t.Error("a restricted environment should not get its own row")
			}
		})
	}
}

func TestNewLinks(t *testing.T) {
	links := dashboard.NewLinks(urls.NewWithBaseURL("https://app.example.com/", "acme"))
	tests := []struct{ got, want string }{
		{links.Environment("care-dev"), "https://app.example.com/orgs/acme/projects/care/environments/dev"},
		{links.Instance("care-dev-app"), "https://app.example.com/orgs/acme/projects/care/environments/dev?package=app"},
		{links.Runtime("ecs"), "https://app.example.com/orgs/acme/runtimes/ecs"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("url = %s, want %s", tc.got, tc.want)
		}
	}
}
