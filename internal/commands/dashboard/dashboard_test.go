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
			err := dashboard.RunSummary(context.Background(), tc.api, &out, now, dashboard.Where{})
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

func TestRunSummaryMoreAlarms(t *testing.T) {
	api := fixture()
	api.envPages[0][0].Instances[1].FiringAlarmCount = 3
	var out bytes.Buffer
	if err := dashboard.RunSummary(context.Background(), api, &out, time.Now(), dashboard.Where{}); err != nil {
		t.Fatal(err)
	}
	if got := ansi.Strip(out.String()); !strings.Contains(got, "+2 more on the instances' alarm pages") {
		t.Errorf("summary should say how many alarms it does not list:\n%s", got)
	}
}

func TestTerminalNotifier(t *testing.T) {
	var out bytes.Buffer
	dashboard.TerminalNotifier(&out)("▲ CPU high", "db\x07 · Care\x1b]")
	if got, want := out.String(), "\a\x1b]9;▲ CPU high: db · Care]\x07"; got != want {
		t.Errorf("notification = %q, want %q", got, want)
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
