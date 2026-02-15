package bundle

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

var (
	// These look eerily similar, the difference being - vs _
	baseRegex        = "^[a-z]+[a-z0-9%s]*[a-z0-9]+$"
	bundleNameFormat = regexp.MustCompile(fmt.Sprintf(baseRegex, "-"))
	linkNameFormat   = regexp.MustCompile(fmt.Sprintf(baseRegex, "_"))

	baseNameError   = "name must be 2 to 53 characters, can only include lowercase letters, numbers and %s, must start with a letter and end with an alphanumeric character [abc%s123, my%scool%sthing]"
	bundleNameError = fmt.Sprintf(baseNameError, "dashes", "-", "-", "-")
	linkNameError   = fmt.Sprintf(baseNameError, "underscores", "_", "_", "_")
)

var massdriverArtifactDefinitions map[string]map[string]any

var promptsNew = []func(t *templatecache.TemplateData) error{
	getName,
	getDescription,
	getTemplate,
	getConnections,
	getArtifacts,
	getOutputDir,
}

// SetMassdriverArtifactDefinitions sets the defs used to specify connections in a bundle
func SetMassdriverArtifactDefinitions(in map[string]map[string]any) {
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
	cache, _ := templatecache.NewBundleTemplateCache(templatecache.GithubTemplatesFetcher)
	templates, err := cache.ListTemplates()

	filteredTemplates := removeIgnoredTemplateDirectories(templates)
	if err != nil {
		return err
	}

	prompt := promptui.Select{
		Label: "Template",
		Items: filteredTemplates,
	}

	_, templateName, err := prompt.Run()
	if err != nil {
		return err
	}

	t.TemplateName = templateName

	paramsPath, paramsErr := getExistingParamsPath(templateName)
	if paramsErr != nil {
		return paramsErr
	}
	t.ExistingParamsPath = paramsPath

	return nil
}

func linkNameValidate(name string) error {
	if len(name) < 2 || len(name) > 53 {
		return errors.New(linkNameError)
	}
	if !linkNameFormat.MatchString(name) {
		return errors.New(linkNameError)
	}
	return nil
}

func getConnections(t *templatecache.TemplateData) error {
	none := "(None)"

	artifactDefinitionsTypes := []string{}
	// in 1.23 we can use maps.Keys(), but until then we'll extract the keys manually
	for adt := range massdriverArtifactDefinitions {
		artifactDefinitionsTypes = append(artifactDefinitionsTypes, adt)
	}
	sort.StringSlice(artifactDefinitionsTypes).Sort()

	var selectedDeps []string
	multiselect := &survey.MultiSelect{
		Message: "What connections do you need?\n  If you don't need any, just hit enter or select (None)\n",
		Options: artifactDefinitionsTypes,
	}

	err := survey.AskOne(multiselect, &selectedDeps)
	if err != nil {
		return err
	}

	var depMap []templatecache.Connection
	envs := map[string]string{}

	for _, currentDep := range selectedDeps {
		if currentDep == none {
			if len(selectedDeps) > 1 {
				return fmt.Errorf("if selecting %v, you cannot select other connections. selected %#v", none, selectedDeps)
			}
			return nil
		}

		fmt.Printf("Please enter a name for the connection: \"%v\"\nThis will be the variable name used to reference it in your app|bundle IaC\n", currentDep)

		prompt := promptui.Prompt{
			Label:    `Name`,
			Validate: linkNameValidate,
		}

		result, errName := prompt.Run()
		if errName != nil {
			return errName
		}

		depMap = append(depMap, templatecache.Connection{Name: result, ArtifactDefinition: currentDep})

		maps.Copy(envs, GetConnectionEnvs(result, massdriverArtifactDefinitions[currentDep]))
	}

	t.Connections = depMap
	t.Envs = envs
	return nil
}

func getArtifacts(t *templatecache.TemplateData) error {
	none := "(None)"

	artifactDefinitionsTypes := []string{}
	// in 1.23 we can use maps.Keys(), but until then we'll extract the keys manually
	for adt := range massdriverArtifactDefinitions {
		artifactDefinitionsTypes = append(artifactDefinitionsTypes, adt)
	}
	sort.StringSlice(artifactDefinitionsTypes).Sort()

	var selectedArts []string
	multiselect := &survey.MultiSelect{
		Message: "What artifacts do you need?\n  If you don't need any, just hit enter or select (None)\n",
		Options: artifactDefinitionsTypes,
	}

	err := survey.AskOne(multiselect, &selectedArts)
	if err != nil {
		return err
	}

	var artMap []templatecache.Artifact

	for _, currentArt := range selectedArts {
		if currentArt == none {
			if len(selectedArts) > 1 {
				return fmt.Errorf("if selecting %v, you cannot select other artifacts. selected %#v", none, selectedArts)
			}
			return nil
		}

		fmt.Printf("Please enter a name for the artifact: \"%v\"\nThis will be the variable name used to reference it in your app|bundle IaC\n", currentArt)

		prompt := promptui.Prompt{
			Label:    `Name`,
			Validate: linkNameValidate,
		}

		result, errName := prompt.Run()
		if errName != nil {
			return errName
		}

		artMap = append(artMap, templatecache.Artifact{Name: result, ArtifactDefinition: currentArt})
	}

	t.Artifacts = artMap
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

//nolint:gocognit
func getExistingParamsPath(templateName string) (string, error) {
	prompt := promptui.Prompt{}

	switch templateName {
	case "terraform-module", "opentofu-module":
		prompt.Label = "Path to an existing Terraform/OpenTofu module to generate a bundle from, leave blank to skip"
		prompt.Validate = func(input string) error {
			if input == "" {
				return nil
			}
			pathInfo, statErr := os.Stat(input)
			if statErr != nil {
				return statErr
			}
			if !pathInfo.IsDir() {
				return errors.New("path must be a directory containing a Terraform/OpenTofu module")
			}
			matches, err := filepath.Glob(path.Join(input, "*.tf"))
			if err != nil {
				return errors.New("unable to read directory")
			}
			if len(matches) == 0 {
				return errors.New("path does not contain any '.tf' files, and therefore isn't a valid Terraform/OpenTofu module")
			}
			return nil
		}
	case "helm-chart":
		prompt.Label = "Path to an existing Helm chart to generate a bundle from, leave blank to skip"
		prompt.Validate = func(input string) error {
			if input == "" {
				return nil
			}
			pathInfo, statErr := os.Stat(input)
			if statErr != nil {
				return statErr
			}
			if !pathInfo.IsDir() {
				return errors.New("path must be a directory containing a helm chart")
			}
			if _, chartErr := os.Stat(path.Join(input, "Chart.yaml")); errors.Is(chartErr, os.ErrNotExist) {
				return errors.New("path does not contain 'Chart.yaml' file, and therefore isn't a valid Helm chart")
			}
			if _, valuesErr := os.Stat(path.Join(input, "values.yaml")); errors.Is(valuesErr, os.ErrNotExist) {
				return errors.New("path does not contain 'values.yaml' file, and therefore isn't a valid Helm chart")
			}
			return nil
		}
	case "bicep-template":
		prompt.Label = "Path to an existing Bicep template file to generate a bundle from, leave blank to skip"
		prompt.Validate = func(input string) error {
			if input == "" {
				return nil
			}
			pathInfo, statErr := os.Stat(input)
			if statErr != nil {
				return statErr
			}
			if pathInfo.IsDir() {
				return errors.New("path must be a file containing a Bicep template")
			}
			return nil
		}
	default:
		return "", nil
	}

	return prompt.Run()
}

func GetConnectionEnvs(connectionName string, artifactDefinition map[string]any) map[string]string {
	envs := map[string]string{}

	mdBlock, mdBlockExists := artifactDefinition["$md"]
	if mdBlockExists {
		envsBlock, envsBlockExists := mdBlock.(map[string]any)["envTemplates"]
		if envsBlockExists {
			for envName, value := range envsBlock.(map[string]any) {
				//nolint:errcheck
				envValue := value.(string)
				envs[envName] = strings.ReplaceAll(envValue, "connection_name", connectionName)
			}
		}
	}

	return envs
}
