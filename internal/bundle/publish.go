package bundle

import (
	"fmt"
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

// ArtifactType is the OCI artifact-type media type for bundles.
const ArtifactType = "application/vnd.massdriver.bundle.v1+json"

// PackageKeep returns the keep predicate used when packaging a bundle. It honors
// a bundle's optional .mdignore file, falling back to a default allowlist that
// only lets the expected bundle files through.
func PackageKeep(bundleDir string) (func(relPath string) bool, error) {
	ignoreMatcher, ignoreErr := getIgnores(filepath.Join(bundleDir, ".mdignore"))
	if ignoreErr != nil {
		return nil, ignoreErr
	}
	return func(relPath string) bool {
		return ignoreMatcher == nil || !ignoreMatcher.MatchesPath(relPath)
	}, nil
}

// Loads patterns from .mdignore file and returns a matcher
func getIgnores(ignorePath string) (*ignore.GitIgnore, error) {
	defaultIgnores := []string{
		// Ignore all files in top level directory except for the following
		"/*",
		"!/massdriver.yaml",
		"!/icon.svg",
		"!/operator.md",
		"!/operator.mdx",
		"!/readme.md",
		"!/README.md",
		"!/CHANGELOG.md",

		// Do NOT ignore directories (preserve all dirs)
		"!/*/",

		// Ignore all hidden files/directories (e.g., .git, .github, .vscode)
		"/.*",
		"/*/.*",

		// Ignore certain terraform/opentofu files
		"**/.terraform",
		"**/*.tfstate*",
		"**/*.tfvars*",
		// Allow terraform lock files
		"!**/*.terraform.lock.hcl",

		// Allow checkov config file
		"!**/.checkov.yml",
		"!**/.checkov.yaml",
	}

	_, err := os.Stat(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ignore.CompileIgnoreLines(defaultIgnores...), nil
		}
		return nil, fmt.Errorf("error checking ignore file: %w", err)
	}

	gi, err := ignore.CompileIgnoreFile(ignorePath)
	if err != nil {
		return nil, fmt.Errorf("invalid ignore file: %w", err)
	}
	return gi, nil
}
