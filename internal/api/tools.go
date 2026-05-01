//go:build tools

package api

import (
	// Keeps genqlient CLI dependencies (alexflint/go-arg, alexflint/go-scalar,
	// agnivade/levenshtein, etc.) from being pruned by `go mod tidy`.
	_ "github.com/Khan/genqlient/generate"
)
