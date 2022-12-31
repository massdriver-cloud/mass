package templatecache

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"text/template"

	"github.com/spf13/afero"
)

type fileManager struct {
	fs                    afero.Fs
	readDirectory         string
	writeDirectory        string
	templateRootDirectory string
	templateData          *TemplateData
}

/*
Copies a bundle template in to the desired directory and writes templated values.
*/
func (f *fileManager) CopyTemplate() error {
	return afero.Walk(f.fs, f.readDirectory, f.mkDirOrWriteFile)
}

func (f *fileManager) mkDirOrWriteFile(filePath string, info fs.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}

	relativeWritePath := relativeWritePath(filePath, f.readDirectory)
	outputPath := path.Join(f.writeDirectory, relativeWritePath)

	if info.IsDir() {
		if isBundleRootDirectory(relativeWritePath) {
			return makeWriteDirectoryAndParents(f.writeDirectory, f.fs)
		}

		return f.fs.Mkdir(outputPath, 0755)
	}

	file, err := afero.ReadFile(f.fs, filePath)

	if err != nil {
		return err
	}

	return f.promptAndWrite(file, outputPath)
}

func (f *fileManager) promptAndWrite(file []byte, outputPath string) error {
	tmpl, errTmpl := template.New("tmpl").Delims("<md", "md>").Parse(string(file))

	if errTmpl != nil {
		return errTmpl
	}

	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("%s exists. Overwrite? (y|N): ", outputPath)
		var response string
		fmt.Scanln(&response)

		if response == "y" || response == "Y" || response == "yes" {
			return f.writeToFile(outputPath, tmpl)
		}
	}

	return f.writeToFile(outputPath, tmpl)
}

func (f *fileManager) writeToFile(outputPath string, tmpl *template.Template) error {
	outputFile, err := f.fs.Create(outputPath)

	if err != nil {
		return err
	}

	defer outputFile.Close()
	return tmpl.Execute(outputFile, f.templateData)
}

func relativeWritePath(currentFilePath, readDirectory string) string {
	path := strings.Replace(currentFilePath, readDirectory, "", 1)
	if path == "" {
		path = "."
	}

	return path
}

func makeWriteDirectoryAndParents(writeDirectory string, fs afero.Fs) error {
	if _, err := fs.Stat(writeDirectory); err != nil {
		return fs.MkdirAll(writeDirectory, 0755)
	}

	return nil
}

func isBundleRootDirectory(realtiveWritePath string) bool {
	return realtiveWritePath == "."
}
