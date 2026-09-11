package grants_test

import (
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/grants"
)

func TestResolveActionDefaults(t *testing.T) {
	repoAction, err := grants.ResolveRepoAction("")
	if err != nil {
		t.Fatalf("ResolveRepoAction returned error: %v", err)
	}
	if repoAction != grants.ActionRepoPull {
		t.Errorf("ResolveRepoAction(\"\") = %q, want %q", repoAction, grants.ActionRepoPull)
	}

	resourceAction, err := grants.ResolveResourceAction("")
	if err != nil {
		t.Fatalf("ResolveResourceAction returned error: %v", err)
	}
	if resourceAction != grants.ActionResourceExport {
		t.Errorf("ResolveResourceAction(\"\") = %q, want %q", resourceAction, grants.ActionResourceExport)
	}
}

func TestResolveActionNormalizes(t *testing.T) {
	for _, input := range []string{"repo:pull", "REPO:PULL", "  Repo:Pull  "} {
		t.Run(input, func(t *testing.T) {
			got, err := grants.ResolveRepoAction(input)
			if err != nil {
				t.Fatalf("ResolveRepoAction(%q) returned error: %v", input, err)
			}
			if got != grants.ActionRepoPull {
				t.Errorf("ResolveRepoAction(%q) = %q, want %q", input, got, grants.ActionRepoPull)
			}
		})
	}
}

func TestResolveActionRejectsUngrantable(t *testing.T) {
	cases := []struct {
		name    string
		resolve func(string) (string, error)
		action  string
	}{
		{"repo view is read-only", grants.ResolveRepoAction, "repo:view"},
		{"repo push is not a sharing concern", grants.ResolveRepoAction, "repo:push"},
		{"resource action on a repo", grants.ResolveRepoAction, grants.ActionResourceExport},
		{"resource view is read-only", grants.ResolveResourceAction, "resource:view"},
		{"repo action on a resource", grants.ResolveResourceAction, grants.ActionRepoPull},
		{"nonsense", grants.ResolveRepoAction, "repo:destroy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.resolve(tc.action)
			if err == nil {
				t.Fatalf("resolving %q succeeded, wanted error", tc.action)
			}
			if !strings.Contains(err.Error(), "unknown action") {
				t.Errorf("error = %q, want it to contain %q", err, "unknown action")
			}
		})
	}
}
