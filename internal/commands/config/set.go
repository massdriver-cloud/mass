package config

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/internal/configfile"
)

// RunSet updates settings on an existing profile. A nil Verifier skips the
// credential check.
func RunSet(ctx context.Context, file *configfile.File, name string, edits ProfileEdits, verifier Verifier, out io.Writer) error {
	profile, found := file.Profile(name)
	if !found {
		return unknownProfileError(file, name)
	}
	if !edits.Any() {
		return errors.New("nothing to update: pass at least one of --org, --api-key, --url, --templates-path")
	}
	edits.Apply(&profile)

	if err := verify(ctx, verifier, profile, out); err != nil {
		return err
	}
	if err := file.SetProfile(name, profile); err != nil {
		return err
	}
	if err := file.Save(); err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Profile `%s` updated in %s\n", name, file.Path())
	return nil
}
