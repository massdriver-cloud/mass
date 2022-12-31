package mockfilesystem

import (
	"fmt"
	"os"
	"path"
	"reflect"

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

	massdriverYamlTemplate, err := os.ReadFile("../mockfilesystem/testdata/massdriver.yaml.txt")

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

	directories := []string{
		rootDir,
		path.Join(rootDir, srcPath),
	}

	massdriverYamlFile, err := os.ReadFile("../mockfilesystem/testdata/massdriver.yaml")

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    fmt.Sprintf("%s/massdriver.yaml", rootDir),
			Content: massdriverYamlFile,
		},
		{
			Path: fmt.Sprintf("%s/main.tf", path.Join(rootDir, srcPath)),
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
