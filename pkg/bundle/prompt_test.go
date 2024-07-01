package bundle

import (
	"reflect"
	"strings"
	"testing"
)

func TestNameValidate(t *testing.T) {
	goodValues := []string{
		"ab", "abc", "a1", "a-1", "a--1--2--b",
		strings.Repeat("a", 53),
	}
	for _, val := range goodValues {
		if err := bundleNameValidate(val); err != nil {
			t.Errorf("expected no error for '%s': %v", val, err)
		}
	}

	badValues := []string{
		"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
		"-", "a-", "-a", "1-", "-1",
		"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
		".", "a.", ".a", "a.b", "1.", ".1", "1.2",
		" ", "a ", " a", "a b", "1 ", " 1", "1 2",
		"1111", "1-1-1", "----", strings.Repeat("a", 54),
	}
	for _, val := range badValues {
		if err := bundleNameValidate(val); err == nil {
			t.Errorf("expected error for '%s'", val)
		}
	}
}

func TestConnNameValidate(t *testing.T) {
	goodValues := []string{
		"ab", "abc", "a1", "a_1", "a__1__2__b",
		strings.Repeat("a", 53),
	}
	for _, val := range goodValues {
		if err := connNameValidate(val); err != nil {
			t.Errorf("expected no error for '%s': %v", val, err)
		}
	}

	badValues := []string{
		"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
		"-", "a-", "-a", "1-", "-1",
		"_", "a_", "_a", "a-b", "1_", "_1", "1_2",
		".", "a.", ".a", "a.b", "1.", ".1", "1.2",
		" ", "a ", " a", "a b", "1 ", " 1", "1 2",
		"1111", "1-1-1", "----", strings.Repeat("a", 54),
	}
	for _, val := range badValues {
		if err := connNameValidate(val); err == nil {
			t.Errorf("expected error for '%s'", val)
		}
	}
}

func TestGetConnectionEnvs(t *testing.T) {
	type test struct {
		name               string
		connectionName     string
		artifactDefinition map[string]interface{}
		want               map[string]string
	}
	tests := []test{
		{
			name:           "Basic",
			connectionName: "foobar",
			artifactDefinition: map[string]interface{}{
				"$md": map[string]interface{}{
					"envTemplates": map[string]interface{}{
						"SOME_ENV":    ".connection_name.data.foo.bar",
						"ANOTHER_ENV": "lol | split() | .connection_name | abc",
					},
				},
			},
			want: map[string]string{
				"SOME_ENV":    ".foobar.data.foo.bar",
				"ANOTHER_ENV": "lol | split() | .foobar | abc",
			},
		},
		{
			name:           "Empty",
			connectionName: "foobar",
			artifactDefinition: map[string]interface{}{
				"$md": map[string]interface{}{},
			},
			want: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetConnectionEnvs(tc.connectionName, tc.artifactDefinition)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
