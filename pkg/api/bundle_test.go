package api_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func TestGetBundle(t *testing.T) {
	bundleId := "aws-vpc"
	version := "1.0.0"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundle": map[string]any{
				"id":          bundleId,
				"name":        "AWS VPC",
				"version":     version,
				"description": "AWS Virtual Private Cloud bundle",
				"spec": map[string]any{
					"name": "aws-vpc",
				},
				"specVersion": "1.0.0",
				"paramsSchema": map[string]any{
					"type": "object",
				},
				"connectionsSchema": map[string]any{
					"type": "object",
				},
				"artifactsSchema": map[string]any{
					"type": "object",
				},
				"operatorGuide": "# AWS VPC\n\nThis is a VPC bundle.",
				"createdAt":     "2024-01-01T00:00:00Z",
				"updatedAt":     "2024-01-01T00:00:00Z",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetBundle(t.Context(), &mdClient, bundleId, &version)

	if err != nil {
		t.Fatal(err)
	}

	want := &api.Bundle{
		ID:          bundleId,
		Name:        "AWS VPC",
		Version:     version,
		Description: "AWS Virtual Private Cloud bundle",
		Spec: map[string]any{
			"name": "aws-vpc",
		},
		SpecVersion: "1.0.0",
		ParamsSchema: map[string]any{
			"type": "object",
		},
		ConnectionsSchema: map[string]any{
			"type": "object",
		},
		ArtifactsSchema: map[string]any{
			"type": "object",
		},
		OperatorGuide: "# AWS VPC\n\nThis is a VPC bundle.",
		CreatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestGetBundleWithoutVersion(t *testing.T) {
	bundleId := "aws-vpc"

	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"bundle": map[string]any{
				"id":        bundleId,
				"name":      "AWS VPC",
				"version":   "latest",
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z",
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.GetBundle(t.Context(), &mdClient, bundleId, nil)

	if err != nil {
		t.Fatal(err)
	}

	want := &api.Bundle{
		ID:        bundleId,
		Name:      "AWS VPC",
		Version:   "latest",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}
