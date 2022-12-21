package commands_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/spf13/afero"
)

func TestGenerateNewBundle(t *testing.T) {
	var hostFs = afero.NewMemMapFs()

	err := buildMockFileSystem(hostFs)

	if err != nil {
		t.Fatal(err)
	}

	templateData := &commands.TemplateData{
		OutputDir:      ".",
		Type:           "infrastructure",
		TemplateName:   "terraform",
		TemplateSource: "massdriver-cloud/infrastructure-templates",
		Name:           "aws-dynamodb",
		Access:         "private",
		Description:    "whatever",
		Connections: map[string]string{
			"massdriver/aws-authentication": "auth",
		},
		CloudPrefix:     "aws",
		RepoName:        "massdriver-cloud/bundle-templates",
		RepoNameEncoded: "massdriver-cloud/bundle-templates",
	}

	got := commands.GenerateNewBundle(hostFs, templateData)

	dir, _ := afero.ReadDir(hostFs, "/")

	for _, d := range dir {

		fmt.Printf("%b", d.IsDir())
	}

	if got != nil {
		t.Errorf("wanted %s but received %s", "nil", got)
	}
}

func buildMockFileSystem(fs afero.Fs) error {
	templateDir := "/home/md-cloud/.massdriver"
	template := "/massdriver-cloud/infrastructure-templates/terraform"
	sourceDir := "/src"
	err := fs.Mkdir(fmt.Sprintf("%s%s", templateDir, template), 0755)
	if err != nil {
		return err
	}
	err = fs.Mkdir(fmt.Sprintf("%s%s%s", templateDir, template, sourceDir), 0755)
	if err != nil {
		return err
	}

	content, err := os.ReadFile("./../../testdata/massdriver.yaml")

	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, fmt.Sprintf("%s%s/massdriver.yaml", templateDir, template), content, 0755)

	if err != nil {
		return err
	}

	_, err = fs.Create(fmt.Sprintf("%s%s%s/main.tf", templateDir, template, sourceDir))

	if err != nil {
		return err
	}

	return nil
}
