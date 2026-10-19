package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"oras.land/oras-go/v2/registry/remote/errcode"
	"sigs.k8s.io/yaml"
)

// RuntimeAPI is the SDK surface needed to resolve and fetch a runtime.
type RuntimeAPI interface {
	// ResolveBundleVersion resolves "<name>@<constraint>" to a concrete version.
	ResolveBundleVersion(ctx context.Context, ref string) (string, error)
	// PullBundle writes the bundle's files into dir and returns its manifest digest.
	PullBundle(ctx context.Context, name, version, dir string) (string, error)
}

// NewRuntimeAPI returns the production [RuntimeAPI] backed by the SDK client.
func NewRuntimeAPI(c *massdriver.Client) RuntimeAPI { return runtimeAPI{c: c} }

type runtimeAPI struct{ c *massdriver.Client }

func (a runtimeAPI) ResolveBundleVersion(ctx context.Context, ref string) (string, error) {
	b, err := a.c.Bundles.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	return b.Version, nil
}

func (a runtimeAPI) PullBundle(ctx context.Context, name, version, dir string) (string, error) {
	desc, err := pullToDirectory(ctx, a.c, name, version, dir)
	if err != nil {
		return "", err
	}
	return desc.Digest.String(), nil
}

// ResolveSource returns the directory to build, lint, or publish from. A
// bundle with no app.runtime is used in place. Otherwise it is assembled in
// the application's local build directory when keep is set, or in a temporary
// directory that cleanup removes.
func ResolveSource(ctx context.Context, api RuntimeAPI, appDir string, keep bool, out io.Writer) (string, func(), error) {
	noop := func() {}
	application, err := bundle.ReadSpec(appDir)
	if err != nil {
		return "", noop, err
	}
	if _, found := bundle.AppRuntime(application); !found {
		return appDir, noop, nil
	}

	buildDir, cleanup, err := buildDirectory(appDir, keep)
	if err != nil {
		return "", noop, err
	}
	if _, err = AssembleApplication(ctx, api, appDir, buildDir, out); err != nil {
		cleanup()
		return "", noop, err
	}
	return buildDir, cleanup, nil
}

func buildDirectory(appDir string, keep bool) (string, func(), error) {
	if keep {
		dir, err := LocalBuildDir(appDir)
		return dir, func() {}, err
	}
	dir, err := os.MkdirTemp("", "mass-bundle-")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// LocalBuildDir is where `mass bundle build` assembles an application on its
// runtime, kept out of the application's step directories and out of git.
func LocalBuildDir(appDir string) (string, error) {
	root := filepath.Join(appDir, ".massdriver")
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*\n"), 0o600); err != nil {
		return "", err
	}
	return filepath.Join(root, "build"), nil
}

// AssembleApplication assembles an application at buildDir by pulling the
// runtime named in its `app.runtime` and merging the two massdriver.yaml
// documents. It reports false, and touches nothing, when the bundle sets no
// `app.runtime`.
func AssembleApplication(ctx context.Context, api RuntimeAPI, appDir, buildDir string, out io.Writer) (bool, error) {
	application, err := bundle.ReadSpec(appDir)
	if err != nil {
		return false, err
	}
	name, constraint, found, err := bundle.ParseAppRuntime(application)
	if err != nil || !found {
		return found, err
	}

	ref := name + "@" + constraint
	version, err := api.ResolveBundleVersion(ctx, ref)
	if err != nil {
		return true, fmt.Errorf("resolving runtime %s: %w", ref, err)
	}

	if err = os.RemoveAll(buildDir); err != nil {
		return true, err
	}
	if err = os.MkdirAll(buildDir, 0o750); err != nil {
		return true, err
	}

	digest, err := api.PullBundle(ctx, name, version, buildDir)
	if err != nil {
		return true, fmt.Errorf("pulling runtime %s@%s: %w", name, version, err)
	}
	fmt.Fprintf(out, "Using runtime %s@%s (%s, %s)\n", prettylogs.Underline(name), prettylogs.Underline(version), constraint, digest)

	runtime, err := bundle.ReadSpec(buildDir)
	if err != nil {
		return true, fmt.Errorf("reading runtime %s@%s: %w", name, version, err)
	}

	runtimeRef := bundle.RuntimeRef{
		Name:       name,
		Constraint: constraint,
		Version:    version,
		Digest:     digest,
	}
	if err = bundle.CheckRuntime(runtime, runtimeRef); err != nil {
		return true, err
	}

	merged, warnings, err := bundle.MergeApplication(runtime, application, runtimeRef)
	if err != nil {
		return true, err
	}
	for _, w := range warnings {
		fmt.Fprintln(out, prettylogs.Orange("Warning: "+w))
	}

	mergedBytes, err := yaml.Marshal(merged)
	if err != nil {
		return true, err
	}
	if err = os.WriteFile(filepath.Join(buildDir, "massdriver.yaml"), mergedBytes, 0o600); err != nil {
		return true, err
	}

	return true, copyDocFiles(appDir, buildDir)
}

func copyDocFiles(appDir, buildDir string) error {
	for _, name := range bundle.DocFiles {
		data, err := os.ReadFile(filepath.Join(appDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		// #nosec G703 -- name comes from the fixed DocFiles list, not user input
		if err = os.WriteFile(filepath.Join(buildDir, name), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func accessHint(err error, repoName string) error {
	var resp *errcode.ErrorResponse
	if errors.As(err, &resp) && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
		return fmt.Errorf("%w (pulling requires repo:pull on %s)", err, repoName)
	}
	return err
}
