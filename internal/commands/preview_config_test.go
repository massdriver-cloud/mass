package commands_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestPreviewConfigGetCredentials(t *testing.T) {
	cfg := commands.PreviewConfig{
		Credentials: map[string]string{
			"massdriver/aws-iam-role": "foo",
		},
	}

	got := cfg.GetCredentials()
	want := []api.Credential{
		api.Credential{ArtifactDefinitionType: "massdriver/aws-iam-role", ArtifactId: "foo"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
