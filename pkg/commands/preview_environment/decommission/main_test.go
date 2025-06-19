package decommission_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/preview_environment/decommission"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestDecommissionPreviewEnvironment(t *testing.T) {
	prNumber := 69
	targetSlug := fmt.Sprintf("p%d", prNumber)
	projectTargetSlug := "ecomm-" + targetSlug
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"decommissionPreviewEnvironment": map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "envuuid1",
					"slug": targetSlug,
				},
				"successful": true,
			},
		},
	})

	mdClient := client.Client{
		GQL: gqlClient,
	}

	environment, err := decommission.Run(context.Background(), &mdClient, projectTargetSlug)

	if err != nil {
		t.Fatal(err)
	}

	got := environment.Slug
	want := "p69"

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
