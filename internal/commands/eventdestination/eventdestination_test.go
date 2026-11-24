package eventdestination_test

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/eventdestination"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/eventdestinations"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

type fakeAPI struct {
	destination  *types.EventDestination
	destinations []types.EventDestination
	deliveries   []types.EventDelivery
	events       []types.AuditLog
	eventTypes   []types.EventType

	gotCreate      eventdestinations.CreateInput
	gotUpdate      eventdestinations.UpdateInput
	gotList        eventdestinations.ListInput
	gotDeliveries  eventdestinations.ListDeliveriesInput
	gotPreview     eventdestinations.PreviewInput
	gotEventTypes  eventdestinations.ListEventTypesInput
	gotRedeliver   []string
	getCalls       int
	deleteCalls    int
	previewFetched int
}

func (f *fakeAPI) Get(_ context.Context, _ string) (*types.EventDestination, error) {
	f.getCalls++
	return f.destination, nil
}

func (f *fakeAPI) Iter(_ context.Context, input eventdestinations.ListInput) iter.Seq2[types.EventDestination, error] {
	f.gotList = input
	return seqOf(f.destinations)
}

func (f *fakeAPI) Create(_ context.Context, input eventdestinations.CreateInput) (*types.EventDestination, error) {
	f.gotCreate = input
	return f.destination, nil
}

func (f *fakeAPI) Update(_ context.Context, _ string, input eventdestinations.UpdateInput) (*types.EventDestination, error) {
	f.gotUpdate = input
	return f.destination, nil
}

func (f *fakeAPI) Delete(_ context.Context, _ string) (*types.EventDestination, error) {
	f.deleteCalls++
	return f.destination, nil
}

func (f *fakeAPI) Pause(_ context.Context, _ string) (*types.EventDestination, error) {
	return f.destination, nil
}

func (f *fakeAPI) Resume(_ context.Context, _ string) (*types.EventDestination, error) {
	return f.destination, nil
}

func (f *fakeAPI) Redeliver(_ context.Context, _ string, deliveryIDs []string) (*types.EventDestination, error) {
	f.gotRedeliver = deliveryIDs
	return f.destination, nil
}

func (f *fakeAPI) IterDeliveries(_ context.Context, _ string, input eventdestinations.ListDeliveriesInput) iter.Seq2[types.EventDelivery, error] {
	f.gotDeliveries = input
	return seqOf(f.deliveries)
}

func (f *fakeAPI) IterPreview(_ context.Context, input eventdestinations.PreviewInput) iter.Seq2[types.AuditLog, error] {
	f.gotPreview = input
	return func(yield func(types.AuditLog, error) bool) {
		for _, e := range f.events {
			f.previewFetched++
			if !yield(e, nil) {
				return
			}
		}
	}
}

func (f *fakeAPI) ListEventTypes(_ context.Context, input eventdestinations.ListEventTypesInput) ([]types.EventType, error) {
	f.gotEventTypes = input
	return f.eventTypes, nil
}

func seqOf[T any](items []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func TestRunCreate(t *testing.T) {
	tests := []struct {
		name      string
		opts      eventdestination.Options
		want      eventdestinations.CreateInput
		wantErr   string
		wantInOut []string
	}{
		{
			name: "url, headers, and selection",
			opts: eventdestination.Options{
				Name:       "Production deployments",
				Adapter:    "http",
				URL:        "https://hooks.example.com/massdriver",
				URLSet:     true,
				Headers:    []string{"X-Team=platform"},
				HeadersSet: true,
				EventTypes: []string{"deployment.*"},
				Attributes: []string{"md-environment=prod"},
				Condition:  `.data.metadata.deployment_action == "PROVISION"`,
				Auth:       `{"headers":{"Authorization":"Bearer token"}}`,
				AuthSet:    true,
			},
			want: eventdestinations.CreateInput{
				Name:       "Production deployments",
				Adapter:    "HTTP",
				Config:     map[string]any{"url": "https://hooks.example.com/massdriver", "headers": map[string]any{"X-Team": "platform"}},
				Auth:       map[string]any{"headers": map[string]any{"Authorization": "Bearer token"}},
				EventTypes: []string{"deployment.*"},
				Attributes: map[string]string{"md-environment": "prod"},
				Condition:  `.data.metadata.deployment_action == "PROVISION"`,
			},
			wantInOut: []string{"Production deployments", "whsec_c2VjcmV0"},
		},
		{
			name: "url overrides config",
			opts: eventdestination.Options{
				Name:      "Audit",
				Adapter:   "HTTP",
				Config:    `{"url":"https://old.example.com","headers":{"X-Team":"platform"}}`,
				ConfigSet: true,
				URL:       "https://new.example.com",
				URLSet:    true,
			},
			want: eventdestinations.CreateInput{
				Name:    "Audit",
				Adapter: "HTTP",
				Config:  map[string]any{"url": "https://new.example.com", "headers": map[string]any{"X-Team": "platform"}},
			},
		},
		{
			name:    "name is required",
			opts:    eventdestination.Options{URL: "https://hooks.example.com", URLSet: true},
			wantErr: "--name is required",
		},
		{
			name:    "config is required",
			opts:    eventdestination.Options{Name: "Audit", Adapter: "HTTP"},
			wantErr: "set --url, --header, or --config",
		},
		{
			name: "bad auth is not echoed",
			opts: eventdestination.Options{
				Name: "Audit", Adapter: "HTTP", URL: "https://hooks.example.com", URLSet: true,
				Auth: `Bearer s3cr3t`, AuthSet: true,
			},
			wantErr: "--auth must be a JSON object or @file holding one",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{destination: &types.EventDestination{ID: "ed-1", Name: "Production deployments", SigningSecret: "whsec_c2VjcmV0"}}
			var out bytes.Buffer

			err := eventdestination.RunCreate(t.Context(), api, tc.opts, &out)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				if strings.Contains(err.Error(), "s3cr3t") {
					t.Errorf("error echoes the auth value: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("RunCreate: %v", err)
			}
			if !reflect.DeepEqual(api.gotCreate, tc.want) {
				t.Errorf("CreateInput = %#v\nwant %#v", api.gotCreate, tc.want)
			}
			for _, want := range tc.wantInOut {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output missing %q:\n%s", want, out.String())
				}
			}
			if strings.Contains(out.String(), "Bearer token") {
				t.Errorf("output echoes the auth value:\n%s", out.String())
			}
		})
	}
}

func TestRunUpdate(t *testing.T) {
	tests := []struct {
		name         string
		opts         eventdestination.Options
		want         eventdestinations.UpdateInput
		wantGetCalls int
		wantErr      string
	}{
		{
			name: "only name",
			opts: eventdestination.Options{Name: "Renamed", NameSet: true},
			want: eventdestinations.UpdateInput{Name: ptr("Renamed")},
		},
		{
			name:         "url changes the current config",
			opts:         eventdestination.Options{URL: "https://new.example.com", URLSet: true},
			want:         eventdestinations.UpdateInput{Config: map[string]any{"url": "https://new.example.com", "headers": map[string]any{"X-Team": "platform"}}},
			wantGetCalls: 1,
		},
		{
			name:         "header adds to the current headers",
			opts:         eventdestination.Options{Headers: []string{"X-Env=prod"}, HeadersSet: true},
			want:         eventdestinations.UpdateInput{Config: map[string]any{"url": "https://old.example.com", "headers": map[string]any{"X-Team": "platform", "X-Env": "prod"}}},
			wantGetCalls: 1,
		},
		{
			name: "config replaces without reading the current one",
			opts: eventdestination.Options{Config: `{"url":"https://new.example.com"}`, ConfigSet: true},
			want: eventdestinations.UpdateInput{Config: map[string]any{"url": "https://new.example.com"}},
		},
		{
			name: "empty values clear",
			opts: eventdestination.Options{
				Auth: `{}`, AuthSet: true,
				EventTypes: []string{""}, EventTypesSet: true,
				Attributes: []string{""}, AttributesSet: true,
				Condition: "", ConditionSet: true,
			},
			want: eventdestinations.UpdateInput{
				Auth:       map[string]any{},
				EventTypes: &[]string{},
				Attributes: map[string]string{},
				Condition:  ptr(""),
			},
		},
		{
			name:    "nothing to update",
			opts:    eventdestination.Options{},
			wantErr: "nothing to update",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{destination: &types.EventDestination{
				ID:     "ed-1",
				Name:   "Production deployments",
				Config: map[string]any{"url": "https://old.example.com", "headers": map[string]any{"X-Team": "platform"}},
			}}
			var out bytes.Buffer

			err := eventdestination.RunUpdate(t.Context(), api, "ed-1", tc.opts, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RunUpdate: %v", err)
			}
			if !reflect.DeepEqual(api.gotUpdate, tc.want) {
				t.Errorf("UpdateInput = %#v\nwant %#v", api.gotUpdate, tc.want)
			}
			if api.getCalls != tc.wantGetCalls {
				t.Errorf("Get calls = %d, want %d", api.getCalls, tc.wantGetCalls)
			}
		})
	}
}

func TestRunDelete(t *testing.T) {
	tests := []struct {
		name        string
		force       bool
		answer      string
		wantDeletes int
	}{
		{name: "confirmed", answer: "Audit\n", wantDeletes: 1},
		{name: "wrong name cancels", answer: "nope\n", wantDeletes: 0},
		{name: "force skips the prompt", force: true, wantDeletes: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{destination: &types.EventDestination{ID: "ed-1", Name: "Audit"}}
			var out bytes.Buffer

			if err := eventdestination.RunDelete(t.Context(), api, "ed-1", tc.force, strings.NewReader(tc.answer), &out); err != nil {
				t.Fatalf("RunDelete: %v", err)
			}
			if api.deleteCalls != tc.wantDeletes {
				t.Errorf("Delete calls = %d, want %d", api.deleteCalls, tc.wantDeletes)
			}
		})
	}
}

func TestRunPauseAndResume(t *testing.T) {
	api := &fakeAPI{destination: &types.EventDestination{ID: "ed-1", Name: "Audit"}}
	var out bytes.Buffer

	if err := eventdestination.RunPause(t.Context(), api, "ed-1", &out); err != nil {
		t.Fatalf("RunPause: %v", err)
	}
	if err := eventdestination.RunResume(t.Context(), api, "ed-1", &out); err != nil {
		t.Fatalf("RunResume: %v", err)
	}
	for _, want := range []string{"`Audit` paused", "`Audit` resumed"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunRedeliver(t *testing.T) {
	tests := []struct {
		name        string
		deliveryIDs []string
		wantOut     string
	}{
		{name: "every failed delivery", wantOut: "Every failed delivery of `Audit`"},
		{name: "given deliveries", deliveryIDs: []string{"d-1", "d-2"}, wantOut: "given deliveries of `Audit` that failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{destination: &types.EventDestination{ID: "ed-1", Name: "Audit"}}
			var out bytes.Buffer

			if err := eventdestination.RunRedeliver(t.Context(), api, "ed-1", tc.deliveryIDs, &out); err != nil {
				t.Fatalf("RunRedeliver: %v", err)
			}
			if !reflect.DeepEqual(api.gotRedeliver, tc.deliveryIDs) {
				t.Errorf("deliveryIDs = %v, want %v", api.gotRedeliver, tc.deliveryIDs)
			}
			if !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tc.wantOut)
			}
		})
	}
}

func TestRunList(t *testing.T) {
	api := &fakeAPI{destinations: []types.EventDestination{
		{ID: "ed-1", Name: "Audit", Adapter: "HTTP", Status: "PAUSED", LastError: "HTTP 503"},
	}}

	out, err := renderTo(t, func(f *os.File) error {
		return eventdestination.RunList(t.Context(), api, []string{"paused"}, "table", f)
	})
	if err != nil {
		t.Fatalf("RunList: %v", err)
	}
	if !reflect.DeepEqual(api.gotList.Statuses, []eventdestinations.Status{eventdestinations.StatusPaused}) {
		t.Errorf("Statuses = %v, want [PAUSED]", api.gotList.Statuses)
	}
	for _, want := range []string{"ed-1", "Audit", "PAUSED", "every event", "HTTP 503"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}

	_, err = renderTo(t, func(f *os.File) error {
		return eventdestination.RunList(t.Context(), api, []string{"broken"}, "table", f)
	})
	if err == nil || !strings.Contains(err.Error(), "ACTIVE, PAUSED") {
		t.Errorf("err = %v, want it to name the allowed statuses", err)
	}
}

func TestRunDeliveries(t *testing.T) {
	api := &fakeAPI{deliveries: []types.EventDelivery{
		{ID: "d-1", Status: "FAILED", Attempts: 8, LastError: "HTTP 503", Event: &types.AuditLog{Type: "deployment.completed"}},
		{ID: "d-2", Status: "FAILED", Attempts: 8},
	}}

	out, err := renderTo(t, func(f *os.File) error {
		return eventdestination.RunDeliveries(t.Context(), api, "ed-1", []string{"FAILED"}, "json", f)
	})
	if err != nil {
		t.Fatalf("RunDeliveries: %v", err)
	}
	if !reflect.DeepEqual(api.gotDeliveries.Statuses, []eventdestinations.DeliveryStatus{eventdestinations.DeliveryFailed}) {
		t.Errorf("Statuses = %v, want [FAILED]", api.gotDeliveries.Statuses)
	}
	var got []types.EventDelivery
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(got) != 2 || got[0].Event.Type != "deployment.completed" {
		t.Errorf("got %+v", got)
	}
}

func TestRunPreview(t *testing.T) {
	api := &fakeAPI{events: []types.AuditLog{{ID: "e-1"}, {ID: "e-2"}, {ID: "e-3"}}}

	out, err := renderTo(t, func(f *os.File) error {
		return eventdestination.RunPreview(t.Context(), api, eventdestination.PreviewOptions{
			EventTypes: []string{"deployment.*"},
			Attributes: []string{"md-environment=prod"},
			Condition:  ".type != null",
			Limit:      2,
			Output:     "json",
		}, f)
	})
	if err != nil {
		t.Fatalf("RunPreview: %v", err)
	}
	want := eventdestinations.PreviewInput{
		EventTypes: []string{"deployment.*"},
		Attributes: map[string]string{"md-environment": "prod"},
		Condition:  ".type != null",
		PageSize:   2,
	}
	if !reflect.DeepEqual(api.gotPreview, want) {
		t.Errorf("PreviewInput = %#v, want %#v", api.gotPreview, want)
	}
	var got []types.AuditLog
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(got) != 2 || api.previewFetched != 2 {
		t.Errorf("listed %d events after fetching %d, want 2 and 2", len(got), api.previewFetched)
	}
}

func TestRunPreviewRejectsLimitBelowOne(t *testing.T) {
	_, err := renderTo(t, func(f *os.File) error {
		return eventdestination.RunPreview(t.Context(), &fakeAPI{}, eventdestination.PreviewOptions{Limit: 0, Output: "table"}, f)
	})
	if err == nil || !strings.Contains(err.Error(), "--limit") {
		t.Errorf("err = %v, want a --limit error", err)
	}
}

func TestRunEventTypeList(t *testing.T) {
	api := &fakeAPI{eventTypes: []types.EventType{
		{ID: "deployment.completed", Entity: "deployment", ChangeType: "UPDATED", Description: "A deployment finished."},
	}}

	out, err := renderTo(t, func(f *os.File) error {
		return eventdestination.RunEventTypeList(t.Context(), api, "deployment", "table", f)
	})
	if err != nil {
		t.Fatalf("RunEventTypeList: %v", err)
	}
	if api.gotEventTypes.Entity != "deployment" {
		t.Errorf("Entity = %q, want deployment", api.gotEventTypes.Entity)
	}
	for _, want := range []string{"deployment.completed", "UPDATED", "A deployment finished."} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}
}

func TestParsePairs(t *testing.T) {
	tests := []struct {
		name    string
		pairs   []string
		want    map[string]string
		wantErr bool
	}{
		{name: "none", pairs: nil, want: nil},
		{name: "empty value clears", pairs: []string{""}, want: map[string]string{}},
		{name: "pairs", pairs: []string{"team=platform", "region=us-east-1,us-west-2"}, want: map[string]string{"team": "platform", "region": "us-east-1,us-west-2"}},
		{name: "value may hold =", pairs: []string{"Authorization=a=b"}, want: map[string]string{"Authorization": "a=b"}},
		{name: "missing =", pairs: []string{"team"}, wantErr: true},
		{name: "missing key", pairs: []string{"=platform"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eventdestination.ParsePairs(tc.pairs, "--attribute")
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParsePairs succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePairs: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestParseJSONObject(t *testing.T) {
	file := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(file, []byte(`{"headers":{"Authorization":"Bearer token"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		in      string
		want    map[string]any
		wantErr bool
	}{
		{name: "inline", in: `{"url":"https://hooks.example.com"}`, want: map[string]any{"url": "https://hooks.example.com"}},
		{name: "file", in: "@" + file, want: map[string]any{"headers": map[string]any{"Authorization": "Bearer token"}}},
		{name: "empty object", in: `{}`, want: map[string]any{}},
		{name: "not JSON", in: `Bearer token`, wantErr: true},
		{name: "null", in: `null`, wantErr: true},
		{name: "array", in: `[]`, wantErr: true},
		{name: "missing file", in: "@" + filepath.Join(t.TempDir(), "missing.json"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eventdestination.ParseJSONObject(tc.in, "--auth")
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParseJSONObject succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseJSONObject: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func renderTo(t *testing.T, fn func(out *os.File) error) (string, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "render.out")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	runErr := fn(file)
	if closeErr := file.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	out, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(out), runErr
}
