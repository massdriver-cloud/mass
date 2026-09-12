package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/internal/configfile"
)

// GetOptions controls how RunGet renders the profile.
type GetOptions struct {
	Output      string
	ShowSecrets bool
	// ProfileOverride is the --profile flag, which decides the profile shown
	// when the caller names none.
	ProfileOverride string
}

// RunGet writes one profile's settings. An empty name shows the active profile.
func RunGet(file *configfile.File, name string, opts GetOptions, out io.Writer) error {
	active := file.ActiveProfileName(opts.ProfileOverride)
	if name == "" {
		name = active
	}

	profile, found := file.Profile(name)
	if !found {
		return unknownProfileError(file, name)
	}
	rendered := newProfileOutput(configfile.NamedProfile{Name: name, Profile: profile}, name == active, opts.ShowSecrets)

	switch opts.Output {
	case "json":
		//nolint:gosec // profileOutput.APIKey is masked unless ShowSecrets was set
		encoded, err := json.MarshalIndent(rendered, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal profile to JSON: %w", err)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	case "text":
		fmt.Fprintf(out, "Profile: %s", rendered.Name)
		if rendered.Active {
			fmt.Fprint(out, " (active)")
		}
		fmt.Fprintln(out)
		fmt.Fprintf(out, "   Organization:   %s\n", rendered.OrganizationID)
		fmt.Fprintf(out, "   API Key:        %s\n", rendered.APIKey)
		if rendered.URL != "" {
			fmt.Fprintf(out, "   URL:            %s\n", rendered.URL)
		}
		if rendered.TemplatesPath != "" {
			fmt.Fprintf(out, "   Templates Path: %s\n", rendered.TemplatesPath)
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", opts.Output)
	}
}
