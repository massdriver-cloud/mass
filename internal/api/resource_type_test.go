package api_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

func newTestClient(t *testing.T, responses ...gqltest.Response) (*massdriver.Client, *gqltest.Client) {
	t.Helper()

	mock := gqltest.NewClient(responses...)
	t.Cleanup(api.SetTransportForTest(mock))

	mdClient, err := massdriver.NewClient(
		massdriver.WithGQLClient(mock),
		massdriver.WithOrganizationID("test-org"),
	)
	if err != nil {
		t.Fatalf("failed to build test client: %v", err)
	}
	return mdClient, mock
}

func TestPublishResourceType(t *testing.T) {
	schema := map[string]any{
		"$md":  map[string]any{"name": "aws-iam-role", "label": "AWS IAM Role"},
		"type": "object",
	}

	mdClient, mock := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"publishResourceType": map[string]any{
			"successful": true,
			"messages":   []any{},
			"result": map[string]any{
				"id":                    "aws-iam-role@0.0.0",
				"name":                  "aws-iam-role",
				"version":               "0.0.0",
				"connectionOrientation": "LINK",
				"schema":                schema,
			},
		},
	}))

	got, err := api.PublishResourceType(t.Context(), mdClient, api.PublishResourceTypeInput{Schema: schema})
	if err != nil {
		t.Fatalf("PublishResourceType returned an error: %v", err)
	}
	if got.Name != "aws-iam-role" {
		t.Errorf("Name = %q, want aws-iam-role", got.Name)
	}
	if got.Version != "0.0.0" {
		t.Errorf("Version = %q, want 0.0.0", got.Version)
	}

	reqs := mock.Requests()
	if len(reqs) != 1 {
		t.Fatalf("got %d requests, want 1", len(reqs))
	}
	if reqs[0].OpName != "publishResourceType" {
		t.Errorf("OpName = %q, want publishResourceType", reqs[0].OpName)
	}
	if reqs[0].Variables["organizationId"] != "test-org" {
		t.Errorf("organizationId = %v, want test-org", reqs[0].Variables["organizationId"])
	}
}

// The `Map!` scalar's wire form is a JSON-encoded string, not a nested object.
// Getting it wrong ships a payload the API rejects.
func TestPublishResourceTypeEncodesSchemaAsScalar(t *testing.T) {
	mdClient, mock := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"publishResourceType": map[string]any{
			"successful": true,
			"result":     map[string]any{"name": "aws-iam-role", "version": "0.0.0"},
		},
	}))

	_, err := api.PublishResourceType(t.Context(), mdClient, api.PublishResourceTypeInput{
		Schema: map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatalf("PublishResourceType returned an error: %v", err)
	}

	input, ok := mock.Requests()[0].Variables["input"].(map[string]any)
	if !ok {
		t.Fatalf("input variable = %T, want map[string]any", mock.Requests()[0].Variables["input"])
	}
	sent, ok := input["schema"].(string)
	if !ok {
		t.Fatalf("input.schema = %T, want a JSON-encoded string", input["schema"])
	}
	if !strings.Contains(sent, `"type":"object"`) {
		t.Errorf("input.schema = %q, want it to contain the encoded schema", sent)
	}
}

func TestPublishResourceTypeUnsuccessful(t *testing.T) {
	mdClient, _ := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"publishResourceType": map[string]any{
			"successful": false,
			"messages": []any{
				map[string]any{"code": "invalid", "field": "schema", "message": "is invalid"},
			},
		},
	}))

	_, err := api.PublishResourceType(t.Context(), mdClient, api.PublishResourceTypeInput{
		Schema: map[string]any{"type": "object"},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, want := range []string{"publish resource type", "schema: is invalid", "(invalid)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

// A successful:true response with a null result would otherwise be dereferenced
// straight into a nil-pointer panic.
func TestPublishResourceTypeSuccessfulWithNoResult(t *testing.T) {
	mdClient, _ := newTestClient(t, gqltest.RespondWithData(map[string]any{
		"publishResourceType": map[string]any{"successful": true, "result": nil},
	}))

	_, err := api.PublishResourceType(t.Context(), mdClient, api.PublishResourceTypeInput{
		Schema: map[string]any{"type": "object"},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "returned no resource type") {
		t.Errorf("unexpected error: %v", err)
	}
}
