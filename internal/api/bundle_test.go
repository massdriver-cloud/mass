package api_test

import (
	"errors"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestGetBundleDependencyCardinalities(t *testing.T) {
	mdClient, mock := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"bundle": map[string]any{
			"version": "1.2.0",
			"dependencies": []any{
				map[string]any{"name": "network", "cardinality": "ONE"},
				map[string]any{"name": "queues", "cardinality": "MANY"},
			},
		},
	}))

	got, err := api.GetBundleDependencyCardinalities(t.Context(), mdClient, "aws-worker")
	if err != nil {
		t.Fatalf("GetBundleDependencyCardinalities returned an error: %v", err)
	}
	if got.Version != "1.2.0" {
		t.Errorf("Version = %q, want 1.2.0", got.Version)
	}
	if got.Cardinalities["network"] != "ONE" || got.Cardinalities["queues"] != "MANY" {
		t.Errorf("Cardinalities = %v, want network=ONE queues=MANY", got.Cardinalities)
	}

	req := mock.Requests()[0]
	if req.Variables["organizationId"] != "test-org" {
		t.Errorf("organizationId = %v, want test-org", req.Variables["organizationId"])
	}
	if req.Variables["id"] != "aws-worker" {
		t.Errorf("id = %v, want aws-worker", req.Variables["id"])
	}
}

func TestGetBundleDependencyCardinalitiesNotFound(t *testing.T) {
	tests := []struct {
		name     string
		response gqltest.Response
	}{
		{
			name:     "null bundle",
			response: gqltest.RespondWithData(map[string]any{"bundle": nil}),
		},
		{
			name: "NOT_FOUND error",
			response: gqltest.RespondWithJSON(map[string]any{
				"data": map[string]any{"bundle": nil},
				"errors": []any{map[string]any{
					"message":    "not found",
					"extensions": map[string]any{"code": "NOT_FOUND"},
				}},
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mdClient, _ := newTestClient(t, tc.response)
			_, err := api.GetBundleDependencyCardinalities(t.Context(), mdClient, "aws-worker")
			if !errors.Is(err, gql.ErrNotFound) {
				t.Errorf("err = %v, want gql.ErrNotFound", err)
			}
		})
	}
}
