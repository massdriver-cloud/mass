package scalars_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api/scalars"
)

func TestMarshalJSON(t *testing.T) {
	data := map[string]any{"foo": "bar"}
	got, _ := scalars.MarshalJSON(data)

	want := `"{\"foo\":\"bar\"}"`

	if string(got) != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}

func TestUnmarshalJSONObject(t *testing.T) {
	want := map[string]any{"foo": "bar"}

	// Test JSON scalar returned as object
	data := []byte(`{"foo": "bar"}`)
	got := map[string]any{}

	if err := scalars.UnmarshalJSON(data, &got); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
