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

var (
	// These look eerily similar, the difference being - vs _
	baseRegex            = "^[a-z]+[a-z0-9%s]*[a-z0-9]+$"
	bundleNameFormat     = regexp.MustCompile(fmt.Sprintf(baseRegex, "-"))
	connectionNameFormat = regexp.MustCompile(fmt.Sprintf(baseRegex, "_"))

	baseNameError   = "name must be 2 to 53 characters, can only include lowercase letters, numbers and %s, must start with a letter and end with an alphanumeric character [abc%s123, my%scool%sthing]"
	bundleNameError = fmt.Sprintf(baseNameError, "dashes", "-", "-", "-")
	connNameError   = fmt.Sprintf(baseNameError, "underscores", "_", "_", "_")
)

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

func bundleNameValidate(name string) error {
	if len(name) < 2 || len(name) > 53 {
		return errors.New(bundleNameError)
	}
	if !bundleNameFormat.MatchString(name) {
		return errors.New(bundleNameError)
	}
	return nil
}

func getName(t *templatecache.TemplateData) error {
	defaultValue := strings.ReplaceAll(strings.ToLower(t.Name), " ", "-")

	prompt := promptui.Prompt{
		Label:    "Name",
		Validate: bundleNameValidate,
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

func connNameValidate(name string) error {
	if len(name) < 2 || len(name) > 53 {
		return errors.New(connNameError)
	}
	if !connectionNameFormat.MatchString(name) {
		return errors.New(connNameError)
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

		fmt.Printf("Please enter a name for the connection: \"%v\"\nThis will be the variable name used to reference it in your app|bundle IaC\n", v)

		prompt := promptui.Prompt{
			Label:    `Name`,
			Validate: connNameValidate,
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
