package api_test

import (
	"context"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetDeployment(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployment": map[string]interface{}{
				"id":     "uuid1",
				"status": "PROVISIONING",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployment, err := api.GetDeployment(context.Background(), &mdClient, "uuid1")

	if err != nil {
		t.Fatal(err)
	}

	got := deployment.Status
	want := "PROVISIONING"

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}
}

func TestDeployPackage(t *testing.T) {
	want := "deployment-uuid1"
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]interface{}{
		"data": map[string]interface{}{
			"deployPackage": map[string]interface{}{
				"result": map[string]interface{}{
					"id": want,
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	deployment, err := api.DeployPackage(context.Background(), &mdClient, "target-id", "manifest-id", "foo")

	if err != nil {
		t.Fatal(err)
	}

	got := deployment.ID

	if got != want {
		t.Errorf("got %s , wanted %s", got, want)
	}
}
