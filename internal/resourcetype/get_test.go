package resourcetype_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func newMockClient(t *testing.T, responses ...gqltest.Response) *massdriver.Client {
	t.Helper()
	mock := gqltest.NewClient(responses...)
	mdClient, err := massdriver.NewClient(
		massdriver.WithGQLClient(mock),
		massdriver.WithOrganizationID("test-org"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return mdClient
}

func TestGet(t *testing.T) {
	mdClient := newMockClient(t, gqltest.RespondWithData(map[string]any{
		"resourceType": map[string]any{
			"id":   "aws-s3-bucket",
			"name": "AWS S3 Bucket",
			"schema": map[string]any{
				"$id":         "https://example.com/schemas/test-schema.json",
				"$schema":     "http://json-schema.org/draft-07/schema#",
				"description": "A test schema for demonstration purposes.",
			},
		},
	}))

	got, err := resourcetype.Get(t.Context(), mdClient, "aws-s3-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "aws-s3-bucket" {
		t.Errorf("ID = %q, want aws-s3-bucket", got.ID)
	}
	if got.Name != "AWS S3 Bucket" {
		t.Errorf("Name = %q, want AWS S3 Bucket", got.Name)
	}
	if _, ok := got.Schema["$id"]; !ok {
		t.Errorf("Schema should carry the resolved JSON schema, got %v", got.Schema)
	}
}

// TestList verifies List sources from the OCI-repo catalog filtered to
// resource-type artifacts and maps each repo into a ResourceType.
func TestList(t *testing.T) {
	mdClient := newMockClient(t, gqltest.RespondWithData(map[string]any{
		"ociRepos": map[string]any{
			"cursor": map[string]any{},
			"items": []map[string]any{
				{"id": "aws-vpc", "name": "aws-vpc", "artifactType": "application/vnd.massdriver.resource-type.v1+json"},
				{"id": "aws-s3", "name": "aws-s3", "artifactType": "application/vnd.massdriver.resource-type.v1+json"},
			},
		},
	}))

	got, err := resourcetype.List(t.Context(), mdClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d resource types, want 2: %+v", len(got), got)
	}
	wantIDs := []string{"aws-vpc", "aws-s3"}
	for i, id := range wantIDs {
		if got[i].ID != id || got[i].Name != id {
			t.Errorf("resource type %d: got id=%q name=%q, want %q", i, got[i].ID, got[i].Name, id)
		}
	}
}
