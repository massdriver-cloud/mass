package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
)

func (f *fakeAPI) DeploymentLogs(_ context.Context, id string) (dashboard.DeploymentLog, error) {
	f.scopeMu.Lock()
	defer f.scopeMu.Unlock()
	if f.logCalls == nil {
		f.logCalls = map[string]int{}
	}
	reads := f.logReads[id]
	if len(reads) == 0 {
		return dashboard.DeploymentLog{}, errors.New("no such deployment")
	}
	read := reads[min(f.logCalls[id], len(reads)-1)]
	f.logCalls[id]++
	return read, nil
}

func (f *fakeAPI) StreamDeploymentLogs(ctx context.Context, id string) (*streaming.Stream[dashboard.LogBatch], error) {
	f.scopeMu.Lock()
	defer f.scopeMu.Unlock()
	if f.liveLogs == nil {
		return nil, streaming.ErrRequiresPAT
	}
	f.logStreams = append(f.logStreams, ctx)
	batches := f.liveLogs[id]
	c := make(chan dashboard.LogBatch, len(batches))
	for _, b := range batches {
		c <- b
	}
	close(c)
	return streaming.New(c, nil), nil
}

func batch(sec int, message string) dashboard.LogBatch {
	return dashboard.LogBatch{Timestamp: fixedNow().Add(time.Duration(sec) * time.Second), Message: message}
}

func logConfig() dashboard.Config {
	cfg := config(&opener{})
	cfg.LogPoll = time.Millisecond
	return cfg
}

func TestFollowLogs(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		setup       func(*fakeAPI)
		deployment  string
		want        []string
		notWant     []string
		wantReads   int
		wantStreams int
	}{
		{
			name: "streams after the history and shows each batch once",
			env:  "billing-dev",
			setup: func(f *fakeAPI) {
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {
					{Status: "RUNNING", Batches: []dashboard.LogBatch{batch(1, "alpha\n"), batch(2, "bravo")}},
					{Status: "COMPLETED", Batches: []dashboard.LogBatch{batch(1, "alpha\n"), batch(2, "bravo"), batch(3, "charlie"), batch(4, "delta")}},
				}}
				f.liveLogs = map[string][]dashboard.LogBatch{"dep-billing": {batch(2, "bravo"), batch(3, "charlie")}}
			},
			deployment:  "dep-billing",
			want:        []string{"LOGS app", "provision · ✓ completed", "alpha", "bravo", "charlie", "delta"},
			wantReads:   2,
			wantStreams: 1,
		},
		{
			name: "polls without a personal access token",
			env:  "billing-dev",
			setup: func(f *fakeAPI) {
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {
					{Status: "RUNNING", Batches: []dashboard.LogBatch{batch(1, "alpha")}},
					{Status: "RUNNING", Batches: []dashboard.LogBatch{batch(1, "alpha"), batch(2, "bravo")}},
					{Status: "FAILED", Batches: []dashboard.LogBatch{batch(1, "alpha"), batch(2, "bravo"), batch(3, "charlie\ndelta\n")}},
				}}
			},
			deployment: "dep-billing",
			want:       []string{"LOGS app", "✕ failed", "alpha", "bravo", "charlie", "delta"},
			wantReads:  3,
		},
		{
			name: "reads a finished deployment once without streaming",
			env:  "care-dev",
			setup: func(f *fakeAPI) {
				f.envPages[0][1].Instances[0].LatestDeployment = &dashboard.Deployment{
					ID: "dep-care", Status: "COMPLETED", Action: "DECOMMISSION", CreatedAt: "2026-11-25T10:00:00Z",
				}
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-care": {{Status: "COMPLETED", Batches: []dashboard.LogBatch{batch(1, "alpha")}}}}
				f.liveLogs = map[string][]dashboard.LogBatch{}
			},
			deployment: "dep-care",
			want:       []string{"LOGS app", "decommission · ✓ completed", "alpha"},
			wantReads:  1,
		},
		{
			name: "strips escapes and control characters",
			env:  "billing-dev",
			setup: func(f *fakeAPI) {
				f.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {
					{Status: "COMPLETED", Batches: []dashboard.LogBatch{batch(1, "\x1b[31mred\x1b[0m alert\x07\n")}},
				}}
			},
			deployment: "dep-billing",
			want:       []string{"red alert"},
			notWant:    []string{"[31m", "[0m"},
			wantReads:  1,
		},
		{
			name:    "says when there is nothing to follow",
			env:     "platform-dev",
			want:    []string{"no deployment to follow here"},
			notWant: []string{"LOGS"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := fixture()
			if tc.setup != nil {
				tc.setup(api)
			}
			m := loaded(t, api, logConfig(), 200, 48)
			m, _ = tabTo(t, m, tc.env)
			m, cmd := pressCmd(m, "f")
			m = drive(t, m, cmd, 60)
			out := view(m)
			for _, want := range tc.want {
				if strings.Count(out, want) != 1 {
					t.Errorf("view shows %q %d times, want once:\n%s", want, strings.Count(out, want), out)
				}
			}
			for _, bad := range tc.notWant {
				if strings.Contains(out, bad) {
					t.Errorf("view shows %q:\n%s", bad, out)
				}
			}
			if got := api.logCalls[tc.deployment]; got != tc.wantReads {
				t.Errorf("read the logs %d times, want %d", got, tc.wantReads)
			}
			if len(api.logStreams) != tc.wantStreams {
				t.Errorf("opened %d streams, want %d", len(api.logStreams), tc.wantStreams)
			}
			for _, ctx := range api.logStreams {
				if ctx.Err() == nil {
					t.Error("the stream should be cancelled once the deployment ends")
				}
			}
		})
	}
}

func TestLogsClose(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "f again", key: "f"},
		{name: "esc", key: "esc"},
		{name: "moving to another environment", key: "tab"},
		{name: "quitting", key: "q"},
		{name: "another tab", key: "2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := fixture()
			api.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {{Status: "RUNNING", Batches: []dashboard.LogBatch{batch(1, "alpha")}}}}
			api.liveLogs = map[string][]dashboard.LogBatch{}
			m := loaded(t, api, logConfig(), 200, 48)
			m, _ = tabTo(t, m, "billing-dev")
			m, cmd := pressCmd(m, "f")
			m = drive(t, m, cmd, 2)
			if out := view(m); !strings.Contains(out, "LOGS app") || !strings.Contains(out, "⟳ running") {
				t.Fatalf("the logs should be open and streaming:\n%s", out)
			}
			m = press(m, tc.key)
			if strings.Contains(view(m), "LOGS app") {
				t.Errorf("the logs should close:\n%s", view(m))
			}
			if len(api.logStreams) != 1 || api.logStreams[0].Err() == nil {
				t.Error("closing the logs should cancel the stream")
			}
		})
	}
}

func TestLogsFollowTheTail(t *testing.T) {
	api := fixture()
	batches := make([]dashboard.LogBatch, 0, 600)
	for i := range 600 {
		batches = append(batches, batch(i, fmt.Sprintf("line %03d", i)))
	}
	api.logReads = map[string][]dashboard.DeploymentLog{"dep-billing": {{Status: "COMPLETED", Batches: batches}}}
	m := loaded(t, api, logConfig(), 200, 48)
	m, _ = tabTo(t, m, "billing-dev")
	m, cmd := pressCmd(m, "f")
	m = drive(t, m, cmd, 10)
	out := view(m)
	if !strings.Contains(out, "line 599") || strings.Contains(out, "line 500") {
		t.Errorf("the panel should show the newest lines:\n%s", out)
	}
}

type fakeDeploys struct {
	stream *streaming.Stream[dashboard.LogBatch]
}

func (f fakeDeploys) StreamLogs(context.Context, string) (*streaming.Stream[dashboard.LogBatch], error) {
	return f.stream, nil
}

func TestDeploymentLogsAPI(t *testing.T) {
	client := &recordingClient{data: map[string]string{
		"dashboardDeploymentLogs": `{"deployment":{"status":"RUNNING","logs":[{"timestamp":"2026-11-26T00:00:01Z","message":"hi"}]}}`,
	}}
	ctx := context.Background()
	api := dashboard.NewAPIWithClient(client, &fakeAudit{}, nil, nil, "acme")
	log, err := api.DeploymentLogs(ctx, "dep-1")
	if err != nil || log.Status != "RUNNING" || len(log.Batches) != 1 || log.Batches[0].Message != "hi" || !log.Batches[0].Timestamp.Equal(fixedNow().Add(time.Second)) {
		t.Errorf("DeploymentLogs = %+v, %v", log, err)
	}
	if vars, ok := client.reqs[0].Variables.(map[string]any); !ok || vars["id"] != "dep-1" || vars["organizationId"] != "acme" {
		t.Errorf("variables = %v", client.reqs[0].Variables)
	}
	if _, err := api.StreamDeploymentLogs(ctx, "dep-1"); !errors.Is(err, streaming.ErrRequiresPAT) {
		t.Errorf("streaming without a deployment service = %v, want ErrRequiresPAT", err)
	}
	want := streaming.New(make(chan dashboard.LogBatch), nil)
	api = dashboard.NewAPIWithClient(client, &fakeAudit{}, nil, fakeDeploys{stream: want}, "acme")
	if got, err := api.StreamDeploymentLogs(ctx, "dep-1"); err != nil || got != want {
		t.Errorf("StreamDeploymentLogs = %v, %v", got, err)
	}
}
