package api_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
)

func TestPreviewConfigGetCredentials(t *testing.T) {
	cfg := api.PreviewConfig{
		Credentials: map[string]string{
			"massdriver/aws-iam-role": "foo",
		},
	}

	got := cfg.GetCredentials()
	want := []api.Credential{
		{ArtifactDefinitionType: "massdriver/aws-iam-role", ArtifactId: "foo"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
