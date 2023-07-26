package mockfilesystem

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/spf13/afero"
)

type VirtualFile struct {
	Path    string
	Content []byte
}

/*
Sets up a mock bundle in the location specified by rootTemplateDir.
Includes a parsable massdriver.yaml template, and an empty src/main.tf
*/
func SetupBundleTemplate(rootTemplateDir string, fs afero.Fs) error {
	repoPath := "massdriver-cloud/infrastructure-templates"
	templatePath := "terraform"
	srcPath := "src"

	directories := []string{
		path.Join(rootTemplateDir, repoPath),
		path.Join(rootTemplateDir, repoPath, templatePath),
		path.Join(rootTemplateDir, repoPath, templatePath, srcPath),
	}

	fixturePath := path.Join(projectRoot(), "/internal/mockfilesystem/testdata/massdriver.yaml.txt")
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

	err = MakeDirectories(directories, fs)

	if err != nil {
		return err
	}

	err = MakeFiles(files, fs)

	if err != nil {
		return err
	}

	return nil
}

func SetupBundle(rootDir string, fs afero.Fs) error {
	srcPath := "src"
	deployPath := "deploy"

	directories := []string{
		rootDir,
		path.Join(rootDir, srcPath),
		path.Join(rootDir, deployPath),
	}

	fixturePath := path.Join(projectRoot(), "/internal/mockfilesystem/testdata/massdriver.yaml")

	massdriverYamlFile, err := os.ReadFile(fixturePath)

	if err != nil {
		return err
	}

	mainTFPath := path.Join(projectRoot(), "/internal/mockfilesystem/testdata/main.tf")
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

	err = MakeDirectories(directories, fs)

	if err != nil {
		return err
	}

	err = MakeFiles(files, fs)

	if err != nil {
		return err
	}

	return nil
}

func WithOperatorGuide(rootDir string, guideType string, fs afero.Fs) error {
	return createDocument(rootDir, "operator", guideType, fs)
}

func WithRunbook(rootDir string, guideType string, fs afero.Fs) error {
	return createDocument(rootDir, "runbook", guideType, fs)
}

func createDocument(rootDir string, name string, guideType string, fs afero.Fs) error {
	documentFilePath := fmt.Sprintf("%s/internal/mockfilesystem/testdata/%s.md", projectRoot(), name)
	documentMdFile, err := os.ReadFile(documentFilePath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    fmt.Sprintf("%s/%s.%s", rootDir, name, guideType),
			Content: documentMdFile,
		},
	}

	err = MakeFiles(files, fs)

	if err != nil {
		return err
	}

	return nil
}

func WithFilesToIgnore(rootDir string, fs afero.Fs) error {
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

	err := MakeDirectories(directories, fs)

	if err != nil {
		return err
	}

	err = MakeFiles(files, fs)

	if err != nil {
		return err
	}

	return nil
}

func MakeFiles(files []VirtualFile, fs afero.Fs) error {
	for _, file := range files {
		err := afero.WriteFile(fs, file.Path, file.Content, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func MakeDirectories(names []string, fs afero.Fs) error {
	for _, name := range names {
		err := fs.MkdirAll(name, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertDirectoryContents(fs afero.Fs, path string, want []string) (string, bool) {
	filesInDirectory, _ := afero.ReadDir(fs, path)
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
