package bundle

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-git/go-git/v5"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
)

var bundleTypeFormat = regexp.MustCompile(`^[a-z0-9-]{2,}`)
var connectionNameFormat = regexp.MustCompile(`^[a-z]+[a-z0-9_]*[a-z0-9]+$`)

// TODO: @coryodaniel Can you add the query we are using for the frontend flow so I can swap this out?
var MassdriverArtifactDefinitions = []string{
	"massdriver/aws-api-gateway-rest-api",
	"massdriver/aws-ecs-cluster",
	"massdriver/aws-dynamodb-table",
	"massdriver/aws-dynamodb-stream",
	"massdriver/aws-efs-file-system",
	"massdriver/aws-eventbridge",
	"massdriver/aws-eventbridge-rule",
	"massdriver/aws-iam-role",
	"massdriver/aws-lambda-function",
	"massdriver/aws-s3-bucket",
	"massdriver/aws-sns-topic",
	"massdriver/aws-sqs-queue",
	"massdriver/aws-vpc",
	"massdriver/azure-cognitive-service-language",
	"massdriver/azure-communication-service",
	"massdriver/azure-data-lake-storage",
	"massdriver/azure-databricks-workspace",
	"massdriver/azure-fhir-service",
	"massdriver/azure-machine-learning-workspace",
	"massdriver/azure-service-principal",
	"massdriver/azure-storage-account",
	"massdriver/azure-virtual-network",
	"massdriver/cosmosdb-sql-authentication",
	"massdriver/elasticsearch-authentication",
	"massdriver/gcp-bucket-https",
	"massdriver/gcp-cloud-function",
	"massdriver/gcp-cloud-tasks-queue",
	"massdriver/gcp-firebase-authentication",
	"massdriver/gcp-gcs-bucket",
	"massdriver/gcp-global-network",
	"massdriver/gcp-pubsub-subscription",
	"massdriver/gcp-pubsub-topic",
	"massdriver/gcp-service-account",
	"massdriver/gcp-subnetwork",
	"massdriver/cosmosdb-sql-authentication",
	"massdriver/kafka-authentication",
	"massdriver/kubernetes-cluster",
	"massdriver/machine-learning-workspace",
	"massdriver/mongo-authentication",
	"massdriver/mysql-authentication",
	"massdriver/opensearch-authentication",
	"massdriver/postgresql-authentication",
	"massdriver/redis-authentication",
	"massdriver/sftp-authentication",
}

var promptsNew = []func(t *templatecache.TemplateData) error{
	getName,
	getDescription,
	getAccessLevel,
	getTemplate,
	GetConnections,
	getOutputDir,
	getSourceUrl,
}

func RunPromptNew(t *templatecache.TemplateData) error {
	var err error

	for _, prompt := range promptsNew {
		err = prompt(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func getName(t *templatecache.TemplateData) error {
	validate := func(input string) error {
		if !bundleTypeFormat.MatchString(input) {
			return errors.New("name must be 2 or more characters and can only include lowercase letters and dashes")
		}
		return nil
	}

	defaultValue := strings.ReplaceAll(strings.ToLower(t.Name), " ", "-")

	prompt := promptui.Prompt{
		Label:    "Name",
		Validate: validate,
		Default:  defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	t.Name = result
	return nil
}

func getAccessLevel(t *templatecache.TemplateData) error {
	if t.Access != "" {
		return nil
	}

	prompt := promptui.Select{
		Label: "Access Level",
		Items: []string{"public", "private"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		return err
	}

	t.Access = result
	return nil
}

func getDescription(t *templatecache.TemplateData) error {
	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("description cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Description",
		Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		return err
	}

	t.Description = result
	return nil
}

var ignoredTemplateDirs = map[string]bool{"alpha": true}

func getTemplate(t *templatecache.TemplateData) error {
	var fs = afero.NewOsFs()
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher, fs)
	templates, err := cache.ListTemplates()

	filteredTemplates := removeIgnoredTemplateDirectories(templates)

	if err != nil {
		return err
	}
	prompt := promptui.Select{
		Label: "Template",
		Items: filteredTemplates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		return err
	}

	t.TemplateName = result
	return nil
}

func GetConnections(t *templatecache.TemplateData) error {
	none := "(None)"

	var selectedDeps []string
	multiselect := &survey.MultiSelect{
		Message: "What connections do you need?\n  If you don't need any, just hit enter or select (None)\n",
		Options: append([]string{none}, MassdriverArtifactDefinitions...),
	}

	err := survey.AskOne(multiselect, &selectedDeps)

	if err != nil {
		return err
	}

	var depMap []templatecache.Connection

	for i, v := range selectedDeps {
		if v == none {
			if len(selectedDeps) > 1 {
				return fmt.Errorf("if selecting %v, you cannot select other dependecies. selected %#v", none, selectedDeps)
			}
			return nil
		}

		validate := func(input string) error {
			if !connectionNameFormat.MatchString(input) {
				return errors.New("name must be at least 2 characters, start with a-z, use lowercase letters, numbers and underscores. It can not end with an underscore")
			}
			return nil
		}

		fmt.Printf("Please enter a name for the connection: \"%v\"\nThis will be the variable name used to reference it in your app|bundle IaC\n", v)

		prompt := promptui.Prompt{
			Label:    `Name`,
			Validate: validate,
		}

		result, errName := prompt.Run()

		if errName != nil {
			return errName
		}

		depMap = append(depMap, templatecache.Connection{Name: result, ArtifactDefinition: selectedDeps[i]})
	}

	t.Connections = depMap
	return nil
}

func removeIgnoredTemplateDirectories(templates []templatecache.TemplateList) []string {
	filteredTemplates := []string{}
	for _, repo := range templates {
		for _, templateName := range repo.Templates {
			if ignoredTemplateDirs[templateName] {
				continue
			}
			filteredTemplates = append(filteredTemplates, templateName)
		}
	}

	return filteredTemplates
}

func getOutputDir(t *templatecache.TemplateData) error {
	prompt := promptui.Prompt{
		Label:   `Output directory`,
		Default: "massdriver",
	}

	result, err := prompt.Run()

	if err != nil {
		return err
	}

	t.OutputDir = result
	return nil
}

func getSourceUrl(t *templatecache.TemplateData) error {
	// No actual prompt here - just check if we are in a github repo so we can automatically
	// set the "source_url". Otherwise use a safe default. Ignore any errors since the user
	// may not be in a git repo.
	t.SourceURL = fmt.Sprintf("github.com/YOUR_ORGANIZATION/%s", t.Name)
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil
	}
	url := remote.Config().URLs[0]
	url = strings.Replace(url, "git@github.com:", "github.com/", 1)
	t.SourceURL = strings.TrimSuffix(url, ".git")

	return nil
}
