package grants_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/grants"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestParseConditions(t *testing.T) {
	cases := []struct {
		name  string
		pairs []string
		want  types.PolicyConditions
	}{
		{
			name:  "single value",
			pairs: []string{"team=platform"},
			want:  types.PolicyConditions{"team": {"platform"}},
		},
		{
			name:  "repeated key unions into a closed set",
			pairs: []string{"team=platform", "team=data"},
			want:  types.PolicyConditions{"team": {"platform", "data"}},
		},
		{
			name:  "repeated identical value is deduped",
			pairs: []string{"team=platform", "team=platform"},
			want:  types.PolicyConditions{"team": {"platform"}},
		},
		{
			name:  "star is the per-key wildcard",
			pairs: []string{"region=*"},
			want:  types.PolicyConditions{"region": nil},
		},
		{
			name:  "distinct keys are independent",
			pairs: []string{"team=platform", "region=*", "stage=prod"},
			want:  types.PolicyConditions{"team": {"platform"}, "region": nil, "stage": {"prod"}},
		},
		{
			name:  "value may contain an equals sign",
			pairs: []string{"expr=a=b"},
			want:  types.PolicyConditions{"expr": {"a=b"}},
		},
		{
			// A comma splits pairs in --attributes; here it is just a character.
			name:  "value may contain a comma",
			pairs: []string{"team=platform,data"},
			want:  types.PolicyConditions{"team": {"platform,data"}},
		},
		{
			name:  "surrounding whitespace is trimmed",
			pairs: []string{"  team  =  platform  "},
			want:  types.PolicyConditions{"team": {"platform"}},
		},
		{
			// Untrimmed this stores a literal " *", which matches no recipient.
			name:  "spaced wildcard is still a wildcard",
			pairs: []string{"region= *"},
			want:  types.PolicyConditions{"region": nil},
		},
		{
			name:  "inner whitespace is preserved",
			pairs: []string{"team=platform team"},
			want:  types.PolicyConditions{"team": {"platform team"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := grants.ParseConditions(tc.pairs)
			if err != nil {
				t.Fatalf("ParseConditions(%v) returned error: %v", tc.pairs, err)
			}
			if !conditionsEqual(got, tc.want) {
				t.Errorf("ParseConditions(%v) = %v, want %v", tc.pairs, got, tc.want)
			}
		})
	}
}

func TestParseConditionsErrors(t *testing.T) {
	cases := []struct {
		name      string
		pairs     []string
		wantError string
	}{
		{"no pairs", nil, "no conditions given"},
		{"missing equals", []string{"team"}, "must be formatted as key=value"},
		{"empty key", []string{"=platform"}, "empty attribute name"},
		{"empty value", []string{"team="}, "empty value"},
		{"whitespace-only value", []string{"team=   "}, "empty value"},
		{"wildcard then specific", []string{"team=*", "team=platform"}, "use one or the other"},
		{"specific then wildcard", []string{"team=platform", "team=*"}, "use one or the other"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := grants.ParseConditions(tc.pairs)
			if err == nil {
				t.Fatalf("ParseConditions(%v) succeeded, wanted error", tc.pairs)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("ParseConditions(%v) error = %q, want it to contain %q", tc.pairs, err, tc.wantError)
			}
		})
	}
}

func TestParseConditionsRepeatedWildcard(t *testing.T) {
	got, err := grants.ParseConditions([]string{"region=*", "region=*"})
	if err != nil {
		t.Fatalf("ParseConditions returned error: %v", err)
	}
	if !conditionsEqual(got, types.PolicyConditions{"region": nil}) {
		t.Errorf("ParseConditions = %v, want region wildcard", got)
	}
}

func TestResolveConditions(t *testing.T) {
	conditions, err := grants.ResolveConditions([]string{"team=platform"}, "", false, "all-projects")
	if err != nil {
		t.Fatalf("ResolveConditions returned error: %v", err)
	}
	if !conditionsEqual(conditions, types.PolicyConditions{"team": {"platform"}}) {
		t.Errorf("ResolveConditions = %v, want team=platform", conditions)
	}
}

func TestResolveConditionsAllIsTheOnlyWildcard(t *testing.T) {
	conditions, err := grants.ResolveConditions(nil, "", true, "all-projects")
	if err != nil {
		t.Fatalf("ResolveConditions returned error: %v", err)
	}
	if conditions != nil {
		t.Errorf("ResolveConditions(all) = %v, want nil", conditions)
	}
}

func TestResolveConditionsErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "conditions.json")
	if err := os.WriteFile(file, []byte(`{"team":["platform"]}`), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		pairs     []string
		file      string
		all       bool
		wantError string
	}{
		{"nothing given", nil, "", false, "--all-projects"},
		{"pairs and file", []string{"team=platform"}, file, false, "mutually exclusive"},
		{"pairs and all", []string{"team=platform"}, "", true, "mutually exclusive"},
		{"file and all", nil, file, true, "mutually exclusive"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := grants.ResolveConditions(tc.pairs, tc.file, tc.all, "all-projects")
			if err == nil {
				t.Fatal("ResolveConditions succeeded, wanted error")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("ResolveConditions error = %q, want it to contain %q", err, tc.wantError)
			}
		})
	}
}

func TestParseConditionsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "conditions.json")
	if err := os.WriteFile(file, []byte(`{"team":["platform"," data "],"region":"*"}`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := grants.ParseConditionsFile(file, "all-projects")
	if err != nil {
		t.Fatalf("ParseConditionsFile returned error: %v", err)
	}
	want := types.PolicyConditions{"team": {"platform", "data"}, "region": nil}
	if !conditionsEqual(got, want) {
		t.Errorf("ParseConditionsFile = %v, want %v", got, want)
	}
}

func TestParseConditionsFileRejectsWildcard(t *testing.T) {
	dir := t.TempDir()

	for _, body := range []string{"null", "{}"} {
		t.Run(body, func(t *testing.T) {
			file := filepath.Join(dir, "wildcard.json")
			if err := os.WriteFile(file, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := grants.ParseConditionsFile(file, "all-projects")
			if err == nil {
				t.Fatal("ParseConditionsFile succeeded, wanted error")
			}
			if !strings.Contains(err.Error(), "--all-projects") {
				t.Errorf("error = %q, want it to name --all-projects", err)
			}
		})
	}
}

func TestParseConditionsFileErrors(t *testing.T) {
	dir := t.TempDir()

	malformed := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(malformed, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	emptyKey := filepath.Join(dir, "empty-key.json")
	if err := os.WriteFile(emptyKey, []byte(`{"":["platform"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	emptyValue := filepath.Join(dir, "empty-value.json")
	if err := os.WriteFile(emptyValue, []byte(`{"team":["  "]}`), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		path      string
		wantError string
	}{
		{"missing file", filepath.Join(dir, "absent.json"), "reading conditions file"},
		{"malformed json", malformed, "parsing conditions file"},
		{"empty attribute name", emptyKey, "empty attribute name"},
		{"empty value", emptyValue, "empty value"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := grants.ParseConditionsFile(tc.path, "all-projects")
			if err == nil {
				t.Fatal("ParseConditionsFile succeeded, wanted error")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantError)
			}
		})
	}
}

func TestFormatConditions(t *testing.T) {
	cases := []struct {
		name       string
		conditions types.PolicyConditions
		want       string
	}{
		{"nil is org-wide", nil, "everyone"},
		{"empty map is org-wide", types.PolicyConditions{}, "everyone"},
		{"per-key wildcard", types.PolicyConditions{"region": nil}, "region=*"},
		{"empty slice is a wildcard too", types.PolicyConditions{"region": {}}, "region=*"},
		{"closed set", types.PolicyConditions{"team": {"platform", "data"}}, "team=platform|data"},
		{
			name:       "keys are sorted for stable output",
			conditions: types.PolicyConditions{"team": {"platform"}, "region": nil, "stage": {"prod"}},
			want:       "region=*, stage=prod, team=platform",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := grants.FormatConditions(tc.conditions); got != tc.want {
				t.Errorf("FormatConditions(%v) = %q, want %q", tc.conditions, got, tc.want)
			}
		})
	}
}

// A nil and an empty value slice are the same per-key wildcard.
func conditionsEqual(got, want types.PolicyConditions) bool {
	if (got == nil) != (want == nil) || len(got) != len(want) {
		return false
	}
	for key := range maps.Keys(want) {
		gotValues, ok := got[key]
		if !ok || !slices.Equal(gotValues, want[key]) {
			return false
		}
	}
	return true
}
