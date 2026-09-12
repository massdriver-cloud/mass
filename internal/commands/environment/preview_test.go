package environment_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/environments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

const sampleConfig = `
project: demo
baseEnvironment: production
copyEnvironmentDefaults: true

environmentDefaults:
  - resourceType: aws-iam-role
    resourceId: res-iam

instances:
  chatdb:
    version: "~2.0+dev"
    params:
      ingress:
        enabled: true
    secrets:
      - name: STRIPE_KEY
        value: FOO
    remoteReferences:
      - resourceId: a1b2c3d4-0000-0000-0000-000000000000
        field: kubernetes_cluster
      - resourceId: demo-production-sharedvpc.vpc
        field: vpc

  noOverrides:
`

// stubPreviewAPI is an in-test stub for environment.PreviewAPI. Each test
// sets the fields it cares about and inspects the captured input on the
// other side.
type stubPreviewAPI struct {
	forkInput  environments.ForkInput
	forkParent string
	forkErr    error

	setDefaultCalls []setDefaultCall

	copyInputs []copyInstanceCall

	updateInputs []updateInstanceCall

	setSecretCalls []setSecretCall

	setRemoteReferenceCalls []setRemoteReferenceCall
	setRemoteReferenceErr   error

	deployed       string
	deployErr      error
	deployCallsLen int
}

type setDefaultCall struct {
	envID    string
	resource string
}

type copyInstanceCall struct {
	source, destination string
	input               instances.CopyInput
}

type updateInstanceCall struct {
	instanceID string
	input      instances.UpdateInput
}

type setSecretCall struct {
	instanceID, name, value string
}

type setRemoteReferenceCall struct {
	instanceID, resourceID, field string
}

func (f *stubPreviewAPI) Fork(_ context.Context, parentID string, input environments.ForkInput) (*types.Environment, error) {
	f.forkParent = parentID
	f.forkInput = input
	if f.forkErr != nil {
		return nil, f.forkErr
	}
	return &types.Environment{ID: "demo-" + input.ID, Name: input.Name}, nil
}

func (f *stubPreviewAPI) SetEnvironmentDefault(_ context.Context, environmentID, resourceID string) error {
	f.setDefaultCalls = append(f.setDefaultCalls, setDefaultCall{envID: environmentID, resource: resourceID})
	return nil
}

func (f *stubPreviewAPI) CopyInstance(_ context.Context, sourceID, destinationID string, input instances.CopyInput) (*types.Instance, error) {
	f.copyInputs = append(f.copyInputs, copyInstanceCall{source: sourceID, destination: destinationID, input: input})
	return &types.Instance{ID: destinationID}, nil
}

func (f *stubPreviewAPI) UpdateInstance(_ context.Context, id string, input instances.UpdateInput) (*types.Instance, error) {
	f.updateInputs = append(f.updateInputs, updateInstanceCall{instanceID: id, input: input})
	return &types.Instance{ID: id}, nil
}

func (f *stubPreviewAPI) SetInstanceSecret(_ context.Context, instanceID, name, value string) error {
	f.setSecretCalls = append(f.setSecretCalls, setSecretCall{instanceID: instanceID, name: name, value: value})
	return nil
}

func (f *stubPreviewAPI) SetInstanceRemoteReference(_ context.Context, instanceID, resourceID, field string) error {
	f.setRemoteReferenceCalls = append(f.setRemoteReferenceCalls, setRemoteReferenceCall{instanceID: instanceID, resourceID: resourceID, field: field})
	return f.setRemoteReferenceErr
}

func (f *stubPreviewAPI) DeployEnvironment(_ context.Context, id string) (*types.Environment, error) {
	f.deployed = id
	f.deployCallsLen++
	if f.deployErr != nil {
		return nil, f.deployErr
	}
	return &types.Environment{ID: id}, nil
}

func TestLoadPreviewConfig_ParsesAllFields(t *testing.T) {
	path := writeConfig(t, sampleConfig)

	cfg, err := environment.LoadPreviewConfig(path)
	if err != nil {
		t.Fatalf("LoadPreviewConfig: %v", err)
	}

	if cfg.Project != "demo" {
		t.Errorf("project = %q, want demo", cfg.Project)
	}
	if cfg.BaseEnvironment != "production" {
		t.Errorf("baseEnvironment = %q, want production", cfg.BaseEnvironment)
	}
	if !cfg.CopyEnvironmentDefaults {
		t.Error("copyEnvironmentDefaults = false, want true")
	}
	if len(cfg.EnvironmentDefaults) != 1 || cfg.EnvironmentDefaults[0].ResourceID != "res-iam" {
		t.Errorf("environmentDefaults parsed wrong: %+v", cfg.EnvironmentDefaults)
	}
	chat, ok := cfg.Instances["chatdb"]
	if !ok {
		t.Fatal("missing chatdb instance")
	}
	if chat.Version != "~2.0+dev" {
		t.Errorf("chatdb version = %q, want ~2.0+dev", chat.Version)
	}
	if len(chat.Secrets) != 1 || chat.Secrets[0].Name != "STRIPE_KEY" {
		t.Errorf("chatdb secrets wrong: %+v", chat.Secrets)
	}
	wantRefs := []environment.PreviewRemoteReference{
		{ResourceID: "a1b2c3d4-0000-0000-0000-000000000000", Field: "kubernetes_cluster"},
		{ResourceID: "demo-production-sharedvpc.vpc", Field: "vpc"},
	}
	if !reflect.DeepEqual(chat.RemoteReferences, wantRefs) {
		t.Errorf("chatdb remoteReferences = %+v, want %+v", chat.RemoteReferences, wantRefs)
	}
}

func TestLoadPreviewConfig_RejectsUnknownKeys(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "top-level typo",
			body: "project: demo\nbaseEnviroment: production\n",
			want: "baseEnviroment",
		},
		{
			name: "instance-level typo",
			body: "project: demo\nbaseEnvironment: production\ninstances:\n  chatsvc:\n    remoteRefs:\n      - resourceId: x\n        field: y\n",
			want: "remoteRefs",
		},
		{
			name: "remote reference entry typo",
			body: "project: demo\nbaseEnvironment: production\ninstances:\n  chatsvc:\n    remoteReferences:\n      - resource: x\n        field: y\n",
			want: "resource",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := environment.LoadPreviewConfig(writeConfig(t, tt.body))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected unknown-key error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}

func TestLoadPreviewConfig_RejectsMissingProject(t *testing.T) {
	path := writeConfig(t, "baseEnvironment: production\n")

	cfg, err := environment.LoadPreviewConfig(path)
	if err != nil {
		t.Fatalf("LoadPreviewConfig: %v", err)
	}

	_, runErr := environment.RunPreview(t.Context(), &stubPreviewAPI{}, cfg, environment.PreviewOptions{ID: "pr1"})
	if runErr == nil || !strings.Contains(runErr.Error(), "project") {
		t.Errorf("expected project required error, got %v", runErr)
	}
}

func TestRunPreview_HappyPath(t *testing.T) {
	path := writeConfig(t, sampleConfig)
	cfg, err := environment.LoadPreviewConfig(path)
	if err != nil {
		t.Fatalf("LoadPreviewConfig: %v", err)
	}

	api := &stubPreviewAPI{}
	env, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{ID: "pr123"})
	if runErr != nil {
		t.Fatalf("RunPreview: %v", runErr)
	}
	if env.ID != "demo-pr123" {
		t.Errorf("env.ID = %q, want demo-pr123", env.ID)
	}

	if api.forkParent != "demo-production" {
		t.Errorf("forkParent = %q, want demo-production", api.forkParent)
	}
	if !api.forkInput.CopyEnvironmentDefaults {
		t.Error("forkInput.CopyEnvironmentDefaults = false, want true")
	}
	if len(api.setDefaultCalls) != 1 || api.setDefaultCalls[0].resource != "res-iam" {
		t.Errorf("setDefault calls wrong: %+v", api.setDefaultCalls)
	}
	if len(api.copyInputs) != 1 {
		t.Errorf("expected 1 copyInstance call (chatdb has params); got %d", len(api.copyInputs))
	}
	if len(api.updateInputs) != 1 || api.updateInputs[0].input.Version != "~2.0+dev" {
		t.Errorf("update calls wrong: %+v", api.updateInputs)
	}
	if len(api.setSecretCalls) != 1 || api.setSecretCalls[0].name != "STRIPE_KEY" {
		t.Errorf("setSecret calls wrong: %+v", api.setSecretCalls)
	}
	wantRefs := []setRemoteReferenceCall{
		{instanceID: "demo-pr123-chatdb", resourceID: "a1b2c3d4-0000-0000-0000-000000000000", field: "kubernetes_cluster"},
		{instanceID: "demo-pr123-chatdb", resourceID: "demo-production-sharedvpc.vpc", field: "vpc"},
	}
	if !reflect.DeepEqual(api.setRemoteReferenceCalls, wantRefs) {
		t.Errorf("setRemoteReference calls = %+v, want %+v", api.setRemoteReferenceCalls, wantRefs)
	}
	if api.deployed != "demo-pr123" {
		t.Errorf("deployed = %q, want demo-pr123", api.deployed)
	}
}

func TestRunPreview_RemoteReferenceValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing resourceId",
			body: "project: demo\nbaseEnvironment: production\ninstances:\n  chatsvc:\n    remoteReferences:\n      - field: vpc\n",
			want: "instances.chatsvc.remoteReferences[0]: `resourceId` is required",
		},
		{
			name: "missing field",
			body: "project: demo\nbaseEnvironment: production\ninstances:\n  chatsvc:\n    remoteReferences:\n      - resourceId: demo-production-sharedvpc.vpc\n",
			want: "instances.chatsvc.remoteReferences[0]: `field` is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := environment.LoadPreviewConfig(writeConfig(t, tt.body))
			if err != nil {
				t.Fatalf("LoadPreviewConfig: %v", err)
			}
			api := &stubPreviewAPI{}
			_, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{ID: "pr1"})
			if runErr == nil || !strings.Contains(runErr.Error(), tt.want) {
				t.Errorf("expected %q, got %v", tt.want, runErr)
			}
			if api.forkParent != "" {
				t.Error("fork should not run when validation fails")
			}
		})
	}
}

func TestRunPreview_PropagatesRemoteReferenceFailure(t *testing.T) {
	cfg, err := environment.LoadPreviewConfig(writeConfig(t, sampleConfig))
	if err != nil {
		t.Fatalf("LoadPreviewConfig: %v", err)
	}

	api := &stubPreviewAPI{setRemoteReferenceErr: errors.New("resource not found")}
	_, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{ID: "pr1"})
	if runErr == nil || !strings.Contains(runErr.Error(), "set remote reference kubernetes_cluster: resource not found") {
		t.Errorf("expected remote reference error, got %v", runErr)
	}
	if api.deployCallsLen != 0 {
		t.Error("deploy should not have been called after remote reference failure")
	}
}

func TestRunPreview_PropagatesForkFailure(t *testing.T) {
	path := writeConfig(t, sampleConfig)
	cfg, _ := environment.LoadPreviewConfig(path)

	api := &stubPreviewAPI{forkErr: errors.New("parent immutable")}
	_, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{ID: "pr1"})
	if runErr == nil || !strings.Contains(runErr.Error(), "parent immutable") {
		t.Errorf("expected fork error, got %v", runErr)
	}
	if api.deployCallsLen != 0 {
		t.Errorf("deploy should not have been called after fork failure")
	}
}

func TestLoadPreviewConfig_ExpandsEnvVars(t *testing.T) {
	t.Setenv("GITHUB_PR", "42")
	body := `
project: demo
baseEnvironment: production
attributes:
  pr: "${GITHUB_PR}"
instances:
  chatsvc:
    params:
      host: "chatty-pr-${GITHUB_PR}.example.com"
`
	cfg, err := environment.LoadPreviewConfig(writeConfig(t, body))
	if err != nil {
		t.Fatalf("LoadPreviewConfig: %v", err)
	}

	if cfg.Attributes["pr"] != "42" {
		t.Errorf("attributes.pr = %q, want 42", cfg.Attributes["pr"])
	}
	if cfg.Instances["chatsvc"].Params["host"] != "chatty-pr-42.example.com" {
		t.Errorf("host = %q, want chatty-pr-42.example.com", cfg.Instances["chatsvc"].Params["host"])
	}
}

func TestRunPreview_AttributesFlowIntoFork(t *testing.T) {
	t.Setenv("GITHUB_PR", "42")
	path := writeConfig(t, `
project: demo
baseEnvironment: production
attributes:
  data_classification: pii
  pr: "${GITHUB_PR}"
`)
	cfg, _ := environment.LoadPreviewConfig(path)

	api := &stubPreviewAPI{}
	if _, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{ID: "pr42"}); runErr != nil {
		t.Fatalf("RunPreview: %v", runErr)
	}
	if api.forkInput.Attributes["data_classification"] != "pii" {
		t.Errorf("attributes.data_classification = %v, want pii", api.forkInput.Attributes["data_classification"])
	}
	if api.forkInput.Attributes["pr"] != "42" {
		t.Errorf("attributes.pr = %v, want 42 (env-expanded)", api.forkInput.Attributes["pr"])
	}
}

func TestRunPreview_CLIAttributesOverrideConfigAttributes(t *testing.T) {
	path := writeConfig(t, `
project: demo
baseEnvironment: production
attributes:
  region: us-east-1
`)
	cfg, _ := environment.LoadPreviewConfig(path)

	api := &stubPreviewAPI{}
	if _, runErr := environment.RunPreview(t.Context(), api, cfg, environment.PreviewOptions{
		ID:         "pr1",
		Attributes: map[string]string{"region": "us-west-2"},
	}); runErr != nil {
		t.Fatalf("RunPreview: %v", runErr)
	}
	if api.forkInput.Attributes["region"] != "us-west-2" {
		t.Errorf("attributes.region = %v, want us-west-2 (CLI override)", api.forkInput.Attributes["region"])
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "preview.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}
	return path
}
