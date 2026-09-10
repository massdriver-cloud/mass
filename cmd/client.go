package cmd

import (
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/spf13/cobra"
)

// newMassdriverClient builds a client for the profile --profile names. An empty
// flag is not an override, leaving the SDK's own precedence untouched.
func newMassdriverClient(cmd *cobra.Command) (*massdriver.Client, error) {
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		return nil, err
	}
	return massdriver.NewClient(massdriver.WithProfile(profile))
}
