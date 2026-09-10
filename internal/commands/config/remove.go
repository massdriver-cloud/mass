package config

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/massdriver-cloud/mass/internal/configfile"
	sdkconfig "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
)

// RunRemove deletes a profile, prompting for confirmation unless force is set.
// The prompt requires the caller to retype the profile name to proceed —
// matches the safety pattern used for `mass resource delete`.
func RunRemove(file *configfile.File, name string, force bool, in io.Reader, out io.Writer) error {
	if _, found := file.Profile(name); !found {
		return unknownProfileError(file, name)
	}

	if !force {
		fmt.Fprintf(out, "WARNING: This will permanently remove profile `%s` from %s.\n", name, file.Path())
		fmt.Fprintf(out, "Type `%s` to confirm removal: ", name)
		reader := bufio.NewReader(in)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(answer) != name {
			fmt.Fprintln(out, "Removal cancelled.")
			return nil
		}
	}

	wasActive := file.CurrentProfile() == name
	file.RemoveProfile(name)
	if err := file.Save(); err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Profile `%s` removed\n", name)
	if wasActive {
		fmt.Fprintf(out, "   No active profile is set; commands will fall back to `%s`\n", sdkconfig.DefaultProfileName)
	}
	return nil
}
