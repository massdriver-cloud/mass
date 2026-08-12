package cmd

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/cli"
	cmdresourcetype "github.com/massdriver-cloud/mass/internal/commands/resourcetype"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
)

//go:embed templates/type.get.md.tmpl
var typeTemplates embed.FS

// NewCmdType returns a cobra command for managing resource types.
func NewCmdType() *cobra.Command {
	typeCmd := &cobra.Command{
		Use:     "resource-type",
		Short:   "Resource type management",
		Long:    helpdocs.MustRender("type"),
		Aliases: []string{"rt", "type", "res-type", "definition", "artifact-definition", "artdef", "def"},
	}

	typeCreateCmd := &cobra.Command{
		Use:     "create <name>",
		Short:   "Create a new resource type OCI repository in your organization's catalog",
		Long:    helpdocs.MustRender("type/create"),
		Example: `mass resource-type create my-resource-type -a owner=data,service=database`,
		Args:    cobra.ExactArgs(1),
		RunE:    runTypeCreate,
	}
	typeCreateCmd.Flags().StringToStringP("attributes", "a", nil, "Custom attributes (e.g. -a owner=data,service=database)")

	typeGetCmd := &cobra.Command{
		Use:   "get [resource-type]",
		Short: "Get a resource type from Massdriver",
		Long:  helpdocs.MustRender("type/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypeGet,
	}
	typeGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")
	typeGetCmd.Flags().Bool("schema", false, "With -o json, output only the resolved JSON schema")

	typeListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List resource types",
		Long:    helpdocs.MustRender("type/list"),
		RunE:    runTypeList,
	}
	typeListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	typePublishCmd := &cobra.Command{
		Use:     "publish [path]",
		Aliases: []string{"push"},
		Short:   "Publish a resource type to Massdriver",
		Long:    helpdocs.MustRender("type/publish"),
		Args:    cobra.MaximumNArgs(1),
		RunE:    runTypePublish,
	}

	typePullCmd := &cobra.Command{
		Use:   "pull <resource-type>",
		Short: "Pull a resource type from Massdriver to a local directory",
		Long:  helpdocs.MustRender("type/pull"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypePull,
	}
	typePullCmd.Flags().StringP("directory", "d", "", "Directory to output the resource type. Defaults to the resource type name.")
	typePullCmd.Flags().BoolP("force", "f", false, "Force pull even if the directory already exists. This will overwrite existing files.")
	typePullCmd.Flags().StringP("version", "v", "latest", "Resource type version or release channel")

	typeDeleteCmd := &cobra.Command{
		Use:   "delete [resource-type]",
		Short: "Delete a resource type from Massdriver",
		Long:  helpdocs.MustRender("type/delete"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypeDelete,
	}
	typeDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	typeConvertCmd := &cobra.Command{
		Use:   "convert <schema-file>",
		Short: "Convert a raw JSON schema resource type into a massdriver.yaml",
		Long:  helpdocs.MustRender("type/convert"),
		Args:  cobra.ExactArgs(1),
		RunE:  runTypeConvert,
	}
	typeConvertCmd.Flags().StringP("output", "o", "", "Path to write the massdriver.yaml (default: alongside the input file)")
	typeConvertCmd.Flags().BoolP("force", "f", false, "Overwrite existing files")

	typeCmd.AddCommand(typeCreateCmd)
	typeCmd.AddCommand(typeGetCmd)
	typeCmd.AddCommand(typeListCmd)
	typeCmd.AddCommand(typePublishCmd)
	typeCmd.AddCommand(typePullCmd)
	typeCmd.AddCommand(typeDeleteCmd)
	typeCmd.AddCommand(typeConvertCmd)

	return typeCmd
}

func runTypeCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return createOciRepoCommon(ctx, mdClient, name, string(ocirepos.ArtifactTypeResourceType), attrs)
}

func runTypeGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	typeName := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	schemaOnly, err := cmd.Flags().GetBool("schema")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	if schemaOnly && outputFormat != "json" {
		return errors.New("--schema requires -o json")
	}

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	rt, err := resourcetype.Get(ctx, mdClient, typeName)
	if err != nil {
		return fmt.Errorf("error getting resource type: %w", err)
	}

	switch outputFormat {
	case "json":
		payload := any(rt)
		if schemaOnly {
			payload = rt.Schema
		}
		jsonBytes, marshalErr := json.MarshalIndent(payload, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resource type to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		if err = renderType(rt); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runTypePublish(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	name, version, publishErr := cmdresourcetype.RunPublish(ctx, mdClient, path)
	if publishErr != nil {
		return fmt.Errorf("error publishing resource type: %w", publishErr)
	}

	fmt.Printf("Resource type %s:%s published successfully!\n", prettylogs.Underline(name), version)
	return nil
}

func runTypePull(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	directory, _ := cmd.Flags().GetString("directory")
	if directory == "" {
		directory = name
	}
	force, _ := cmd.Flags().GetBool("force")
	version, _ := cmd.Flags().GetString("version")
	cmd.SilenceUsage = true

	// Warn before overwriting an existing resource type in the target directory.
	mdYamlPath := filepath.Join(directory, "massdriver.yaml")
	if _, statErr := os.Stat(mdYamlPath); statErr == nil && !force {
		fmt.Printf("Resource type already exists at %s. Continuing will overwrite its contents. Continue? (y/N): ", mdYamlPath)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Resource type pull aborted!")
			return nil
		}
	}

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	tag, digest, pullErr := cmdresourcetype.RunPull(ctx, mdClient, name, version, directory)
	if pullErr != nil {
		return fmt.Errorf("error pulling resource type: %w", pullErr)
	}

	fmt.Printf("Resource type %s:%s pulled successfully to %s (Digest: %s)\n",
		prettylogs.Underline(name),
		prettylogs.Underline(tag),
		prettylogs.Underline(directory),
		prettylogs.Underline(digest),
	)
	return nil
}

func runTypeList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	seq := mdClient.OciRepos.Iter(ctx, ocirepos.ListInput{
		ArtifactType: ocirepos.ArtifactTypeResourceType,
	})

	switch output {
	case "json":
		repos, collectErr := types.Collect(seq)
		if collectErr != nil {
			return fmt.Errorf("failed to list resource types: %w", collectErr)
		}
		jsonBytes, marshalErr := json.MarshalIndent(repos, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resource types to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "table":
		return cli.Paginate(seq, cli.PagerConfig[ocirepos.OciRepo]{
			Columns: []string{"Name", "Latest", "Created At"},
			Row: func(repo ocirepos.OciRepo) []string {
				return []string{repo.Name, repo.LatestTag, repo.CreatedAt.Format("2006-01-02 15:04:05")}
			},
		})
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}

	return nil
}

func runTypeDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	// Confirm the repository exists (and surface its canonical name) before prompting.
	repo, getErr := mdClient.OciRepos.Get(ctx, name)
	if getErr != nil {
		return fmt.Errorf("error getting resource type: %w", getErr)
	}

	// Fail before the confirmation prompt if the repo is immutable (has published
	// versions) — no point making the user type the name for a delete that can't
	// succeed. RunDelete re-checks to guard against a version being published
	// during the prompt.
	if len(repo.Tags) > 0 {
		return fmt.Errorf("resource type %s has published versions and is immutable; its repository cannot be deleted", repo.Name)
	}

	if !force {
		fmt.Printf("WARNING: This will permanently delete resource type `%s`.\n", repo.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", repo.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != repo.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deleted, deleteErr := cmdresourcetype.RunDelete(ctx, mdClient, name)
	if deleteErr != nil {
		return fmt.Errorf("error deleting resource type: %w", deleteErr)
	}

	fmt.Printf("Resource type %s deleted successfully!\n", prettylogs.Underline(deleted.Name))
	return nil
}

func runTypeConvert(cmd *cobra.Command, args []string) error {
	schemaPath := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	result, convertErr := cmdresourcetype.RunConvert(schemaPath, output, force)
	if convertErr != nil {
		return fmt.Errorf("error converting resource type: %w", convertErr)
	}

	fmt.Printf("Wrote %s\n", prettylogs.Underline(result.MassdriverYAML))
	for _, f := range result.ExtraFiles {
		fmt.Printf("Wrote %s\n", prettylogs.Underline(f))
	}
	fmt.Println(prettylogs.Orange("Remember to set a real `version` in the massdriver.yaml before publishing."))
	return nil
}

func renderType(restype *resourcetype.ResourceType) error {
	schemaJSON, err := json.MarshalIndent(restype.Schema, "", "  ")
	if err != nil {
		return err
	}

	tmplBytes, err := typeTemplates.ReadFile("templates/type.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("type").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ID         string
		Name       string
		SchemaJSON string
	}{
		ID:         restype.ID,
		Name:       restype.Name,
		SchemaJSON: string(schemaJSON),
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, data); renderErr != nil {
		return fmt.Errorf("failed to execute template: %w", renderErr)
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(buf.String())
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}
