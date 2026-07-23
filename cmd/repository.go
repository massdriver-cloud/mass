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
	cmdrepository "github.com/massdriver-cloud/mass/internal/commands/repository"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
)

//go:embed templates/repository.get.md.tmpl
var repositoryTemplates embed.FS

// NewCmdRepository returns a cobra command for managing OCI repositories.
func NewCmdRepository() *cobra.Command {
	repositoryCmd := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repo"},
		Short:   "Manage OCI repositories (bundles and resource types)",
	}

	listInput := repositoryListInput{}
	repositoryListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List OCI repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runRepositoryList(&listInput, cmd.Flags().Changed("order"))
		},
	}
	repositoryListCmd.Flags().StringVarP(&listInput.name, "name", "n", "", "Filter by exact repository name")
	repositoryListCmd.Flags().StringVar(&listInput.prefix, "prefix", "", "Filter by repository name prefix")
	repositoryListCmd.Flags().StringVarP(&listInput.search, "search", "s", "", "Full-text search across name, readme, and changelog")
	repositoryListCmd.Flags().StringVarP(&listInput.kind, "type", "t", "", "Filter by artifact type (bundle, resource-type)")
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
	repositoryCreateCmd.Flags().StringP("type", "t", "", "Artifact type (bundle, resource-type)")
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

func runRepositoryList(input *repositoryListInput, orderChanged bool) error {
	ctx := context.Background()

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	listInput, buildErr := buildOciReposListInput(input, orderChanged)
	if buildErr != nil {
		return buildErr
	}

	seq := mdClient.OciRepos.Iter(ctx, listInput)

	switch input.output {
	case "json":
		// JSON output is consumed by scripts that expect a single document, so
		// buffer the full result set before marshaling.
		repos, collectErr := types.Collect(seq)
		if collectErr != nil {
			return fmt.Errorf("failed to list repositories: %w", collectErr)
		}
		jsonBytes, marshalErr := json.MarshalIndent(repos, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal repositories to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "table":
		// Interactive pager on a TTY, streamed table otherwise — pages fetched
		// on demand rather than buffering every repository.
		return cli.Paginate(seq, cli.PagerConfig[ocirepos.OciRepo]{
			Columns: []string{"Name", "Type", "Latest", "Created At"},
			Row: func(repo ocirepos.OciRepo) []string {
				return []string{repo.Name, cmdrepository.ArtifactTypeLabel(repo.ArtifactType), repo.LatestTag, repo.CreatedAt.Format("2006-01-02 15:04:05")}
			},
		})
	default:
		return fmt.Errorf("unsupported output format: %s", input.output)
	}

	return nil
}

func buildOciReposListInput(input *repositoryListInput, orderChanged bool) (ocirepos.ListInput, error) {
	out := ocirepos.ListInput{
		Search: input.search,
	}

	if input.kind != "" {
		artifactType, resolveErr := cmdrepository.ResolveArtifactType(input.kind)
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

	if input.sortField != "" || orderChanged {
		field, fieldErr := parseRepoSortField(input.sortField)
		if fieldErr != nil {
			return out, fieldErr
		}
		order, orderErr := parseRepoSortOrder(input.sortOrder)
		if orderErr != nil {
			return out, orderErr
		}
		out.SortBy = field
		out.SortOrder = order
	}

	return out, nil
}

func parseRepoSortField(s string) (ocirepos.SortField, error) {
	switch strings.ToLower(s) {
	case "", "name":
		return ocirepos.SortByName, nil
	case "created_at":
		return ocirepos.SortByCreatedAt, nil
	default:
		return "", fmt.Errorf("unknown sort field %q (valid: name, created_at)", s)
	}
}

func parseRepoSortOrder(s string) (ocirepos.SortOrder, error) {
	switch strings.ToLower(s) {
	case "", "asc":
		return ocirepos.SortAsc, nil
	case "desc":
		return ocirepos.SortDesc, nil
	default:
		return "", fmt.Errorf("unknown sort order %q (valid: asc, desc)", s)
	}
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
		ShownTags  []types.OciRepoTag
		TotalTags  int
		Truncated  bool
		FormatTime func(time.Time) string
	}{
		OciRepo:    repo,
		TypeLabel:  cmdrepository.ArtifactTypeLabel(repo.ArtifactType),
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
	artifactType, resolveErr := cmdrepository.ResolveArtifactType(typeFlag)
	if resolveErr != nil {
		return resolveErr
	}

	created, err := mdClient.OciRepos.Create(ctx, ocirepos.CreateInput{
		ID:           name,
		ArtifactType: artifactType,
		Attributes:   cli.AttributesToAnyMap(attrs),
	})
	if err != nil {
		return err
	}

	fmt.Printf("✅ Repository `%s` created (type: %s)\n", created.Name, cmdrepository.ArtifactTypeLabel(created.ArtifactType))
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
