package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/configfile"
)

// ListOptions controls how RunList renders the profiles.
type ListOptions struct {
	Output      string
	ShowSecrets bool
	// ProfileOverride is the --profile flag, which decides the profile the
	// listing marks as active.
	ProfileOverride string
}

// RunList writes every configured profile, marking the active one.
func RunList(file *configfile.File, opts ListOptions, out io.Writer) error {
	profiles := file.Profiles()
	active := file.ActiveProfileName(opts.ProfileOverride)

	switch opts.Output {
	case "json":
		rendered := make([]profileOutput, 0, len(profiles))
		for _, p := range profiles {
			rendered = append(rendered, newProfileOutput(p, p.Name == active, opts.ShowSecrets))
		}
		//nolint:gosec // profileOutput.APIKey is masked unless ShowSecrets was set
		encoded, err := json.MarshalIndent(rendered, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal profiles to JSON: %w", err)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	case "table":
		if len(profiles) == 0 {
			fmt.Fprintf(out, "No profiles configured in %s\n", file.Path())
			fmt.Fprintln(out, "Add one with `mass config add default --org <organization> --api-key <key>`")
			return nil
		}
		fmt.Fprintf(out, "Profiles in %s (* = active)\n\n", file.Path())
		tbl := cli.NewTable("", "Name", "Organization", "API Key", "URL", "Templates Path").WithWriter(out)
		for _, p := range profiles {
			marker := ""
			if p.Name == active {
				marker = "*"
			}
			tbl.AddRow(marker, p.Name, p.OrganizationID, secret(p.APIKey, opts.ShowSecrets), p.URL, p.TemplatesPath)
		}
		tbl.Print()
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", opts.Output)
	}
}
