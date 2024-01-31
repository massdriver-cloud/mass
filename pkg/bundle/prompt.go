package bundle

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
	"github.com/spf13/afero"
)

var bundleTypeFormat = regexp.MustCompile(`^[a-z0-9-]{2,}`)
var connectionNameFormat = regexp.MustCompile(`^[a-z]+[a-z0-9_]*[a-z0-9]+$`)

var massdriverArtifactDefinitions []string

var promptsNew = []func(t *templatecache.TemplateData) error{
	getName,
	getDescription,
	getTemplate,
	GetConnections,
	getOutputDir,
}

// SetMassdriverArtifactDefinitions sets the defs used to specify connections in a bundle
func SetMassdriverArtifactDefinitions(in []string) {
	massdriverArtifactDefinitions = in
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

	// "helm-chart" doesn't exist yet but seems like the right thing to call the template
	if result == "terraform-module" || result == "helm-chart" {
		paramPath, paramsErr := getExistingParamsPath(result)
		if paramsErr != nil {
			return paramsErr
		}
		t.ExistingParamsPath = paramPath
	}

	return nil
}

func GetConnections(t *templatecache.TemplateData) error {
	none := "(None)"

	var selectedDeps []string
	multiselect := &survey.MultiSelect{
		Message: "What connections do you need?\n  If you don't need any, just hit enter or select (None)\n",
		Options: append([]string{none}, massdriverArtifactDefinitions...),
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

func getExistingParamsPath(in string) (string, error) {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Path to an existing %s to generate params from, leave blank to skip", in),
	}

	return prompt.Run()
}
