package stash_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	sdkstash "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

type setCall struct {
	scope sdkstash.Scope
	key   string
	input sdkstash.SetInput
}

type fakeAPI struct {
	environments map[string]bool
	components   map[string]bool
	entries      map[string]types.StashEntry

	gets    []sdkstash.Scope
	sets    []setCall
	deletes []string
	lists   []sdkstash.Scope
}

func (f *fakeAPI) Get(_ context.Context, scope sdkstash.Scope, key string) (*types.StashEntry, error) {
	f.gets = append(f.gets, scope)
	e, ok := f.entries[key]
	if !ok {
		return nil, gql.ErrNotFound
	}
	return &e, nil
}

func (f *fakeAPI) Iter(_ context.Context, scope sdkstash.Scope, _ sdkstash.ListInput) iter.Seq2[types.StashEntry, error] {
	f.lists = append(f.lists, scope)
	return func(yield func(types.StashEntry, error) bool) {
		for _, e := range f.entries {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func (f *fakeAPI) Set(_ context.Context, scope sdkstash.Scope, key string, input sdkstash.SetInput) (*types.StashEntry, error) {
	f.sets = append(f.sets, setCall{scope: scope, key: key, input: input})
	return &types.StashEntry{Key: key, Value: &input.Value}, nil
}

func (f *fakeAPI) Delete(_ context.Context, scope sdkstash.Scope, key string) error {
	f.deletes = append(f.deletes, scope.String()+"."+key)
	return nil
}

func (f *fakeAPI) EnvironmentExists(_ context.Context, id string) (bool, error) {
	return f.environments[id], nil
}

func (f *fakeAPI) ComponentExists(_ context.Context, id string) (bool, error) {
	return f.components[id], nil
}

func ptr(s string) *string { return &s }

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address   string
		wantScope string
		wantKey   string
		wantErr   bool
	}{
		{address: "api-prod.db/hostname", wantScope: "api-prod", wantKey: "db/hostname"},
		{address: "env:api-prod.config.v2", wantScope: "env:api-prod", wantKey: "config.v2"},
		{address: "a1b2c3d4-0000-0000-0000-000000000001.migrations/ran", wantScope: "a1b2c3d4-0000-0000-0000-000000000001", wantKey: "migrations/ran"},
		{address: "api", wantErr: true},
		{address: ".key", wantErr: true},
		{address: "api.", wantErr: true},
		{address: "api.has space", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.address, func(t *testing.T) {
			scope, key, err := stash.ParseAddress(tc.address)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAddress: %v", err)
			}
			if scope != tc.wantScope || key != tc.wantKey {
				t.Errorf("got (%q, %q), want (%q, %q)", scope, key, tc.wantScope, tc.wantKey)
			}
		})
	}
}

func TestResolveScope(t *testing.T) {
	api := &fakeAPI{
		environments: map[string]bool{"api-prod": true, "api-db": true},
		components:   map[string]bool{"api-cache": true, "api-db": true},
	}
	tests := []struct {
		ref     string
		want    string
		wantErr string
	}{
		{ref: "api", want: "project:api"},
		{ref: "api-prod", want: "environment:api-prod"},
		{ref: "api-cache", want: "component:api-cache"},
		{ref: "api-prod-db", want: "instance:api-prod-db"},
		{ref: "a1b2c3d4-0000-0000-0000-000000000001", want: "deployment:a1b2c3d4-0000-0000-0000-000000000001"},
		{ref: "project:api", want: "project:api"},
		{ref: "env:api-db", want: "environment:api-db"},
		{ref: "environment:api-db", want: "environment:api-db"},
		{ref: "component:api-db", want: "component:api-db"},
		{ref: "instance:api-prod-db", want: "instance:api-prod-db"},
		{ref: "deployment:a1b2c3d4-0000-0000-0000-000000000001", want: "deployment:a1b2c3d4-0000-0000-0000-000000000001"},
		{ref: "api-db", wantErr: "api-db matches an environment and a component; use env:api-db or component:api-db"},
		{ref: "api-gone", wantErr: "no environment or component api-gone: not found"},
		{ref: "org:api", wantErr: `unknown scope prefix "org"`},
		{ref: "API", wantErr: `invalid scope "API"`},
		{ref: "a-b-c-d", wantErr: `invalid scope "a-b-c-d"`},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			got, err := stash.ResolveScope(t.Context(), api, tc.ref)
			if tc.wantErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want prefix %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveScope: %v", err)
			}
			if got.String() != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestResolveScope_NeitherIsNotFound(t *testing.T) {
	_, err := stash.ResolveScope(t.Context(), &fakeAPI{}, "api-gone")
	if !errors.Is(err, gql.ErrNotFound) {
		t.Errorf("err = %v, want it to wrap gql.ErrNotFound", err)
	}
}

func TestRunGet(t *testing.T) {
	api := &fakeAPI{entries: map[string]types.StashEntry{
		"db/hostname": {Key: "db/hostname", Value: ptr("db.internal"), Format: "TEXT"},
		"password":    {Key: "password", Sensitive: true, Format: "TEXT"},
	}}

	tests := []struct {
		name    string
		address string
		output  string
		want    string
		wantErr string
	}{
		{name: "raw value", address: "api.db/hostname", output: "text", want: "db.internal"},
		{name: "json entry", address: "api.db/hostname", output: "json", want: "{\n  \"key\": \"db/hostname\",\n  \"value\": \"db.internal\",\n  \"format\": \"TEXT\",\n  \"immutable\": false,\n  \"sensitive\": false\n}\n"},
		{name: "redacted", address: "api.password", output: "text", wantErr: "password is sensitive: reading it needs write access to project:api"},
		{name: "missing", address: "api.nope", output: "text", wantErr: "not found"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := stash.RunGet(t.Context(), api, tc.address, tc.output, &out)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RunGet: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("out = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestRunSet(t *testing.T) {
	tests := []struct {
		name    string
		address string
		value   string
		stdin   string
		opts    stash.SetOptions
		want    setCall
		wantErr string
	}{
		{
			name:    "text with ttl",
			address: "api-staging.ami-id",
			value:   "ami-0abc",
			opts:    stash.SetOptions{TTL: 86400},
			want: setCall{
				scope: sdkstash.Scope{Kind: sdkstash.ScopeEnvironment, ID: "api-staging"},
				key:   "ami-id",
				input: sdkstash.SetInput{Value: "ami-0abc", Format: sdkstash.FormatText, TTL: 24 * time.Hour},
			},
		},
		{
			name:    "json immutable sensitive",
			address: "a1b2c3d4-0000-0000-0000-000000000001.stats",
			value:   `{"duration_s":412,"changed":3}`,
			opts:    stash.SetOptions{Format: "JSON", Immutable: true, Sensitive: true},
			want: setCall{
				scope: sdkstash.Scope{Kind: sdkstash.ScopeDeployment, ID: "a1b2c3d4-0000-0000-0000-000000000001"},
				key:   "stats",
				input: sdkstash.SetInput{Value: `{"duration_s":412,"changed":3}`, Format: sdkstash.FormatJSON, Immutable: true, Sensitive: true},
			},
		},
		{
			name:    "stdin",
			address: "api.notes",
			value:   "-",
			stdin:   "line one\nline two\n",
			want: setCall{
				scope: sdkstash.Scope{Kind: sdkstash.ScopeProject, ID: "api"},
				key:   "notes",
				input: sdkstash.SetInput{Value: "line one\nline two\n", Format: sdkstash.FormatText},
			},
		},
		{name: "json not an object", address: "api.stats", value: `[1,2]`, opts: stash.SetOptions{Format: "json"}, wantErr: "--format json needs the value to be a JSON object"},
		{name: "bad format", address: "api.stats", value: "x", opts: stash.SetOptions{Format: "yaml"}, wantErr: `invalid --format "yaml": use text or json`},
		{name: "negative ttl", address: "api.stats", value: "x", opts: stash.SetOptions{TTL: -1}, wantErr: "invalid --ttl -1: use a number of seconds greater than zero"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{environments: map[string]bool{"api-staging": true}}
			var out bytes.Buffer
			err := stash.RunSet(t.Context(), api, tc.address, tc.value, tc.opts, strings.NewReader(tc.stdin), &out)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				if len(api.sets) != 0 {
					t.Errorf("sets = %+v, want none", api.sets)
				}
				return
			}
			if err != nil {
				t.Fatalf("RunSet: %v", err)
			}
			if len(api.sets) != 1 || api.sets[0] != tc.want {
				t.Errorf("sets = %+v, want [%+v]", api.sets, tc.want)
			}
		})
	}
}

func TestRunDelete(t *testing.T) {
	tests := []struct {
		name        string
		force       bool
		stdin       string
		wantDeletes []string
		wantOut     string
	}{
		{name: "confirmed", stdin: "db/hostname\n", wantDeletes: []string{"instance:api-prod-db.db/hostname"}, wantOut: "Type `db/hostname` to delete it from instance `api-prod-db`: Deleted `db/hostname` from instance `api-prod-db`\n"},
		{name: "cancelled", stdin: "no\n", wantOut: "Type `db/hostname` to delete it from instance `api-prod-db`: Deletion cancelled.\n"},
		{name: "forced", force: true, wantDeletes: []string{"instance:api-prod-db.db/hostname"}, wantOut: "Deleted `db/hostname` from instance `api-prod-db`\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &fakeAPI{entries: map[string]types.StashEntry{"db/hostname": {Key: "db/hostname", Value: ptr("db.internal")}}}
			var out bytes.Buffer
			if err := stash.RunDelete(t.Context(), api, "api-prod-db.db/hostname", tc.force, strings.NewReader(tc.stdin), &out); err != nil {
				t.Fatalf("RunDelete: %v", err)
			}
			if strings.Join(api.deletes, ",") != strings.Join(tc.wantDeletes, ",") {
				t.Errorf("deletes = %v, want %v", api.deletes, tc.wantDeletes)
			}
			if out.String() != tc.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tc.wantOut)
			}
		})
	}
}

func TestRunDelete_Missing(t *testing.T) {
	api := &fakeAPI{}
	err := stash.RunDelete(t.Context(), api, "api.nope", true, strings.NewReader(""), &bytes.Buffer{})
	if !errors.Is(err, gql.ErrNotFound) {
		t.Fatalf("err = %v, want gql.ErrNotFound", err)
	}
	if len(api.deletes) != 0 {
		t.Errorf("deletes = %v, want none", api.deletes)
	}
}

func TestRunList(t *testing.T) {
	api := &fakeAPI{
		components: map[string]bool{"api-db": true},
		entries: map[string]types.StashEntry{
			"password": {Key: "password", Format: "TEXT", Sensitive: true, Immutable: true},
		},
	}

	tests := []struct {
		output string
		want   []string
	}{
		{output: "json", want: []string{`"key": "password"`, `"value": null`, `"sensitive": true`}},
		{output: "table", want: []string{"Key", "password", "(sensitive)", "immutable,sensitive"}},
	}
	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "out")
			if err != nil {
				t.Fatal(err)
			}
			if err := stash.RunList(t.Context(), api, "api-db", tc.output, f); err != nil {
				t.Fatalf("RunList: %v", err)
			}
			got, _ := os.ReadFile(f.Name())
			for _, w := range tc.want {
				if !strings.Contains(string(got), w) {
					t.Errorf("output missing %q:\n%s", w, got)
				}
			}
			if tc.output == "json" {
				var entries []types.StashEntry
				if err := json.Unmarshal(got, &entries); err != nil || len(entries) != 1 {
					t.Errorf("json = %s, err = %v", got, err)
				}
			}
		})
	}
	if api.lists[0].String() != "component:api-db" {
		t.Errorf("listed %s, want component:api-db", api.lists[0])
	}
}
