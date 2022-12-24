package templatecache_test

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/massdriver-cloud/mass/internal/templatecache"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type fileToWrite struct {
	path    string
	content []byte
}

func TestBundleTemplateRefresh(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	err := bundleCache.RefreshTemplates()

	if err != nil {
		t.Error(err)
	}

	got, _ := afero.Glob(fs, fmt.Sprintf("%s/**/**/*", rootTemplateDir))

	want := []string{
		"/home/md-cloud/massdriver-cloud/application-templates/aws-lambda",
		"/home/md-cloud/massdriver-cloud/application-templates/aws-vm",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestListTemplates(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	directories := []string{
		rootTemplateDir,
		fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform", rootTemplateDir),
		fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi", rootTemplateDir),
	}

	err := makeTemplateDirectories(directories, fs)

	if err != nil {
		t.Fatal(err)
	}

	files := []fileToWrite{
		{path: fmt.Sprintf("%s/massdriver-cloud/application-templates/kubernetes-cronjob/massdriver.yaml", rootTemplateDir)},
		{path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/terraform/massdriver.yaml", rootTemplateDir)},
		{path: fmt.Sprintf("%s/massdriver-cloud/infrastructure-templates/palumi/massdriver.yaml", rootTemplateDir)},
	}

	err = makeFiles(files, fs)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	got, _ := bundleCache.ListTemplates()

	want := []templatecache.TemplateList{
		{
			Repository: "massdriver-cloud/application-templates",
			Templates:  []string{"kubernetes-cronjob"},
		},
		{
			Repository: "massdriver-cloud/infrastructure-templates",
			Templates:  []string{"palumi", "terraform"},
		},
	}

	sort.Slice(got, func(i int, j int) bool { return got[i].Repository < got[j].Repository })

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestTemplatePath(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	var fs = afero.NewMemMapFs()

	bundleCache := newMockClient(rootTemplateDir, fs, t)

	got, _ := bundleCache.GetTemplatePath()

	if got != rootTemplateDir {
		t.Errorf("got %v, wanted %v", got, rootTemplateDir)
	}
}

func TestCopyFilesFromTemplate(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	repoPath := "massdriver-cloud/infrastructure-templates"
	templatePath := "terraform"
	srcPath := "src"
	writePath := "."

	directories := []string{
		path.Join(rootTemplateDir, repoPath),
		path.Join(rootTemplateDir, repoPath, templatePath),
		path.Join(rootTemplateDir, repoPath, templatePath, srcPath),
	}

	files := []fileToWrite{
		{path: fmt.Sprintf("%s/massdriver.yaml", path.Join(rootTemplateDir, repoPath, templatePath))},
		{path: fmt.Sprintf("%s/main.tf", path.Join(rootTemplateDir, repoPath, templatePath, srcPath))},
	}

	var fs = afero.NewMemMapFs()

	err := setupMockFileSystem(directories, files, fs)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        func(filePath string) error { return nil },
		Fs:           fs,
	}

	templateData := &templatecache.TemplateData{
		OutputDir:      writePath,
		Type:           "infrastructure",
		TemplateName:   "terraform",
		TemplateRepo:   "massdriver-cloud/infrastructure-templates",
		TemplateSource: "/home/md-cloud",
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

	err = bundleCache.RenderTemplate(writePath, templateData)

	if err != nil {
		t.Fatal(err)
	}

	wantTopLevel := []string{
		"home",
		"massdriver.yaml",
		"src",
	}

	if errorString, assertion := assertDirectoryContents(fs, writePath, wantTopLevel); assertion != true {
		t.Errorf(errorString)
	}

	wantSecondLevel := []string{"main.tf"}

	if errorString, assertion := assertDirectoryContents(fs, path.Join(writePath, "src"), wantSecondLevel); assertion != true {
		t.Errorf(errorString)
	}
}

func TestTemplateRender(t *testing.T) {
	rootTemplateDir := "/home/md-cloud"
	repoPath := "massdriver-cloud/infrastructure-templates"
	templatePath := "terraform"
	srcPath := "src"
	writePath := "."

	directories := []string{
		path.Join(rootTemplateDir, repoPath),
		path.Join(rootTemplateDir, repoPath, templatePath),
		path.Join(rootTemplateDir, repoPath, templatePath, srcPath),
	}

	massdriverYamlTemplate, err := os.ReadFile("./testdata/massdriver.yaml")

	if err != nil {
		t.Fatal(err)
	}

	files := []fileToWrite{
		{
			path:    fmt.Sprintf("%s/massdriver.yaml", path.Join(rootTemplateDir, repoPath, templatePath)),
			content: massdriverYamlTemplate,
		},
		{
			path: fmt.Sprintf("%s/main.tf", path.Join(rootTemplateDir, repoPath, templatePath, srcPath)),
		},
	}

	var fs = afero.NewMemMapFs()

	err = setupMockFileSystem(directories, files, fs)

	if err != nil {
		t.Fatal(err)
	}

	bundleCache := &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        func(filePath string) error { return nil },
		Fs:           fs,
	}

	templateData := &templatecache.TemplateData{
		OutputDir:      writePath,
		Type:           "infrastructure",
		TemplateName:   "terraform",
		TemplateRepo:   "massdriver-cloud/infrastructure-templates",
		TemplateSource: "/home/md-cloud",
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

	err = bundleCache.RenderTemplate(writePath, templateData)

	if err != nil {
		t.Fatal(err)
	}

	renderedTemplate, err := afero.ReadFile(fs, "massdriver.yaml")

	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]interface{})

	err = yaml.Unmarshal(renderedTemplate, got)

	if err != nil {
		t.Fatal(err)
	}

	if got["name"] != templateData.Name {
		t.Errorf("Expected rendered template's name field to be %s but got %s", templateData.Name, got["name"])
	}
}

func assertDirectoryContents(fs afero.Fs, path string, want []string) (string, bool) {
	filesInDirectory, _ := afero.ReadDir(fs, path)
	got := []string{}
	for _, file := range filesInDirectory {
		got = append(got, file.Name())
	}

	return fmt.Sprintf("Wanted %v but got %v", want, got), reflect.DeepEqual(got, want)
}

func newMockClient(rootTemplateDir string, fs afero.Fs, t *testing.T) templatecache.TemplateCache {
	fetcher := func(filePath string) error {
		directories := []string{
			filePath,
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-lambda", filePath),
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-vm", filePath),
		}

		err := makeTemplateDirectories(directories, fs)

		if err != nil {
			t.Fatal(err)
		}

		return nil
	}

	return &templatecache.BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        fetcher,
		Fs:           fs,
	}
}

func setupMockFileSystem(directories []string, files []fileToWrite, fs afero.Fs) error {
	err := makeTemplateDirectories(directories, fs)

	if err != nil {
		return err
	}

	err = makeFiles(files, fs)

	if err != nil {
		return err
	}

	return nil
}

func makeFiles(files []fileToWrite, fs afero.Fs) error {
	for _, file := range files {
		err := afero.WriteFile(fs, file.path, file.content, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeTemplateDirectories(names []string, fs afero.Fs) error {
	for _, name := range names {
		err := fs.Mkdir(name, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}
