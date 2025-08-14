package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"

	"github.com/stretchr/testify/assert"
)

func TestGetBundleVersions(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundle": map[string]any{
				"versions": []string{
					"1.0.0",
					"1.1.0",
				},
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetBundleVersions(t.Context(), &mdClient, "aws-ecs-cluster")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"1.0.0",
		"1.1.0",
	}

	assert.ElementsMatch(t, got, want)
}
