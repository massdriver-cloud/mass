package config

import (
	"fmt"
	"io"
	"os"

	"github.com/massdriver-cloud/mass/internal/configfile"
)

// RunUse points the config file's current_profile at name. The profile must
// exist: the SDK treats current_profile as an explicit selection and fails
// every later command when it names nothing.
func RunUse(file *configfile.File, name string, out io.Writer) error {
	if _, found := file.Profile(name); !found {
		return unknownProfileError(file, name)
	}
	if err := file.SetCurrentProfile(name); err != nil {
		return err
	}
	if err := file.Save(); err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Active profile is now `%s`\n", name)
	if env := os.Getenv(configfile.EnvProfile); env != "" && env != name {
		fmt.Fprintf(out, "⚠️  %s is set to `%s` in this shell and takes precedence; unset it to use `%s`\n",
			configfile.EnvProfile, env, name)
	}
	return nil
}
