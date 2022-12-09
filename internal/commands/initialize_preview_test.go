package commands_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
)

func TestInitializePreview(t *testing.T) {
	projectSlug := "ecomm"
	responses := []interface{}{
		mockQueryResponse("project", map[string]interface{}{
			"slug": projectSlug,
			"defaultParams": map[string]interface{}{
				"database": map[string]interface{}{"username": "root"},
			},
		}),
	}
	client := mockClientWithJSONResponseArray(responses)

	// TODO: this previously took the file writing path, where do we want to handle that?
	previewCfg, err := commands.InitializePreview(client, "faux-org-id", projectSlug)

	if err != nil {
		t.Fatal(err)
	}

	got := previewCfg.PackageParams
	want := map[string]interface{}{
		"database": map[string]interface{}{
			"username": "root",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
