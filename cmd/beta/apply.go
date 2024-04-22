package beta

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/massdriver-cloud/mass/pkg/beta"
	"github.com/massdriver-cloud/mass/pkg/bundle"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func NewCmdApply() *cobra.Command {
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Execute a bundle deployment",
		Args:  cobra.ExactArgs(1),
		RunE:  runBundleApply,
	}

	applyCmd.Flags().StringP("params", "p", "", "Path to params.json file")
	applyCmd.Flags().StringP("connections", "c", "", "Path to connections.json file")
	applyCmd.Flags().StringP("values", "v", "", "Path to values.json file")
	applyCmd.MarkFlagsMutuallyExclusive("values", "params")
	applyCmd.MarkFlagsRequiredTogether("params", "connections")
	applyCmd.MarkFlagsOneRequired("values", "params")

	return applyCmd
}

func runBundleApply(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	name := args[0]

	paramsFile, _ := cmd.Flags().GetString("params")
	connectionsFile, _ := cmd.Flags().GetString("connections")
	valuesFile, _ := cmd.Flags().GetString("values")

	var params map[string]interface{}
	var connections map[string]interface{}

	if valuesFile != "" {
		values := map[string]interface{}{}

		valuesBytes, err := os.ReadFile(valuesFile)
		if err != nil {
			return errors.New("unable to open values file")
		}
		err = json.Unmarshal(valuesBytes, &values)
		if err != nil {
			return errors.New("unable to unmarshal values file (is it proper JSON?)")
		}

		params = values["params"].(map[string]interface{})
		connections = values["connections"].(map[string]interface{})
	} else {
		params = map[string]interface{}{}
		connections = map[string]interface{}{}

		paramsBytes, err := os.ReadFile(paramsFile)
		if err != nil {
			return errors.New("unable to open params file")
		}
		err = json.Unmarshal(paramsBytes, &params)
		if err != nil {
			return errors.New("unable to unmarshal params file (is it proper JSON?)")
		}

		connectionsBytes, err := os.ReadFile(connectionsFile)
		if err != nil {
			return errors.New("unable to open connections file")
		}
		err = json.Unmarshal(connectionsBytes, &connections)
		if err != nil {
			return errors.New("unable to unmarshal connections file (is it proper JSON?)")
		}
	}

	return beta.Apply(ctx, cli, name, params, connections)
}

func applyOverrides(bundle *bundle.Bundle, cmd *cobra.Command) {
	access, err := cmd.Flags().GetString("access")
	if err == nil {
		bundle.Access = access
	}
}
