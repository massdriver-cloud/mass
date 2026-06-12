package resourcetype_test

import (
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func TestGet(t *testing.T) {
	type test struct {
		name         string
		resourceType map[string]any
		want         resourcetype.ResourceType
	}
	tests := []test{
		{
			name: "simple",
			resourceType: map[string]any{
				"id":   "123-456",
				"name": "massdriver/test-schema",
				"schema": map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
			},
			want: resourcetype.ResourceType{
				ID:   "123-456",
				Name: "massdriver/test-schema",
				Schema: map[string]any{
					"$id":         "https://example.com/schemas/test-schema.json",
					"$schema":     "http://json-schema.org/draft-07/schema#",
					"description": "A test schema for demonstration purposes.",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := gqltest.NewClient(
				gqltest.RespondWithData(map[string]any{
					"resourceType": tc.resourceType,
				}),
			)
			t.Cleanup(api.SetTransportForTest(mock))
			mdClient, err := massdriver.NewClient(
				massdriver.WithGQLClient(mock),
				massdriver.WithOrganizationID("test-org"),
			)
			if err != nil {
				t.Fatal(err)
			}

			got, err := resourcetype.Get(t.Context(), mdClient, "massdriver/test-schema")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %v, want %v", *got, tc.want)
			}
		})
	}
}

// TestListWalksPages verifies that List follows the API's cursor pagination and
// accumulates every page, rather than returning only the server's first page.
func TestListWalksPages(t *testing.T) {
	page := func(items []map[string]any, next string) map[string]any {
		return map[string]any{
			"resourceTypes": map[string]any{
				"items": items,
				"cursor": map[string]any{
					"next":     next,
					"previous": "",
				},
			},
		}
	}

	mock := gqltest.NewClient(
		gqltest.RespondWithData(page([]map[string]any{
			{"id": "rt-1", "name": "aws/vpc"},
			{"id": "rt-2", "name": "aws/s3"},
		}, "cursor-2")),
		gqltest.RespondWithData(page([]map[string]any{
			{"id": "rt-3", "name": "gcp/bucket"},
		}, "")),
	)
	t.Cleanup(api.SetTransportForTest(mock))
	mdClient, err := massdriver.NewClient(
		massdriver.WithGQLClient(mock),
		massdriver.WithOrganizationID("test-org"),
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := resourcetype.List(t.Context(), mdClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All three resource types, across both pages, must be accumulated.
	wantIDs := []string{"rt-1", "rt-2", "rt-3"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d resource types, want %d (page walk should accumulate all pages): %+v", len(got), len(wantIDs), got)
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("resource type %d: got id %q, want %q", i, got[i].ID, id)
		}
	}

	// Two requests, each carrying an explicit page-size limit (a null cursor
	// 500s the server). The first has no `next`; the second carries the prior
	// page's next cursor.
	reqs := mock.Requests()
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2 (should follow cursor.next)", len(reqs))
	}
	cursor1, ok := reqs[0].Variables["cursor"].(map[string]any)
	if !ok || cursor1["limit"] == nil || cursor1["next"] != nil {
		t.Errorf("first request should send a limit and no next, got %v", reqs[0].Variables["cursor"])
	}
	cursor2, ok := reqs[1].Variables["cursor"].(map[string]any)
	if !ok || cursor2["next"] != "cursor-2" {
		t.Errorf("second request should carry next=cursor-2, got %v", reqs[1].Variables["cursor"])
	}
	if pending := mock.Pending(); pending != 0 {
		t.Errorf("expected all queued responses consumed, %d pending", pending)
	}
}
