// Package config provides command implementations for managing the CLI
// configuration file's profiles.
package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/configfile"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// ProfileEdits are the profile fields a command was asked to change. A nil
// field was not passed and leaves the stored value alone, which is what lets
// `mass config set` change one setting without blanking the rest.
type ProfileEdits struct {
	OrganizationID *string
	APIKey         *string
	URL            *string
	TemplatesPath  *string
}

// Any reports whether any field was passed.
func (e ProfileEdits) Any() bool {
	return e.OrganizationID != nil || e.APIKey != nil || e.URL != nil || e.TemplatesPath != nil
}

// Apply writes the passed fields onto p.
func (e ProfileEdits) Apply(p *sdkconfig.Profile) {
	for _, field := range []struct {
		edit  *string
		value *string
	}{
		{e.OrganizationID, &p.OrganizationID},
		{e.APIKey, &p.APIKey},
		{e.URL, &p.URL},
		{e.TemplatesPath, &p.TemplatesPath},
	} {
		if field.edit != nil {
			*field.value = *field.edit
		}
	}
}

// Prompter collects a value the user did not pass as a flag.
type Prompter interface {
	Prompt(label string, secret bool) (string, error)
}

// NewPrompter returns a terminal Prompter, or nil when stdout is not
// interactive so scripted callers fail on the missing flag rather than block
// on a prompt nobody can answer.
func NewPrompter() Prompter {
	if !cli.IsInteractive(os.Stdout) {
		return nil
	}
	return terminalPrompter{}
}

type terminalPrompter struct{}

func (terminalPrompter) Prompt(label string, secret bool) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("%s cannot be empty", label)
			}
			return nil
		},
	}
	if secret {
		prompt.Mask = '*'
	}
	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// Verifier confirms a profile's credentials authenticate, returning the
// identity they belong to.
type Verifier interface {
	Verify(ctx context.Context, p sdkconfig.Profile) (string, error)
}

// NewVerifier returns a Verifier that authenticates against Massdriver.
func NewVerifier() Verifier { return apiVerifier{} }

type apiVerifier struct{}

func (apiVerifier) Verify(ctx context.Context, p sdkconfig.Profile) (string, error) {
	mdClient, err := massdriver.NewClient(
		massdriver.WithOrganizationID(p.OrganizationID),
		massdriver.WithAPIKey(p.APIKey),
		massdriver.WithBaseURL(p.URL),
	)
	if err != nil {
		return "", err
	}

	v, err := mdClient.Viewer.Get(ctx)
	if err != nil {
		return "", err
	}
	if v.Email != "" {
		return v.Email, nil
	}
	return v.Name, nil
}

// verify runs the credential check unless the caller opted out by passing a nil
// Verifier, pointing at --no-verify in the failure so the profile can still be
// saved when the API is unreachable.
func verify(ctx context.Context, verifier Verifier, p sdkconfig.Profile, out io.Writer) error {
	if verifier == nil {
		return nil
	}
	identity, err := verifier.Verify(ctx, p)
	if err != nil {
		return fmt.Errorf("could not verify credentials: %w (pass --no-verify to save anyway)", err)
	}
	fmt.Fprintf(out, "✅ Authenticated as %s\n", identity)
	return nil
}

// profileOutput is the rendered shape of a profile, with the API key masked
// unless the caller opted into seeing it.
type profileOutput struct {
	Name           string `json:"name"`
	Active         bool   `json:"active"`
	OrganizationID string `json:"organization_id"`
	APIKey         string `json:"api_key"`
	URL            string `json:"url"`
	TemplatesPath  string `json:"templates_path,omitempty"`
}

func newProfileOutput(p configfile.NamedProfile, active, showSecrets bool) profileOutput {
	return profileOutput{
		Name:           p.Name,
		Active:         active,
		OrganizationID: p.OrganizationID,
		APIKey:         secret(p.APIKey, showSecrets),
		URL:            p.URL,
		TemplatesPath:  p.TemplatesPath,
	}
}

// secret masks all but the last four characters of a credential, keeping
// enough to tell two keys apart without printing either.
func secret(value string, show bool) string {
	if value == "" || show {
		return value
	}
	if len(value) <= 8 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

// unknownProfileError reports a missing profile alongside the names that do
// exist, since the usual cause is a typo or the wrong config file. It wraps the
// SDK's sentinel so a caller classifying this matches the error the SDK raises
// for the same mistake.
func unknownProfileError(file *configfile.File, name string) error {
	names := file.ProfileNames()
	if len(names) == 0 {
		return fmt.Errorf("%w: %q, and %s defines no profiles — add one with `mass config add %s`",
			sdkconfig.ErrProfileNotFound, name, file.Path(), name)
	}
	return fmt.Errorf("%w: %q is not in %s, which defines %s",
		sdkconfig.ErrProfileNotFound, name, file.Path(), strings.Join(names, ", "))
}
