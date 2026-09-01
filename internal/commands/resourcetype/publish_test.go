package resourcetype //nolint:testpackage // needs access to unexported checkDuplicateVersion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql/gqltest"
)

// TestRunPublishValidation covers the local validation RunPublish performs
// before it touches the OCI registry: rejecting unsupported file types and
// requiring a name in the massdriver.yaml. These paths short-circuit before the
// massdriver client is used, so a nil client is fine. (A missing version is not
// an error — it warns and defaults to 0.0.0.)
func TestRunPublishValidation(t *testing.T) {
	dir := t.TempDir()

	unsupported := filepath.Join(dir, "schema.txt")
	if err := os.WriteFile(unsupported, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	noNameDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(noNameDir, "massdriver.yaml"), []byte("version: 1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	emptyDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{name: "unsupported file type", path: unsupported, contains: "unsupported resource type path"},
		{name: "directory without massdriver.yaml", path: emptyDir, contains: "no massdriver.yaml"},
		{name: "missing name", path: noNameDir, contains: "name is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := RunPublish(t.Context(), nil, tc.path)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("expected error to contain %q, got: %v", tc.contains, err)
			}
		})
	}
}

// schemaDir writes the two json-schemas documents RunPublish validates
// against and returns a file:// base URL pointing at them. The schema loader
// handles file:// the same as https://, so this exercises the real validation
// path without standing up an HTTP server.
func schemaDir(t *testing.T, resourceTypeSchema map[string]any) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "json-schemas"), 0750); err != nil {
		t.Fatal(err)
	}
	write := func(name string, schema map[string]any) {
		body, err := json.Marshal(schema)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "json-schemas", name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("resource-type.json", resourceTypeSchema)
	write("draft-7.json", map[string]any{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"})

	return "file://" + dir
}

// legacyClient wires a client at the schema directory with the gql mock
// installed as both the SDK and api-package transport.
func legacyClient(t *testing.T, baseURL string, responses ...gqltest.Response) *massdriver.Client {
	t.Helper()

	mock := gqltest.NewClient(responses...)
	t.Cleanup(api.SetTransportForTest(mock))

	mdClient, err := massdriver.NewClient(
		massdriver.WithGQLClient(mock),
		massdriver.WithOrganizationID("test-org"),
		massdriver.WithBaseURL(baseURL),
	)
	if err != nil {
		t.Fatalf("failed to build test client: %v", err)
	}
	return mdClient
}

// captureStdout runs fn with os.Stdout pointed at a temp file and returns what
// it printed. The deprecation notice is the whole point of the legacy path, so
// it has to be asserted on.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = f                       //nolint:reassign // capturing the deprecation notice
	defer func() { os.Stdout = orig }() //nolint:reassign // restore

	fn()

	out, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func writeSchema(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

const legacySchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schemas.massdriver.cloud/aws-iam-role.json",
  "title": "AWS IAM Role",
  "type": "object",
  "properties": {"arn": {"type": "string"}}
}`

// TestRunPublishLegacySchema covers the deprecated raw-schema flow end to end:
// it must route away from OCI, validate, publish through the legacy mutation,
// and warn the user. This path was dropped and re-added once already (commit
// 1e16b3b), so it is pinned here.
func TestRunPublishLegacySchema(t *testing.T) {
	path := writeSchema(t, "aws-iam-role.json", legacySchemaJSON)

	mdClient := legacyClient(t, schemaDir(t, map[string]any{"type": "object"}), gqltest.RespondWithData(map[string]any{
		"publishResourceType": map[string]any{
			"successful": true,
			"messages":   []any{},
			"result": map[string]any{
				"id":      "aws-iam-role@0.0.0",
				"name":    "aws-iam-role",
				"version": "0.0.0",
			},
		},
	}))

	var name, version string
	var publishErr error
	out := captureStdout(t, func() {
		name, version, publishErr = RunPublish(t.Context(), mdClient, path)
	})

	if publishErr != nil {
		t.Fatalf("RunPublish returned an error: %v", publishErr)
	}
	if name != "aws-iam-role" {
		t.Errorf("name = %q, want aws-iam-role", name)
	}
	// A raw schema has no version of its own; the API stores it as the
	// unversioned 0.0.0 document.
	if version != "0.0.0" {
		t.Errorf("version = %q, want 0.0.0", version)
	}
	if !strings.Contains(out, "deprecated") {
		t.Errorf("expected a deprecation warning on stdout, got: %q", out)
	}
	if !strings.Contains(out, "mass resource-type convert "+path) {
		t.Errorf("expected the warning to point at the convert command for %s, got: %q", path, out)
	}
}

// repoWithTags builds a client whose OciRepos.Get returns a repository carrying
// the given published tags. Get selects tags through a paginated `items`
// envelope, which is the shape the SDK unwraps.
func repoWithTags(t *testing.T, name string, tags ...string) *massdriver.Client {
	t.Helper()

	items := make([]map[string]any, 0, len(tags))
	for _, tag := range tags {
		items = append(items, map[string]any{"tag": tag})
	}

	return newMockClient(t, gqltest.RespondWithData(map[string]any{
		"ociRepo": map[string]any{
			"id":           name,
			"name":         name,
			"artifactType": "application/vnd.massdriver.resource-type.v1+json",
			"tags":         map[string]any{"items": items},
		},
	}))
}

func newMockClient(t *testing.T, responses ...gqltest.Response) *massdriver.Client {
	t.Helper()
	mdClient, err := massdriver.NewClient(
		massdriver.WithGQLClient(gqltest.NewClient(responses...)),
		massdriver.WithOrganizationID("test-org"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return mdClient
}

// TestCheckDuplicateVersion pins the local half of publish immutability. Too
// strict and valid publishes are blocked before they reach the API; too loose
// and the user only learns of the collision after the packaging work.
func TestCheckDuplicateVersion(t *testing.T) {
	tests := []struct {
		name     string
		client   func(t *testing.T) *massdriver.Client
		version  string
		wantErr  string
		wantPass bool
	}{
		{
			name:     "version not yet published",
			client:   func(t *testing.T) *massdriver.Client { return repoWithTags(t, "aws-s3-bucket", "1.0.0", "1.1.0") },
			version:  "2.0.0",
			wantPass: true,
		},
		{
			name:    "version already published",
			client:  func(t *testing.T) *massdriver.Client { return repoWithTags(t, "aws-s3-bucket", "1.0.0", "2.0.0") },
			version: "2.0.0",
			wantErr: "version 2.0.0 already exists for resource type aws-s3-bucket",
		},
		{
			name:     "repository has no published versions",
			client:   func(t *testing.T) *massdriver.Client { return repoWithTags(t, "aws-s3-bucket") },
			version:  "1.0.0",
			wantPass: true,
		},
		{
			name:    "api failure is surfaced",
			client:  func(t *testing.T) *massdriver.Client { return newMockClient(t, gqltest.RespondWithError("boom")) },
			version: "1.0.0",
			wantErr: "fetching OCI repo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkDuplicateVersion(t.Context(), tc.client(t), "aws-s3-bucket", tc.version)
			if tc.wantPass {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q missing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestCheckDuplicateVersionDevTagSkipsAPI pins that 0.0.0 is always
// republishable. The nil client is the assertion: if the carve-out ever stops
// short-circuiting, this panics rather than silently costing a round trip.
func TestCheckDuplicateVersionDevTagSkipsAPI(t *testing.T) {
	if err := checkDuplicateVersion(t.Context(), nil, "aws-s3-bucket", "0.0.0"); err != nil {
		t.Fatalf("0.0.0 should always be republishable, got: %v", err)
	}
}
