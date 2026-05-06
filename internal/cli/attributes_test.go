package cli_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/cli"
)

func TestAttributesToAnyMap(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]any
	}{
		{name: "nil returns nil", in: nil, want: nil},
		{name: "empty returns nil", in: map[string]string{}, want: nil},
		{
			name: "single entry",
			in:   map[string]string{"team": "ops"},
			want: map[string]any{"team": "ops"},
		},
		{
			name: "multiple entries",
			in:   map[string]string{"team": "ops", "system": "api"},
			want: map[string]any{"team": "ops", "system": "api"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cli.AttributesToAnyMap(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("AttributesToAnyMap(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestAttributesToAnyMap_NilForServer confirms the contract that a nil/empty
// input produces nil output — required so the eventual JSON serialization
// elides the field entirely (or sends bare null), rather than sending an
// empty map the server would reject.
func TestAttributesToAnyMap_NilForServer(t *testing.T) {
	if cli.AttributesToAnyMap(nil) != nil {
		t.Error("nil input must produce nil output")
	}
	if cli.AttributesToAnyMap(map[string]string{}) != nil {
		t.Error("empty input must produce nil output")
	}
}

func TestStringMapToAnyMap(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]any
	}{
		{name: "nil returns nil", in: nil, want: nil},
		{name: "empty returns nil", in: map[string]string{}, want: nil},
		{
			name: "round-trips a stored attribute map",
			in:   map[string]string{"team": "ops", "cost_center": "infra"},
			want: map[string]any{"team": "ops", "cost_center": "infra"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cli.StringMapToAnyMap(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("StringMapToAnyMap(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
