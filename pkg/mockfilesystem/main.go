package mockfilesystem

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
)

type VirtualFile struct {
	Path    string
	Content []byte
}

/*
Sets up a test bundle in the location specified by rootTemplateDir.
Includes a parsable massdriver.yaml template, and an empty src/main.tf
*/
func SetupBundleTemplate(rootTemplateDir string) error {
	repoPath := "massdriver-cloud/infrastructure-templates"
	templatePath := "terraform"
	srcPath := "src"

	directories := []string{
		path.Join(rootTemplateDir, repoPath),
		path.Join(rootTemplateDir, repoPath, templatePath),
		path.Join(rootTemplateDir, repoPath, templatePath, srcPath),
	}

	fixturePath := path.Join(projectRoot(), "/pkg/mockfilesystem/testdata/massdriver.yaml.txt")
	massdriverYamlTemplate, err := os.ReadFile(fixturePath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    fmt.Sprintf("%s/massdriver.yaml", path.Join(rootTemplateDir, repoPath, templatePath)),
			Content: massdriverYamlTemplate,
		},
		{
			Path: fmt.Sprintf("%s/main.tf", path.Join(rootTemplateDir, repoPath, templatePath, srcPath)),
		},
	}

	err = MakeDirectories(directories)

	if err != nil {
		return err
	}

	err = MakeFiles(files)

	if err != nil {
		return err
	}

	return nil
}

func SetupBundle(rootDir string) error {
	srcPath := "src"
	deployPath := "deploy"

	directories := []string{
		rootDir,
		path.Join(rootDir, srcPath),
		path.Join(rootDir, deployPath),
	}

	fixturePath := path.Join(projectRoot(), "/pkg/mockfilesystem/testdata/massdriver.yaml")

	massdriverYamlFile, err := os.ReadFile(fixturePath)

	if err != nil {
		return err
	}

	mainTFPath := path.Join(projectRoot(), "/pkg/mockfilesystem/testdata/main.tf")
	mainTF, err := os.ReadFile(mainTFPath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    fmt.Sprintf("%s/massdriver.yaml", rootDir),
			Content: massdriverYamlFile,
		},
		{
			Path:    fmt.Sprintf("%s/main.tf", path.Join(rootDir, srcPath)),
			Content: mainTF,
		},
		{
			Path: fmt.Sprintf("%s/main.tf", path.Join(rootDir, deployPath)),
		},
	}

	err = MakeDirectories(directories)

	if err != nil {
		return err
	}

	err = MakeFiles(files)

	if err != nil {
		return err
	}

	return nil
}

func WithOperatorGuide(rootDir string, guideType string) error {
	operatorGuideFilePath := fmt.Sprintf("%s/pkg/mockfilesystem/testdata/operator.md", projectRoot())
	operatorGuideMd, err := os.ReadFile(operatorGuideFilePath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    fmt.Sprintf("%s/operator.%s", rootDir, guideType),
			Content: operatorGuideMd,
		},
	}

	err = MakeFiles(files)

	if err != nil {
		return err
	}

	return nil
}

func WithFilesToIgnore(rootDir string) error {
	directories := []string{
		path.Join(rootDir, "shouldntexist"),
	}

	files := []VirtualFile{
		{
			Path: fmt.Sprintf("%s/shouldntexist.txt", rootDir),
		},
		{
			Path: fmt.Sprintf("%s/src/.tfstate", rootDir),
		},
	}

	err := MakeDirectories(directories)

	if err != nil {
		return err
	}

	err = MakeFiles(files)

	if err != nil {
		return err
	}

	return nil
}

func MakeFiles(files []VirtualFile) error {
	for _, file := range files {
		err := os.WriteFile(file.Path, file.Content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func MakeDirectories(names []string) error {
	for _, name := range names {
		err := os.MkdirAll(name, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertDirectoryContents(path string, want []string) (string, bool) {
	filesInDirectory, _ := os.ReadDir(path)
	got := []string{}
	for _, file := range filesInDirectory {
		got = append(got, file.Name())
	}

	return fmt.Sprintf("Wanted %v but got %v", want, got), reflect.DeepEqual(got, want)
}

func projectRoot() string {
	_, b, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(b), "../..")
}
