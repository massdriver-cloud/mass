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
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
	"github.com/spf13/cobra"
)

//go:embed templates/repository.get.md.tmpl
var repositoryTemplates embed.FS

// artifactTypeAliases maps the user-facing --type flag values to the SDK's
// typed artifact-type enum. Today only "bundle" is supported by the platform.
var artifactTypeAliases = map[string]ocirepos.ArtifactType{
	"bundle": ocirepos.ArtifactTypeBundle,
}

// artifactTypeLabels is the reverse lookup for table/output rendering — turn
// the enum the server returns back into the friendly name the user typed.
var artifactTypeLabels = map[ocirepos.ArtifactType]string{
	ocirepos.ArtifactTypeBundle: "bundle",
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	listInput, buildErr := buildOciReposListInput(input)
	if buildErr != nil {
		return buildErr
	}

	repos, err := mdClient.OciRepos.List(ctx, listInput)
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

func buildOciReposListInput(input *repositoryListInput) (ocirepos.ListInput, error) {
	out := ocirepos.ListInput{
		Search: input.search,
	}

	if input.kind != "" {
		artifactType, resolveErr := resolveArtifactType(input.kind)
		if resolveErr != nil {
			return out, resolveErr
		}
		out.ArtifactType = artifactType
	}

	switch {
	case input.name != "" && input.prefix != "":
		return out, errors.New("--name and --prefix are mutually exclusive")
	case input.name != "":
		out.NameEquals = input.name
	case input.prefix != "":
		out.NameStartsWith = input.prefix
	}

	if input.sortField != "" {
		field := ocirepos.SortByName
		if strings.EqualFold(input.sortField, "created_at") {
			field = ocirepos.SortByCreatedAt
		}
		order := ocirepos.SortAsc
		if strings.EqualFold(input.sortOrder, "desc") {
			order = ocirepos.SortDesc
		}
		out.SortBy = field
		out.SortOrder = order
	}

	return out, nil
}

func resolveArtifactType(s string) (ocirepos.ArtifactType, error) {
	if at, ok := artifactTypeAliases[strings.ToLower(s)]; ok {
		return at, nil
	}
	return "", fmt.Errorf("unknown artifact type %q (valid: bundle)", s)
}

func artifactTypeLabel(at string) string {
	if label, ok := artifactTypeLabels[ocirepos.ArtifactType(at)]; ok {
		return label
	}
	return at
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	repo, err := mdClient.OciRepos.Get(ctx, name)
	if err != nil {
		return err
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

func renderRepository(repo *ocirepos.OciRepo, tagCount int) error {
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
		*ocirepos.OciRepo
		TypeLabel  string
		ShownTags  []string
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return createOciRepoCommon(ctx, mdClient, name, typeFlag, attrs)
}

// createOciRepoCommon is shared by `mass repository create` and
// `mass bundle create`. It resolves the friendly type name into the
// ArtifactType enum, calls the SDK, and prints a success line.
func createOciRepoCommon(ctx context.Context, mdClient *massdriver.Client, name, typeFlag string, attrs map[string]string) error {
	artifactType, resolveErr := resolveArtifactType(typeFlag)
	if resolveErr != nil {
		valid := make([]string, 0, len(artifactTypeAliases))
		for k := range artifactTypeAliases {
			valid = append(valid, k)
		}
		return fmt.Errorf("unknown artifact type %q (valid: %s)", typeFlag, strings.Join(valid, ", "))
	}

	created, err := mdClient.OciRepos.Create(ctx, ocirepos.CreateInput{
		ID:           name,
		ArtifactType: artifactType,
		Attributes:   cli.AttributesToAnyMap(attrs),
	})
	if err != nil {
		return err
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	current, err := mdClient.OciRepos.Get(ctx, name)
	if err != nil {
		return err
	}

	var attributes map[string]any
	if cmd.Flags().Changed("attributes") {
		attributes = cli.AttributesToAnyMap(attrs)
	} else {
		attributes = current.Attributes
	}

	updated, err := mdClient.OciRepos.Update(ctx, name, ocirepos.UpdateInput{
		Attributes: attributes,
	})
	if err != nil {
		return err
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
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

	deleted, err := mdClient.OciRepos.Delete(ctx, name)
	if err != nil {
		return err
	}

	fmt.Printf("Repository %s deleted successfully\n", deleted.Name)
	return nil
}
