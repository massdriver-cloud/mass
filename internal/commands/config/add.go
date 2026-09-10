package config

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// AddOptions carries everything `mass config add` needs beyond the profile name.
type AddOptions struct {
	Edits ProfileEdits
	Force bool
	Use   bool
	// Prompter fills in required values the caller omitted; nil disables
	// prompting and turns a missing value into an error.
	Prompter Prompter
	// Verifier checks the credentials before they are written; nil skips the
	// check.
	Verifier Verifier
}

// RunAdd writes a new profile to the config file.
func RunAdd(ctx context.Context, file *configfile.File, name string, opts AddOptions, out io.Writer) error {
	if _, exists := file.Profile(name); exists && !opts.Force {
		return fmt.Errorf("profile %q already exists; use `mass config set %s` to change it, or --force to overwrite", name, name)
	}

	var profile sdkconfig.Profile
	opts.Edits.Apply(&profile)

	if err := promptRequired(&profile, opts.Prompter); err != nil {
		return err
	}
	if profile.OrganizationID == "" {
		return errors.New("organization is required: pass --org")
	}
	if profile.APIKey == "" {
		return errors.New("API key is required: pass --api-key")
	}

	if err := verify(ctx, opts.Verifier, profile, out); err != nil {
		return err
	}
	if err := file.SetProfile(name, profile); err != nil {
		return err
	}

	// The first profile added is the one every command would otherwise fail
	// looking for, so adopt it as active without being asked.
	use := opts.Use || len(file.Profiles()) == 1
	if use {
		if err := file.SetCurrentProfile(name); err != nil {
			return err
		}
	}
	if err := file.Save(); err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Profile `%s` saved to %s\n", name, file.Path())
	if use {
		fmt.Fprintf(out, "   Active profile is now `%s`\n", name)
	}
	return nil
}

func promptRequired(profile *sdkconfig.Profile, prompter Prompter) error {
	if prompter == nil {
		return nil
	}
	if profile.OrganizationID == "" {
		value, err := prompter.Prompt("Organization abbreviation", false)
		if err != nil {
			return err
		}
		profile.OrganizationID = value
	}
	if profile.APIKey == "" {
		value, err := prompter.Prompt("API key or personal access token", true)
		if err != nil {
			return err
		}
		profile.APIKey = value
	}
	return nil
}
