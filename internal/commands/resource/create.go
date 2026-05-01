// Package resource provides command implementations for resource operations.
package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/manifoldco/promptui"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/jsonschema"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunCreate reads an resource from a file, validates it, and creates it in Massdriver.
func RunCreate(ctx context.Context, mdClient *client.Client, resourceName string, resourceType string, resourceFile string) (string, error) {
	bytes, readErr := os.ReadFile(resourceFile)
	if readErr != nil {
		return "", readErr
	}

	var payload map[string]any
	unmarshalErr := json.Unmarshal(bytes, &payload)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}

	validateErr := validateResource(ctx, mdClient, resourceType, payload)
	if validateErr != nil {
		return "", validateErr
	}

	input := api.CreateResourceInput{
		Name:    resourceName,
		Payload: payload,
	}

	fmt.Printf("Creating resource %s of type %s...\n", resourceName, resourceType)
	resp, createErr := api.CreateResource(ctx, mdClient, resourceType, input)
	if createErr != nil {
		return "", createErr
	}
	fmt.Printf("Resource %s created! (Resource ID: %s)\n", resp.Name, resp.ID)

	return resp.ID, nil
}

func validateResource(ctx context.Context, mdClient *client.Client, resourceTypeName string, resource map[string]any) error {
	resourceType, typeErr := api.GetResourceType(ctx, mdClient, resourceTypeName)
	if typeErr != nil {
		return typeErr
	}

	sch, schemaErr := jsonschema.LoadSchemaFromGo(resourceType.Schema)
	if schemaErr != nil {
		return fmt.Errorf("failed to compile resource definition schema: %w", schemaErr)
	}
	return jsonschema.ValidateGo(sch, resource)
}

// CreatePrompt holds the user-supplied data needed to create a resource.
type CreatePrompt struct {
	Name string `json:"name"`
	Type string `json:"type"`
	File string `json:"file"`
}

var promptsNew = []func(t *CreatePrompt) error{
	getName,
	getType,
	getFile,
}

var resourceTypeNames = []string{}

// RunCreatePrompt interactively prompts the user to fill in any missing resource import fields.
func RunCreatePrompt(ctx context.Context, mdClient *client.Client, t *CreatePrompt) error {
	var err error

	rts, err := api.ListResourceTypes(ctx, mdClient, nil)
	if err != nil {
		return err
	}

	resourceTypeNames = make([]string, len(rts))
	for idx, rt := range rts {
		resourceTypeNames[idx] = rt.Name
	}
	sort.Strings(resourceTypeNames)

	for _, prompt := range promptsNew {
		err = prompt(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func getName(t *CreatePrompt) error {
	var resourceNameFormat = regexp.MustCompile(`[a-z][a-z0-9-]*[a-z0-9]`)

	validate := func(input string) error {
		if !resourceNameFormat.MatchString(input) {
			return errors.New("name must be 2 or more characters and can only include lowercase letters, numbers and dashes")
		}
		return nil
	}

	if t.Name != "" {
		return validate(t.Name)
	}

	prompt := promptui.Prompt{
		Label:    "Resource name",
		Validate: validate,
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	t.Name = result
	return nil
}

func getType(t *CreatePrompt) error {
	if t.Type != "" {
		return nil
	}

	typeSelect := &survey.Select{
		Message: "What is the type of the resource\n",
		Options: resourceTypeNames,
	}

	var selectedType string
	err := survey.AskOne(typeSelect, &selectedType)
	if err != nil {
		return err
	}

	t.Type = selectedType
	return nil
}

func getFile(t *CreatePrompt) error {
	if t.File != "" {
		return nil
	}

	prompt := promptui.Prompt{
		Label:   `Resource file`,
		Default: "resource.json",
	}

	result, err := prompt.Run()
	if err != nil {
		return err
	}

	t.File = result
	return nil
}
