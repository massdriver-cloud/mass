package bundle

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/pkg/templates"
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

var massdriverArtifactDefinitions map[string]map[string]any

var promptsNew = []func(t *templates.TemplateData) error{
	getName,
	getDescription,
	getTemplate,
	GetConnections,
	getOutputDir,
}

// SetMassdriverArtifactDefinitions sets the defs used to specify connections in a bundle
func SetMassdriverArtifactDefinitions(in map[string]map[string]any) {
	massdriverArtifactDefinitions = in
}

// RunPromptNew interactively prompts the user to fill in all fields of a new bundle template.
func RunPromptNew(t *templates.TemplateData) error {
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

func getName(t *templates.TemplateData) error {
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

func getDescription(t *templates.TemplateData) error {
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

func getTemplate(t *templates.TemplateData) error {
	templateList, err := templates.List()
	if err != nil {
		if errors.Is(err, templates.ErrNotConfigured) {
			fmt.Println()
			fmt.Println("💡 Did you know? You can set up bundle templates for faster development!")
			fmt.Println("   Set MASSDRIVER_TEMPLATES_PATH or templates_path in your config profile.")
			fmt.Println("   See: https://docs.massdriver.cloud/guides/bundle-templates")
			fmt.Println()
			fmt.Println("   Continuing without a template - we'll generate a basic massdriver.yaml for you.")
			fmt.Println()
			t.TemplateName = ""
			return nil
		}
		return err
	}

	filteredTemplates := removeIgnoredTemplateDirectories(templateList)

	if len(filteredTemplates) == 0 {
		fmt.Println("No templates found in templates path. Generating a basic massdriver.yaml.")
		t.TemplateName = ""
		return nil
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

func connNameValidate(name string) error {
	if len(name) < 2 || len(name) > 53 {
		return errors.New(connNameError)
	}
	if !connectionNameFormat.MatchString(name) {
		return errors.New(connNameError)
	}
	return nil
}

// GetConnections prompts the user to select and name the connections for the bundle.
func GetConnections(t *templates.TemplateData) error {
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

	var depMap []templates.Connection
	envs := map[string]string{}

	for _, currentDep := range selectedDeps {
		if currentDep == none {
			if len(selectedDeps) > 1 {
				return fmt.Errorf("if selecting %v, you cannot select other dependecies. selected %#v", none, selectedDeps)
			}
			return nil
		}

		fmt.Printf("Please enter a name for the connection: \"%v\"\nThis will be the variable name used to reference it in your app|bundle IaC\n", currentDep)

		prompt := promptui.Prompt{
			Label:    `Name`,
			Validate: connNameValidate,
		}

		result, errName := prompt.Run()
		if errName != nil {
			return errName
		}

		depMap = append(depMap, templates.Connection{Name: result, ArtifactDefinition: currentDep})

		maps.Copy(envs, GetConnectionEnvs(result, massdriverArtifactDefinitions[currentDep]))
	}

	t.Connections = depMap
	t.Envs = envs
	return nil
}

func removeIgnoredTemplateDirectories(templates []string) []string {
	filteredTemplates := []string{}
	for _, templateName := range templates {
		if ignoredTemplateDirs[templateName] {
			continue
		}
		filteredTemplates = append(filteredTemplates, templateName)
	}

	return filteredTemplates
}

func getOutputDir(t *templates.TemplateData) error {
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
			matches, err := filepath.Glob(filepath.Join(input, "*.tf"))
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
			if _, chartErr := os.Stat(filepath.Join(input, "Chart.yaml")); errors.Is(chartErr, os.ErrNotExist) {
				return errors.New("path does not contain 'Chart.yaml' file, and therefore isn't a valid Helm chart")
			}
			if _, valuesErr := os.Stat(filepath.Join(input, "values.yaml")); errors.Is(valuesErr, os.ErrNotExist) {
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

// GetConnectionEnvs extracts environment variable templates from an artifact definition for the given connection name.
func GetConnectionEnvs(connectionName string, artifactDefinition map[string]any) map[string]string {
	envs := map[string]string{}

	mdBlock, mdBlockExists := artifactDefinition["$md"]
	if mdBlockExists {
		mdBlockMap, mdBlockMapOk := mdBlock.(map[string]any)
		if !mdBlockMapOk {
			return envs
		}
		envsBlock, envsBlockExists := mdBlockMap["envTemplates"]
		if envsBlockExists {
			envsBlockMap, envsBlockMapOk := envsBlock.(map[string]any)
			if !envsBlockMapOk {
				return envs
			}

			for envName, value := range envsBlockMap {
				//nolint:errcheck // value type is string as enforced by the surrounding map range
				envValue := value.(string)
				envs[envName] = strings.ReplaceAll(envValue, "connection_name", connectionName)
			}
		}
	}

	return envs
}
