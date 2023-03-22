package commands_test

import (
	"fmt"
	"path"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/mockfilesystem"
	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func TestCopyFilesFromTemplateToCurrentDirectory(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	writePath := "."

	var fs = afero.NewMemMapFs()

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir, fs)
	checkErr(err, t)

	bundleCache := &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        func(filePath string) error { return nil },
		Fs:           fs,
	}

	templateData := mockTemplateData(writePath)

	err = commands.GenerateNewBundle(bundleCache, templateData)

	checkErr(err, t)

	wantTopLevel := []string{
		"home",
		"massdriver.yaml",
		"src",
	}

	if errorString, assertion := mockfilesystem.AssertDirectoryContents(fs, writePath, wantTopLevel); assertion != true {
		t.Errorf(errorString)
	}

	wantSecondLevel := []string{"main.tf"}

	if errorString, assertion := mockfilesystem.AssertDirectoryContents(fs, path.Join(writePath, "src"), wantSecondLevel); assertion != true {
		t.Errorf(errorString)
	}
}

func TestCopyFilesFromTemplateToNonExistentDirectory(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	writePath := "./bundles/aws-sqs-queue"

	var fs = afero.NewMemMapFs()

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir, fs)

	checkErr(err, t)

	bundleCache := &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        func(filePath string) error { return nil },
		Fs:           fs,
	}

	templateData := mockTemplateData(writePath)

	err = commands.GenerateNewBundle(bundleCache, templateData)

	checkErr(err, t)

	wantTopLevel := []string{
		"massdriver.yaml",
		"src",
	}

	if errorString, assertion := mockfilesystem.AssertDirectoryContents(fs, writePath, wantTopLevel); assertion != true {
		t.Errorf(errorString)
	}

	wantSecondLevel := []string{"main.tf"}

	if errorString, assertion := mockfilesystem.AssertDirectoryContents(fs, path.Join(writePath, "src"), wantSecondLevel); assertion != true {
		t.Errorf(errorString)
	}
}

func TestTemplateRender(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	writePath := "."

	var fs = afero.NewMemMapFs()

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir, fs)

	checkErr(err, t)

	bundleCache := &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        func(filePath string) error { return nil },
		Fs:           fs,
	}

	templateData := mockTemplateData(writePath)

	err = bundleCache.RenderTemplate(templateData)

	checkErr(err, t)

	renderedTemplate, err := afero.ReadFile(fs, "massdriver.yaml")
	fmt.Println(string(renderedTemplate))

	checkErr(err, t)

	got := make(map[string]interface{})

	err = yaml.Unmarshal(renderedTemplate, got)

	checkErr(err, t)

	wantConnections := map[string]interface{}{
		"properties": map[string]interface{}{
			"aws_authentication": map[string]interface{}{
				"$ref": "massdriver/aws-iam-role",
			},
			"dynamo": map[string]interface{}{
				"$ref": "massdriver/aws-dynamodb-table",
			},
		},
		"required": []interface{}{"aws_authentication", "dynamo"},
	}

	if got["name"] != templateData.Name {
		t.Errorf("Expected rendered template's name field to be %s but got %s", templateData.Name, got["name"])
	}

	if !reflect.DeepEqual(got["connections"], wantConnections) {
		t.Errorf("Expected rendered template's connections field to be %v but got %v", wantConnections, got["connections"])
	}
}

func mockTemplateData(writePath string) *templatecache.TemplateData {
	return &templatecache.TemplateData{
		OutputDir:      writePath,
		Type:           "infrastructure",
		TemplateName:   "terraform",
		TemplateRepo:   "massdriver-cloud/infrastructure-templates",
		TemplateSource: "/home/md-cloud",
		Name:           "aws-dynamodb",
		Access:         "private",
		Description:    "whatever",
		Connections: []templatecache.Connection{
			{ArtifactDefinition: "massdriver/aws-dynamodb-table", Name: "dynamo"},
		},
		CloudAbbreviation: "aws",
		RepoName:          "massdriver-cloud/bundle-templates",
		RepoNameEncoded:   "massdriver-cloud/bundle-templates",
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}
