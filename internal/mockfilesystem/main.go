// Package mockfilesystem provides helpers for creating virtual file structures in tests.
package mockfilesystem

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
)

// VirtualFile represents a file path and its contents for use in test fixture setup.
type VirtualFile struct {
	Path    string
	Content []byte
}

// SetupBundleTemplate sets up a test bundle template in rootTemplateDir with a parsable massdriver.yaml and empty src/main.tf.
func SetupBundleTemplate(rootTemplateDir string) error {
	templatePath := "opentofu"
	srcPath := "src"

	directories := []string{
		rootTemplateDir,
		path.Join(rootTemplateDir, templatePath),
		path.Join(rootTemplateDir, templatePath, srcPath),
	}

	fixturePath := path.Join(projectRoot(), "/internal/mockfilesystem/testdata/massdriver.yaml.txt")
	massdriverYamlTemplate, err := os.ReadFile(fixturePath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    path.Join(rootTemplateDir, templatePath) + "/massdriver.yaml",
			Content: massdriverYamlTemplate,
		},
		{
			Path: path.Join(rootTemplateDir, templatePath, srcPath) + "/main.tf",
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

// SetupBundle sets up a complete test bundle directory at rootDir with massdriver.yaml and source files.
func SetupBundle(rootDir string) error {
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
			Path:    rootDir + "/massdriver.yaml",
			Content: massdriverYamlFile,
		},
		{
			Path:    path.Join(rootDir, srcPath) + "/main.tf",
			Content: mainTF,
		},
		{
			Path: path.Join(rootDir, deployPath) + "/main.tf",
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

// WithOperatorGuide adds an operator guide file of the given type to rootDir.
func WithOperatorGuide(rootDir string, guideType string) error {
	operatorGuideFilePath := projectRoot() + "/internal/mockfilesystem/testdata/operator.md"
	operatorGuideMd, err := os.ReadFile(operatorGuideFilePath)

	if err != nil {
		return err
	}

	files := []VirtualFile{
		{
			Path:    rootDir + "/operator." + guideType,
			Content: operatorGuideMd,
		},
	}

	err = MakeFiles(files)

	if err != nil {
		return err
	}

	return nil
}

// WithFilesToIgnore adds files and directories to rootDir that should be excluded during bundle operations.
func WithFilesToIgnore(rootDir string) error {
	directories := []string{
		path.Join(rootDir, "shouldntexist"),
	}

	files := []VirtualFile{
		{
			Path: rootDir + "/shouldntexist.txt",
		},
		{
			Path: rootDir + "/src/.tfstate",
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

// MakeFiles writes each VirtualFile to disk.
func MakeFiles(files []VirtualFile) error {
	for _, file := range files {
		// #nosec G306
		err := os.WriteFile(file.Path, file.Content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// MakeDirectories creates each named directory, including any missing parents.
func MakeDirectories(names []string) error {
	for _, name := range names {
		err := os.MkdirAll(name, 0750)
		if err != nil {
			return err
		}
	}

	return nil
}

// AssertDirectoryContents returns a diff message and whether the directory contents match the expected list.
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
