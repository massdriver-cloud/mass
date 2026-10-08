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

// ExtendsAPI is the SDK surface needed to resolve and fetch a parent bundle.
type ExtendsAPI interface {
	// ResolveBundleVersion resolves "<name>@<constraint>" to a concrete version.
	ResolveBundleVersion(ctx context.Context, ref string) (string, error)
	// PullBundle writes the bundle's files into dir and returns its manifest digest.
	PullBundle(ctx context.Context, name, version, dir string) (string, error)
}

// NewExtendsAPI returns the production [ExtendsAPI] backed by the SDK client.
func NewExtendsAPI(c *massdriver.Client) ExtendsAPI { return extendsAPI{c: c} }

type extendsAPI struct{ c *massdriver.Client }

func (a extendsAPI) ResolveBundleVersion(ctx context.Context, ref string) (string, error) {
	b, err := a.c.Bundles.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	return b.Version, nil
}

func (a extendsAPI) PullBundle(ctx context.Context, name, version, dir string) (string, error) {
	desc, err := pullToDirectory(ctx, a.c, name, version, dir)
	if err != nil {
		return "", err
	}
	return desc.Digest.String(), nil
}

// ResolveSource returns the directory to build, lint, or publish from. A
// bundle that extends nothing is used in place. Otherwise it is assembled in
// the child's local build directory when keep is set, or in a temporary
// directory that cleanup removes.
func ResolveSource(ctx context.Context, api ExtendsAPI, childDir string, keep bool, out io.Writer) (string, func(), error) {
	noop := func() {}
	child, err := bundle.ReadSpec(childDir)
	if err != nil {
		return "", noop, err
	}
	if _, found := child[bundle.ExtendsKey]; !found {
		return childDir, noop, nil
	}

	buildDir, cleanup, err := buildDirectory(childDir, keep)
	if err != nil {
		return "", noop, err
	}
	if _, err = RunExtends(ctx, api, childDir, buildDir, out); err != nil {
		cleanup()
		return "", noop, err
	}
	return buildDir, cleanup, nil
}

func buildDirectory(childDir string, keep bool) (string, func(), error) {
	if keep {
		dir, err := LocalBuildDir(childDir)
		return dir, func() {}, err
	}
	dir, err := os.MkdirTemp("", "mass-bundle-")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// LocalBuildDir is where `mass bundle build` assembles a bundle that extends
// another, kept out of the child's step directories and out of git.
func LocalBuildDir(childDir string) (string, error) {
	root := filepath.Join(childDir, ".massdriver")
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*\n"), 0o600); err != nil {
		return "", err
	}
	return filepath.Join(root, "build"), nil
}

// RunExtends assembles a child bundle at buildDir by pulling the parent named
// in the child's `extends` and merging the two massdriver.yaml documents. It
// reports false, and touches nothing, when the child extends nothing.
func RunExtends(ctx context.Context, api ExtendsAPI, childDir, buildDir string, out io.Writer) (bool, error) {
	child, err := bundle.ReadSpec(childDir)
	if err != nil {
		return false, err
	}
	name, constraint, found, err := bundle.ParseExtends(child)
	if err != nil || !found {
		return found, err
	}

	ref := name + "@" + constraint
	version, err := api.ResolveBundleVersion(ctx, ref)
	if err != nil {
		return true, fmt.Errorf("resolving parent bundle %s: %w", ref, err)
	}

	if err = os.RemoveAll(buildDir); err != nil {
		return true, err
	}
	if err = os.MkdirAll(buildDir, 0o750); err != nil {
		return true, err
	}

	digest, err := api.PullBundle(ctx, name, version, buildDir)
	if err != nil {
		return true, fmt.Errorf("pulling parent bundle %s@%s: %w", name, version, err)
	}
	fmt.Fprintf(out, "Extending %s@%s (%s, %s)\n", prettylogs.Underline(name), prettylogs.Underline(version), constraint, digest)

	parent, err := bundle.ReadSpec(buildDir)
	if err != nil {
		return true, fmt.Errorf("reading parent bundle %s@%s: %w", name, version, err)
	}

	merged, warnings, err := bundle.MergeExtends(parent, child, bundle.ParentRef{
		Name:       name,
		Constraint: constraint,
		Version:    version,
		Digest:     digest,
	})
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

	return true, copyDocFiles(childDir, buildDir)
}

func copyDocFiles(childDir, buildDir string) error {
	for _, name := range bundle.DocFiles {
		data, err := os.ReadFile(filepath.Join(childDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
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
