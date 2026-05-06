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
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/repository.get.md.tmpl
var repositoryTemplates embed.FS

// artifactTypeAliases maps friendly names users might type into the canonical
// OCI media type the server stores. The reverse mapping (`mediaTypeLabels`)
// is used for display in `mass repository list`.
var artifactTypeAliases = map[string]string{
	"bundle": "application/vnd.massdriver.bundle.v1+json",
}

var mediaTypeLabels = map[string]string{
	"application/vnd.massdriver.bundle.v1+json": "bundle",
}

// createTypeEnums maps the friendly --type flag values into the
// OciArtifactType enum the createOciRepo mutation expects.
var createTypeEnums = map[string]api.OciArtifactType{
	"bundle": api.OciArtifactTypeBundle,
}

// NewCmdRepository returns a cobra command for managing OCI repositories.
func NewCmdRepository() *cobra.Command {
	repositoryCmd := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repo"},
		Short:   "Manage OCI repositories (bundles and, in future, resource types and provisioners)",
	}

	listInput := repositoryListInput{}
	repositoryListCmd := &cobra.Command{
		Use:   "list",
		Short: "List OCI repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runRepositoryList(&listInput)
		},
	}
	repositoryListCmd.Flags().StringVarP(&listInput.name, "name", "n", "", "Filter by exact repository name")
	repositoryListCmd.Flags().StringVar(&listInput.prefix, "prefix", "", "Filter by repository name prefix")
	repositoryListCmd.Flags().StringVarP(&listInput.search, "search", "s", "", "Full-text search across name, readme, and changelog")
	repositoryListCmd.Flags().StringVarP(&listInput.kind, "type", "t", "", "Filter by artifact type (bundle)")
	repositoryListCmd.Flags().StringVar(&listInput.sortField, "sort", "", "Sort field (name, created_at)")
	repositoryListCmd.Flags().StringVar(&listInput.sortOrder, "order", "asc", "Sort order (asc, desc)")
	repositoryListCmd.Flags().StringVarP(&listInput.output, "output", "o", "table", "Output format (table, json)")

	repositoryGetCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get an OCI repository by name",
		Args:  cobra.ExactArgs(1),
		RunE:  runRepositoryGet,
	}
	repositoryGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")
	repositoryGetCmd.Flags().IntP("tags", "n", 10, "Number of recent tags to display (newest first)")

	repositoryCreateCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new OCI repository",
		Args:  cobra.ExactArgs(1),
		RunE:  runRepositoryCreate,
	}
	repositoryCreateCmd.Flags().StringP("type", "t", "", "Artifact type (bundle)")
	repositoryCreateCmd.Flags().StringToStringP("attributes", "a", nil, "Custom attributes (e.g. -a owner=data,service=database)")
	_ = repositoryCreateCmd.MarkFlagRequired("type")

	repositoryUpdateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an OCI repository's attributes",
		Args:  cobra.ExactArgs(1),
		RunE:  runRepositoryUpdate,
	}
	repositoryUpdateCmd.Flags().StringToStringP("attributes", "a", nil, "Replacement custom attributes (e.g. -a owner=data,service=database)")

	repositoryDeleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an OCI repository",
		Args:  cobra.ExactArgs(1),
		RunE:  runRepositoryDelete,
	}
	repositoryDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	repositoryCmd.AddCommand(repositoryListCmd)
	repositoryCmd.AddCommand(repositoryGetCmd)
	repositoryCmd.AddCommand(repositoryCreateCmd)
	repositoryCmd.AddCommand(repositoryUpdateCmd)
	repositoryCmd.AddCommand(repositoryDeleteCmd)

	return repositoryCmd
}

type repositoryListInput struct {
	name      string
	prefix    string
	search    string
	kind      string
	sortField string
	sortOrder string
	output    string
}

func runRepositoryList(input *repositoryListInput) error {
	ctx := context.Background()

	mdClient, err := client.New()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	filter, filterErr := buildOciReposFilter(input)
	if filterErr != nil {
		return filterErr
	}

	sort := buildOciReposSort(input.sortField, input.sortOrder)

	repos, err := api.ListOciRepos(ctx, mdClient, filter, sort)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	switch input.output {
	case "json":
		jsonBytes, err := json.MarshalIndent(repos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal repositories to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "table":
		tbl := cli.NewTable("Name", "Type", "Latest", "Created At")
		for _, repo := range repos {
			latest := ""
			for _, rc := range repo.ReleaseChannels {
				if rc.Name == "latest" {
					latest = rc.Tag
					break
				}
			}
			tbl.AddRow(repo.Name, artifactTypeLabel(repo.ArtifactType), latest, repo.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		tbl.Print()
	default:
		return fmt.Errorf("unsupported output format: %s", input.output)
	}

	return nil
}

func buildOciReposFilter(input *repositoryListInput) (*api.OciReposFilter, error) {
	filter := &api.OciReposFilter{}
	hasFilter := false

	if input.kind != "" {
		mediaType, resolveErr := resolveArtifactType(input.kind)
		if resolveErr != nil {
			return nil, resolveErr
		}
		filter.ArtifactType = mediaType
		hasFilter = true
	}
	if input.search != "" {
		filter.Search = input.search
		hasFilter = true
	}
	switch {
	case input.name != "" && input.prefix != "":
		return nil, errors.New("--name and --prefix are mutually exclusive")
	case input.name != "":
		filter.Name = &api.OciRepoNameFilter{Eq: input.name}
		hasFilter = true
	case input.prefix != "":
		filter.Name = &api.OciRepoNameFilter{StartsWith: input.prefix}
		hasFilter = true
	}
	if !hasFilter {
		return nil, nil //nolint:nilnil // explicit nil filter is the no-filter signal to the API
	}
	return filter, nil
}

func buildOciReposSort(sortField, sortOrder string) *api.OciReposSort {
	if sortField == "" {
		return nil
	}
	field := api.OciReposSortFieldName
	if strings.EqualFold(sortField, "created_at") {
		field = api.OciReposSortFieldCreatedAt
	}
	order := api.SortOrderAsc
	if strings.EqualFold(sortOrder, "desc") {
		order = api.SortOrderDesc
	}
	return &api.OciReposSort{Field: field, Order: order}
}

func resolveArtifactType(s string) (string, error) {
	if mediaType, ok := artifactTypeAliases[strings.ToLower(s)]; ok {
		return mediaType, nil
	}
	if strings.Contains(s, "/") {
		// already a media type
		return s, nil
	}
	return "", fmt.Errorf("unknown artifact type %q (valid: bundle)", s)
}

func artifactTypeLabel(mediaType string) string {
	if label, ok := mediaTypeLabels[mediaType]; ok {
		return label
	}
	return mediaType
}

func runRepositoryGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	tagCount, err := cmd.Flags().GetInt("tags")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	repo, getErr := api.GetOciRepo(ctx, mdClient, name)
	if getErr != nil {
		return getErr
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(repo, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal repository to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		return renderRepository(repo, tagCount)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func renderRepository(repo *api.OciRepo, tagCount int) error {
	tmplBytes, err := repositoryTemplates.ReadFile("templates/repository.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("repository").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	tags := repo.Tags
	if tagCount > 0 && len(tags) > tagCount {
		tags = tags[:tagCount]
	}

	data := struct {
		*api.OciRepo
		TypeLabel  string
		ShownTags  []api.OciRepoTag
		TotalTags  int
		Truncated  bool
		FormatTime func(time.Time) string
	}{
		OciRepo:    repo,
		TypeLabel:  artifactTypeLabel(repo.ArtifactType),
		ShownTags:  tags,
		TotalTags:  len(repo.Tags),
		Truncated:  tagCount > 0 && len(repo.Tags) > tagCount,
		FormatTime: func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
	}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return fmt.Errorf("failed to execute template: %w", execErr)
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}
	out, renderErr := r.Render(buf.String())
	if renderErr != nil {
		return renderErr
	}
	fmt.Print(out)
	return nil
}

func runRepositoryCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	typeFlag, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	return createOciRepoCommon(ctx, mdClient, name, typeFlag, attrs)
}

// createOciRepoCommon is shared by `mass repository create` and
// `mass bundle create`. It resolves the friendly type name into the
// OciArtifactType enum, calls the API, and prints a success line.
func createOciRepoCommon(ctx context.Context, mdClient *client.Client, name, typeFlag string, attrs map[string]string) error {
	enumValue, ok := createTypeEnums[strings.ToLower(typeFlag)]
	if !ok {
		valid := make([]string, 0, len(createTypeEnums))
		for k := range createTypeEnums {
			valid = append(valid, k)
		}
		return fmt.Errorf("unknown artifact type %q (valid: %s)", typeFlag, strings.Join(valid, ", "))
	}

	created, createErr := api.CreateOciRepo(ctx, mdClient, api.CreateOciRepoInput{
		Id:           name,
		ArtifactType: enumValue,
		Attributes:   cli.AttributesToAnyMap(attrs),
	})
	if createErr != nil {
		return createErr
	}

	fmt.Printf("✅ Repository `%s` created (type: %s)\n", created.Name, artifactTypeLabel(created.ArtifactType))
	return nil
}

func runRepositoryUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	current, getErr := api.GetOciRepo(ctx, mdClient, name)
	if getErr != nil {
		return getErr
	}

	var attributes map[string]any
	if cmd.Flags().Changed("attributes") {
		attributes = cli.AttributesToAnyMap(attrs)
	} else {
		attributes = cli.StringMapToAnyMap(current.Attributes)
	}

	updated, updateErr := api.UpdateOciRepo(ctx, mdClient, name, api.UpdateOciRepoInput{
		Attributes: attributes,
	})
	if updateErr != nil {
		return updateErr
	}

	fmt.Printf("✅ Repository `%s` updated\n", updated.Name)
	return nil
}

func runRepositoryDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	if !force {
		fmt.Printf("WARNING: This will permanently delete repository `%s`.\n", name)
		fmt.Printf("Type `%s` to confirm deletion: ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deleted, deleteErr := api.DeleteOciRepo(ctx, mdClient, name)
	if deleteErr != nil {
		return deleteErr
	}

	fmt.Printf("Repository %s deleted successfully\n", deleted.Name)
	return nil
}
