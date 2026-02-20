package api_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/gqlmock"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

func TestCreateArtifact(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"createArtifact": map[string]any{
				"result": map[string]any{
					"id":   "artifact-id",
					"name": "artifact-name",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.CreateArtifact(t.Context(), &mdClient, "artifact-name", "artifact-type", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	want := &api.Artifact{
		Name: "artifact-name",
		ID:   "artifact-id",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %v but got %v", want, got)
	}
}

func TestUpdateArtifact(t *testing.T) {
	gqlClient := gqlmock.NewClientWithSingleJSONResponse(map[string]any{
		"data": map[string]any{
			"updateArtifact": map[string]any{
				"result": map[string]any{
					"id":   "artifact-id",
					"name": "updated-name",
				},
				"successful": true,
			},
		},
	})
	mdClient := client.Client{
		GQL: gqlClient,
	}

	got, err := api.UpdateArtifact(t.Context(), &mdClient, "artifact-id", "updated-name", map[string]any{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}

	want := &api.Artifact{
		Name: "updated-name",
		ID:   "artifact-id",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %v but got %v", want, got)
	}
}

func TestGetArtifact(t *testing.T) {
	type test struct {
		name     string
		artifact map[string]any
		want     api.Artifact
	}
	tests := []test{
		{
			name: "simple",
			artifact: map[string]any{
				"id":    "123-456",
				"name":  "my-artifact",
				"type":  "aws-s3",
				"field": "bucket",
				"payload": map[string]any{
					"bucket": "my-bucket",
				},
				"formats":   []string{"json", "yaml"},
				"createdAt": time.Now().Format(time.RFC3339),
				"updatedAt": time.Now().Format(time.RFC3339),
				"origin":    "IMPORTED",
				"artifactDefinition": map[string]any{
					"id":    "def-123",
					"name":  "aws-s3",
					"label": "AWS S3",
				},
				"package": map[string]any{
					"id":   "pkg-123",
					"slug": "my-package",
				},
			},
			want: api.Artifact{
				ID:    "123-456",
				Name:  "my-artifact",
				Type:  "aws-s3",
				Field: "bucket",
				Payload: map[string]any{
					"bucket": "my-bucket",
				},
				Formats: []string{"json", "yaml"},
				Origin:  "IMPORTED",
				ArtifactDefinition: &api.ArtifactDefinitionWithSchema{
					ID:    "def-123",
					Name:  "aws-s3",
					Label: "AWS S3",
				},
				Package: &api.ArtifactPackage{
					ID:   "pkg-123",
					Slug: "my-package",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockQueryResponse("artifact", tc.artifact),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					OrganizationID: "org-123",
				},
			}

			got, err := api.GetArtifact(t.Context(), &mdClient, "123-456")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare fields that we can easily compare
			if got.ID != tc.want.ID {
				t.Errorf("got ID %v, want %v", got.ID, tc.want.ID)
			}
			if got.Name != tc.want.Name {
				t.Errorf("got Name %v, want %v", got.Name, tc.want.Name)
			}
			if got.Type != tc.want.Type {
				t.Errorf("got Type %v, want %v", got.Type, tc.want.Type)
			}
			if got.Field != tc.want.Field {
				t.Errorf("got Field %v, want %v", got.Field, tc.want.Field)
			}
			if got.Origin != tc.want.Origin {
				t.Errorf("got Origin %v, want %v", got.Origin, tc.want.Origin)
			}
			if !reflect.DeepEqual(got.Payload, tc.want.Payload) {
				t.Errorf("got Payload %v, want %v", got.Payload, tc.want.Payload)
			}
			if !reflect.DeepEqual(got.Formats, tc.want.Formats) {
				t.Errorf("got Formats %v, want %v", got.Formats, tc.want.Formats)
			}
			if got.ArtifactDefinition != nil && tc.want.ArtifactDefinition != nil {
				if got.ArtifactDefinition.ID != tc.want.ArtifactDefinition.ID {
					t.Errorf("got ArtifactDefinition.ID %v, want %v", got.ArtifactDefinition.ID, tc.want.ArtifactDefinition.ID)
				}
			}
			if got.Package != nil && tc.want.Package != nil {
				if got.Package.ID != tc.want.Package.ID {
					t.Errorf("got Package.ID %v, want %v", got.Package.ID, tc.want.Package.ID)
				}
			}
		})
	}
}

func TestDownloadArtifact(t *testing.T) {
	type test struct {
		name     string
		format   string
		response map[string]any
		want     string
	}
	tests := []test{
		{
			name:   "json format",
			format: "json",
			response: map[string]any{
				"renderedArtifact": `{"bucket": "my-bucket"}`,
			},
			want: `{"bucket": "my-bucket"}`,
		},
		{
			name:   "yaml format",
			format: "yaml",
			response: map[string]any{
				"renderedArtifact": "bucket: my-bucket\n",
			},
			want: "bucket: my-bucket\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := []any{
				gqlmock.MockQueryResponse("downloadArtifact", tc.response),
			}

			mdClient := client.Client{
				GQL: gqlmock.NewClientWithJSONResponseArray(responses),
				Config: config.Config{
					OrganizationID: "org-123",
				},
			}

			got, err := api.DownloadArtifact(t.Context(), &mdClient, "123-456", tc.format)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
