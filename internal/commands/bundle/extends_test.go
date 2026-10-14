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

type fakeExtendsAPI struct {
	versions    map[string]string
	parentFiles map[string]string
	pulled      []string
}

func (f *fakeExtendsAPI) ResolveBundleVersion(_ context.Context, ref string) (string, error) {
	v, ok := f.versions[ref]
	if !ok {
		return "", errors.New("bundle not found")
	}
	return v, nil
}

func (f *fakeExtendsAPI) PullBundle(_ context.Context, name, version, dir string) (string, error) {
	f.pulled = append(f.pulled, name+"@"+version)
	for rel, content := range f.parentFiles {
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
	parentFiles := map[string]string{
		"massdriver.yaml":       "name: base\ndescription: The base bundle.\nversion: 1.2.3\nsteps:\n  - path: src\n    provisioner: opentofu\nparams:\n  properties: {}\nui: {}\n",
		"src/main.tf":           "# parent code\n",
		"src/artifact_queue.jq": ".outputs.queue\n",
		"README.md":             "parent readme\n",
		"icon.svg":              "<svg/>\n",
	}

	tests := []struct {
		name        string
		childFiles  map[string]string
		keep        bool
		versions    map[string]string
		wantInPlace bool
		wantErr     string
		check       func(t *testing.T, childDir, sourceDir string)
	}{
		{
			name:        "bundle without extends builds in place",
			childFiles:  map[string]string{"massdriver.yaml": "name: plain\nversion: 1.0.0\n"},
			wantInPlace: true,
			check: func(t *testing.T, childDir, _ string) {
				if _, err := os.Stat(filepath.Join(childDir, ".massdriver")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf(".massdriver created for a bundle without extends")
				}
			},
		},
		{
			name: "keep assembles into the git-ignored local build directory",
			childFiles: map[string]string{
				"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\n",
				"README.md":       "child readme\n",
				"src/main.tf":     "# child code that must not leak\n",
			},
			keep:     true,
			versions: map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, childDir, sourceDir string) {
				if sourceDir != filepath.Join(childDir, ".massdriver", "build") {
					t.Fatalf("sourceDir = %s, want the local build directory", sourceDir)
				}
				ignore, err := os.ReadFile(filepath.Join(childDir, ".massdriver", ".gitignore"))
				if err != nil || string(ignore) != "*\n" {
					t.Errorf(".gitignore = %q, %v", ignore, err)
				}
				assertFile(t, sourceDir, "src/main.tf", "# parent code\n")
				assertFile(t, sourceDir, "README.md", "child readme\n")
				assertFile(t, sourceDir, "icon.svg", "<svg/>\n")

				merged, err := bundle.ReadSpec(sourceDir)
				if err != nil {
					t.Fatal(err)
				}
				if merged["name"] != "myapp" || merged["description"] != "The base bundle." {
					t.Errorf("merged identity = %v / %v", merged["name"], merged["description"])
				}
				stamp, _ := merged["extends"].(map[string]any)
				if stamp["version"] != "1.2.3" || stamp["digest"] != "sha256:feed" || stamp["constraint"] != "~1.2" {
					t.Errorf("extends stamp = %v", stamp)
				}
			},
		},
		{
			name:       "temporary build directory is removed by cleanup",
			childFiles: map[string]string{"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\n"},
			versions:   map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, childDir, sourceDir string) {
				if strings.HasPrefix(sourceDir, childDir) {
					t.Errorf("sourceDir %s is inside the child directory", sourceDir)
				}
			},
		},
		{
			name: "step resource templates are written into the parent step",
			childFiles: map[string]string{
				"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: src\n    resources:\n      billing: '{url: .outputs.url}'\n",
			},
			versions: map[string]string{"base@~1.2": "1.2.3"},
			check: func(t *testing.T, _, sourceDir string) {
				assertFile(t, sourceDir, "src/resource_billing.jq", "{url: .outputs.url}")
				assertFile(t, sourceDir, "src/main.tf", "# parent code\n")
			},
		},
		{
			name: "step resource template collides with a parent template file",
			childFiles: map[string]string{
				"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\nresources:\n  queue:\n    resource_type: sqs@~1\nsteps:\n  - path: src\n    resources:\n      queue: .outputs\n",
			},
			versions: map[string]string{"base@~1.2": "1.2.3"},
			wantErr:  "parent step 'src' already emits resource 'queue' (artifact_queue.jq)",
		},
		{
			name:       "unresolvable parent",
			childFiles: map[string]string{"massdriver.yaml": "extends: base@~9\nname: myapp\nversion: 0.1.0\n"},
			wantErr:    "resolving parent bundle base@~9: bundle not found",
		},
		{
			name:       "invalid child",
			childFiles: map[string]string{"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\nparams:\n  properties: {}\n"},
			versions:   map[string]string{"base@~1.2": "1.2.3"},
			wantErr:    "remove: params",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			childDir := t.TempDir()
			writeFiles(t, childDir, tc.childFiles)
			api := &fakeExtendsAPI{versions: tc.versions, parentFiles: parentFiles}

			var out bytes.Buffer
			sourceDir, cleanup, err := cmdbundle.ResolveSource(context.Background(), api, childDir, tc.keep, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (sourceDir == childDir) != tc.wantInPlace {
				t.Fatalf("sourceDir = %s, in place = %v, want %v", sourceDir, sourceDir == childDir, tc.wantInPlace)
			}
			if tc.check != nil {
				tc.check(t, childDir, sourceDir)
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

func TestRunExtendsRebuildsFromScratch(t *testing.T) {
	childDir := t.TempDir()
	writeFiles(t, childDir, map[string]string{"massdriver.yaml": "extends: base@~1.2\nname: myapp\nversion: 0.1.0\n"})
	buildDir := filepath.Join(t.TempDir(), "build")
	writeFiles(t, buildDir, map[string]string{"stale.tf": "# from a previous parent\n"})

	api := &fakeExtendsAPI{
		versions:    map[string]string{"base@~1.2": "1.2.3"},
		parentFiles: map[string]string{"massdriver.yaml": "name: base\ndescription: The base bundle.\n"},
	}
	found, err := cmdbundle.RunExtends(context.Background(), api, childDir, buildDir, &bytes.Buffer{})
	if err != nil || !found {
		t.Fatalf("RunExtends = %v, %v", found, err)
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
