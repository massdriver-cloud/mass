package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/gqlmock"
)

func TestGetProject(t *testing.T) {
	client := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"project": map[string]interface{}{
				"id":   "uuid1",
				"slug": "sluggy",
				"defaultParams": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	})

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
