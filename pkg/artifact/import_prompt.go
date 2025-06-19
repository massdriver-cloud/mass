package artifact

import (
	"context"
	"errors"
	"regexp"
	"sort"

	"github.com/massdriver-cloud/mass/pkg/api"

	"github.com/AlecAivazis/survey/v2"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

var artifactNameFormat = regexp.MustCompile(`[a-z][a-z0-9-]*[a-z0-9]`)
var artifactDefinitions = []string{}

type ImportedArtifact struct {
	Name string `json:"name"`
	Type string `json:"type"`
	File string `json:"file"`
}

var promptsNew = []func(t *ImportedArtifact) error{
	getName,
	getType,
	getFile,
}

func RunArtifactImportPrompt(ctx context.Context, mdClient *client.Client, t *ImportedArtifact) error {
	var err error

	ads, err := api.ListArtifactDefinitions(ctx, mdClient)
	if err != nil {
		return err
	}

	artifactDefinitions = make([]string, len(ads))
	for idx, ad := range ads {
		artifactDefinitions[idx] = ad.Name
	}
	sort.Strings(artifactDefinitions)

	for _, prompt := range promptsNew {
		err = prompt(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func getName(t *ImportedArtifact) error {
	validate := func(input string) error {
		if !artifactNameFormat.MatchString(input) {
			return errors.New("name must be 2 or more characters and can only include lowercase letters, numbers and dashes")
		}
		return nil
	}

	if t.Name != "" {
		return validate(t.Name)
	}

	prompt := promptui.Prompt{
		Label:    "Artifact name",
		Validate: validate,
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	t.Name = result
	return nil
}

func getType(t *ImportedArtifact) error {
	if t.Type != "" {
		return nil
	}

	typeSelect := &survey.Select{
		Message: "What is the type of the artifact\n",
		Options: artifactDefinitions,
	}

	var selectedType string
	err := survey.AskOne(typeSelect, &selectedType)
	if err != nil {
		return err
	}

	t.Type = selectedType
	return nil
}

func getFile(t *ImportedArtifact) error {
	if t.File != "" {
		return nil
	}

	prompt := promptui.Prompt{
		Label:   `Artifact file`,
		Default: "artifact.json",
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	t.File = result
	return nil
}
