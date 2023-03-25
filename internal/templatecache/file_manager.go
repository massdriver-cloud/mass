package templatecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/massdriver-cloud/liquid"
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
		if _, checkDirExistsErr := f.fs.Stat(outputPath); errors.Is(checkDirExistsErr, os.ErrNotExist) {
			if isBundleRootDirectory(relativeWritePath) {
				return makeWriteDirectoryAndParents(f.writeDirectory, f.fs)
			}

			return f.fs.Mkdir(outputPath, 0755)
		}

		return nil
	}

	readBytes, readErr := afero.ReadFile(f.fs, filePath)
	if readErr != nil {
		return readErr
	}

	// only templatize files in the bundle root directory (to not conflict w/ helm templates)
	var outBytes []byte
	if isInsideBundleRootDirectory(relativeWritePath) {
		var renderErr error
		outBytes, renderErr = f.renderFile(readBytes)
		if renderErr != nil {
			return renderErr
		}
	} else {
		outBytes = readBytes
	}

	return f.promptAndWrite(outBytes, outputPath)
}

func (f *fileManager) promptAndWrite(template []byte, outputPath string) error {
	if _, err := f.fs.Stat(outputPath); err == nil {
		fmt.Printf("%s exists. Overwrite? (y|N): ", outputPath)
		var response string
		fmt.Scanln(&response)

		if !(response == "y" || response == "Y" || response == "yes") {
			fmt.Println("keeping existing file")
			return nil
		}

		return f.writeToFile(outputPath, template)
	}

	return f.writeToFile(outputPath, template)
}

func (f *fileManager) renderFile(template []byte) ([]byte, error) {
	engine := liquid.NewEngine()

	var bindings map[string]interface{}
	inrec, _ := json.Marshal(f.templateData)

	err := json.Unmarshal(inrec, &bindings)

	if err != nil {
		return nil, err
	}

	return engine.ParseAndRender(template, bindings)
}

func (f *fileManager) writeToFile(outputPath string, outBytes []byte) error {
	return afero.WriteFile(f.fs, outputPath, outBytes, 0600)
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

func isBundleRootDirectory(relativeWritePath string) bool {
	return relativeWritePath == "."
}

func isInsideBundleRootDirectory(relativeWritePath string) bool {
	return filepath.Dir(relativeWritePath) == "/"
}
