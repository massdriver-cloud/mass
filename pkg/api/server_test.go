package api_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/stretchr/testify/assert"
)

func TestGetServer(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"server": map[string]any{
				"version": "1.2.3",
				"mode":    "MANAGED",
				"appUrl":  "https://app.massdriver.cloud",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetServer(t.Context(), &mdClient)
	if err != nil {
		t.Fatal(err)
	}

	want := api.Server{
		Version: "1.2.3",
		Mode:    "MANAGED",
		AppURL:  "https://app.massdriver.cloud",
	}

	assert.Equal(t, &want, got)
}
