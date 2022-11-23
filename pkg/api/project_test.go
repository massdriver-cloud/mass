package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
)

func TestGetProject(t *testing.T) {
	mux := muxWithJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"project": map[string]string{
				"id":            "uuid1",
				"slug":          "sluggy",
				"defaultParams": `{"foo":"bar"}`,
			},
		},
	})
	client := mockClient(mux)
	project, err := api.GetProject(client, "faux-org-id", "sluggy")

	if err != nil {
		t.Fatal(err)
	}

	got := project.Slug

	want := "sluggy"

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}
