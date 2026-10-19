package bundle_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	cmdbundle "github.com/massdriver-cloud/mass/internal/commands/bundle"
)

type fakeRuntimeAPI struct {
	versions     map[string]string
	runtimeFiles map[string]string
	pulled       []string
}

func (f *fakeRuntimeAPI) ResolveBundleVersion(_ context.Context, ref string) (string, error) {
	v, ok := f.versions[ref]
	if !ok {
		return "", errors.New("bundle not found")
	}
	return v, nil
}

func (f *fakeRuntimeAPI) PullBundle(_ context.Context, name, version, dir string) (string, error) {
	f.pulled = append(f.pulled, name+"@"+version)
	for rel, content := range f.runtimeFiles {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return "", err
		}
	}
	return "sha256:feed", nil
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveSource(t *testing.T) {
	runtimeFiles := map[string]string{
		"massdriver.yaml": "name: base\ndescription: The base bundle.\nversion: 1.2.3\nprovides: [runtime]\napp:\n  envs: {}\nsteps:\n  - path: src\n    provisioner: opentofu\nparams:\n  properties: {}\nui: {}\n",
		"src/main.tf":     "# runtime code\n",
		"README.md":       "runtime readme\n",
		"icon.svg":        "<svg/>\n",
	}

	tests := []struct {
		name         string
		appFiles     map[string]string
		runtimeFiles map[string]string
		keep         bool
		versions     map[string]string
		wantInPlace  bool
		wantErr      string
		check        func(t *testing.T, appDir, sourceDir string)
	}{
		{
			name:        "bundle without app.runtime builds in place",
			appFiles:    map[string]string{"massdriver.yaml": "name: plain\nversion: 1.0.0\n"},
			wantInPlace: true,
			check: func(t *testing.T, appDir, _ string) {
				if _, err := os.Stat(filepath.Join(appDir, ".massdriver")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf(".massdriver created for a bundle without app.runtime")
				}
			},
		},
		{
			name: "keep assembles into the git-ignored local build directory",
			appFiles: map[string]string{
				"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\n",
				"README.md":       "application readme\n",
				"src/main.tf":     "# application code that must not leak\n",
			},
			keep:     true,
			versions: map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, appDir, sourceDir string) {
				if sourceDir != filepath.Join(appDir, ".massdriver", "build") {
					t.Fatalf("sourceDir = %s, want the local build directory", sourceDir)
				}
				ignore, err := os.ReadFile(filepath.Join(appDir, ".massdriver", ".gitignore"))
				if err != nil || string(ignore) != "*\n" {
					t.Errorf(".gitignore = %q, %v", ignore, err)
				}
				assertFile(t, sourceDir, "src/main.tf", "# runtime code\n")
				assertFile(t, sourceDir, "README.md", "application readme\n")
				assertFile(t, sourceDir, "icon.svg", "<svg/>\n")

				merged, err := bundle.ReadSpec(sourceDir)
				if err != nil {
					t.Fatal(err)
				}
				if merged["name"] != "myapp" || merged["description"] != "The base bundle." {
					t.Errorf("merged identity = %v / %v", merged["name"], merged["description"])
				}
				app, _ := merged["app"].(map[string]any)
				if app["runtime"] != "base@1.2.3" {
					t.Errorf("app.runtime = %v, want base@1.2.3", app["runtime"])
				}
			},
		},
		{
			name:     "temporary build directory is removed by cleanup",
			appFiles: map[string]string{"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\n"},
			versions: map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, appDir, sourceDir string) {
				if strings.HasPrefix(sourceDir, appDir) {
					t.Errorf("sourceDir %s is inside the application directory", sourceDir)
				}
			},
		},
		{
			name: "step resource templates stay in massdriver.yaml",
			appFiles: map[string]string{
				"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: src\n    resources:\n      billing:\n        template: '{url: .outputs.url}'\n",
			},
			versions: map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, _, sourceDir string) {
				merged, err := bundle.ReadSpec(sourceDir)
				if err != nil {
					t.Fatal(err)
				}
				steps, _ := merged["steps"].([]any)
				step, _ := steps[0].(map[string]any)
				resources, _ := step["resources"].(map[string]any)
				billing, _ := resources["billing"].(map[string]any)
				if billing["template"] != "{url: .outputs.url}" {
					t.Errorf("steps[0].resources = %v", step["resources"])
				}
				matches, _ := filepath.Glob(filepath.Join(sourceDir, "src", "*.jq"))
				if len(matches) != 0 {
					t.Errorf("jq files written to the build: %v", matches)
				}
			},
		},
		{
			name:     "unresolvable runtime",
			appFiles: map[string]string{"massdriver.yaml": "app:\n  runtime: base@~9\nname: myapp\nversion: 0.1.0\n"},
			wantErr:  "resolving runtime base@~9: bundle not found",
		},
		{
			name:     "invalid application",
			appFiles: map[string]string{"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\nparams:\n  properties: {}\n"},
			versions: map[string]string{"base@~1.2": "1.2.3"},
			wantErr:  "remove: params",
		},
		{
			name:         "bundle is not a runtime",
			appFiles:     map[string]string{"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\n"},
			runtimeFiles: map[string]string{"massdriver.yaml": "name: base\nversion: 1.2.3\napp:\n  envs: {}\n"},
			versions:     map[string]string{"base@~1.2": "1.2.3"},
			wantErr:      "base@1.2.3 does not provide a runtime",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			writeFiles(t, appDir, tc.appFiles)
			runtime := runtimeFiles
			if tc.runtimeFiles != nil {
				runtime = tc.runtimeFiles
			}
			api := &fakeRuntimeAPI{versions: tc.versions, runtimeFiles: runtime}

			var out bytes.Buffer
			sourceDir, cleanup, err := cmdbundle.ResolveSource(context.Background(), api, appDir, tc.keep, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (sourceDir == appDir) != tc.wantInPlace {
				t.Fatalf("sourceDir = %s, in place = %v, want %v", sourceDir, sourceDir == appDir, tc.wantInPlace)
			}
			if tc.check != nil {
				tc.check(t, appDir, sourceDir)
			}

			cleanup()
			if !tc.keep && !tc.wantInPlace {
				if _, statErr := os.Stat(sourceDir); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("temporary build directory %s survived cleanup", sourceDir)
				}
			}
		})
	}
}

func TestAssembleApplicationRebuildsFromScratch(t *testing.T) {
	appDir := t.TempDir()
	writeFiles(t, appDir, map[string]string{"massdriver.yaml": "app:\n  runtime: base@~1.2\nname: myapp\nversion: 0.1.0\n"})
	buildDir := filepath.Join(t.TempDir(), "build")
	writeFiles(t, buildDir, map[string]string{"stale.tf": "# from a previous runtime\n"})

	api := &fakeRuntimeAPI{
		versions:     map[string]string{"base@~1.2": "1.2.3"},
		runtimeFiles: map[string]string{"massdriver.yaml": "name: base\ndescription: The base bundle.\nprovides: [runtime]\napp:\n  envs: {}\n"},
	}
	found, err := cmdbundle.AssembleApplication(context.Background(), api, appDir, buildDir, &bytes.Buffer{})
	if err != nil || !found {
		t.Fatalf("AssembleApplication = %v, %v", found, err)
	}
	if _, statErr := os.Stat(filepath.Join(buildDir, "stale.tf")); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("stale file from a previous build survived")
	}
	if strings.Join(api.pulled, ",") != "base@1.2.3" {
		t.Errorf("pulled = %v, want the resolved version", api.pulled)
	}
}

func assertFile(t *testing.T, dir, rel, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", rel, got, want)
	}
}
